"""Contrast, hex, and ANSI validation."""

import tomllib
from copy import deepcopy
from dataclasses import fields, replace

import pytest

import sf2_theme.model as theme_model
from sf2_theme.errors import ThemeError
from sf2_theme.model import UI_COLOR_FIELDS, HexColor, Theme, UiColors, parse_hex, theme_to_toml
from sf2_theme.parse import parse_theme
from sf2_theme.validation import Severity, contrast_ratio, validate_catalog, validate_theme


def _canonical_theme_raw() -> dict[str, object]:
    return tomllib.loads(
        """
schema_version = 1

[meta]
id = "ryu"
display_name = "Street Fighter II - Ryu"
kind = "character"
introduced_in = "world-warrior"
character = "Ryu"
aliases = []
name = "Ryu"
variant = "dark"
family = "sf2"
stage = "Japan - Suzaku Castle rooftop"

[ui]
background = "#141a23"
surface = "#1f252f"
overlay = "#2b333f"
border = "#2b333f"
foreground = "#cad2df"
muted = "#7b8598"
subtle = "#9ba7bd"
accent = "#da6a6a"
accent_secondary = "#5b8ac6"
cursor = "#da6a6a"
cursor_text = "#141a23"
selection_background = "#522d2c"
selection_foreground = "#cad2df"

[semantic]
red = "#c86e6c"
green = "#5e9465"
yellow = "#9e8625"
blue = "#558acc"
magenta = "#a278b5"
cyan = "#1195ad"
orange = "#c2753c"

[ansi.normal]
black = "#2b323f"
red = "#c86e6c"
green = "#5e9465"
yellow = "#9e8625"
blue = "#558acc"
magenta = "#a278b5"
cyan = "#1195ad"
white = "#aeb6c2"

[ansi.bright]
black = "#7b8598"
red = "#ee8582"
green = "#70b27a"
yellow = "#bda131"
blue = "#69a6f3"
magenta = "#c291d9"
cyan = "#1bb2ce"
white = "#d6dce5"
"""
    )


def _canonical_theme() -> Theme:
    return parse_theme(_canonical_theme_raw(), source="canonical.toml")


def test_parse_theme_retains_revised_accent_secondary() -> None:
    # Given: a complete canonical theme carrying the revised secondary accent.
    raw = _canonical_theme_raw()

    # When: the TOML boundary parses the theme.
    theme = parse_theme(raw, source="canonical.toml")

    # Then: the revised field remains first-class canonical data.
    assert theme.ui.accent_secondary == HexColor("#5b8ac6")


def test_adapter_projection_derives_current_roles() -> None:
    # Given: a complete canonical theme with independently recognizable colors.
    projection_factory = theme_model.project_adapter_colors
    theme = parse_theme(_canonical_theme_raw(), source="canonical.toml")

    # When: the central adapter projection is requested.
    projection = projection_factory(theme.ui)

    # Then: current operational roles are deterministic derived values.
    assert projection.panel_bg == HexColor("#1f252f")
    assert projection.cursor_bg == HexColor("#da6a6a")
    assert projection.selection_bg == HexColor("#522d2c")


def test_adapter_projection_covers_every_current_role() -> None:
    # Given: a canonical palette whose source colors are independently recognizable.
    theme = _canonical_theme()

    # When: its current adapter roles are projected centrally.
    projection = theme_model.project_adapter_colors(theme.ui)

    # Then: every current role has one documented deterministic source.
    assert projection == theme_model.AdapterColors(
        cursor_bg=HexColor("#da6a6a"),
        cursor_fg=HexColor("#141a23"),
        selection_bg=HexColor("#522d2c"),
        selection_fg=HexColor("#cad2df"),
        panel_bg=HexColor("#1f252f"),
        sidebar_bg=HexColor("#141a23"),
        active_row_bg=HexColor("#522d2c"),
        navigate_row_bg=HexColor("#2b333f"),
        surface_dim=HexColor("#141a23"),
        surface0=HexColor("#1f252f"),
        surface1=HexColor("#2b333f"),
        overlay0=HexColor("#2b333f"),
        overlay1=HexColor("#7b8598"),
        subtext=HexColor("#9ba7bd"),
    )


def test_canonical_toml_round_trips_in_source_and_model_order() -> None:
    # Given: a parsed theme containing every canonical metadata and palette field.
    theme = _canonical_theme()

    # When: the model is serialized and parsed through the TOML boundary again.
    serialized = theme_to_toml(theme)
    serialized_raw = tomllib.loads(serialized)
    round_tripped = parse_theme(serialized_raw, source="round-trip.toml")

    # Then: no value is lost and canonical field order remains stable.
    assert round_tripped == theme
    assert tuple(serialized_raw["meta"]) == (
        "id",
        "display_name",
        "kind",
        "introduced_in",
        "character",
        "aliases",
        "name",
        "variant",
        "family",
        "stage",
    )
    assert tuple(serialized_raw["ui"]) == UI_COLOR_FIELDS
    assert tuple(field.name for field in fields(UiColors)) == UI_COLOR_FIELDS


@pytest.mark.parametrize(
    ("role", "source_field"),
    (
        ("cursor_bg", "cursor"),
        ("cursor_fg", "cursor_text"),
        ("selection_bg", "selection_background"),
        ("selection_fg", "selection_foreground"),
        ("panel_bg", "surface"),
        ("sidebar_bg", "background"),
        ("active_row_bg", "selection_background"),
        ("navigate_row_bg", "overlay"),
        ("surface_dim", "background"),
        ("surface0", "surface"),
        ("surface1", "overlay"),
        ("overlay0", "border"),
        ("overlay1", "muted"),
        ("subtext", "subtle"),
    ),
)
def test_projection_role_tracks_its_canonical_source(role: str, source_field: str) -> None:
    # Given: a canonical palette with one independently changed source token.
    theme = _canonical_theme()
    revised = replace(theme.ui, **{source_field: HexColor("#123456")})

    # When: an owning canonical source token changes.
    projection = theme_model.project_adapter_colors(revised)

    # Then: the named projection follows that source without a driftable copy.
    assert role not in {field.name for field in fields(UiColors)}
    assert getattr(projection, role) == HexColor("#123456")


@pytest.mark.parametrize(
    ("section", "field"),
    (
        ("meta", "name"),
        ("meta", "variant"),
        ("meta", "family"),
        ("meta", "stage"),
        ("meta", "aliases"),
        *(("ui", field) for field in UI_COLOR_FIELDS),
    ),
)
def test_parse_theme_names_each_missing_revised_field(section: str, field: str) -> None:
    # Given: a canonical fixture missing exactly one newly required revised field.
    raw = deepcopy(_canonical_theme_raw())
    table = raw[section]
    assert isinstance(table, dict)
    table.pop(field)

    # When: the malformed source crosses the parser boundary.
    # Then: its source-qualified field name is the binary failure observable.
    with pytest.raises(ThemeError, match=rf"missing malformed\.toml\.{section}\.{field}$"):
        parse_theme(raw, source="malformed.toml")


@pytest.mark.parametrize("field", ("name", "family", "stage"))
def test_parse_theme_rejects_blank_designer_metadata(field: str) -> None:
    # Given: a complete fixture with one blank designer metadata value.
    raw = deepcopy(_canonical_theme_raw())
    meta = raw["meta"]
    assert isinstance(meta, dict)
    meta[field] = "  "

    # When: the malformed source crosses the parser boundary.
    # Then: the exact source-qualified field fails closed.
    with pytest.raises(ThemeError, match=rf"malformed\.toml\.meta\.{field} must not be empty$"):
        parse_theme(raw, source="malformed.toml")


def test_parse_theme_rejects_unknown_revised_field() -> None:
    # Given: the canonical UI table with one obsolete projection field added.
    raw = deepcopy(_canonical_theme_raw())
    ui = raw["ui"]
    assert isinstance(ui, dict)
    ui["panel_bg"] = "#123456"

    # When: the expanded canonical boundary parses the source.
    # Then: the unknown field is rejected by its source-qualified name.
    with pytest.raises(ThemeError, match=r"unknown malformed\.toml\.ui keys: panel_bg$"):
        parse_theme(raw, source="malformed.toml")


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
    errors = [issue for issue in validate_theme(_canonical_theme()) if issue.severity is Severity.ERROR]
    assert errors == []


def test_identical_bright_and_normal_is_an_error() -> None:
    theme = _canonical_theme()
    theme = replace(theme, ansi_bright=replace(theme.ansi_bright, black=theme.ansi_normal.black))
    messages = [issue.message for issue in validate_theme(theme) if issue.severity is Severity.ERROR]
    assert any("ansi.bright.black matches" in message for message in messages)


def test_semantic_slot_must_match_its_ansi_meaning() -> None:
    # Given: a semantic error color drifted away from ANSI red.
    theme = _canonical_theme()
    malformed = replace(theme, semantic=replace(theme.semantic, red=HexColor("#123456")))

    # When: theme validation checks stable semantic meanings.
    messages = [issue.message for issue in validate_theme(malformed) if issue.severity is Severity.ERROR]

    # Then: the mismatched named meaning fails closed.
    assert "ryu: semantic.red must match ansi.normal.red" in messages


def test_ansi_chromatic_slot_requires_text_contrast() -> None:
    # Given: one ANSI chromatic slot with unreadable background contrast.
    theme = _canonical_theme()
    malformed = replace(theme, ansi_normal=replace(theme.ansi_normal, red=HexColor("#151b24")))

    # When: ANSI validation runs.
    messages = [issue.message for issue in validate_theme(malformed) if issue.severity is Severity.ERROR]

    # Then: the exact row and slot own the contrast failure.
    assert any(message.startswith("ryu: ansi.normal.red/background contrast") for message in messages)


def test_duplicate_alias_is_an_error() -> None:
    theme = _canonical_theme()
    issues = validate_catalog((theme, theme))
    assert any(issue.severity is Severity.ERROR and "duplicate lookup key" in issue.message for issue in issues)
