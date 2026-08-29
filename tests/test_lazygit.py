"""Lazygit rendering and filesystem behavior."""

from pathlib import Path

import pytest

from sf2_theme.adapters.lazygit import apply_lazygit, merge_config, render_theme
from sf2_theme.catalog import get_theme, parse_catalog
from sf2_theme.errors import ThemeError


def test_render_theme_covers_current_lazygit_schema() -> None:
    theme = get_theme("vega", parse_catalog())
    rendered = render_theme(theme)

    for key in (
        "activeBorderColor",
        "inactiveBorderColor",
        "searchingActiveBorderColor",
        "optionsTextColor",
        "selectedLineBgColor",
        "inactiveViewSelectedLineBgColor",
        "cherryPickedCommitFgColor",
        "cherryPickedCommitBgColor",
        "markedBaseCommitFgColor",
        "markedBaseCommitBgColor",
        "unstagedChangesColor",
        "defaultFgColor",
    ):
        assert rendered.count(f"    {key}:") == 1
    assert "  authorColors:\n    '*':" in rendered
    assert "      - bold" in rendered
    assert str(theme.ui.accent) in rendered
    assert str(theme.semantic.red) in rendered


def test_merge_config_preserves_unrelated_gui_and_author_settings() -> None:
    theme = get_theme("main", parse_catalog())
    existing = "gui:\n  sidePanelWidth: 0.3\n  authorColors:\n    Alice: '#fff'\n"

    merged = merge_config(existing, theme, adopt=False)

    assert "sidePanelWidth: 0.3" in merged
    assert "Alice: '#fff'" in merged
    assert merged.count("sf2-themes managed theme") == 2
    assert merged.count("sf2-themes: sf2-main") == 2


def test_merge_config_recognizes_sections_with_comments_and_trailing_whitespace() -> None:
    theme = get_theme("main", parse_catalog())
    existing = "gui: # ui settings\n  sidePanelWidth: 0.3\n"

    merged = merge_config(existing, theme, adopt=False)

    assert merged.count("gui:") == 1
    assert merged.count("theme:") == 1
    assert merged.count("authorColors:") == 1

    commented_theme = "gui:  \n  theme:   # custom\n    activeBorderColor: ['#ffffff']\n"
    with pytest.raises(ThemeError, match="--adopt"):
        merge_config(commented_theme, theme, adopt=False)


def test_merge_config_replaces_double_quoted_wildcard_author_entry() -> None:
    theme = get_theme("main", parse_catalog())
    existing = "gui:\n  authorColors:\n    Alice: '#fff'\n    \"*\": '#abc'\n"

    merged = merge_config(existing, theme, adopt=False)

    assert "Alice: '#fff'" in merged
    assert "\"*\": '#abc'" not in merged
    assert merged.count("'*':") == 1


def test_merge_config_requires_adopt_for_an_existing_theme() -> None:
    theme = get_theme("main", parse_catalog())
    existing = "gui:\n  theme:\n    activeBorderColor: ['#ffffff']\n"

    with pytest.raises(ThemeError, match="--adopt"):
        merge_config(existing, theme, adopt=False)


def test_apply_lazygit_is_idempotent_and_dry_run_is_non_mutating(tmp_path: Path) -> None:
    theme = get_theme("main", parse_catalog())
    themes = parse_catalog()

    first = apply_lazygit(
        theme,
        themes,
        config_dir=tmp_path,
        dry_run=False,
        follow_symlinks=False,
        adopt=False,
    )
    second = apply_lazygit(
        theme,
        themes,
        config_dir=tmp_path,
        dry_run=False,
        follow_symlinks=False,
        adopt=False,
    )
    assert all(result.action.value in {"created", "unchanged"} for result in first + second)
    assert not list(tmp_path.glob("*.bak.*"))

    dry_run_dir = tmp_path / "dry-run"
    apply_lazygit(
        theme,
        themes,
        config_dir=dry_run_dir,
        dry_run=True,
        follow_symlinks=False,
        adopt=False,
    )
    assert not dry_run_dir.exists()
