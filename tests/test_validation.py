"""Contrast, hex, and ANSI validation."""

from pathlib import Path

import pytest

from sf2_theme.catalog import parse_catalog
from sf2_theme.errors import ThemeError
from sf2_theme.model import HexColor, parse_hex
from sf2_theme.parse import parse_theme
from sf2_theme.validation import Severity, contrast_ratio, validate_catalog, validate_theme


def test_parse_hex_accepts_rrggbb_and_lowercases() -> None:
    assert parse_hex("#FFF4D6") == HexColor("#fff4d6")


@pytest.mark.parametrize("raw", ["#fff", "#ffffffff", "red", "fff4d6", "#GGG000"])
def test_parse_hex_rejects_non_rrggbb(raw: str) -> None:
    with pytest.raises(ThemeError, match="#RRGGBB"):
        parse_hex(raw)


def test_cream_on_navy_meets_primary_contrast() -> None:
    assert contrast_ratio(HexColor("#fff4d6"), HexColor("#101a3a")) >= 7.0


def test_cream_on_teal_fails_text_contrast() -> None:
    assert contrast_ratio(HexColor("#fff4d6"), HexColor("#18a6a6")) < 4.5


def test_navy_on_gold_passes_selection_contrast() -> None:
    assert contrast_ratio(HexColor("#101a3a"), HexColor("#f2b134")) >= 4.5


def test_main_theme_has_no_errors() -> None:
    themes = parse_catalog()
    main = next(theme for theme in themes if theme.metadata.id == "main")
    errors = [issue for issue in validate_theme(main) if issue.severity is Severity.ERROR]
    assert errors == []


def test_identical_bright_and_normal_is_an_error() -> None:
    raw = Path("themes/main.toml").read_text(encoding="utf-8")
    raw = raw.replace('black = "#7f89aa"', 'black = "#101a3a"', 1)
    theme = parse_theme(__import__("tomllib").loads(raw), source="dup.toml")
    messages = [issue.message for issue in validate_theme(theme) if issue.severity is Severity.ERROR]
    assert any("ansi.bright.black matches" in message for message in messages)


def test_duplicate_alias_is_an_error() -> None:
    themes = parse_catalog()
    cloned = themes[0]
    issues = validate_catalog((*themes, cloned))
    assert any(issue.severity is Severity.ERROR and "duplicate lookup key" in issue.message for issue in issues)
