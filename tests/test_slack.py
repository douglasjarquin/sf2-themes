import re
from pathlib import Path

from sf2_theme.adapters.slack import render_theme
from sf2_theme.catalog import get_theme
from sf2_theme.cli import dispatch

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


def test_apps_lists_slack(capsys) -> None:
    assert dispatch(["apps"]) == 0

    assert "slack" in capsys.readouterr().out.splitlines()


def test_help_says_slack_prints_a_paste_string(capsys) -> None:
    assert dispatch(["--help"]) == 0

    assert "Slack prints a paste string and does not install an sf2- identity." in capsys.readouterr().out


def test_apply_slack_ken_prints_the_import_line(capsys) -> None:
    assert dispatch(["apply", "slack", "--theme", "ken"]) == 0
    captured = capsys.readouterr()

    assert captured.out == f"{KEN_LINE}\n"
    assert "ken" in captured.err
    assert "sf2-ken" not in captured.err
    assert re.search(r"\bDark\b", captured.err)
    assert "Darker sidebars off" in captured.err
    assert "Appearance" in captured.err
    assert "Import theme" in captured.err


def test_apply_slack_ken_light_says_light_and_sidebars_off(capsys) -> None:
    assert dispatch(["apply", "slack", "--theme", "ken-light"]) == 0
    captured = capsys.readouterr()

    assert captured.out == f"{KEN_LIGHT_LINE}\n"
    assert re.search(r"\bLight\b", captured.err)
    assert "Darker sidebars off" in captured.err


def test_apply_slack_defaults_to_main(capsys) -> None:
    assert dispatch(["apply", "slack"]) == 0
    captured = capsys.readouterr()

    assert captured.out == f"{MAIN_LINE}\n"
    assert re.search(r"\bDark\b", captured.err)


def test_setup_slack_matches_apply_stdout(capsys) -> None:
    assert dispatch(["setup", "slack", "--theme", "ken"]) == 0
    setup_out = capsys.readouterr().out
    assert dispatch(["apply", "slack", "--theme", "ken"]) == 0

    assert capsys.readouterr().out == setup_out == f"{KEN_LINE}\n"


def test_apply_slack_dry_run_creates_no_file(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "slack"

    assert dispatch(["apply", "slack", "--theme", "ken", "--dry-run", "--config-dir", str(config_dir)]) == 0

    assert capsys.readouterr().out == f"{KEN_LINE}\n"
    assert not config_dir.exists()


def test_current_slack_fails_closed(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "slack"

    assert dispatch(["current", "slack", "--theme", "ken", "--config-dir", str(config_dir)]) == 1
    captured = capsys.readouterr()

    assert captured.out == ""
    assert "unsupported app" not in captured.err
    assert "ken" not in captured.err
    assert "sf2-" not in captured.err
    assert not config_dir.exists()


def test_apply_slack_unknown_theme_uses_existing_error(capsys) -> None:
    assert dispatch(["apply", "slack", "--theme", "no-such"]) == 1

    assert "unknown theme: no-such" in capsys.readouterr().err


def test_install_slack_warns_and_prints_only_the_color_line(capsys) -> None:
    assert dispatch(["install", "slack", "--theme", "ken"]) == 0
    captured = capsys.readouterr()

    assert captured.out == f"{KEN_LINE}\n"
    assert "deprecated" in captured.err


def test_apply_slack_ignores_adopt_and_config_dir(tmp_path: Path, capsys) -> None:
    config_dir = tmp_path / "slack"

    assert dispatch(["apply", "slack", "--theme", "ken", "--adopt", "--config-dir", str(config_dir)]) == 0

    assert capsys.readouterr().out == f"{KEN_LINE}\n"
    assert not config_dir.exists()
