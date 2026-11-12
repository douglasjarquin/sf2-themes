"""Planned mutations: preflight, dry-run, removals, and committed writes."""

from pathlib import Path

import pytest

from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import WriteAction, commit
from sf2_theme.plan import Remove, Write


def test_symlink_anywhere_in_plan_commits_nothing(tmp_path: Path) -> None:
    target = tmp_path / "real.toml"
    target.write_text("old\n", encoding="utf-8")
    (tmp_path / "c.toml").symlink_to(target)
    steps = [
        Write(tmp_path / "a.toml", "a\n"),
        Write(tmp_path / "b.toml", "b\n"),
        Write(tmp_path / "c.toml", "c\n"),
    ]
    with pytest.raises(ThemeError, match="symlink"):
        commit(steps)
    assert not (tmp_path / "a.toml").exists()
    assert not (tmp_path / "b.toml").exists()
    assert target.read_text(encoding="utf-8") == "old\n"


def test_follow_symlinks_writes_every_step_through(tmp_path: Path) -> None:
    target = tmp_path / "real.toml"
    target.write_text("old\n", encoding="utf-8")
    link = tmp_path / "c.toml"
    link.symlink_to(target)
    steps = [
        Write(tmp_path / "a.toml", "a\n"),
        Write(tmp_path / "b.toml", "b\n"),
        Write(link, "c\n"),
    ]
    results = commit(steps, follow_symlinks=True)
    assert [result.action for result in results] == [WriteAction.CREATED, WriteAction.CREATED, WriteAction.UPDATED]
    assert (tmp_path / "a.toml").read_text(encoding="utf-8") == "a\n"
    assert (tmp_path / "b.toml").read_text(encoding="utf-8") == "b\n"
    assert link.is_symlink()
    assert target.read_text(encoding="utf-8") == "c\n"


def test_dry_run_reports_every_outcome_and_touches_nothing(tmp_path: Path) -> None:
    root = tmp_path / "config"
    root.mkdir()
    changed = root / "changed.toml"
    changed.write_text("old\n", encoding="utf-8")
    unchanged = root / "unchanged.toml"
    unchanged.write_text("same\n", encoding="utf-8")
    legacy = root / "legacy.toml"
    legacy.write_text("legacy\n", encoding="utf-8")
    before = {path.name: path.read_text(encoding="utf-8") for path in root.iterdir()}
    steps = [
        Write(root / "new" / "created.toml", "new\n"),
        Write(changed, "new\n"),
        Write(unchanged, "same\n"),
        Remove(legacy),
    ]
    results = commit(steps, dry_run=True)
    assert [result.action for result in results] == [
        WriteAction.WOULD_CREATE,
        WriteAction.WOULD_UPDATE,
        WriteAction.UNCHANGED,
        WriteAction.WOULD_REMOVE,
    ]
    assert {path.name: path.read_text(encoding="utf-8") for path in root.iterdir()} == before


def test_removing_a_missing_file_reports_nothing(tmp_path: Path) -> None:
    assert commit([Remove(tmp_path / "gone.toml")]) == []


def test_commit_removes_existing_file(tmp_path: Path) -> None:
    legacy = tmp_path / "legacy.toml"
    legacy.write_text("legacy\n", encoding="utf-8")
    results = commit([Remove(legacy)])
    assert [(result.path, result.action) for result in results] == [(legacy, WriteAction.REMOVED)]
    assert not legacy.exists()


def test_commit_backs_up_changed_file_and_preserves_mode(tmp_path: Path) -> None:
    path = tmp_path / "config.toml"
    path.write_text("old\n", encoding="utf-8")
    path.chmod(0o600)
    commit([Write(path, "new\n")])
    assert path.read_text(encoding="utf-8") == "new\n"
    assert path.stat().st_mode & 0o777 == 0o600
    backups = list(tmp_path.glob("config.toml.bak.*"))
    assert [backup.read_text(encoding="utf-8") for backup in backups] == ["old\n"]
