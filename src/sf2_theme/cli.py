"""Command-line interface for the Street Fighter II theme pack."""

import argparse
import json
import sys
import tomllib
from collections import Counter
from collections.abc import Callable, Sequence
from dataclasses import dataclass
from enum import StrEnum, unique
from pathlib import Path

from sf2_theme import __version__
from sf2_theme.adapters.claude import inspect_claude, plan_claude
from sf2_theme.adapters.codex import inspect_codex, plan_codex
from sf2_theme.adapters.herdr import inspect_herdr, plan_herdr
from sf2_theme.adapters.herdr_app import inspect_herdr_app, plan_herdr_app
from sf2_theme.adapters.lazygit import inspect_lazygit, plan_lazygit
from sf2_theme.adapters.nvim import inspect_nvim, plan_nvim
from sf2_theme.adapters.slack import render_theme as slack_import_line
from sf2_theme.adapters.starship import inspect_starship, plan_starship
from sf2_theme.adapters.wezterm import inspect_wezterm, plan_wezterm
from sf2_theme.adapters.zsh_syntax import SOURCE_HINT
from sf2_theme.catalog import (
    catalog_issues,
    default_theme,
    get_theme,
    load_catalog,
    parse_catalog,
    show_theme,
    theme_pair,
)
from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import WriteAction, WriteResult, commit
from sf2_theme.model import Theme, ThemeVariant
from sf2_theme.plan import Inspection, Ownership, Refusal, Step
from sf2_theme.validation import validate_theme


@dataclass(frozen=True, slots=True)
class App:
    """How the CLI drives one configurable application."""

    label: str
    inspect: Callable[[Sequence[Theme], Path | None], Inspection]
    plan: Callable[[Theme, Sequence[Theme], Path | None, bool], list[Step]]
    pair: bool = False
    owns_files_on_unknown_shape: bool = False
    reload: str = ""
    stderr_hint: str = ""


def _plan_nvim(theme: Theme, themes: Sequence[Theme], config_dir: Path | None, _slot: bool) -> list[Step]:
    return plan_nvim(theme, themes, config_dir)


APPS: dict[str, App] = {
    "wezterm": App(
        "WezTerm",
        inspect_wezterm,
        plan_wezterm,
        pair=True,
        owns_files_on_unknown_shape=True,
        reload="restart WezTerm, or let a config reload pick up the managed pointer",
    ),
    "herdr": App(
        "Herdr",
        inspect_herdr,
        plan_herdr,
        pair=True,
        reload="run `herdr server reload-config` to apply the selection",
    ),
    "herdr-app": App(
        "Herdr app",
        inspect_herdr_app,
        plan_herdr_app,
        pair=True,
        reload="settings apply live; appearance, layout themes, and the theme picker can override the pair",
    ),
    "nvim": App("Neovim", inspect_nvim, _plan_nvim, pair=True, reload="restart Neovim to load the new colorscheme"),
    "codex": App("Codex", inspect_codex, plan_codex, reload="restart Codex, or reselect with /theme"),
    "starship": App(
        "Starship",
        inspect_starship,
        plan_starship,
        stderr_hint=f"Source the zsh highlight snippet after zsh-syntax-highlighting:\n  {SOURCE_HINT}\n",
    ),
    "lazygit": App("Lazygit", inspect_lazygit, plan_lazygit, reload="restart lazygit to render the new theme"),
    "claude": App(
        "Claude Code", inspect_claude, plan_claude, reload="reselect with /theme in a running Claude Code session"
    ),
}
APP_NAMES = (*APPS, "slack")

HELP_EPILOG = """\
notes:
  apply prepares theme assets, installs or repairs the app integration, and
  selects a theme (default: main) in one idempotent step.
  setup performs the same integration but keeps an existing selection unless
  --theme is given. install is a deprecated alias for apply.
  When an app already uses a theme sf2-themes does not manage, apply and
  setup --theme refuse unless --adopt is given; plain setup installs the
  theme files and keeps that theme.
  starship also refreshes ~/.config/sf2-theme/zsh-syntax-highlighting.zsh.
  Lazygit installs one YAML fragment per catalog theme under its themes directory.
  Herdr app installs JSON themes in ~/.config/herdr-app/themes/ and selects
  a light/dark pair in settings.json. Layout themes and the picker take precedence.
  WezTerm, Herdr TUI/app, and Neovim apply both dark and light siblings and auto-switch
  with host appearance. Claude installs one theme file per catalog entry under
  ~/.claude/themes/ and selects it in settings.json; select it or switch
  siblings yourself in /theme, since Claude Code has no host-appearance
  auto-switch.
  Slack prints a paste string and does not install an sf2- identity.
"""

_ACTION_TEXT = {
    WriteAction.CREATED: "created",
    WriteAction.UPDATED: "updated",
    WriteAction.UNCHANGED: "unchanged",
    WriteAction.WOULD_CREATE: "would create",
    WriteAction.WOULD_UPDATE: "would update",
    WriteAction.REMOVED: "removed",
    WriteAction.WOULD_REMOVE: "would remove",
}


@unique
class Selection(StrEnum):
    """How a run arrived at the theme it writes."""

    SELECTED = "selected"
    KEPT = "kept"
    NONE_FOUND = "none found"
    RESET = "reset"


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="sf2-themes",
        description=(
            "Street Fighter II theme pack for WezTerm, Herdr TUI, Herdr app, Neovim, "
            "Codex, Claude Code, Starship, Lazygit, and Slack."
        ),
        epilog=HELP_EPILOG,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--version", "-V", action="version", version=__version__)
    commands = parser.add_subparsers(dest="command", metavar="COMMAND")
    commands.add_parser("apps", help="list supported applications")
    commands.add_parser("themes", help="list catalog themes")
    show = commands.add_parser("show", help="print a theme's canonical TOML")
    show.add_argument("theme", help="theme id, alias, or display name")
    validate = commands.add_parser("validate", help="validate theme definitions")
    validate.add_argument("themes", nargs="*", metavar="THEME", help="catalog ids (default: the whole catalog)")
    validate.add_argument("--all", dest="all_themes", action="store_true", help="validate every catalog theme")
    mutation = (
        commands.add_parser(
            "setup", help="install or repair app integration (keeps an existing selection unless --theme is given)"
        ),
        commands.add_parser("apply", help="integrate the app and select a theme in one step"),
        commands.add_parser("install", help="deprecated alias for apply"),
    )
    for sub in mutation:
        _mutation_args(sub)
    current = commands.add_parser("current", help="print the selected theme id, or none")
    current.add_argument("app", choices=APP_NAMES)
    current.add_argument("--config-dir", type=Path, metavar="PATH")
    current.add_argument("--theme", help=argparse.SUPPRESS)
    return parser


def _mutation_args(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("app", choices=APP_NAMES, help="application to configure")
    parser.add_argument("--theme", metavar="THEME", help="theme id, alias, or display name (default: main)")
    parser.add_argument(
        "--config-dir", type=Path, metavar="PATH", help="write under PATH instead of the app's config location"
    )
    parser.add_argument("--dry-run", action="store_true", help="print the full plan without writing anything")
    parser.add_argument("--follow-symlinks", action="store_true", help="write through destination symlinks")
    parser.add_argument("--adopt", action="store_true", help="replace a theme setting sf2-themes does not manage")
    parser.add_argument("--verbose", "-v", action="store_true", help="print every file operation, not just the summary")


def _validate(ns: argparse.Namespace) -> int:
    catalog = parse_catalog()
    if ns.all_themes or not ns.themes:
        themes = catalog
        lines = catalog_issues(themes)
    else:
        themes = tuple(get_theme(name, catalog) for name in ns.themes)
        lines = []
        for theme in themes:
            for issue in validate_theme(theme):
                lines.append(f"{issue.severity.value}: {issue.message}")
    failed = False
    for line in lines:
        if line.startswith("error:"):
            print(line, file=sys.stderr)
            failed = True
        else:
            print(line)
    if not failed:
        labels = ", ".join(theme.metadata.id for theme in themes)
        print(f"valid: {labels}")
    return 1 if failed else 0


def _refusal(name: str, app: App, inspection: Inspection) -> Refusal:
    """Build the one refusal message for an ownership class."""
    match inspection.ownership:
        case Ownership.FOREIGN:
            message = (
                f"{app.label} already uses a theme sf2-themes does not manage; pass --adopt to replace it, "
                f"or run `setup {name}` without --theme to keep it and install the theme files"
            )
        case Ownership.UNKNOWN_SHAPE:
            message = (
                f"{app.label} config has a shape sf2-themes cannot edit safely.\n"
                f"{inspection.detail.rstrip()}\n\nThen run the command again."
            )
        case _:
            message = f"{app.label} config cannot be changed safely: {inspection.detail}"
    return Refusal(message, inspection.ownership)


def _select(
    requested: Theme | None, selects_theme: bool, inspection: Inspection, catalog: Sequence[Theme]
) -> tuple[Theme, Selection]:
    if selects_theme:
        return requested or default_theme(tuple(catalog)), Selection.SELECTED
    if inspection.current is not None:
        return inspection.current, Selection.KEPT
    return default_theme(tuple(catalog)), Selection.RESET if inspection.unreadable else Selection.NONE_FOUND


def _report(results: Sequence[WriteResult], *, verbose: bool) -> None:
    """Print the plan. Dry-run always lists planned writes; otherwise --verbose."""
    for result in results:
        planned = result.action in (WriteAction.WOULD_CREATE, WriteAction.WOULD_UPDATE, WriteAction.WOULD_REMOVE)
        if not (verbose or planned):
            continue
        print(f"{_ACTION_TEXT[result.action]}: {result.path}")
        if result.diff:
            print(result.diff, end="")


def _changes(results: Sequence[WriteResult]) -> str:
    counts = Counter(result.action for result in results)
    if all(action is WriteAction.UNCHANGED for action in counts):
        return "no changes"
    order = (
        WriteAction.CREATED,
        WriteAction.UPDATED,
        WriteAction.REMOVED,
        WriteAction.WOULD_CREATE,
        WriteAction.WOULD_UPDATE,
        WriteAction.WOULD_REMOVE,
        WriteAction.UNCHANGED,
    )
    return ", ".join(f"{counts[action]} {_ACTION_TEXT[action]}" for action in order if counts[action])


def _selection_text(app: App, theme: Theme, selection: Selection, catalog: Sequence[Theme]) -> str:
    if app.pair:
        dark, light = theme_pair(theme, catalog)
        label = f"{dark.metadata.selectable_id} + {light.metadata.selectable_id} (follows host appearance)"
    else:
        label = theme.metadata.selectable_id
    match selection:
        case Selection.SELECTED:
            return label
        case Selection.KEPT:
            return f"kept {label}"
        case Selection.NONE_FOUND:
            return f"no selection found, selected {label}"
        case Selection.RESET:
            return f"selection unreadable, reset to {label}"


def _summarize(
    verb: str,
    name: str,
    app: App,
    ownership: Ownership,
    slot: bool,
    selection: str,
    results: Sequence[WriteResult],
) -> None:
    if ownership is Ownership.UNKNOWN_SHAPE:
        print(f"{verb} {name}: incomplete (config left unchanged)")
    elif not slot:
        print(f"{verb} {name}: kept your own theme (config left unchanged)")
    else:
        print(f"{verb} {name}: {selection}")
    if not slot and app.owns_files_on_unknown_shape:
        print(f"  pointer: {selection}")
    print(f"  changes: {_changes(results)}")
    if slot and app.reload:
        print(f"  note: {app.reload}")


def _print_slack(theme: Theme) -> None:
    mode = "Light" if theme.metadata.variant is ThemeVariant.LIGHT else "Dark"
    print(slack_import_line(theme))
    print(
        "\n".join(
            (
                theme.metadata.id,
                f"Set Slack Color Mode to {mode}.",
                "Turn Darker sidebars off.",
                "Paste the line into Preferences, Appearance, Custom theme, Import theme.",
            )
        ),
        file=sys.stderr,
    )


def _mutate(command: str, ns: argparse.Namespace) -> int:
    if command == "install":
        print("warning: install is deprecated; use apply", file=sys.stderr)
    catalog = load_catalog()
    requested = None if ns.theme is None else get_theme(ns.theme, catalog)
    if ns.app == "slack":
        _print_slack(requested or default_theme(catalog))
        return 0
    app = APPS[ns.app]
    config_dir = ns.config_dir.expanduser() if ns.config_dir is not None else None
    selects_theme = command != "setup" or requested is not None
    inspection = app.inspect(catalog, config_dir)
    ownership = inspection.ownership
    keeps_foreign = ownership is Ownership.FOREIGN and not ns.adopt
    if (
        ownership is Ownership.BROKEN
        or (ownership is Ownership.UNKNOWN_SHAPE and not app.owns_files_on_unknown_shape)
        or (keeps_foreign and selects_theme)
    ):
        raise _refusal(ns.app, app, inspection)
    slot = ownership is not Ownership.UNKNOWN_SHAPE and not keeps_foreign
    theme, selection = _select(requested, selects_theme, inspection, catalog)
    results = commit(app.plan(theme, catalog, config_dir, slot), dry_run=ns.dry_run, follow_symlinks=ns.follow_symlinks)
    _report(results, verbose=ns.verbose)
    verb = "apply" if command == "install" else command
    _summarize(verb, ns.app, app, ownership, slot, _selection_text(app, theme, selection, catalog), results)
    if ownership is Ownership.UNKNOWN_SHAPE:
        raise _refusal(ns.app, app, inspection)
    if app.stderr_hint and slot:
        print(app.stderr_hint, file=sys.stderr)
    return 0


def _current(name: str, config_dir: Path | None) -> str:
    if name == "slack":
        raise ThemeError("no Slack theme to read")
    app = APPS[name]
    inspection = app.inspect(load_catalog(), config_dir)
    if inspection.current is not None:
        return inspection.current.metadata.selectable_id
    match inspection.ownership:
        case Ownership.BROKEN:
            raise ThemeError(inspection.detail)
        case Ownership.FOREIGN if not app.owns_files_on_unknown_shape:
            raise ThemeError(f"{app.label} uses a theme sf2-themes does not manage")
    if inspection.unreadable:
        raise ThemeError(f"could not read the sf2 theme {app.label} selects")
    raise ThemeError(f"no {app.label} theme applied yet")


def dispatch(arguments: list[str]) -> int:
    """Run one CLI command. Returns a process exit code."""
    parser = _build_parser()
    try:
        ns = parser.parse_args(arguments)
        if ns.command is None:
            parser.print_help()
            return 0
        match ns.command:
            case "apps":
                print("\n".join(APP_NAMES))
            case "themes":
                for theme in load_catalog():
                    print(f"{theme.metadata.id}\t{theme.metadata.display_name}")
            case "show":
                print(show_theme(get_theme(ns.theme)), end="")
            case "validate":
                return _validate(ns)
            case "current":
                print(_current(ns.app, ns.config_dir.expanduser() if ns.config_dir else None))
            case "setup" | "apply" | "install":
                return _mutate(ns.command, ns)
    except SystemExit as exit_:
        return int(exit_.code) if isinstance(exit_.code, int) else 0
    except (ThemeError, OSError, tomllib.TOMLDecodeError, json.JSONDecodeError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 1
    return 0


def main(arguments: list[str] | None = None) -> int:
    """CLI entry point."""
    return dispatch(sys.argv[1:] if arguments is None else arguments)
