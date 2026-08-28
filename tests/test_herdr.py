"""Herdr managed-block integration."""

from pathlib import Path

import pytest

from sf2_theme.adapters.herdr import HERDR_TOKENS, apply_herdr, merge_theme, render_block
from sf2_theme.catalog import get_theme, parse_catalog
from sf2_theme.errors import ThemeError


def test_render_includes_every_token() -> None:
    theme = get_theme("main", parse_catalog())
    block = render_block(theme)
    assert 'name = "catppuccin"' in block
    for token, _, _ in HERDR_TOKENS:
        assert f"{token} =" in block
    assert "overlay0" in block and "mauve" in block and "peach" in block
    assert f'mauve = "{theme.ui.accent_secondary}"' in block
    assert f'accent = "{theme.ui.accent}"' in block


def test_light_variant_uses_latte_base() -> None:
    theme = get_theme("ryu-light", parse_catalog())
    assert 'name = "catppuccin-latte"' in render_block(theme)


def test_sidebar_rows_use_surfaces_not_selection_tint() -> None:
    theme = get_theme("vega", parse_catalog())
    block = render_block(theme)
    assert f'active_row_bg = "{theme.ui.surface}"' in block
    assert f'selection_bg = "{theme.ui.overlay}"' in block
    assert f'active_row_bg = "{theme.ui.selection_background}"' not in block


def test_overlay_tokens_are_muted_text_not_border() -> None:
    # Herdr uses overlay0/1 as dim fg; shared adapter.overlay0 is border chrome.
    theme = get_theme("vega", parse_catalog())
    block = render_block(theme)
    assert f'overlay0 = "{theme.ui.muted}"' in block
    assert f'overlay1 = "{theme.ui.subtle}"' in block
    assert f'overlay0 = "{theme.ui.border}"' not in block


def test_herdr_managed_identity_is_prefixed() -> None:
    theme = get_theme("main", parse_catalog())

    assert "# sf2-themes: sf2-main" in render_block(theme)


def test_preserves_unrelated_sections() -> None:
    theme = get_theme("main", parse_catalog())
    existing = '[ui]\ntheme = "follow-system"\n\n[keys]\nprefix = "ctrl+b"\n'
    merged = merge_theme(existing, theme, adopt=False)
    assert 'theme = "follow-system"' in merged
    assert 'prefix = "ctrl+b"' in merged
    assert "# >>> sf2-themes managed theme" in merged


def test_refuses_unmarked_theme_without_adopt() -> None:
    theme = get_theme("main", parse_catalog())
    existing = '[theme]\nname = "catppuccin"\n'
    with pytest.raises(ThemeError, match="--adopt"):
        merge_theme(existing, theme, adopt=False)


def test_adopt_replaces_unmarked_theme() -> None:
    theme = get_theme("main", parse_catalog())
    existing = '[ui]\nconfirm_close = true\n\n[theme]\nname = "dracula"\n'
    merged = merge_theme(existing, theme, adopt=True)
    assert "dracula" not in merged
    assert 'name = "catppuccin"' in merged
    assert "# >>> sf2-themes managed theme" in merged
    assert "confirm_close = true" in merged


def test_second_apply_replaces_marked_block() -> None:
    catalog = parse_catalog()
    main = get_theme("main", catalog)
    first = merge_theme("", main, adopt=False)
    second = merge_theme(first, main, adopt=False)
    assert second.count("# >>> sf2-themes managed theme") == 1


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
