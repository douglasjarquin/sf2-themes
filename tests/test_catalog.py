"""Catalog discovery, aliases, and default theme."""

import tomllib
from dataclasses import replace
from pathlib import Path

import pytest

from sf2_theme.catalog import get_theme
from sf2_theme.errors import ThemeError
from sf2_theme.model import HexColor, IntroducedIn, Theme, ThemeKind, ThemeMetadata, ThemeVariant
from test_validation import _canonical_theme

CATALOG_IDS = (
    "main",
    "main-light",
    "akuma-light",
    "akuma",
    "balrog-light",
    "balrog",
    "blanka-light",
    "blanka",
    "cammy-light",
    "cammy",
    "chun-li-light",
    "chun-li",
    "dee-jay-light",
    "dee-jay",
    "dhalsim-light",
    "dhalsim",
    "e-honda-light",
    "e-honda",
    "fei-long-light",
    "fei-long",
    "guile-light",
    "guile",
    "ken-light",
    "ken",
    "m-bison-light",
    "m-bison",
    "ryu-light",
    "ryu",
    "sagat-light",
    "sagat",
    "t-hawk-light",
    "t-hawk",
    "vega-light",
    "vega",
    "zangief-light",
    "zangief",
)


def _catalog_contract() -> tuple[Theme, ...]:
    paths = [Path("themes/main.toml"), Path("themes/main-light.toml")]
    paths.extend(sorted(Path("themes/characters").glob("*.toml")))
    palette = _canonical_theme()
    themes: list[Theme] = []
    for path in paths:
        raw = tomllib.loads(path.read_text(encoding="utf-8"))
        meta = raw["meta"]
        theme_id = meta["id"]
        metadata = ThemeMetadata(
            id=theme_id,
            display_name=meta["display_name"],
            kind=ThemeKind(meta["kind"]),
            introduced_in=IntroducedIn(meta["introduced_in"]),
            character=meta.get("character"),
            aliases=tuple(meta.get("aliases", ())),
            name=meta.get("character", meta["display_name"]),
            variant=ThemeVariant.LIGHT if theme_id.endswith("-light") else ThemeVariant.DARK,
            family="sf2",
            stage="catalog contract",
        )
        ui = palette.ui
        if metadata.variant is ThemeVariant.LIGHT:
            ui = replace(ui, background=HexColor("#f4f1eb"), foreground=HexColor("#272b33"))
        themes.append(replace(palette, metadata=metadata, ui=ui))
    return tuple(themes)


def test_main_is_default_and_has_legacy_alias() -> None:
    catalog = _catalog_contract()
    main = get_theme("main", catalog)
    assert get_theme("street-fighter-2", catalog) is main
    assert get_theme("street-fighter-ii-main", catalog) is main
    assert main.metadata.display_name == "Street Fighter II - Main"


def test_unknown_theme_errors() -> None:
    with pytest.raises(ThemeError, match="unknown theme"):
        get_theme("bison", _catalog_contract())


def test_catalog_has_thirty_six_themes_and_role_aliases() -> None:
    catalog = _catalog_contract()
    assert len(catalog) == 36
    assert tuple(theme.metadata.id for theme in catalog) == CATALOG_IDS
    assert get_theme("boxer", catalog).metadata.id == "balrog"
    assert get_theme("claw", catalog).metadata.id == "vega"
    assert get_theme("dictator", catalog).metadata.id == "m-bison"


def test_catalog_has_light_variant_for_every_dark_theme() -> None:
    catalog = _catalog_contract()
    by_id = {theme.metadata.id: theme for theme in catalog}
    dark_themes = tuple(theme for theme in catalog if not theme.metadata.id.endswith("-light"))

    assert len(catalog) == 36
    assert len(dark_themes) == 18
    for dark in dark_themes:
        light = by_id[f"{dark.metadata.id}-light"]
        assert light.metadata.display_name == f"{dark.metadata.display_name} Light"
        assert light.ui.background != dark.ui.background
        assert light.ui.foreground != dark.ui.foreground
