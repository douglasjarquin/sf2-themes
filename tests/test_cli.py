"""CLI dispatch for list, validate, apply, and deprecated install."""

from pathlib import Path

from sf2_theme.cli import dispatch


def test_apps_and_version(capsys) -> None:
    assert dispatch(["apps"]) == 0
    apps = capsys.readouterr().out
    assert "wezterm" in apps
    assert "herdr" in apps
    assert "nvim" in apps
    assert "starship" in apps
    assert "lazygit" in apps
    assert dispatch(["--version"]) == 0
    assert capsys.readouterr().out.strip() == "1.0.1"


def test_validate_main(capsys) -> None:
    assert dispatch(["validate", "main"]) == 0
    assert "valid: main" in capsys.readouterr().out


def test_apply_wezterm_defaults_to_main(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    monkeypatch.setenv("WEZTERM_CONFIG_DIR", str(tmp_path / "xdg" / "wezterm"))
    assert dispatch(["apply", "wezterm"]) == 0
    pointer = (tmp_path / "xdg" / "sf2-theme" / "wezterm-current.lua").read_text(encoding="utf-8")
    assert "sf2-themes: sf2-main" in pointer
    assert 'return "sf2-main"' in pointer
    assert 'return "sf2-main-light"' in pointer
    assert "get_appearance" in pointer
    captured = capsys.readouterr().out
    assert "sf2-main.toml" in captured


def test_install_warns_and_applies(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    herdr = tmp_path / "herdr"
    herdr.mkdir()
    assert dispatch(["install", "herdr", "--config-dir", str(herdr)]) == 0
    err = capsys.readouterr().err
    assert "deprecated" in err
    text = (herdr / "config.toml").read_text(encoding="utf-8")
    assert "sf2-themes: sf2-main" in text
    assert "auto_switch = true" in text
    assert any(line.strip() == "[theme.custom]" for line in text.splitlines())
    assert not any(line.strip() == "[theme.custom.dark]" for line in text.splitlines())
    assert not any(line.strip() == "[theme.custom.light]" for line in text.splitlines())


def test_apply_herdr_light_selection_current_is_family_id(tmp_path: Path, capsys) -> None:
    herdr = tmp_path / "herdr"
    herdr.mkdir()
    assert dispatch(["apply", "herdr", "--theme", "chun-li-light", "--config-dir", str(herdr)]) == 0
    text = (herdr / "config.toml").read_text(encoding="utf-8")
    assert "auto_switch = true" in text
    assert 'light_name = "catppuccin-latte"' in text
    assert "# sf2-themes: sf2-chun-li" in text
    assert any(line.strip() == "[theme.custom]" for line in text.splitlines())
    assert not any(line.strip() == "[theme.custom.dark]" for line in text.splitlines())
    assert not any(line.strip() == "[theme.custom.light]" for line in text.splitlines())
    assert dispatch(["current", "herdr", "--config-dir", str(herdr)]) == 0
    assert capsys.readouterr().out.strip().endswith("sf2-chun-li")


def test_apply_wezterm_light_selection_writes_appearance_pair(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    assert dispatch(["apply", "wezterm", "--theme", "ryu-light", "--config-dir", str(tmp_path / "wezterm")]) == 0
    pointer = (tmp_path / "xdg" / "sf2-theme" / "wezterm-current.lua").read_text(encoding="utf-8")
    assert "-- sf2-themes: sf2-ryu" in pointer
    assert 'return "sf2-ryu"' in pointer
    assert 'return "sf2-ryu-light"' in pointer
    assert dispatch(["current", "wezterm"]) == 0
    assert capsys.readouterr().out.strip().endswith("sf2-ryu")


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
    captured = capsys.readouterr()
    assert "WezTerm config was left unchanged" in captured.err


def test_apply_lazygit_writes_all_catalog_themes_and_preserves_config(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "lazygit"
    config_dir.mkdir()
    (config_dir / "config.yml").write_text(
        "gui:\n  sidePanelWidth: 0.3\n\nnotATheme: true\n",
        encoding="utf-8",
    )

    assert dispatch(["apply", "lazygit", "--theme", "vega", "--config-dir", str(config_dir)]) == 0

    theme_files = sorted((config_dir / "themes").glob("sf2-*.yml"))
    assert len(theme_files) == 36
    selected = (config_dir / "themes" / "sf2-vega.yml").read_text(encoding="utf-8")
    for key in (
        "activeBorderColor",
        "inactiveBorderColor",
        "searchingActiveBorderColor",
        "optionsTextColor",
        "selectedLineBgColor",
        "inactiveViewSelectedLineBgColor",
        "cherryPickedCommitFgColor",
        "cherryPickedCommitBgColor",
        "markedBaseCommitFgColor",
        "markedBaseCommitBgColor",
        "unstagedChangesColor",
        "defaultFgColor",
        "authorColors",
    ):
        assert key in selected
    config = (config_dir / "config.yml").read_text(encoding="utf-8")
    assert "sidePanelWidth: 0.3" in config
    assert "notATheme: true" in config
    assert "# sf2-themes: sf2-vega" in config
    assert "sf2-vega.yml" in capsys.readouterr().out


def test_setup_lazygit_selects_light_theme_and_current_reads_it(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "lazygit"

    assert dispatch(["setup", "lazygit", "--theme", "ryu-light", "--config-dir", str(config_dir)]) == 0
    assert dispatch(["current", "lazygit", "--config-dir", str(config_dir)]) == 0
    assert capsys.readouterr().out.strip().endswith("sf2-ryu-light")
