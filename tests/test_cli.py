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
    monkeypatch.delenv("WEZTERM_CONFIG_FILE", raising=False)
    assert dispatch(["apply", "wezterm"]) == 0
    pointer = (tmp_path / "xdg" / "sf2-theme" / "wezterm-current.lua").read_text(encoding="utf-8")
    assert "sf2-themes: sf2-main" in pointer
    assert 'return "sf2-main"' in pointer
    assert 'return "sf2-main-light"' in pointer
    assert "get_appearance" in pointer
    lua = (tmp_path / "xdg" / "wezterm" / "wezterm.lua").read_text(encoding="utf-8")
    assert "wezterm.config_builder()" in lua
    assert "dofile(sf2_current)" in lua
    captured = capsys.readouterr().out
    assert "apply wezterm: sf2-main + sf2-main-light (follows host appearance)" in captured
    assert "created" in captured


def test_apply_wezterm_leaves_unknown_lua_incomplete(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    lua_dir = tmp_path / "wezterm"
    lua_dir.mkdir()
    original = 'return {\n  color_scheme = "Builtin Dark",\n  font_size = 13,\n}\n'
    (lua_dir / "wezterm.lua").write_text(original, encoding="utf-8")
    assert dispatch(["apply", "wezterm", "--config-dir", str(lua_dir)]) == 1
    assert (lua_dir / "wezterm.lua").read_text(encoding="utf-8") == original
    assert (tmp_path / "xdg" / "sf2-theme" / "wezterm-current.lua").is_file()
    captured = capsys.readouterr()
    assert "apply wezterm: incomplete" in captured.out
    assert "sf2-" not in captured.out
    assert "restart WezTerm" not in captured.out
    assert "apply incomplete" in captured.err
    assert "dofile" in captured.err
    assert "--adopt" not in captured.err


def test_apply_wezterm_foreign_scheme_hints_adopt(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    lua_dir = tmp_path / "wezterm"
    lua_dir.mkdir()
    original = "\n".join(
        (
            'local wezterm = require("wezterm")',
            "local config = wezterm.config_builder()",
            'config.color_scheme = "Builtin Dark"',
            "return config",
            "",
        )
    )
    (lua_dir / "wezterm.lua").write_text(original, encoding="utf-8")
    assert dispatch(["apply", "wezterm", "--config-dir", str(lua_dir)]) == 1
    assert (lua_dir / "wezterm.lua").read_text(encoding="utf-8") == original
    captured = capsys.readouterr()
    assert "apply wezterm: incomplete" in captured.out
    assert "--adopt" in captured.err


def test_apply_wezterm_adopt_replaces_foreign_scheme(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    lua_dir = tmp_path / "wezterm"
    lua_dir.mkdir()
    (lua_dir / "wezterm.lua").write_text(
        "\n".join(
            (
                'local wezterm = require("wezterm")',
                "local config = wezterm.config_builder()",
                'config.color_scheme = "Builtin Dark"',
                "config.font_size = 13",
                "return config",
                "",
            )
        ),
        encoding="utf-8",
    )
    assert dispatch(["apply", "wezterm", "--config-dir", str(lua_dir), "--adopt"]) == 0
    lua = (lua_dir / "wezterm.lua").read_text(encoding="utf-8")
    assert "Builtin Dark" not in lua
    assert "dofile(sf2_current)" in lua
    assert "config.font_size = 13" in lua


def test_apply_wezterm_repeat_reports_no_changes(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    monkeypatch.setenv("WEZTERM_CONFIG_DIR", str(tmp_path / "xdg" / "wezterm"))
    monkeypatch.delenv("WEZTERM_CONFIG_FILE", raising=False)
    assert dispatch(["apply", "wezterm"]) == 0
    assert dispatch(["apply", "wezterm"]) == 0
    assert "no changes" in capsys.readouterr().out
    assert not list(tmp_path.rglob("*.bak.*"))


def test_apply_nvim_writes_loader_without_setup(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "nvim"
    assert dispatch(["apply", "nvim", "--theme", "ryu", "--config-dir", str(config_dir)]) == 0
    loader = config_dir / "plugin" / "sf2-theme.lua"
    assert loader.is_file()
    assert "sf2-theme/current.lua" in loader.read_text(encoding="utf-8")
    assert "apply nvim: sf2-ryu + sf2-ryu-light" in capsys.readouterr().out


def test_apply_dry_run_prints_plan_and_writes_nothing(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "nvim"
    assert dispatch(["apply", "nvim", "--config-dir", str(config_dir), "--dry-run"]) == 0
    assert not config_dir.exists()
    captured = capsys.readouterr().out
    assert "would create" in captured


def test_adopt_rejected_for_apps_without_it(tmp_path: Path, capsys) -> None:
    assert dispatch(["apply", "nvim", "--adopt", "--config-dir", str(tmp_path)]) != 0
    assert "--adopt" in capsys.readouterr().err


def test_apply_help_lists_command_options(capsys) -> None:
    assert dispatch(["apply", "--help"]) == 0
    out = capsys.readouterr().out
    for flag in ("--theme", "--config-dir", "--dry-run", "--follow-symlinks", "--adopt", "--verbose"):
        assert flag in out


def test_verbose_lists_every_file(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "codex"
    assert dispatch(["apply", "codex", "--config-dir", str(config_dir), "--verbose"]) == 0
    assert "sf2-main.tmTheme" in capsys.readouterr().out


def test_sf2_entry_point_declared() -> None:
    import tomllib

    scripts = tomllib.loads(Path("pyproject.toml").read_text(encoding="utf-8"))["project"]["scripts"]
    assert scripts["sf2"] == "sf2_theme.cli:main"
    assert scripts["sf2-themes"] == "sf2_theme.cli:main"


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
    assert any(line.strip() == "[theme.custom.dark]" for line in text.splitlines())
    assert any(line.strip() == "[theme.custom.light]" for line in text.splitlines())


def test_apply_herdr_light_selection_current_is_family_id(tmp_path: Path, capsys) -> None:
    herdr = tmp_path / "herdr"
    herdr.mkdir()
    assert dispatch(["apply", "herdr", "--theme", "chun-li-light", "--config-dir", str(herdr)]) == 0
    text = (herdr / "config.toml").read_text(encoding="utf-8")
    assert "auto_switch = true" in text
    assert 'light_name = "catppuccin-latte"' in text
    assert "# sf2-themes: sf2-chun-li" in text
    assert any(line.strip() == "[theme.custom.dark]" for line in text.splitlines())
    assert any(line.strip() == "[theme.custom.light]" for line in text.splitlines())
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

    assert dispatch(["apply", "lazygit", "--theme", "vega", "--config-dir", str(config_dir), "--verbose"]) == 0

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


def test_setup_keeps_existing_selection_for_block_adapters(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    for app in ("herdr", "starship", "lazygit"):
        config_dir = tmp_path / app
        assert dispatch(["apply", app, "--theme", "chun-li", "--config-dir", str(config_dir)]) == 0
        capsys.readouterr()
        assert dispatch(["setup", app, "--config-dir", str(config_dir)]) == 0
        assert "existing selection kept" in capsys.readouterr().out
        assert dispatch(["current", app, "--config-dir", str(config_dir)]) == 0
        assert capsys.readouterr().out.strip().endswith("sf2-chun-li")


def test_setup_without_selection_installs_default(tmp_path: Path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    assert dispatch(["setup", "starship", "--config-dir", str(tmp_path / "starship")]) == 0
    out = capsys.readouterr().out
    assert "setup starship: sf2-main" in out
    assert "existing selection kept" not in out
