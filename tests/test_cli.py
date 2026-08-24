"""CLI dispatch for list, validate, apply, and deprecated install."""

from pathlib import Path

from sf2_theme.cli import dispatch


def test_apps_and_version(capsys) -> None:
    assert dispatch(["apps"]) == 0
    assert "wezterm" in capsys.readouterr().out
    assert dispatch(["--version"]) == 0
    assert capsys.readouterr().out.strip() == "1.0.0"


def test_validate_main(capsys) -> None:
    assert dispatch(["validate", "main"]) == 0
    assert "valid: main" in capsys.readouterr().out


def test_apply_wezterm_defaults_to_main(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    assert dispatch(["apply", "wezterm"]) == 0
    pointer = (tmp_path / "xdg" / "sf2-theme" / "wezterm-current.lua").read_text(encoding="utf-8")
    assert "sf2-theme: main" in pointer
    assert "Street Fighter II - Main" in pointer
    captured = capsys.readouterr().out
    assert "street-fighter-ii-main.toml" in captured


def test_install_warns_and_applies(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    herdr = tmp_path / "herdr"
    herdr.mkdir()
    assert dispatch(["install", "herdr", "--config-dir", str(herdr)]) == 0
    err = capsys.readouterr().err
    assert "deprecated" in err
    assert "sf2-theme: main" in (herdr / "config.toml").read_text(encoding="utf-8")


def test_setup_leaves_unknown_lua(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    lua_dir = tmp_path / "wezterm"
    lua_dir.mkdir()
    original = "\n".join(
        (
            'local wezterm = require("wezterm")',
            "local config = wezterm.config_builder()",
            "config.color_scheme = scheme_for_appearance(wezterm.gui.get_appearance())",
            "return config",
            "",
        )
    )
    (lua_dir / "wezterm.lua").write_text(original, encoding="utf-8")
    assert dispatch(["setup", "wezterm", "--config-dir", str(lua_dir)]) == 0
    assert (lua_dir / "wezterm.lua").read_text(encoding="utf-8") == original
    assert "WezTerm config was left unchanged" in capsys.readouterr().out
