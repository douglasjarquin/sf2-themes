import subprocess
import sys
from pathlib import Path


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
