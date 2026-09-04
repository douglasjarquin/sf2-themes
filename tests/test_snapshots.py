"""Golden WezTerm, Herdr, and Neovim renders for main and one character theme."""

from pathlib import Path

from sf2_theme.adapters.herdr import render_block
from sf2_theme.adapters.lazygit import render_theme as render_lazygit_theme
from sf2_theme.adapters.nvim import render_scheme as render_nvim_scheme
from sf2_theme.adapters.wezterm import render_scheme
from sf2_theme.catalog import get_theme, parse_catalog, theme_pair

SNAPSHOTS = Path(__file__).parent / "snapshots"


def _assert_snapshot(name: str, rendered: str) -> None:
    path = SNAPSHOTS / name
    assert path.is_file(), f"missing snapshot {path}"
    assert rendered == path.read_text(encoding="utf-8")


def test_main_wezterm_snapshot() -> None:
    theme = get_theme("main", parse_catalog())
    _assert_snapshot("wezterm-main.toml", render_scheme(theme))


def test_main_herdr_snapshot() -> None:
    catalog = parse_catalog()
    dark, light = theme_pair(get_theme("main", catalog), catalog)
    _assert_snapshot("herdr-main.toml", render_block(dark, light))


def test_ryu_wezterm_snapshot() -> None:
    theme = get_theme("ryu", parse_catalog())
    _assert_snapshot("wezterm-ryu.toml", render_scheme(theme))


def test_main_nvim_snapshot() -> None:
    theme = get_theme("main", parse_catalog())
    _assert_snapshot("nvim-main.lua", render_nvim_scheme(theme))


def test_ryu_light_nvim_snapshot() -> None:
    theme = get_theme("ryu-light", parse_catalog())
    _assert_snapshot("nvim-ryu-light.lua", render_nvim_scheme(theme))


def test_main_lazygit_snapshot() -> None:
    theme = get_theme("main", parse_catalog())
    _assert_snapshot("lazygit-main.yml", render_lazygit_theme(theme))
