import json
from pathlib import Path

from sf2_theme.adapters.claude import setup_claude
from sf2_theme.catalog import get_theme, parse_catalog
from sf2_theme.cli import dispatch


def test_apps_lists_claude(capsys) -> None:
    assert dispatch(["apps"]) == 0

    assert "claude" in capsys.readouterr().out.splitlines()


def test_apply_claude_defaults_to_main_and_switches_character(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "claude"

    assert dispatch(["apply", "claude", "--config-dir", str(config_dir)]) == 0
    assert dispatch(["current", "claude", "--config-dir", str(config_dir)]) == 0
    assert capsys.readouterr().out.splitlines()[-1] == "sf2-main"

    assert dispatch(["apply", "claude", "--theme", "ryu-light", "--config-dir", str(config_dir)]) == 0
    assert dispatch(["current", "claude", "--config-dir", str(config_dir)]) == 0
    assert capsys.readouterr().out.splitlines()[-1] == "sf2-ryu-light"
    assert (config_dir / "themes" / "sf2-ryu-light.json").is_file()
    backups = list(config_dir.glob("settings.json.bak.*"))
    assert len(backups) == 1
    assert json.loads(backups[0].read_text(encoding="utf-8"))["theme"] == "custom:sf2-main"


def test_setup_claude_installs_catalog_and_selects_default(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "claude"

    assert dispatch(["setup", "claude", "--config-dir", str(config_dir)]) == 0
    assert dispatch(["current", "claude", "--config-dir", str(config_dir)]) == 0

    assert capsys.readouterr().out.splitlines()[-1] == "sf2-main"
    assert (config_dir / "themes" / "sf2-main-light.json").is_file()


def test_setup_claude_preserves_existing_selection(tmp_path: Path) -> None:
    config_dir = tmp_path / "claude"
    config_dir.mkdir()
    (config_dir / "settings.json").write_text(json.dumps({"theme": "custom:sf2-ken"}), encoding="utf-8")
    catalog = parse_catalog()

    setup_claude(
        get_theme("main", catalog),
        catalog,
        config_dir=config_dir,
        dry_run=False,
        follow_symlinks=False,
        replace_theme=False,
    )

    assert json.loads((config_dir / "settings.json").read_text(encoding="utf-8"))["theme"] == "custom:sf2-ken"


def test_apply_claude_writes_every_catalog_theme_with_palette_values(tmp_path: Path) -> None:
    config_dir = tmp_path / "claude"

    assert dispatch(["apply", "claude", "--config-dir", str(config_dir)]) == 0

    settings = json.loads((config_dir / "settings.json").read_text(encoding="utf-8"))
    assert settings["theme"] == "custom:sf2-main"
    expected_ids = {f"sf2-{theme.metadata.id}" for theme in parse_catalog()}
    written_ids = {path.stem for path in (config_dir / "themes").glob("*.json")}
    assert written_ids == expected_ids

    main = get_theme("main", parse_catalog())
    document = json.loads((config_dir / "themes" / "sf2-main.json").read_text(encoding="utf-8"))
    assert document["base"] == "dark"
    assert document["overrides"]["error"] == main.semantic.red
    assert document["overrides"]["success"] == main.semantic.green
    assert document["overrides"]["claude"] == main.ui.accent


def test_apply_claude_dry_run_does_not_change_settings_or_create_backup(tmp_path: Path) -> None:
    config_dir = tmp_path / "claude"
    assert dispatch(["apply", "claude", "--config-dir", str(config_dir)]) == 0
    settings_path = config_dir / "settings.json"
    original = settings_path.read_text(encoding="utf-8")

    assert (
        dispatch(
            ["apply", "claude", "--theme", "ryu-light", "--config-dir", str(config_dir), "--dry-run"]
        )
        == 0
    )

    assert settings_path.read_text(encoding="utf-8") == original
    assert not list(config_dir.glob("settings.json.bak.*"))


def test_apply_claude_preserves_other_settings(tmp_path: Path) -> None:
    config_dir = tmp_path / "claude"
    config_dir.mkdir()
    settings_path = config_dir / "settings.json"
    settings_path.write_text(json.dumps({"model": "sonnet", "permissions": {"allow": ["Bash(ls)"]}}), encoding="utf-8")

    assert dispatch(["apply", "claude", "--theme", "ryu", "--config-dir", str(config_dir)]) == 0

    settings = json.loads(settings_path.read_text(encoding="utf-8"))
    assert settings["model"] == "sonnet"
    assert settings["permissions"] == {"allow": ["Bash(ls)"]}
    assert settings["theme"] == "custom:sf2-ryu"


def test_apply_claude_refuses_settings_symlink_without_follow_flag(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "claude"
    config_dir.mkdir()
    target = tmp_path / "real-settings.json"
    target.write_text(json.dumps({"theme": "dark"}), encoding="utf-8")
    (config_dir / "settings.json").symlink_to(target)

    assert dispatch(["apply", "claude", "--config-dir", str(config_dir)]) == 1

    assert "symlink" in capsys.readouterr().err
    assert json.loads(target.read_text(encoding="utf-8")) == {"theme": "dark"}


def test_apply_claude_follows_settings_symlink_when_requested(tmp_path: Path) -> None:
    config_dir = tmp_path / "claude"
    config_dir.mkdir()
    target = tmp_path / "real-settings.json"
    target.write_text(json.dumps({"theme": "dark"}), encoding="utf-8")
    (config_dir / "settings.json").symlink_to(target)

    assert (
        dispatch(
            ["apply", "claude", "--theme", "ryu-light", "--config-dir", str(config_dir), "--follow-symlinks"]
        )
        == 0
    )

    assert json.loads(target.read_text(encoding="utf-8"))["theme"] == "custom:sf2-ryu-light"


def test_claude_config_dir_environment_override(tmp_path: Path, monkeypatch) -> None:
    config_dir = tmp_path / "env-claude"
    monkeypatch.setenv("CLAUDE_CONFIG_DIR", str(config_dir))

    assert dispatch(["apply", "claude"]) == 0

    assert (config_dir / "settings.json").is_file()
    assert not (tmp_path / ".claude" / "settings.json").exists()
