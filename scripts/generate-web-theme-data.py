#!/usr/bin/env python3
import argparse
import json
import os
import sys
import tempfile
import tomllib
from dataclasses import asdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "src"))

from sf2_theme.catalog import load_catalog  # noqa: E402
from sf2_theme.errors import ThemeError  # noqa: E402
from sf2_theme.model import Theme, project_adapter_colors  # noqa: E402

DEFAULT_SOURCE = ROOT / "themes"
DEFAULT_OUTPUT = ROOT / "web" / "src" / "data" / "generated-theme-data.json"


def browser_theme(theme: Theme) -> dict[str, object]:
    metadata = asdict(theme.metadata)
    if metadata["character"] is None:
        del metadata["character"]
    return {
        "schema_version": 1,
        "meta": metadata,
        "ui": asdict(theme.ui) | asdict(project_adapter_colors(theme.ui)),
        "semantic": asdict(theme.semantic),
        "ansi": {
            "normal": asdict(theme.ansi_normal),
            "bright": asdict(theme.ansi_bright),
        },
    }


def render(source: Path) -> str:
    os.environ["SF2_THEME_DIR"] = str(source.resolve())
    payload = {
        "schema_version": 1,
        "themes": [browser_theme(theme) for theme in load_catalog()],
    }
    return json.dumps(payload, ensure_ascii=False, indent=2) + "\n"


def write_atomic(output: Path, content: str) -> None:
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(
        "w",
        dir=output.parent,
        encoding="utf-8",
        delete=False,
    ) as staged:
        staged.write(content)
        staged_path = Path(staged.name)
    staged_path.replace(output)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate browser theme data from the canonical Python model."
    )
    parser.add_argument("--source", type=Path, default=DEFAULT_SOURCE)
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT)
    parser.add_argument("--check", action="store_true")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        content = render(args.source)
        if args.check:
            if not args.output.is_file() or args.output.read_text(encoding="utf-8") != content:
                print(f"stale generated theme data: {args.output}", file=sys.stderr)
                return 1
            print(f"verified generated theme data: {args.output}")
            return 0
        write_atomic(args.output, content)
    except (OSError, ThemeError, tomllib.TOMLDecodeError) as error:
        print(error, file=sys.stderr)
        return 1
    print(f"wrote {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
