"""Herdr managed-block integration."""

from pathlib import Path

import pytest

from sf2_theme.adapters.herdr import HERDR_TOKENS, apply_herdr, merge_theme, render_block
from sf2_theme.catalog import get_theme, parse_catalog, theme_pair
from sf2_theme.errors import ThemeError


def _assert_stable_custom_table(block: str) -> None:
    assert any(line.strip() == "[theme.custom]" for line in block.splitlines())
    assert "[theme.custom.dark]" not in block
    assert "[theme.custom.light]" not in block


def test_render_includes_every_token() -> None:
    catalog = parse_catalog()
    dark, light = theme_pair(get_theme("main", catalog), catalog)
    block = render_block(dark, light)
    _assert_stable_custom_table(block)
    assert 'name = "catppuccin"' in block
    assert "auto_switch = true" in block
    assert 'light_name = "catppuccin-latte"' in block
    assert 'dark_name = "catppuccin"' in block
    for token, _, _ in HERDR_TOKENS:
        assert f"{token} =" in block
    assert "overlay0" in block and "mauve" in block and "peach" in block
    assert f'mauve = "{dark.ui.accent_secondary}"' in block
    assert f'accent = "{dark.ui.accent}"' in block


def test_pair_uses_latte_base_for_light_and_mocha_for_dark() -> None:
    catalog = parse_catalog()
    dark, light = theme_pair(get_theme("ryu-light", catalog), catalog)
    block = render_block(dark, light)
    _assert_stable_custom_table(block)
    assert 'name = "catppuccin"' in block
    assert 'light_name = "catppuccin-latte"' in block
    assert 'dark_name = "catppuccin"' in block
    assert "# sf2-themes: sf2-ryu" in block


def test_custom_overlay_uses_dark_sibling_on_stable_herdr() -> None:
    catalog = parse_catalog()
    dark, light = theme_pair(get_theme("vega", catalog), catalog)
    block = render_block(dark, light)
    _assert_stable_custom_table(block)
    assert f'sidebar_bg = "{dark.ui.background}"' in block
    assert f'text = "{dark.ui.foreground}"' in block
    assert f'sidebar_bg = "{light.ui.background}"' not in block
    assert f'text = "{light.ui.foreground}"' not in block


def test_sidebar_rows_use_surfaces_not_selection_tint() -> None:
    catalog = parse_catalog()
    dark, light = theme_pair(get_theme("vega", catalog), catalog)
    block = render_block(dark, light)
    assert f'active_row_bg = "{dark.ui.surface}"' in block
    assert f'selection_bg = "{dark.ui.overlay}"' in block
    assert f'active_row_bg = "{dark.ui.selection_background}"' not in block


def test_overlay_tokens_are_muted_text_not_border() -> None:
    # Herdr uses overlay0/1 as dim fg; shared adapter.overlay0 is border chrome.
    catalog = parse_catalog()
    dark, light = theme_pair(get_theme("vega", catalog), catalog)
    block = render_block(dark, light)
    assert f'overlay0 = "{dark.ui.muted}"' in block
    assert f'overlay1 = "{dark.ui.subtle}"' in block
    assert f'overlay0 = "{dark.ui.border}"' not in block


def test_herdr_managed_identity_is_family_dark_id() -> None:
    catalog = parse_catalog()
    dark, light = theme_pair(get_theme("main-light", catalog), catalog)
    assert "# sf2-themes: sf2-main" in render_block(dark, light)


def test_selecting_light_or_dark_writes_the_same_pair() -> None:
    catalog = parse_catalog()
    from_dark = merge_theme("", get_theme("chun-li", catalog), catalog, adopt=False)
    from_light = merge_theme("", get_theme("chun-li-light", catalog), catalog, adopt=False)
    assert from_dark == from_light
    _assert_stable_custom_table(from_dark)
    assert "auto_switch = true" in from_dark
    assert "# sf2-themes: sf2-chun-li" in from_dark


def test_preserves_unrelated_sections() -> None:
    catalog = parse_catalog()
    theme = get_theme("main", catalog)
    existing = '[ui]\ntheme = "follow-system"\n\n[keys]\nprefix = "ctrl+b"\n'
    merged = merge_theme(existing, theme, catalog, adopt=False)
    assert 'theme = "follow-system"' in merged
    assert 'prefix = "ctrl+b"' in merged
    assert "# >>> sf2-themes managed theme" in merged
    assert "auto_switch = true" in merged
    _assert_stable_custom_table(merged)


def test_refuses_unmarked_theme_without_adopt() -> None:
    catalog = parse_catalog()
    theme = get_theme("main", catalog)
    existing = '[theme]\nname = "catppuccin"\n'
    with pytest.raises(ThemeError, match="--adopt"):
        merge_theme(existing, theme, catalog, adopt=False)


def test_adopt_replaces_unmarked_theme() -> None:
    catalog = parse_catalog()
    theme = get_theme("main", catalog)
    existing = '[ui]\nconfirm_close = true\n\n[theme]\nname = "dracula"\n'
    merged = merge_theme(existing, theme, catalog, adopt=True)
    assert "dracula" not in merged
    assert 'name = "catppuccin"' in merged
    assert "auto_switch = true" in merged
    assert "# >>> sf2-themes managed theme" in merged
    assert "confirm_close = true" in merged
    _assert_stable_custom_table(merged)


def test_second_apply_replaces_marked_block() -> None:
    catalog = parse_catalog()
    main = get_theme("main", catalog)
    first = merge_theme("", main, catalog, adopt=False)
    second = merge_theme(first, main, catalog, adopt=False)
    assert second.count("# >>> sf2-themes managed theme") == 1
    assert sum(line.strip() == "[theme.custom]" for line in second.splitlines()) == 1
    _assert_stable_custom_table(second)


def test_apply_herdr_round_trip(tmp_path: Path) -> None:
    catalog = parse_catalog()
    theme = get_theme("main", catalog)
    config_dir = tmp_path / "herdr"
    config_dir.mkdir()
    (config_dir / "config.toml").write_text('[ui]\ntheme = "follow-system"\n', encoding="utf-8")
    apply_herdr(theme, catalog, config_dir=config_dir, dry_run=False, follow_symlinks=False, adopt=False)
    apply_herdr(theme, catalog, config_dir=config_dir, dry_run=False, follow_symlinks=False, adopt=False)
    text = (config_dir / "config.toml").read_text(encoding="utf-8")
    _assert_stable_custom_table(text)
    assert sum(line.strip() == "[theme.custom]" for line in text.splitlines()) == 1
    assert 'theme = "follow-system"' in text
    assert "auto_switch = true" in text
