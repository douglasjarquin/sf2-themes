"""One outcome per ownership class, and the selection a run reports."""

import json
import tomllib
from pathlib import Path

import pytest

from sf2_theme.cli import dispatch

BUILDER_LUA = "\n".join(
    (
        'local wezterm = require("wezterm")',
        "local config = wezterm.config_builder()",
        "{body}",
        "return config",
        "",
    )
)

# The file each app reads its theme setting from, and foreign content for it.
FOREIGN = {
    "wezterm": ("wezterm.lua", BUILDER_LUA.format(body='config.color_scheme = "Builtin Dark"')),
    "herdr": ("config.toml", '[theme]\nname = "dracula"\n'),
    "herdr-app": ("settings.json", json.dumps({"theme": {"dark": "graphite", "light": "paper"}})),
    "codex": ("config.toml", '[tui]\ntheme = "dracula"\n'),
    "claude": ("settings.json", json.dumps({"theme": "dark"})),
    "starship": ("starship.toml", 'palette = "mine"\n\n[palettes.mine]\nblue = "#0000ff"\n'),
    "lazygit": ("config.yml", "gui:\n  theme:\n    activeBorderColor:\n      - red\n"),
}


@pytest.fixture
def xdg(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    root = tmp_path / "xdg"
    monkeypatch.setenv("XDG_CONFIG_HOME", str(root))
    return root


def _config(tmp_path: Path, app: str, content: str) -> tuple[Path, Path]:
    config_dir = tmp_path / app
    config_dir.mkdir(exist_ok=True)
    path = config_dir / FOREIGN[app][0]
    path.write_text(content, encoding="utf-8")
    return config_dir, path


def _current(app: str, config_dir: Path, capsys) -> str:
    capsys.readouterr()
    assert dispatch(["current", app, "--config-dir", str(config_dir)]) == 0
    return capsys.readouterr().out.strip()


@pytest.mark.parametrize("app", sorted(FOREIGN))
def test_foreign_theme_refuses_apply_the_same_way_in_every_app(app: str, tmp_path: Path, xdg: Path, capsys) -> None:
    config_dir, path = _config(tmp_path, app, FOREIGN[app][1])
    original = path.read_text(encoding="utf-8")
    assert dispatch(["apply", app, "--config-dir", str(config_dir)]) == 1
    err = capsys.readouterr().err
    assert err.startswith("error: ")
    assert "already uses a theme sf2-themes does not manage; pass --adopt to replace it" in err
    assert path.read_text(encoding="utf-8") == original
    assert sorted(p.name for p in config_dir.iterdir()) == [path.name]
    assert not xdg.exists()


@pytest.mark.parametrize("app", sorted(FOREIGN))
def test_setup_with_theme_refuses_a_foreign_theme(app: str, tmp_path: Path, xdg: Path) -> None:
    config_dir, path = _config(tmp_path, app, FOREIGN[app][1])
    assert dispatch(["setup", app, "--theme", "ryu", "--config-dir", str(config_dir)]) == 1
    assert sorted(p.name for p in config_dir.iterdir()) == [path.name]


@pytest.mark.parametrize("app", sorted(FOREIGN))
def test_plain_setup_keeps_a_foreign_theme(app: str, tmp_path: Path, xdg: Path, capsys) -> None:
    config_dir, path = _config(tmp_path, app, FOREIGN[app][1])
    original = path.read_text(encoding="utf-8")
    assert dispatch(["setup", app, "--config-dir", str(config_dir)]) == 0
    assert f"setup {app}: kept your own theme (config left unchanged)" in capsys.readouterr().out
    assert path.read_text(encoding="utf-8") == original


def test_codex_adopt_replaces_only_the_theme_line(tmp_path: Path) -> None:
    config_dir, path = _config(
        tmp_path, "codex", '# mine\nmodel = "o3"\n\n[tui]\ntheme = "dracula"\nanimations = false\n'
    )
    assert dispatch(["apply", "codex", "--config-dir", str(config_dir), "--adopt"]) == 0
    assert path.read_text(encoding="utf-8") == '# mine\nmodel = "o3"\n\n[tui]\ntheme = "sf2-main"\nanimations = false\n'


def test_plain_setup_claude_installs_theme_files_and_keeps_settings(tmp_path: Path) -> None:
    config_dir, path = _config(tmp_path, "claude", json.dumps({"theme": "dark"}))
    original = path.read_bytes()
    assert dispatch(["setup", "claude", "--config-dir", str(config_dir)]) == 0
    assert path.read_bytes() == original
    assert len(list((config_dir / "themes").glob("sf2-*.json"))) == 36


def test_unknown_wezterm_shape_writes_sf2_files_and_exits_nonzero(tmp_path: Path, xdg: Path, capsys) -> None:
    config_dir, path = _config(tmp_path, "wezterm", "return {\n  font_size = 13,\n}\n")
    original = path.read_bytes()
    assert dispatch(["apply", "wezterm", "--theme", "ryu", "--config-dir", str(config_dir)]) == 1
    assert path.read_bytes() == original
    assert (config_dir / "colors" / "sf2-ryu.toml").is_file()
    assert "-- sf2-themes: sf2-ryu" in (xdg / "sf2-theme" / "wezterm-current.lua").read_text(encoding="utf-8")
    captured = capsys.readouterr()
    assert "apply wezterm: incomplete (config left unchanged)" in captured.out
    assert "restart WezTerm" not in captured.out
    assert "dofile(sf2_current)" in captured.err
    assert "Then run the command again." in captured.err


@pytest.mark.parametrize("command", ["apply", "setup"])
def test_pasted_snippet_in_a_non_builder_config_is_managed(command: str, tmp_path: Path, xdg: Path) -> None:
    pointer = xdg / "sf2-theme" / "wezterm-current.lua"
    snippet = "\n".join(
        (
            f'local sf2_current = "{pointer}"',
            "wezterm.add_to_config_reload_watch_list(sf2_current)",
            "local sf2_scheme = dofile(sf2_current)",
            "local sf2_env = {}",
            'sf2_env.TERM_THEME = "dark"',
        )
    )
    lua = f'local wezterm = require("wezterm")\n{snippet}\nreturn {{ color_scheme = sf2_scheme }}\n'
    config_dir, path = _config(tmp_path, "wezterm", lua)
    assert dispatch([command, "wezterm", "--config-dir", str(config_dir)]) == 0
    assert path.read_text(encoding="utf-8") == lua


def test_commented_out_integration_does_not_count(tmp_path: Path, xdg: Path) -> None:
    pointer = xdg / "sf2-theme" / "wezterm-current.lua"
    body = f'-- local sf2_current = "{pointer}" -- dofile(sf2_current) TERM_THEME'
    config_dir, path = _config(tmp_path, "wezterm", BUILDER_LUA.format(body=body))
    assert dispatch(["apply", "wezterm", "--config-dir", str(config_dir)]) == 0
    lua = path.read_text(encoding="utf-8")
    assert "\nlocal sf2_scheme = dofile(sf2_current)\n" in lua
    assert body in lua


def test_inline_codex_tui_refuses_with_the_manual_fix(tmp_path: Path, capsys) -> None:
    config_dir, path = _config(tmp_path, "codex", 'tui = { theme = "dracula" }\n')
    assert dispatch(["apply", "codex", "--config-dir", str(config_dir)]) == 1
    err = capsys.readouterr().err
    assert "into a [tui] table" in err
    assert "Then run the command again." in err
    assert sorted(p.name for p in config_dir.iterdir()) == ["config.toml"]


def test_starship_palette_is_a_top_level_key(tmp_path: Path, xdg: Path) -> None:
    config_dir, path = _config(tmp_path, "starship", 'add_newline = false\n\n[directory]\nstyle = "blue"\n')
    assert dispatch(["apply", "starship", "--config-dir", str(config_dir)]) == 0
    config = tomllib.loads(path.read_text(encoding="utf-8"))
    assert config["palette"] == "sf2"
    assert "palette" not in config["directory"]
    assert config["add_newline"] is False
    assert dispatch(["apply", "starship", "--theme", "ken", "--config-dir", str(config_dir)]) == 0
    assert path.read_text(encoding="utf-8").count("palette = ") == 1


def test_starship_block_appended_by_older_versions_moves_to_the_top_level(tmp_path: Path, xdg: Path, capsys) -> None:
    config_dir = tmp_path / "starship"
    assert dispatch(["apply", "starship", "--theme", "ken", "--config-dir", str(config_dir)]) == 0
    path = config_dir / "starship.toml"
    block = path.read_text(encoding="utf-8")
    path.write_text(f'[directory]\nstyle = "blue"\n\n{block}', encoding="utf-8")
    assert _current("starship", config_dir, capsys) == "sf2-ken"
    assert dispatch(["setup", "starship", "--config-dir", str(config_dir)]) == 0
    config = tomllib.loads(path.read_text(encoding="utf-8"))
    assert config["palette"] == "sf2"
    assert "palette" not in config["directory"]


def test_starship_user_palette_is_foreign_until_adopted(tmp_path: Path, xdg: Path) -> None:
    config_dir, path = _config(tmp_path, "starship", FOREIGN["starship"][1])
    assert dispatch(["apply", "starship", "--config-dir", str(config_dir), "--adopt"]) == 0
    config = tomllib.loads(path.read_text(encoding="utf-8"))
    assert config["palette"] == "sf2"
    assert config["palettes"]["mine"] == {"blue": "#0000ff"}


def test_mixed_herdr_app_pair_is_foreign(tmp_path: Path) -> None:
    pair = json.dumps({"theme": {"dark": "sf2-ryu", "light": "sf2-ken-light"}})
    config_dir, path = _config(tmp_path, "herdr-app", pair)
    assert dispatch(["apply", "herdr-app", "--config-dir", str(config_dir)]) == 1
    assert path.read_text(encoding="utf-8") == pair


def test_adopt_without_theme_selects_main(tmp_path: Path) -> None:
    config_dir, path = _config(tmp_path, "herdr", FOREIGN["herdr"][1])
    assert dispatch(["setup", "herdr", "--adopt", "--config-dir", str(config_dir)]) == 0
    text = path.read_text(encoding="utf-8")
    assert "dracula" not in text
    assert "# sf2-themes: sf2-main" in text


@pytest.mark.parametrize(
    ("app", "content"),
    [
        ("herdr", '# >>> sf2-themes managed theme\n[theme]\nname = "x"\n'),
        ("lazygit", "gui:\n  # >>> sf2-themes managed theme\n  theme:\n"),
        ("starship", "# <<< sf2-themes managed theme\n"),
    ],
)
@pytest.mark.parametrize("command", [["apply"], ["setup"]])
def test_incomplete_markers_refuse(app: str, content: str, command: list[str], tmp_path: Path, capsys) -> None:
    config_dir, path = _config(tmp_path, app, content)
    assert dispatch([*command, app, "--config-dir", str(config_dir)]) == 1
    assert "incomplete sf2-themes markers" in capsys.readouterr().err
    assert sorted(p.name for p in config_dir.iterdir()) == [path.name]


@pytest.mark.parametrize("app", ["nvim", "starship"])
def test_adopt_is_harmless_where_nothing_is_foreign(app: str, tmp_path: Path, xdg: Path, capsys) -> None:
    assert dispatch(["apply", app, "--adopt", "--config-dir", str(tmp_path / app)]) == 0
    assert f"apply {app}: sf2-main" in capsys.readouterr().out


def test_unreadable_wezterm_pointer_resets_to_main(tmp_path: Path, xdg: Path, capsys) -> None:
    pointer = xdg / "sf2-theme" / "wezterm-current.lua"
    pointer.parent.mkdir(parents=True)
    pointer.write_text('return "sf2-ryu"\n', encoding="utf-8")
    config_dir = tmp_path / "wezterm"
    assert dispatch(["setup", "wezterm", "--config-dir", str(config_dir)]) == 0
    assert "setup wezterm: selection unreadable, reset to sf2-main" in capsys.readouterr().out
    assert _current("wezterm", config_dir, capsys) == "sf2-main"


def test_plain_setup_on_foreign_wezterm_writes_a_main_pointer(tmp_path: Path, xdg: Path, capsys) -> None:
    config_dir, path = _config(tmp_path, "wezterm", FOREIGN["wezterm"][1])
    original = path.read_bytes()
    assert dispatch(["setup", "wezterm", "--config-dir", str(config_dir)]) == 0
    out = capsys.readouterr().out
    assert "setup wezterm: kept your own theme (config left unchanged)" in out
    assert "  pointer: no selection found, selected sf2-main" in out
    assert path.read_bytes() == original
    assert _current("wezterm", config_dir, capsys) == "sf2-main"


def test_plain_setup_keeps_nvim_pointer(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "nvim"
    assert dispatch(["apply", "nvim", "--theme", "ryu-light", "--config-dir", str(config_dir)]) == 0
    pointer = config_dir / "sf2-theme" / "current.lua"
    original = pointer.read_bytes()
    capsys.readouterr()
    assert dispatch(["setup", "nvim", "--config-dir", str(config_dir)]) == 0
    assert "setup nvim: kept sf2-ryu + sf2-ryu-light" in capsys.readouterr().out
    assert pointer.read_bytes() == original


def test_plain_setup_upgrades_a_legacy_codex_id(tmp_path: Path, capsys) -> None:
    config_dir, path = _config(tmp_path, "codex", '[tui]\ntheme = "ryu"\n')
    assert dispatch(["setup", "codex", "--config-dir", str(config_dir), "--verbose"]) == 0
    out = capsys.readouterr().out
    assert "setup codex: kept sf2-ryu" in out
    assert f"updated: {path}" in out
    assert path.read_text(encoding="utf-8") == '[tui]\ntheme = "sf2-ryu"\n'


def test_unreadable_claude_selection_resets_to_main(tmp_path: Path, capsys) -> None:
    config_dir, path = _config(tmp_path, "claude", json.dumps({"theme": "custom:sf2-not-a-theme"}))
    assert dispatch(["setup", "claude", "--config-dir", str(config_dir)]) == 0
    assert "setup claude: selection unreadable, reset to sf2-main" in capsys.readouterr().out
    assert json.loads(path.read_text(encoding="utf-8"))["theme"] == "custom:sf2-main"


@pytest.mark.parametrize("app", ["wezterm", "herdr", "herdr-app", "nvim", "codex", "starship", "lazygit", "claude"])
def test_kept_selection_is_what_current_prints(app: str, tmp_path: Path, xdg: Path, capsys) -> None:
    config_dir = tmp_path / app
    assert dispatch(["apply", app, "--theme", "chun-li-light", "--config-dir", str(config_dir)]) == 0
    capsys.readouterr()
    assert dispatch(["setup", app, "--config-dir", str(config_dir)]) == 0
    kept = capsys.readouterr().out.splitlines()[0].removeprefix(f"setup {app}: kept ").split(" ")[0]
    assert kept == _current(app, config_dir, capsys)
