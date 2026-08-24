"""Symlink, mode, backup, dry-run, and idempotent writes."""

from pathlib import Path

from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import WriteAction, write_file


def test_refuses_symlink_by_default(tmp_path: Path) -> None:
    target = tmp_path / "real.toml"
    target.write_text("old\n", encoding="utf-8")
    link = tmp_path / "link.toml"
    link.symlink_to(target)
    try:
        write_file(link, "new\n")
    except ThemeError as error:
        assert "symlink" in str(error)
    else:
        raise AssertionError("expected symlink refusal")
    assert link.is_symlink()
    assert target.read_text(encoding="utf-8") == "old\n"


def test_follow_symlinks_writes_target(tmp_path: Path) -> None:
    target = tmp_path / "real.toml"
    target.write_text("old\n", encoding="utf-8")
    link = tmp_path / "link.toml"
    link.symlink_to(target)
    result = write_file(link, "new\n", follow_symlinks=True)
    assert result.action is WriteAction.UPDATED
    assert link.is_symlink()
    assert target.read_text(encoding="utf-8") == "new\n"


def test_preserves_mode(tmp_path: Path) -> None:
    path = tmp_path / "config.toml"
    path.write_text("old\n", encoding="utf-8")
    path.chmod(0o600)
    write_file(path, "new\n")
    assert path.stat().st_mode & 0o777 == 0o600


def test_backup_on_change_not_on_noop(tmp_path: Path) -> None:
    path = tmp_path / "config.toml"
    path.write_text("old\n", encoding="utf-8")
    write_file(path, "new\n")
    backups = list(tmp_path.glob("config.toml.bak.*"))
    assert len(backups) == 1
    assert backups[0].read_text(encoding="utf-8") == "old\n"
    write_file(path, "new\n")
    assert len(list(tmp_path.glob("config.toml.bak.*"))) == 1


def test_dry_run_writes_nothing(tmp_path: Path) -> None:
    path = tmp_path / "config.toml"
    result = write_file(path, "new\n", dry_run=True)
    assert result.action is WriteAction.WOULD_CREATE
    assert not path.exists()
