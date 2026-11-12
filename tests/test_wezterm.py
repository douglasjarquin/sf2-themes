"""WezTerm rendering, Lua safety, and path discovery."""

import tomllib
from pathlib import Path

import pytest

from sf2_theme.adapters.wezterm import (
    current_pointer_path,
    plan_wezterm,
    render_pointer,
    render_scheme,
    wezterm_lua_path,
)
from sf2_theme.adapters.wezterm_lua import classify, integrate
from sf2_theme.catalog import get_theme, parse_catalog, theme_pair
from sf2_theme.cli import dispatch
from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import commit
from sf2_theme.parse import parse_theme
from sf2_theme.plan import Ownership


def test_ansi_order_and_distinct_brights() -> None:
    theme = get_theme("main", parse_catalog())
    rendered = render_scheme(theme)
    assert 'selection_bg = "#453714"' in rendered
    assert 'selection_fg = "#cad1de"' in rendered
    assert theme.ansi_normal.green != theme.ansi_normal.yellow
    assert theme.ansi_normal.blue.startswith("#5b")
    assert theme.ansi_bright.red != theme.ansi_normal.red
    assert 'name = "sf2-main"' in rendered
    assert '"sf2-street-fighter-2"' in rendered
    expected = ", ".join(f'"{color}"' for color in theme.ansi_normal.as_tuple())
    assert f"ansi = [{expected}]" in rendered


def test_scheme_covers_compose_tab_bar_and_indexed_slots() -> None:
    theme = get_theme("main", parse_catalog())
    rendered = render_scheme(theme)

    assert f'compose_cursor = "{theme.ui.accent_secondary}"' in rendered
    assert f'visual_bell = "{theme.ui.overlay}"' in rendered
    assert f'scrollbar_thumb = "{theme.ui.muted}"' in rendered
    assert f'16 = "{theme.semantic.orange}"' in rendered
    assert f'17 = "{theme.ui.accent}"' in rendered
    assert "[colors.tab_bar.active_tab]" in rendered
    assert f'bg_color = "{theme.ui.accent}"' in rendered


@pytest.mark.parametrize("missing_field", ["accent_secondary", "border"])
def test_incomplete_palette_fails_before_wezterm_write(tmp_path: Path, missing_field: str) -> None:
    raw = tomllib.loads(Path("themes/main.toml").read_text(encoding="utf-8"))
    raw["ui"].pop(missing_field)
    config_dir = tmp_path / "wezterm"

    with pytest.raises(ThemeError, match=rf"ui\.{missing_field}"):
        theme = parse_theme(raw, source="incomplete.toml")
        commit(
            plan_wezterm(
                theme,
                (theme,),
                config_dir=config_dir,
                slot=False,
            )
        )

    assert not config_dir.exists()


def test_light_variant_resolves_and_renders_light_metadata() -> None:
    theme = get_theme("ryu-light", parse_catalog())
    rendered = render_scheme(theme)
    assert theme.metadata.display_name == "Street Fighter II - Ryu Light"
    assert 'name = "sf2-ryu-light"' in rendered
    assert 'background = "#' in rendered


def test_pointer_auto_switches_dark_and_light_siblings() -> None:
    catalog = parse_catalog()
    dark, light = theme_pair(get_theme("ryu-light", catalog), catalog)
    pointer = render_pointer(dark, light)
    assert "-- sf2-themes: sf2-ryu" in pointer
    assert "get_appearance" in pointer
    assert 'return "sf2-ryu"' in pointer
    assert 'return "sf2-ryu-light"' in pointer


def test_safe_config_builder_gets_dofile() -> None:
    existing = "\n".join(
        (
            'local wezterm = require("wezterm")',
            "local config = wezterm.config_builder()",
            "return config",
            "",
        )
    )
    pointer = Path("/tmp/xdg/sf2-theme/wezterm-current.lua")
    assert classify(existing, pointer) is Ownership.ABSENT
    content = integrate(existing, pointer)
    assert "local sf2_scheme = dofile(sf2_current)" in content
    assert "config.color_scheme = sf2_scheme" in content
    assert "TERM_THEME" in content


def test_existing_integration_gains_term_theme_guard() -> None:
    pointer = Path("/tmp/xdg/sf2-theme/wezterm-current.lua")
    existing = "\n".join(
        (
            'local wezterm = require("wezterm")',
            "local config = wezterm.config_builder()",
            f'local sf2_current = "{pointer}"',
            "wezterm.add_to_config_reload_watch_list(sf2_current)",
            "config.color_scheme = dofile(sf2_current)",
            "return config",
            "",
        )
    )
    assert classify(existing, pointer) is Ownership.LEGACY
    content = integrate(existing, pointer)
    assert "TERM_THEME" in content
    assert "config.color_scheme = sf2_scheme" in content
    assert "config.font" not in content


def test_return_cfg_uses_discovered_builder_name() -> None:
    existing = "\n".join(
        (
            'local wezterm = require("wezterm")',
            "local cfg = wezterm.config_builder()",
            "return cfg",
            "",
        )
    )
    content = integrate(existing, Path("/tmp/sf2-theme/wezterm-current.lua"))
    assert "cfg.color_scheme = sf2_scheme" in content
    assert "config.color_scheme" not in content
    assert "TERM_THEME" in content


def test_existing_color_scheme_is_foreign() -> None:
    existing = "\n".join(
        (
            'local wezterm = require("wezterm")',
            "local config = wezterm.config_builder()",
            'config.color_scheme = "Catppuccin Mocha"',
            "return config",
            "",
        )
    )
    assert classify(existing, Path("/tmp/pointer.lua")) is Ownership.FOREIGN


def test_previous_sf2_scheme_assignment_is_upgraded() -> None:
    existing = Path("tests/fixtures/wezterm-previous-sf2.lua").read_text(encoding="utf-8")
    pointer = Path("/Users/douglasjarquin/.config/sf2-theme/wezterm-current.lua")
    assert classify(existing, pointer) is Ownership.LEGACY
    content = integrate(existing, pointer)
    assert 'config.color_scheme = "street-fighter-2"' not in content
    assert "config.color_scheme = sf2_scheme" in content
    assert "TERM_THEME" in content
    assert "wezterm.add_to_config_reload_watch_list(sf2_current)" in content
    assert 'config.font = wezterm.font("Monaspace Neon")' in content
    assert "scheme_for_appearance" in content


def test_integrate_replaces_an_adopted_color_scheme() -> None:
    existing = "\n".join(
        (
            'local wezterm = require("wezterm")',
            "local config = wezterm.config_builder()",
            'config.color_scheme = "Catppuccin Mocha"',
            "return config",
            "",
        )
    )
    content = integrate(existing, Path("/tmp/pointer.lua"))
    assert "Catppuccin Mocha" not in content
    assert "config.color_scheme = sf2_scheme" in content
    assert "TERM_THEME" in content


def test_apply_does_not_touch_lua(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    lua_dir = tmp_path / "wezterm"
    lua_dir.mkdir()
    lua = lua_dir / "wezterm.lua"
    original = 'return { color_scheme = "other" }\n'
    lua.write_text(original, encoding="utf-8")
    catalog = parse_catalog()
    commit(
        plan_wezterm(
            get_theme("main", catalog),
            catalog,
            config_dir=lua_dir,
            slot=False,
        )
    )
    assert lua.read_text(encoding="utf-8") == original
    assert (lua_dir / "colors" / "sf2-main.toml").is_file()
    assert (tmp_path / "xdg" / "sf2-theme" / "wezterm-current.lua").is_file()


def test_wezterm_generated_ids_are_prefixed_and_old_files_are_replaced(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    config_dir = tmp_path / "wezterm"
    colors = config_dir / "colors"
    colors.mkdir(parents=True)
    (colors / "street-fighter-ii-main.toml").write_text("old managed scheme", encoding="utf-8")
    catalog = parse_catalog()

    commit(
        plan_wezterm(
            get_theme("main", catalog),
            catalog,
            config_dir=config_dir,
            slot=False,
        )
    )

    written_ids = {path.stem for path in colors.glob("*.toml")}
    expected_ids = {f"sf2-{theme.metadata.id}" for theme in catalog}
    assert written_ids == expected_ids
    assert all(theme_id.startswith("sf2-") for theme_id in written_ids)
    assert "-- sf2-themes: sf2-main" in (tmp_path / "xdg" / "sf2-theme" / "wezterm-current.lua").read_text(
        encoding="utf-8"
    )
    assert not (colors / "street-fighter-ii-main.toml").exists()


def test_setup_writes_starter_when_missing(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    monkeypatch.delenv("WEZTERM_CONFIG_DIR", raising=False)
    monkeypatch.delenv("WEZTERM_CONFIG_FILE", raising=False)
    catalog = parse_catalog()
    results = commit(plan_wezterm(get_theme("main", catalog), catalog, config_dir=None, slot=True))
    written = (tmp_path / "xdg" / "wezterm" / "wezterm.lua").read_text(encoding="utf-8")
    assert "dofile(sf2_current)" in written
    assert "TERM_THEME" in written
    assert any(result.path.name == "sf2-main.toml" for result in results)


def test_setup_without_theme_keeps_existing_pointer(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    catalog = parse_catalog()
    commit(
        plan_wezterm(
            get_theme("ken", catalog),
            catalog,
            config_dir=tmp_path / "wezterm",
            slot=False,
        )
    )
    assert "sf2-ken" in current_pointer_path().read_text(encoding="utf-8")


def test_setup_wezterm_migrates_existing_unprefixed_pointer(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    monkeypatch.delenv("WEZTERM_CONFIG_DIR", raising=False)
    pointer = current_pointer_path()
    pointer.parent.mkdir(parents=True)
    pointer.write_text('-- sf2-themes: ken\nreturn "Street Fighter II - Ken"\n', encoding="utf-8")
    setup = ["setup", "wezterm", "--config-dir", str(tmp_path / "wezterm")]

    assert dispatch(setup) == 0

    assert "-- sf2-themes: sf2-ken" in pointer.read_text(encoding="utf-8")
    assert 'return "sf2-ken"' in pointer.read_text(encoding="utf-8")
    assert 'return "sf2-ken-light"' in pointer.read_text(encoding="utf-8")
    assert "get_appearance" in pointer.read_text(encoding="utf-8")
    assert dispatch(setup) == 0
    assert "sf2-ken" in current_pointer_path().read_text(encoding="utf-8")


def test_wezterm_config_file_env(tmp_path: Path, monkeypatch) -> None:
    lua = tmp_path / "custom.lua"
    lua.write_text("return {}\n", encoding="utf-8")
    monkeypatch.setenv("WEZTERM_CONFIG_FILE", str(lua))
    monkeypatch.delenv("XDG_CONFIG_HOME", raising=False)
    assert wezterm_lua_path(None) == lua
