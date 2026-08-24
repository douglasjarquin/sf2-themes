"""Safe configuration writes: symlinks, modes, backups, dry-run."""

import os
import stat
import tempfile
from dataclasses import dataclass
from datetime import UTC, datetime
from difflib import unified_diff
from enum import StrEnum, unique
from pathlib import Path

from sf2_theme.errors import ThemeError


@unique
class WriteAction(StrEnum):
    CREATED = "created"
    UPDATED = "updated"
    UNCHANGED = "unchanged"
    WOULD_CREATE = "would_create"
    WOULD_UPDATE = "would_update"


@dataclass(frozen=True, slots=True)
class WriteResult:
    """Outcome of a configuration write."""

    path: Path
    action: WriteAction
    diff: str


def write_file(
    path: Path,
    content: str,
    *,
    dry_run: bool = False,
    follow_symlinks: bool = False,
) -> WriteResult:
    """Atomically write `content` with symlink, mode, and backup safety."""
    destination = _resolve_destination(path, follow_symlinks=follow_symlinks)
    existing = destination.read_text(encoding="utf-8") if destination.is_file() else None
    if existing == content:
        return WriteResult(path=destination, action=WriteAction.UNCHANGED, diff="")
    diff = _diff(destination, existing, content)
    if dry_run:
        action = WriteAction.WOULD_CREATE if existing is None else WriteAction.WOULD_UPDATE
        return WriteResult(path=destination, action=action, diff=diff)
    if existing is not None:
        _backup(destination, existing)
    destination.parent.mkdir(parents=True, exist_ok=True)
    mode = destination.stat().st_mode & 0o777 if destination.exists() else 0o644
    _replace(destination, content, mode)
    action = WriteAction.CREATED if existing is None else WriteAction.UPDATED
    return WriteResult(path=destination, action=action, diff=diff)


def _resolve_destination(path: Path, *, follow_symlinks: bool) -> Path:
    if not path.is_symlink():
        return path
    if not follow_symlinks:
        raise ThemeError(
            f"refusing to replace symlink {path}; pass --follow-symlinks to write through"
        )
    return path.resolve()


def _backup(path: Path, existing: str) -> None:
    stamp = datetime.now(UTC).strftime("%Y%m%dT%H%M%SZ")
    backup = path.with_name(f"{path.name}.bak.{stamp}")
    backup.write_text(existing, encoding="utf-8")
    backup.chmod(path.stat().st_mode & 0o777)


def _replace(path: Path, content: str, mode: int) -> None:
    descriptor, temporary_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent, text=True)
    temporary = Path(temporary_name)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8") as handle:
            handle.write(content)
        os.chmod(temporary, mode | stat.S_IRUSR | stat.S_IWUSR)
        os.replace(temporary, path)
        path.chmod(mode)
    except OSError:
        temporary.unlink(missing_ok=True)
        raise


def _diff(path: Path, existing: str | None, content: str) -> str:
    before = existing.splitlines(keepends=True) if existing is not None else []
    after = content.splitlines(keepends=True)
    from_name = "/dev/null" if existing is None else str(path)
    return "".join(unified_diff(before, after, fromfile=from_name, tofile=str(path)))
