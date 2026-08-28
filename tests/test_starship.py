"""Starship managed palette and zsh-syntax companion."""

from pathlib import Path

from sf2_theme.adapters.starship import apply_starship, merge_theme, render_block
from sf2_theme.adapters.zsh_syntax import apply_zsh_syntax, render_highlight
from sf2_theme.catalog import get_theme, parse_catalog
from sf2_theme.cli import dispatch


def test_starship_palette_maps_blue_to_accent() -> None:
    theme = get_theme("vega", parse_catalog())
    block = render_block(theme)
    assert 'palette = "sf2"' in block
    assert f'blue = "{theme.ui.accent}"' in block
    assert f'purple = "{theme.ui.accent_secondary}"' in block
    assert f'green = "{theme.semantic.green}"' in block


def test_starship_merge_preserves_user_styles() -> None:
    theme = get_theme("main", parse_catalog())
    existing = 'format = "$directory"\n\n[directory]\nstyle = "blue"\n'
    merged = merge_theme(existing, theme)
    assert 'style = "blue"' in merged
    assert merged.count("# >>> sf2-themes managed theme") == 1
    again = merge_theme(merged, theme)
    assert again.count("# >>> sf2-themes managed theme") == 1


def test_zsh_highlight_uses_accent_for_commands() -> None:
    theme = get_theme("vega", parse_catalog())
    snippet = render_highlight(theme)
    assert f"ZSH_HIGHLIGHT_STYLES[command]='fg={theme.ui.accent}'" in snippet
    assert f"ZSH_HIGHLIGHT_STYLES[arg0]='fg={theme.ui.accent}'" in snippet


def test_apply_starship_writes_palette_and_zsh(tmp_path: Path, monkeypatch) -> None:
    theme = get_theme("vega", parse_catalog())
    config_dir = tmp_path / "config"
    config_dir.mkdir()
    (config_dir / "starship.toml").write_text('[directory]\nstyle = "blue"\n', encoding="utf-8")
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path / "xdg"))
    apply_starship(theme, config_dir=config_dir, dry_run=False, follow_symlinks=False)
    apply_zsh_syntax(theme, dry_run=False, follow_symlinks=False)
    starship = (config_dir / "starship.toml").read_text(encoding="utf-8")
    assert f'blue = "{theme.ui.accent}"' in starship
    highlight = (tmp_path / "xdg" / "sf2-theme" / "zsh-syntax-highlighting.zsh").read_text(
        encoding="utf-8"
    )
    assert theme.ui.accent in highlight


def test_apps_lists_starship(capsys) -> None:
    assert dispatch(["apps"]) == 0
    assert "starship" in capsys.readouterr().out
