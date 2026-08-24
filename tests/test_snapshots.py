"""Golden WezTerm and Herdr renders for main and one character theme."""

from pathlib import Path

from sf2_theme.adapters.herdr import render_block
from sf2_theme.adapters.wezterm import render_scheme
from sf2_theme.catalog import get_theme, parse_catalog

SNAPSHOTS = Path(__file__).parent / "snapshots"


def _assert_snapshot(name: str, rendered: str) -> None:
    path = SNAPSHOTS / name
    assert path.is_file(), f"missing snapshot {path}"
    assert rendered == path.read_text(encoding="utf-8")


def test_main_wezterm_snapshot() -> None:
    theme = get_theme("main", parse_catalog())
    _assert_snapshot("wezterm-main.toml", render_scheme(theme))


def test_main_herdr_snapshot() -> None:
    theme = get_theme("main", parse_catalog())
    _assert_snapshot("herdr-main.toml", render_block(theme))


def test_ryu_wezterm_snapshot() -> None:
    theme = get_theme("ryu", parse_catalog())
    _assert_snapshot("wezterm-ryu.toml", render_scheme(theme))
