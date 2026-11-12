"""Lazygit rendering and filesystem behavior."""

from pathlib import Path

from sf2_theme.adapters.lazygit import inspect_lazygit, merge_config, plan_lazygit, render_theme
from sf2_theme.catalog import get_theme, parse_catalog
from sf2_theme.filesystem import commit
from sf2_theme.plan import Ownership


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

    merged = merge_config(existing, theme)

    assert "sidePanelWidth: 0.3" in merged
    assert "Alice: '#fff'" in merged
    assert merged.count("sf2-themes managed theme") == 2
    assert merged.count("sf2-themes: sf2-main") == 2


def test_merge_config_recognizes_sections_with_comments_and_trailing_whitespace(tmp_path: Path) -> None:
    theme = get_theme("main", parse_catalog())
    existing = "gui: # ui settings\n  sidePanelWidth: 0.3\n"

    merged = merge_config(existing, theme)

    assert merged.count("gui:") == 1
    assert merged.count("theme:") == 1
    assert merged.count("authorColors:") == 1

    commented_theme = "gui:  \n  theme:   # custom\n    activeBorderColor: ['#ffffff']\n"
    config_dir = tmp_path / "lazygit"
    config_dir.mkdir()
    (config_dir / "config.yml").write_text(commented_theme, encoding="utf-8")
    assert inspect_lazygit(parse_catalog(), config_dir).ownership is Ownership.FOREIGN


def test_merge_config_replaces_double_quoted_wildcard_author_entry() -> None:
    theme = get_theme("main", parse_catalog())
    existing = "gui:\n  authorColors:\n    Alice: '#fff'\n    \"*\": '#abc'\n"

    merged = merge_config(existing, theme)

    assert "Alice: '#fff'" in merged
    assert "\"*\": '#abc'" not in merged
    assert merged.count("'*':") == 1


def test_merge_config_replaces_an_adopted_theme() -> None:
    theme = get_theme("main", parse_catalog())
    existing = "gui:\n  theme:\n    activeBorderColor: ['#ffffff']\n"

    merged = merge_config(existing, theme)

    assert "#ffffff" not in merged
    assert "# sf2-themes: sf2-main" in merged


def test_apply_lazygit_is_idempotent_and_dry_run_is_non_mutating(tmp_path: Path) -> None:
    theme = get_theme("main", parse_catalog())
    themes = parse_catalog()

    first = commit(
        plan_lazygit(
            theme,
            themes,
            config_dir=tmp_path,
            slot=True,
        )
    )
    second = commit(
        plan_lazygit(
            theme,
            themes,
            config_dir=tmp_path,
            slot=True,
        )
    )
    assert all(result.action.value in {"created", "unchanged"} for result in first + second)
    assert not list(tmp_path.glob("*.bak.*"))

    dry_run_dir = tmp_path / "dry-run"
    commit(
        plan_lazygit(
            theme,
            themes,
            config_dir=dry_run_dir,
            slot=True,
        ),
        dry_run=True,
    )
    assert not dry_run_dir.exists()
