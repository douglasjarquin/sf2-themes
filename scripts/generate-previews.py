#!/usr/bin/env python3
"""Generate SVG swatches for every packaged theme."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from sf2_theme.catalog import parse_catalog  # noqa: E402
from sf2_theme.model import Theme  # noqa: E402

OUT = Path(__file__).resolve().parents[1] / "docs" / "previews"


def swatch(theme: Theme) -> str:
    ui = theme.ui
    ansi = list(theme.ansi_normal.as_tuple()) + list(theme.ansi_bright.as_tuple())
    cells = []
    for index, color in enumerate(ansi):
        x = 16 + (index % 8) * 36
        y = 92 + (index // 8) * 36
        cells.append(f'<rect x="{x}" y="{y}" width="32" height="32" fill="{color}"/>')
    rows = (
        ('background', ui.background),
        ('foreground', ui.foreground),
        ('accent', ui.accent),
        ('active row', ui.active_row_bg),
        ('navigate', ui.navigate_row_bg),
        ('selection', ui.selection_bg),
    )
    chips = []
    for index, (label, color) in enumerate(rows):
        x = 16 + index * 52
        chips.append(
            f'<rect x="{x}" y="36" width="48" height="40" fill="{color}"/>'
            f'<text x="{x + 24}" y="88" text-anchor="middle" font-size="7" fill="{ui.foreground}">{label}</text>'
        )
    return "\n".join(
        (
            '<svg xmlns="http://www.w3.org/2000/svg" width="320" height="176" viewBox="0 0 320 176">',
            f'<rect width="320" height="176" fill="{ui.background}"/>',
            (
                f'<text x="16" y="22" font-family="ui-monospace, monospace" '
                f'font-size="12" fill="{ui.foreground}">{theme.metadata.display_name}</text>'
            ),
            *chips,
            *cells,
            "</svg>",
            "",
        )
    )


def main() -> int:
    OUT.mkdir(parents=True, exist_ok=True)
    for theme in parse_catalog():
        path = OUT / f"{theme.metadata.id}.svg"
        path.write_text(swatch(theme), encoding="utf-8")
        print(f"wrote {path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
