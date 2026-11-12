import json
from pathlib import Path

from sf2_theme.catalog import load_catalog
from sf2_theme.cli import dispatch


def test_apply_desktop_pair_and_catalog(tmp_path: Path, capsys) -> None:
    assert dispatch(["apply", "herdr-app", "--theme", "ryu-light", "--config-dir", str(tmp_path)]) == 0
    settings = json.loads((tmp_path / "settings.json").read_text())
    assert settings == {"theme": {"dark": "sf2-ryu", "light": "sf2-ryu-light"}}
    catalog = load_catalog()
    assert {p.stem for p in (tmp_path / "themes").glob("*.json")} == {t.metadata.selectable_id for t in catalog}
    for theme in catalog:
        data = json.loads((tmp_path / "themes" / f"{theme.metadata.selectable_id}.json").read_text())
        assert data["ansi"] == list(theme.ansi_normal.as_tuple() + theme.ansi_bright.as_tuple())
        assert data["background"] == theme.ui.background
        assert data["foreground"] == theme.ui.foreground
        assert data["selection"] == theme.ui.selection_background
        assert data["cursor"] == theme.ui.cursor
        assert data["colors"] == dict(
            canvas=theme.ui.background,
            chrome=theme.ui.surface,
            surface=theme.ui.overlay,
            border=theme.ui.border,
            accent=theme.ui.accent,
        )
    capsys.readouterr()
    assert dispatch(["current", "herdr-app", "--config-dir", str(tmp_path)]) == 0
    assert capsys.readouterr().out == "sf2-ryu\n"


def test_apply_preserves_settings_and_is_idempotent(tmp_path: Path, capsys) -> None:
    path = tmp_path / "settings.json"
    original = {
        "appearance": "light",
        "colors": {"accent": "#ffffff"},
        "layout": "work",
        "theme": {"dark": "graphite", "light": "paper", "extra": True},
    }
    path.write_text(json.dumps(original))
    args = ["apply", "herdr-app", "--config-dir", str(tmp_path)]
    assert dispatch(args) == 1
    assert json.loads(path.read_text()) == original
    args.append("--adopt")
    assert dispatch(args) == 0
    expected = {**original, "theme": {"dark": "sf2-main", "light": "sf2-main-light", "extra": True}}
    assert json.loads(path.read_text()) == expected
    assert json.loads(next(tmp_path.glob("settings.json.bak.*")).read_text()) == original
    capsys.readouterr()
    assert dispatch(args) == 0
    assert "no changes" in capsys.readouterr().out


def test_setup_preserves_foreign_mixed_and_single_selections(tmp_path: Path) -> None:
    path = tmp_path / "settings.json"
    for selection in ["paper", {"light": "sf2-ken-light", "dark": "sf2-ryu"}, {"light": "paper"}]:
        original = json.dumps({"theme": selection}, separators=(",", ":"))
        path.write_text(original)
        assert dispatch(["setup", "herdr-app", "--config-dir", str(tmp_path)]) == 0
        assert path.read_text() == original
    assert dispatch(["setup", "herdr-app", "--theme", "ken", "--config-dir", str(tmp_path)]) == 1
    assert path.read_text() == original
    assert dispatch(["setup", "herdr-app", "--theme", "ken", "--adopt", "--config-dir", str(tmp_path)]) == 0
    assert json.loads(path.read_text())["theme"] == {"light": "sf2-ken-light", "dark": "sf2-ken"}


def test_default_setup_and_dry_run(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path))
    assert dispatch(["setup", "herdr-app", "--dry-run"]) == 0
    assert not (tmp_path / "herdr-app").exists()
    assert dispatch(["setup", "herdr-app"]) == 0
    path = tmp_path / "herdr-app" / "settings.json"
    original = path.read_text()
    assert dispatch(["apply", "herdr-app", "--theme", "ken", "--dry-run"]) == 0
    assert path.read_text() == original
    assert not list(path.parent.rglob("*.bak.*"))


def test_invalid_settings_fail_before_writing_assets(tmp_path: Path, capsys) -> None:
    path = tmp_path / "settings.json"
    for invalid in ["{", "[]", "null", '{"theme": []}', '{"theme": 42}']:
        path.write_text(invalid)
        assert dispatch(["apply", "herdr-app", "--config-dir", str(tmp_path)]) == 1
        assert "settings.json" in capsys.readouterr().err
        assert path.read_text() == invalid
        assert not (tmp_path / "themes").exists()


def test_current_rejects_absent_foreign_and_mixed_pairs(tmp_path: Path, capsys) -> None:
    args = ["current", "herdr-app", "--config-dir", str(tmp_path)]
    assert dispatch(args) == 1
    capsys.readouterr()
    for selection in [
        "paper",
        {"light": "paper", "dark": "graphite"},
        {"light": "sf2-ken-light", "dark": "sf2-ryu"},
        {"light": "sf2-ryu", "dark": "sf2-ryu-light"},
        {"light": "sf2-unknown-light", "dark": "sf2-unknown"},
    ]:
        (tmp_path / "settings.json").write_text(json.dumps({"theme": selection}))
        assert dispatch(args) == 1
        assert capsys.readouterr().out == ""


def test_symlink_policy_for_settings_and_themes(tmp_path: Path) -> None:
    config = tmp_path / "config"
    config.mkdir()
    target = tmp_path / "settings.json"
    target.write_text("{}")
    (config / "settings.json").symlink_to(target)
    args = ["apply", "herdr-app", "--config-dir", str(config)]
    assert dispatch(args) == 1
    assert target.read_text() == "{}"
    assert dispatch([*args, "--follow-symlinks"]) == 0
    assert json.loads(target.read_text())["theme"]["dark"] == "sf2-main"
    theme_path = config / "themes" / "sf2-main.json"
    theme_path.unlink()
    theme_target = tmp_path / "theme.json"
    theme_target.write_text("{}")
    theme_path.symlink_to(theme_target)
    assert dispatch(args) == 1
    assert theme_target.read_text() == "{}"
    assert dispatch([*args, "--follow-symlinks"]) == 0
    assert len(json.loads(theme_target.read_text())["ansi"]) == 16
