"""WezTerm rendering, Lua safety, and path discovery."""

import tomllib
from pathlib import Path

import pytest

from sf2_theme.adapters.wezterm import (
    apply_wezterm,
    current_pointer_path,
    render_scheme,
    setup_wezterm,
    wezterm_lua_path,
)
from sf2_theme.adapters.wezterm_lua import setup_lua
from sf2_theme.catalog import get_theme, parse_catalog
from sf2_theme.errors import ThemeError
from sf2_theme.parse import parse_theme


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
        apply_wezterm(
            theme,
            (theme,),
            config_dir=config_dir,
            dry_run=False,
            follow_symlinks=False,
        )

    assert not config_dir.exists()


def test_light_variant_resolves_and_renders_light_metadata() -> None:
    theme = get_theme("ryu-light", parse_catalog())
    rendered = render_scheme(theme)
    assert theme.metadata.display_name == "Street Fighter II - Ryu Light"
    assert 'name = "sf2-ryu-light"' in rendered
    assert 'background = "#' in rendered


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
    result = setup_lua(existing, pointer)
    assert result.mutated is True
    assert "local sf2_scheme = dofile(sf2_current)" in result.content
    assert "config.color_scheme = sf2_scheme" in result.content
    assert "TERM_THEME" in result.content
    assert result.snippet is None


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
    result = setup_lua(existing, pointer)
    assert result.mutated is True
    assert "TERM_THEME" in result.content
    assert "config.color_scheme = sf2_scheme" in result.content
    assert 'config.font' not in result.content


def test_return_cfg_uses_discovered_builder_name() -> None:
    existing = "\n".join(
        (
            'local wezterm = require("wezterm")',
            "local cfg = wezterm.config_builder()",
            "return cfg",
            "",
        )
    )
    result = setup_lua(existing, Path("/tmp/sf2-theme/wezterm-current.lua"))
    assert result.mutated is True
    assert "cfg.color_scheme = sf2_scheme" in result.content
    assert "config.color_scheme" not in result.content
    assert "TERM_THEME" in result.content


def test_existing_color_scheme_is_not_stolen() -> None:
    existing = "\n".join(
        (
            'local wezterm = require("wezterm")',
            "local config = wezterm.config_builder()",
            'config.color_scheme = "Catppuccin Mocha"',
            "return config",
            "",
        )
    )
    result = setup_lua(existing, Path("/tmp/pointer.lua"))
    assert result.mutated is False
    assert 'config.color_scheme = "Catppuccin Mocha"' in result.content


def test_previous_sf2_scheme_assignment_is_upgraded() -> None:
    existing = Path("tests/fixtures/wezterm-previous-sf2.lua").read_text(encoding="utf-8")
    pointer = Path("/Users/douglasjarquin/.config/sf2-theme/wezterm-current.lua")
    result = setup_lua(existing, pointer)
    assert result.mutated is True
    assert result.snippet is None
    assert 'config.color_scheme = "street-fighter-2"' not in result.content
    assert "config.color_scheme = sf2_scheme" in result.content
    assert "TERM_THEME" in result.content
    assert "wezterm.add_to_config_reload_watch_list(sf2_current)" in result.content
    assert 'config.font = wezterm.font("Monaspace Neon")' in result.content
    assert "scheme_for_appearance" in result.content


def test_adopt_replaces_foreign_color_scheme() -> None:
    existing = "\n".join(
        (
            'local wezterm = require("wezterm")',
            "local config = wezterm.config_builder()",
            'config.color_scheme = "Catppuccin Mocha"',
            "return config",
            "",
        )
    )
    result = setup_lua(existing, Path("/tmp/pointer.lua"), adopt=True)
    assert result.mutated is True
    assert "Catppuccin Mocha" not in result.content
    assert "config.color_scheme = sf2_scheme" in result.content
    assert "TERM_THEME" in result.content


def test_apply_does_not_touch_lua(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    lua_dir = tmp_path / "wezterm"
    lua_dir.mkdir()
    lua = lua_dir / "wezterm.lua"
    original = 'return { color_scheme = "other" }\n'
    lua.write_text(original, encoding="utf-8")
    catalog = parse_catalog()
    apply_wezterm(
        get_theme("main", catalog),
        catalog,
        config_dir=lua_dir,
        dry_run=False,
        follow_symlinks=False,
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

    apply_wezterm(
        get_theme("main", catalog),
        catalog,
        config_dir=config_dir,
        dry_run=False,
        follow_symlinks=False,
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
    results, lua = setup_wezterm(
        get_theme("main", catalog),
        catalog,
        config_dir=None,
        dry_run=False,
        follow_symlinks=False,
    )
    assert lua.mutated is True
    written = (tmp_path / "xdg" / "wezterm" / "wezterm.lua").read_text(encoding="utf-8")
    assert "dofile(sf2_current)" in written
    assert "TERM_THEME" in written
    assert any(result.path.name == "sf2-main.toml" for result in results)


def test_setup_without_theme_keeps_existing_pointer(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    catalog = parse_catalog()
    apply_wezterm(
        get_theme("ken", catalog),
        catalog,
        config_dir=tmp_path / "wezterm",
        dry_run=False,
        follow_symlinks=False,
    )
    assert "sf2-ken" in current_pointer_path().read_text(encoding="utf-8")


def test_setup_wezterm_migrates_existing_unprefixed_pointer(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    monkeypatch.delenv("WEZTERM_CONFIG_DIR", raising=False)
    pointer = current_pointer_path()
    pointer.parent.mkdir(parents=True)
    pointer.write_text('-- sf2-themes: ken\nreturn "Street Fighter II - Ken"\n', encoding="utf-8")
    catalog = parse_catalog()

    setup_wezterm(
        get_theme("main", catalog),
        catalog,
        config_dir=tmp_path / "wezterm",
        dry_run=False,
        follow_symlinks=False,
        replace_pointer=False,
    )

    assert "-- sf2-themes: sf2-ken" in pointer.read_text(encoding="utf-8")
    assert 'return "sf2-ken"' in pointer.read_text(encoding="utf-8")
    setup_wezterm(
        get_theme("main", catalog),
        catalog,
        config_dir=tmp_path / "wezterm",
        dry_run=False,
        follow_symlinks=False,
        replace_pointer=False,
    )
    assert "sf2-ken" in current_pointer_path().read_text(encoding="utf-8")


def test_wezterm_config_file_env(tmp_path: Path, monkeypatch) -> None:
    lua = tmp_path / "custom.lua"
    lua.write_text("return {}\n", encoding="utf-8")
    monkeypatch.setenv("WEZTERM_CONFIG_FILE", str(lua))
    monkeypatch.delenv("XDG_CONFIG_HOME", raising=False)
    assert wezterm_lua_path(None) == lua
