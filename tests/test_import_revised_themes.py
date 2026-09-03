import importlib.machinery
import importlib.util
import os
import subprocess
import sys
from pathlib import Path
from types import ModuleType

import pytest

IMPORTER_PATH = Path("mise-tasks/import-revised-themes")


def load_importer() -> ModuleType:
    loader = importlib.machinery.SourceFileLoader("import_revised_themes", str(IMPORTER_PATH))
    spec = importlib.util.spec_from_file_location("import_revised_themes", IMPORTER_PATH, loader=loader)
    assert spec is not None
    assert spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def catalog_snapshot(destination: Path) -> dict[Path, bytes]:
    return {path.relative_to(destination): path.read_bytes() for path in destination.rglob("*") if path.is_file()}


def test_importer_fails_before_writes_when_source_file_is_missing(tmp_path: Path) -> None:
    # Given: an empty source and a destination containing operator-owned data.
    source = tmp_path / "source"
    destination = tmp_path / "themes"
    source.mkdir()
    destination.mkdir()
    marker = destination / "operator-owned.txt"
    marker.write_text("preserve me\n", encoding="utf-8")

    # When: the real importer preflights that incomplete source.
    result = subprocess.run(
        (
            sys.executable,
            "mise-tasks/import-revised-themes",
            "--source",
            str(source),
            "--destination",
            str(destination),
            "--check",
        ),
        check=False,
        capture_output=True,
        text=True,
    )

    # Then: the command fails closed before changing the destination.
    assert result.returncode != 0
    assert "expected 36 TOML files" in result.stderr
    assert marker.read_text(encoding="utf-8") == "preserve me\n"
    assert tuple(destination.iterdir()) == (marker,)


def test_persistent_replacement_failure_never_exposes_mixed_catalog(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    # Given: a complete source and destination catalog with an operator-owned file.
    importer = load_importer()
    source = tmp_path / "source"
    destination = tmp_path / "themes"
    source.mkdir()
    destination.mkdir()
    (destination / "AGENTS.md").write_text("operator owned\n", encoding="utf-8")
    transformed: list[tuple[str, str]] = []
    for name in importer.expected_names():
        (source / name).write_text(f"source {name}\n", encoding="utf-8")
        current_path = importer.destination_path(destination, name)
        current_path.parent.mkdir(parents=True, exist_ok=True)
        current_path.write_text(f"old {name}\n", encoding="utf-8")
        transformed.append((name, f"new {name}\n"))
    inventory = importer.source_inventory(source)
    before = catalog_snapshot(destination)
    monkeypatch.setattr(importer, "require_bound_inventory", lambda _inventory: None)
    real_replace = os.replace
    replacement_calls = 0

    def fail_from_third_replacement(source_path: Path, destination_path: Path) -> None:
        nonlocal replacement_calls
        replacement_calls += 1
        if replacement_calls >= 3:
            raise OSError("injected persistent replacement failure")
        real_replace(source_path, destination_path)

    monkeypatch.setattr(importer.os, "replace", fail_from_third_replacement)

    # When: the legacy per-file replacement primitive fails persistently from call three onward.
    try:
        importer.promote_catalog(source, destination, inventory, tuple(transformed))
    except OSError as error:
        assert "injected persistent replacement failure" in str(error)

    # Then: the live catalog is one complete generation, never a mixture.
    after = {
        **before,
        **{
            importer.destination_path(destination, name).relative_to(destination): content.encode()
            for name, content in transformed
        },
    }
    assert replacement_calls == 0
    assert catalog_snapshot(destination) == after


def test_persistent_atomic_exchange_failure_leaves_pre_import_catalog(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    # Given: a complete destination and an atomic exchange seam that always fails.
    importer = load_importer()
    source = tmp_path / "source"
    destination = tmp_path / "themes"
    source.mkdir()
    destination.mkdir()
    (destination / "AGENTS.md").write_text("operator owned\n", encoding="utf-8")
    transformed: list[tuple[str, str]] = []
    for name in importer.expected_names():
        (source / name).write_text(f"source {name}\n", encoding="utf-8")
        current_path = importer.destination_path(destination, name)
        current_path.parent.mkdir(parents=True, exist_ok=True)
        current_path.write_text(f"old {name}\n", encoding="utf-8")
        transformed.append((name, f"new {name}\n"))
    inventory = importer.source_inventory(source)
    before = catalog_snapshot(destination)
    monkeypatch.setattr(importer, "require_bound_inventory", lambda _inventory: None)
    exchange_calls = 0

    def fail_atomic_exchange(_first: Path, _second: Path) -> None:
        nonlocal exchange_calls
        exchange_calls += 1
        raise OSError("injected persistent atomic exchange failure")

    monkeypatch.setattr(importer, "atomic_exchange", fail_atomic_exchange)

    # When: catalog promotion reaches the persistently failing atomic boundary.
    with pytest.raises(OSError, match="injected persistent atomic exchange failure"):
        importer.promote_catalog(source, destination, inventory, tuple(transformed))

    # Then: there was one indivisible attempt and the complete old catalog remains live.
    assert exchange_calls == 1
    assert catalog_snapshot(destination) == before


def test_interruption_after_atomic_exchange_leaves_complete_new_catalog(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    # Given: a complete source and destination plus an interruption immediately after exchange.
    importer = load_importer()
    source = tmp_path / "source"
    destination = tmp_path / "themes"
    source.mkdir()
    destination.mkdir()
    (destination / "AGENTS.md").write_text("operator owned\n", encoding="utf-8")
    transformed: list[tuple[str, str]] = []
    for name in importer.expected_names():
        (source / name).write_text(f"source {name}\n", encoding="utf-8")
        current_path = importer.destination_path(destination, name)
        current_path.parent.mkdir(parents=True, exist_ok=True)
        current_path.write_text(f"old {name}\n", encoding="utf-8")
        transformed.append((name, f"new {name}\n"))
    inventory = importer.source_inventory(source)
    before = catalog_snapshot(destination)
    after = {
        **before,
        **{
            importer.destination_path(destination, name).relative_to(destination): content.encode()
            for name, content in transformed
        },
    }
    monkeypatch.setattr(importer, "require_bound_inventory", lambda _inventory: None)
    real_atomic_exchange = importer.atomic_exchange

    def interrupt_after_exchange(first: Path, second: Path) -> None:
        real_atomic_exchange(first, second)
        raise KeyboardInterrupt

    monkeypatch.setattr(importer, "atomic_exchange", interrupt_after_exchange)

    # When: interruption lands immediately after the one catalog-level switch.
    with pytest.raises(KeyboardInterrupt):
        importer.promote_catalog(source, destination, inventory, tuple(transformed))

    # Then: operator data and all 36 revised files form the complete new generation.
    assert catalog_snapshot(destination) == after
