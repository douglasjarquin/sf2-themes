"""Herdr managed-block integration."""

from pathlib import Path

import pytest

from sf2_theme.adapters.herdr import HERDR_TOKENS, apply_herdr, merge_theme, render_block
from sf2_theme.catalog import get_theme, parse_catalog
from sf2_theme.errors import ThemeError


def test_render_includes_every_token() -> None:
    theme = get_theme("main", parse_catalog())
    block = render_block(theme)
    assert 'name = "terminal"' in block
    for token, _, _ in HERDR_TOKENS:
        assert f"{token} =" in block
    assert "overlay0" in block and "mauve" in block and "peach" in block


def test_preserves_unrelated_sections() -> None:
    theme = get_theme("main", parse_catalog())
    existing = '[ui]\ntheme = "follow-system"\n\n[keys]\nprefix = "ctrl+b"\n'
    merged = merge_theme(existing, theme, adopt=False)
    assert 'theme = "follow-system"' in merged
    assert 'prefix = "ctrl+b"' in merged
    assert "# >>> sf2-theme managed theme" in merged


def test_refuses_unmarked_theme_without_adopt() -> None:
    theme = get_theme("main", parse_catalog())
    existing = '[theme]\nname = "catppuccin"\n'
    with pytest.raises(ThemeError, match="--adopt"):
        merge_theme(existing, theme, adopt=False)


def test_adopt_replaces_unmarked_theme() -> None:
    theme = get_theme("main", parse_catalog())
    existing = '[ui]\nconfirm_close = true\n\n[theme]\nname = "catppuccin"\n'
    merged = merge_theme(existing, theme, adopt=True)
    assert "catppuccin" not in merged
    assert 'name = "terminal"' in merged
    assert "confirm_close = true" in merged


def test_second_apply_replaces_marked_block() -> None:
    catalog = parse_catalog()
    main = get_theme("main", catalog)
    first = merge_theme("", main, adopt=False)
    second = merge_theme(first, main, adopt=False)
    assert second.count("# >>> sf2-theme managed theme") == 1


def test_apply_herdr_round_trip(tmp_path: Path) -> None:
    theme = get_theme("main", parse_catalog())
    config_dir = tmp_path / "herdr"
    config_dir.mkdir()
    (config_dir / "config.toml").write_text('[ui]\ntheme = "follow-system"\n', encoding="utf-8")
    apply_herdr(theme, config_dir=config_dir, dry_run=False, follow_symlinks=False, adopt=False)
    apply_herdr(theme, config_dir=config_dir, dry_run=False, follow_symlinks=False, adopt=False)
    text = (config_dir / "config.toml").read_text(encoding="utf-8")
    assert text.count("[theme.custom]") == 1
    assert 'theme = "follow-system"' in text
