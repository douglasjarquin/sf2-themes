"""Catalog discovery, aliases, and default theme."""

import pytest

from sf2_theme.catalog import get_theme, parse_catalog
from sf2_theme.errors import ThemeError


def test_main_is_default_and_has_legacy_alias() -> None:
    catalog = parse_catalog()
    main = get_theme("main", catalog)
    assert get_theme("street-fighter-2", catalog) is main
    assert get_theme("street-fighter-ii-main", catalog) is main
    assert main.metadata.display_name == "Street Fighter II - Main"


def test_unknown_theme_errors() -> None:
    with pytest.raises(ThemeError, match="unknown theme"):
        get_theme("bison", parse_catalog())


def test_catalog_has_eighteen_themes_and_role_aliases() -> None:
    catalog = parse_catalog()
    assert len(catalog) == 18
    assert get_theme("boxer", catalog).metadata.id == "balrog"
    assert get_theme("claw", catalog).metadata.id == "vega"
    assert get_theme("dictator", catalog).metadata.id == "m-bison"
