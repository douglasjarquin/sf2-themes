from sf2_theme.adapters.slack import render_theme
from sf2_theme.catalog import get_theme

KEN_LINE = "#221616,#de6767,#549657,#c76e67"
KEN_LIGHT_LINE = "#faeeed,#af3c40,#2d7633,#a94a46"
MAIN_LINE = "#131927,#ad8705,#4e975b,#c86e67"


def test_ken_renders_import_line() -> None:
    assert render_theme(get_theme("ken")) == KEN_LINE


def test_ken_light_renders_import_line() -> None:
    assert render_theme(get_theme("ken-light")) == KEN_LIGHT_LINE


def test_main_renders_import_line() -> None:
    assert render_theme(get_theme("main")) == MAIN_LINE


def test_import_line_is_one_lowercase_line_without_spaces() -> None:
    line = render_theme(get_theme("ken"))

    assert "\n" not in line
    assert " " not in line
    assert line == line.lower()
    assert line.count(",") == 3


def test_ken_selected_slot_is_accent_not_selection_background() -> None:
    theme = get_theme("ken")
    slots = render_theme(theme).split(",")

    assert slots[1] == theme.ui.accent
    assert slots[1] != theme.ui.selection_background


def test_import_line_is_not_the_ten_color_template() -> None:
    line = render_theme(get_theme("ken"))

    assert line.count(",") == 3
    assert "#f8f8fa" not in line


def test_ken_line_uses_four_different_colors() -> None:
    slots = render_theme(get_theme("ken")).split(",")

    assert len(slots) == 4
    assert len(set(slots)) == 4
