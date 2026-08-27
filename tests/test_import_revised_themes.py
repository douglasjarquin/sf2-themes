import importlib.util
import os
import subprocess
import sys
from pathlib import Path
from types import ModuleType

import pytest

IMPORTER_PATH = Path("scripts/import-revised-themes.py")


def load_importer() -> ModuleType:
    spec = importlib.util.spec_from_file_location("import_revised_themes", IMPORTER_PATH)
    assert spec is not None
    assert spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def catalog_snapshot(destination: Path) -> dict[Path, bytes]:
    return {
        path.relative_to(destination): path.read_bytes()
        for path in destination.rglob("*")
        if path.is_file()
    }


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
            "scripts/import-revised-themes.py",
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


def test_promotion_failure_restores_exact_pre_import_catalog(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
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

    def fail_third_replacement(source_path: Path, destination_path: Path) -> None:
        nonlocal replacement_calls
        replacement_calls += 1
        if replacement_calls == 3:
            raise OSError("injected third replacement failure")
        real_replace(source_path, destination_path)

    monkeypatch.setattr(importer.os, "replace", fail_third_replacement)

    # When: the real promotion seam is interrupted during its third replacement.
    with pytest.raises(OSError, match="injected third replacement failure"):
        importer.promote_catalog(source, destination, inventory, tuple(transformed))

    # Then: every destination byte is exactly the pre-import catalog, never a mixture.
    assert replacement_calls >= 3
    assert catalog_snapshot(destination) == before
