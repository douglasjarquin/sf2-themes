import tomllib
from pathlib import Path
from xml.etree import ElementTree

from sf2_theme.adapters.codex import setup_codex
from sf2_theme.catalog import get_theme, parse_catalog
from sf2_theme.cli import dispatch


def test_apps_lists_codex(capsys) -> None:
    assert dispatch(["apps"]) == 0

    assert "codex" in capsys.readouterr().out.splitlines()


def test_apply_codex_defaults_to_main_and_switches_character(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "codex"

    assert dispatch(["apply", "codex", "--config-dir", str(config_dir)]) == 0
    assert dispatch(["current", "codex", "--config-dir", str(config_dir)]) == 0
    assert capsys.readouterr().out.splitlines()[-1] == "sf2-main"

    assert (
        dispatch(
            [
                "apply",
                "codex",
                "--theme",
                "ryu-light",
                "--config-dir",
                str(config_dir),
            ]
        )
        == 0
    )
    assert dispatch(["current", "codex", "--config-dir", str(config_dir)]) == 0
    assert capsys.readouterr().out.splitlines()[-1] == "sf2-ryu-light"
    assert (config_dir / "themes" / "sf2-ryu-light.tmTheme").is_file()
    backups = list(config_dir.glob("config.toml.bak.*"))
    assert len(backups) == 1
    assert tomllib.loads(backups[0].read_text(encoding="utf-8"))["tui"]["theme"] == "sf2-main"


def test_setup_codex_installs_catalog_and_selects_default(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "codex"

    assert dispatch(["setup", "codex", "--config-dir", str(config_dir)]) == 0
    assert dispatch(["current", "codex", "--config-dir", str(config_dir)]) == 0

    assert capsys.readouterr().out.splitlines()[-1] == "sf2-main"
    assert (config_dir / "themes" / "sf2-main-light.tmTheme").is_file()


def test_setup_codex_migrates_existing_unprefixed_selection(tmp_path: Path) -> None:
    config_dir = tmp_path / "codex"
    config_dir.mkdir()
    (config_dir / "config.toml").write_text('[tui]\ntheme = "ken"\n', encoding="utf-8")
    catalog = parse_catalog()

    setup_codex(
        get_theme("main", catalog),
        catalog,
        config_dir=config_dir,
        dry_run=False,
        follow_symlinks=False,
        replace_theme=False,
    )

    assert tomllib.loads((config_dir / "config.toml").read_text(encoding="utf-8"))["tui"]["theme"] == "sf2-ken"


def test_apply_codex_writes_every_catalog_theme_with_palette_values(tmp_path: Path) -> None:
    config_dir = tmp_path / "codex"

    assert dispatch(["apply", "codex", "--config-dir", str(config_dir)]) == 0

    config = tomllib.loads((config_dir / "config.toml").read_text(encoding="utf-8"))
    assert config["tui"]["theme"] == "sf2-main"
    expected_ids = {f"sf2-{theme.metadata.id}" for theme in parse_catalog()}
    written_ids = {path.stem for path in (config_dir / "themes").glob("*.tmTheme")}
    assert written_ids == expected_ids

    main = get_theme("main", parse_catalog())
    document = ElementTree.parse(config_dir / "themes" / "sf2-main.tmTheme")
    rendered = ElementTree.tostring(document.getroot(), encoding="unicode")
    assert main.ui.background in rendered
    assert main.ui.foreground in rendered
    assert main.semantic.red in rendered


def test_codex_generated_ids_are_prefixed_and_old_files_are_replaced(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "codex"
    themes_dir = config_dir / "themes"
    themes_dir.mkdir(parents=True)
    (themes_dir / "main.tmTheme").write_text("old managed theme", encoding="utf-8")

    assert dispatch(["apply", "codex", "--theme", "ken", "--config-dir", str(config_dir)]) == 0

    config = tomllib.loads((config_dir / "config.toml").read_text(encoding="utf-8"))
    written_ids = {path.stem for path in themes_dir.glob("*.tmTheme")}
    expected_ids = {f"sf2-{theme.metadata.id}" for theme in parse_catalog()}
    assert written_ids == expected_ids
    assert all(theme_id.startswith("sf2-") for theme_id in written_ids)
    assert config["tui"]["theme"] == "sf2-ken"
    assert not (themes_dir / "main.tmTheme").exists()

    assert dispatch(["current", "codex", "--config-dir", str(config_dir)]) == 0
    assert capsys.readouterr().out.splitlines()[-1] == "sf2-ken"


def test_apply_codex_dry_run_does_not_change_config_or_create_backup(tmp_path: Path) -> None:
    config_dir = tmp_path / "codex"
    assert dispatch(["apply", "codex", "--config-dir", str(config_dir)]) == 0
    config_path = config_dir / "config.toml"
    original = config_path.read_text(encoding="utf-8")

    assert (
        dispatch(
            [
                "apply",
                "codex",
                "--theme",
                "ryu-light",
                "--config-dir",
                str(config_dir),
                "--dry-run",
            ]
        )
        == 0
    )

    assert config_path.read_text(encoding="utf-8") == original
    assert not list(config_dir.glob("config.toml.bak.*"))


def test_apply_codex_preserves_other_config_settings(tmp_path: Path) -> None:
    config_dir = tmp_path / "codex"
    config_dir.mkdir()
    config_path = config_dir / "config.toml"
    config_path.write_text('[model]\nname = "gpt-5"\n\n[tui]\nanimations = false\n', encoding="utf-8")

    assert dispatch(["apply", "codex", "--theme", "ryu", "--config-dir", str(config_dir)]) == 0

    config = tomllib.loads(config_path.read_text(encoding="utf-8"))
    assert config["model"]["name"] == "gpt-5"
    assert config["tui"] == {"animations": False, "theme": "sf2-ryu"}


def test_apply_codex_refuses_config_symlink_without_follow_flag(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "codex"
    config_dir.mkdir()
    target = tmp_path / "real-config.toml"
    target.write_text('[tui]\ntheme = "main"\n', encoding="utf-8")
    (config_dir / "config.toml").symlink_to(target)

    assert dispatch(["apply", "codex", "--config-dir", str(config_dir)]) == 1

    assert "symlink" in capsys.readouterr().err
    assert target.read_text(encoding="utf-8") == '[tui]\ntheme = "main"\n'


def test_apply_codex_follows_config_symlink_when_requested(tmp_path: Path) -> None:
    config_dir = tmp_path / "codex"
    config_dir.mkdir()
    target = tmp_path / "real-config.toml"
    target.write_text('[tui]\ntheme = "catppuccin-mocha"\n', encoding="utf-8")
    (config_dir / "config.toml").symlink_to(target)

    assert (
        dispatch(
            [
                "apply",
                "codex",
                "--theme",
                "ryu-light",
                "--config-dir",
                str(config_dir),
                "--follow-symlinks",
            ]
        )
        == 0
    )

    assert 'theme = "sf2-ryu-light"' in target.read_text(encoding="utf-8")


def test_codex_home_environment_override(tmp_path: Path, monkeypatch) -> None:
    config_dir = tmp_path / "env-codex"
    monkeypatch.setenv("CODEX_HOME", str(config_dir))

    assert dispatch(["apply", "codex"]) == 0

    assert (config_dir / "config.toml").is_file()
    assert not (tmp_path / ".codex" / "config.toml").exists()
