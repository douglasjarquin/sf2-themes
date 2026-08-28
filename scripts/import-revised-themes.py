#!/usr/bin/env python3
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///

# ─── How to run ───
# 1. Install uv (if not installed):
#      curl -LsSf https://astral.sh/uv/install.sh | sh
# 2. Run directly:
#      python3 scripts/import-revised-themes.py --source PATH --destination themes [--check]
# 3. Or run through uv:
#      uv run scripts/import-revised-themes.py --source PATH --destination themes [--check]
# ──────────────────

from __future__ import annotations

import argparse
import ctypes
import hashlib
import os
import shutil
import sys
import tempfile
import tomllib
from dataclasses import dataclass
from enum import StrEnum, unique
from pathlib import Path
from typing import Final, assert_never

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from sf2_theme.errors import ThemeError  # noqa: E402
from sf2_theme.model import Theme, theme_to_toml  # noqa: E402
from sf2_theme.parse import parse_theme  # noqa: E402
from sf2_theme.validation import EXPECTED_DARK_IDS  # noqa: E402

EXPECTED_INVENTORY_DIGEST: Final = "0d562e06f71b335bab44112d8dd7f7f5f9071547bb1675505467211ed1af64cf"
DESIGNER_META_FIELDS: Final = frozenset({"id", "name", "variant", "family", "stage"})
OPERATIONAL_META_FIELDS: Final = ("display_name", "kind", "introduced_in", "character", "aliases")
AT_FDCWD: Final = -100
RENAME_EXCHANGE: Final = 2
RENAME_SWAP: Final = 2


@unique
class AtomicExchangePlatform(StrEnum):
    DARWIN = "darwin"
    LINUX = "linux"


@dataclass(frozen=True, slots=True)
class RevisedThemeImportError(Exception):
    detail: str

    def __str__(self) -> str:
        return self.detail


def expected_names() -> tuple[str, ...]:
    names = {f"{theme_id}.toml" for theme_id in EXPECTED_DARK_IDS}
    names.update(f"{theme_id}-light.toml" for theme_id in EXPECTED_DARK_IDS)
    return tuple(sorted(names))


def destination_path(destination: Path, name: str) -> Path:
    if name in {"main.toml", "main-light.toml"}:
        return destination / name
    return destination / "characters" / name


def source_inventory(source: Path) -> tuple[tuple[str, str], ...]:
    actual_names = tuple(sorted(path.name for path in source.glob("*.toml")))
    expected = expected_names()
    if actual_names != expected:
        missing = sorted(set(expected) - set(actual_names))
        unexpected = sorted(set(actual_names) - set(expected))
        details = [f"expected 36 TOML files, found {len(actual_names)}"]
        if missing:
            details.append(f"missing: {', '.join(missing)}")
        if unexpected:
            details.append(f"unexpected: {', '.join(unexpected)}")
        raise RevisedThemeImportError("; ".join(details))
    return tuple((name, hashlib.sha256((source / name).read_bytes()).hexdigest()) for name in actual_names)


def inventory_text(inventory: tuple[tuple[str, str], ...]) -> str:
    return "".join(f"{digest}  {name}\n" for name, digest in inventory)


def require_bound_inventory(inventory: tuple[tuple[str, str], ...]) -> None:
    digest = hashlib.sha256(inventory_text(inventory).encode()).hexdigest()
    if digest != EXPECTED_INVENTORY_DIGEST:
        raise RevisedThemeImportError(
            f"source inventory digest changed: expected {EXPECTED_INVENTORY_DIGEST}, got {digest}"
        )


def merged_theme(source_path: Path, current_path: Path) -> Theme:
    revised = tomllib.loads(source_path.read_text(encoding="utf-8"))
    current = tomllib.loads(current_path.read_text(encoding="utf-8"))
    revised_meta = revised.get("meta")
    current_meta = current.get("meta")
    if not isinstance(revised_meta, dict) or set(revised_meta) != DESIGNER_META_FIELDS:
        raise RevisedThemeImportError(f"{source_path}: expected exact revised meta fields")
    if not isinstance(current_meta, dict):
        raise RevisedThemeImportError(f"{current_path}: missing operational metadata")
    source_id = revised_meta.get("id")
    expected_id = source_path.stem
    if source_id != expected_id:
        raise RevisedThemeImportError(f"{source_path}: meta.id must be {expected_id!r}, got {source_id!r}")
    operational = {field: current_meta[field] for field in OPERATIONAL_META_FIELDS if field in current_meta}
    canonical = {
        "schema_version": current.get("schema_version"),
        "meta": {**operational, **revised_meta},
        "ui": revised.get("ui"),
        "semantic": revised.get("semantic"),
        "ansi": revised.get("ansi"),
    }
    return parse_theme(canonical, source=str(source_path))


def transformed_catalog(source: Path, destination: Path) -> tuple[tuple[str, str], ...]:
    transformed: list[tuple[str, str]] = []
    ids: set[str] = set()
    for name in expected_names():
        theme = merged_theme(source / name, destination_path(destination, name))
        if theme.metadata.id in ids:
            raise RevisedThemeImportError(f"duplicate source id: {theme.metadata.id}")
        ids.add(theme.metadata.id)
        transformed.append((name, theme_to_toml(theme)))
    if len(ids) != 36:
        raise RevisedThemeImportError(f"expected 36 unique IDs, found {len(ids)}")
    return tuple(transformed)


def destination_matches(destination: Path, transformed: tuple[tuple[str, str], ...]) -> bool:
    return all(
        destination_path(destination, name).read_text(encoding="utf-8") == content
        for name, content in transformed
    )


def atomic_exchange(first: Path, second: Path) -> None:
    libc = ctypes.CDLL(None, use_errno=True)
    try:
        platform = AtomicExchangePlatform(sys.platform)
    except ValueError:
        raise RevisedThemeImportError(f"atomic catalog exchange is unsupported on {sys.platform}") from None
    match platform:
        case AtomicExchangePlatform.DARWIN:
            exchange = libc.renamex_np
            exchange.argtypes = (ctypes.c_char_p, ctypes.c_char_p, ctypes.c_uint)
            exchange.restype = ctypes.c_int
            result = exchange(os.fsencode(first), os.fsencode(second), RENAME_SWAP)
        case AtomicExchangePlatform.LINUX:
            exchange = libc.renameat2
            exchange.argtypes = (
                ctypes.c_int,
                ctypes.c_char_p,
                ctypes.c_int,
                ctypes.c_char_p,
                ctypes.c_uint,
            )
            exchange.restype = ctypes.c_int
            result = exchange(AT_FDCWD, os.fsencode(first), AT_FDCWD, os.fsencode(second), RENAME_EXCHANGE)
        case unreachable:
            assert_never(unreachable)
    if result != 0:
        error_number = ctypes.get_errno()
        raise OSError(error_number, os.strerror(error_number), str(first), str(second))


def promote_catalog(
    source: Path,
    destination: Path,
    inventory: tuple[tuple[str, str], ...],
    transformed: tuple[tuple[str, str], ...],
) -> None:
    with tempfile.TemporaryDirectory(prefix=".import-revised-themes-", dir=destination.parent) as temporary:
        staging = Path(temporary) / "staged"
        shutil.copytree(destination, staging, symlinks=True)
        for name, content in transformed:
            staged_path = destination_path(staging, name)
            staged_path.write_text(content, encoding="utf-8")
        final_inventory = source_inventory(source)
        require_bound_inventory(final_inventory)
        if final_inventory != inventory:
            raise RevisedThemeImportError("source changed while import was staged")
        atomic_exchange(staging, destination)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Import the approved revised SF2 theme palette drop.")
    parser.add_argument("--source", required=True, type=Path)
    parser.add_argument("--destination", required=True, type=Path)
    parser.add_argument("--check", action="store_true")
    return parser.parse_args()


def run() -> int:
    args = parse_args()
    inventory = source_inventory(args.source)
    transformed = transformed_catalog(args.source, args.destination)
    final_inventory = source_inventory(args.source)
    require_bound_inventory(final_inventory)
    if final_inventory != inventory:
        raise RevisedThemeImportError("source changed during preflight")
    print(inventory_text(inventory), end="")
    print(f"inventory-sha256  {EXPECTED_INVENTORY_DIGEST}")
    print(f"preflight: {len(inventory)} files, {len(transformed)} IDs")
    matches = destination_matches(args.destination, transformed)
    if args.check:
        print(f"destination-match: {'yes' if matches else 'no'}")
        return 0
    promote_catalog(args.source, args.destination, inventory, transformed)
    print("installed: 36 themes")
    return 0


def main() -> int:
    try:
        return run()
    except (OSError, KeyError, ThemeError, tomllib.TOMLDecodeError, RevisedThemeImportError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
