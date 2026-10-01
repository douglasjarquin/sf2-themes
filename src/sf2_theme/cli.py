"""Command-line interface for the Street Fighter II theme pack."""

import argparse
import json
import sys
import tomllib
from collections import Counter
from collections.abc import Sequence
from pathlib import Path

from sf2_theme import __version__
from sf2_theme.adapters.claude import read_current_id as claude_current
from sf2_theme.adapters.claude import setup_claude
from sf2_theme.adapters.codex import read_current_id as codex_current
from sf2_theme.adapters.codex import setup_codex
from sf2_theme.adapters.herdr import apply_herdr
from sf2_theme.adapters.herdr import read_current_id as herdr_current
from sf2_theme.adapters.lazygit import read_current_id as lazygit_current
from sf2_theme.adapters.lazygit import setup_lazygit
from sf2_theme.adapters.nvim import current_pointer_path as nvim_pointer_path
from sf2_theme.adapters.nvim import read_current_id as nvim_current
from sf2_theme.adapters.nvim import setup_nvim
from sf2_theme.adapters.starship import apply_starship
from sf2_theme.adapters.starship import read_current_id as starship_current
from sf2_theme.adapters.wezterm import current_pointer_path as wezterm_pointer_path
from sf2_theme.adapters.wezterm import read_current_id as wezterm_current
from sf2_theme.adapters.wezterm import setup_wezterm
from sf2_theme.adapters.zsh_syntax import SOURCE_HINT, apply_zsh_syntax
from sf2_theme.catalog import (
    catalog_issues,
    default_theme,
    get_theme,
    installed_theme,
    load_catalog,
    parse_catalog,
    show_theme,
    theme_pair,
)
from sf2_theme.errors import CliError, ThemeError
from sf2_theme.filesystem import WriteAction, WriteResult
from sf2_theme.model import Theme
from sf2_theme.validation import validate_theme

APP_NAMES = ("wezterm", "herdr", "nvim", "codex", "starship", "lazygit", "claude")
ADOPT_APPS = ("wezterm", "herdr", "lazygit")
# Apps whose selection is appearance-aware: both siblings are installed and the
# managed identity follows the host's light/dark mode.
PAIR_APPS = ("wezterm", "herdr", "nvim")
# Apps whose adapter always writes the theme it is given, so setup without
# --theme preserves by re-resolving the managed selection before dispatch.
# The other adapters preserve internally through their replace_* flags.
RESOLVED_PRESERVE_APPS = ("herdr", "starship", "lazygit")

HELP_EPILOG = """\
notes:
  apply prepares theme assets, installs or repairs the app integration, and
  selects a theme (default: main) in one idempotent step.
  setup performs the same integration but keeps an existing selection unless
  --theme is given. install is a deprecated alias for apply.
  starship also refreshes ~/.config/sf2-theme/zsh-syntax-highlighting.zsh.
  Lazygit installs one YAML fragment per catalog theme under its themes directory.
  WezTerm, Herdr, and Neovim apply both dark and light siblings and auto-switch
  with host appearance. Claude installs one theme file per catalog entry under
  ~/.claude/themes/ and selects it in settings.json; select it or switch
  siblings yourself in /theme, since Claude Code has no host-appearance
  auto-switch.
"""

_ACTION_TEXT = {
    WriteAction.CREATED: "created",
    WriteAction.UPDATED: "updated",
    WriteAction.UNCHANGED: "unchanged",
    WriteAction.WOULD_CREATE: "would create",
    WriteAction.WOULD_UPDATE: "would update",
}

# What the user still has to do by hand after a completed apply.
_RELOAD_NOTES = {
    "wezterm": "restart WezTerm, or let a config reload pick up the managed pointer",
    "herdr": "run `herdr server reload-config` to apply the selection",
    "nvim": "restart Neovim to load the new colorscheme",
    "codex": "restart Codex, or reselect with /theme",
    "claude": "reselect with /theme in a running Claude Code session",
    "lazygit": "restart lazygit to render the new theme",
}


def _build_parser() -> tuple[argparse.ArgumentParser, dict[str, argparse.ArgumentParser]]:
    parser = argparse.ArgumentParser(
        prog="sf2-themes",
        description=(
            "Street Fighter II theme pack for WezTerm, Herdr, Neovim, Codex, Claude Code, Starship, and Lazygit."
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
    mutation: dict[str, argparse.ArgumentParser] = {}
    mutation["setup"] = commands.add_parser(
        "setup", help="install or repair app integration (keeps an existing selection unless --theme is given)"
    )
    mutation["apply"] = commands.add_parser("apply", help="integrate the app and select a theme in one step")
    mutation["install"] = commands.add_parser("install", help="deprecated alias for apply")
    for sub in mutation.values():
        _mutation_args(sub)
    current = commands.add_parser("current", help="print the selected theme id, or none")
    current.add_argument("app", choices=APP_NAMES)
    current.add_argument("--config-dir", type=Path, metavar="PATH")
    return parser, mutation


def _mutation_args(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("app", choices=APP_NAMES, help="application to configure")
    parser.add_argument("--theme", metavar="THEME", help="theme id, alias, or display name (default: main)")
    parser.add_argument(
        "--config-dir", type=Path, metavar="PATH", help="write under PATH instead of the app's config location"
    )
    parser.add_argument("--dry-run", action="store_true", help="print the full plan without writing anything")
    parser.add_argument("--follow-symlinks", action="store_true", help="write through destination symlinks")
    parser.add_argument(
        "--adopt", action="store_true", help=f"replace an unmanaged theme setting ({', '.join(ADOPT_APPS)} only)"
    )
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


def _starship_results(theme: Theme, ns: argparse.Namespace, config_dir: Path | None) -> list[WriteResult]:
    return [
        apply_starship(theme, config_dir=config_dir, dry_run=ns.dry_run, follow_symlinks=ns.follow_symlinks),
        apply_zsh_syntax(theme, dry_run=ns.dry_run, follow_symlinks=ns.follow_symlinks),
    ]


def _report(results: Sequence[WriteResult], *, verbose: bool) -> None:
    """Print the plan. Dry-run always lists planned writes; otherwise --verbose."""
    for result in results:
        planned = result.action in (WriteAction.WOULD_CREATE, WriteAction.WOULD_UPDATE)
        if not (verbose or planned):
            continue
        print(f"{_ACTION_TEXT[result.action]}: {result.path}")
        if result.diff:
            print(result.diff, end="")


def _summarize(
    command: str,
    app: str,
    select: bool,
    theme: Theme,
    catalog: Sequence[Theme],
    results: Sequence[WriteResult],
    *,
    kept: bool,
    blocked: bool,
) -> None:
    counts = Counter(result.action for result in results)
    changed = sum(
        counts[action]
        for action in (WriteAction.CREATED, WriteAction.UPDATED, WriteAction.WOULD_CREATE, WriteAction.WOULD_UPDATE)
    )
    if changed:
        order = (
            WriteAction.CREATED,
            WriteAction.UPDATED,
            WriteAction.WOULD_CREATE,
            WriteAction.WOULD_UPDATE,
            WriteAction.UNCHANGED,
        )
        changes = ", ".join(f"{counts[action]} {_ACTION_TEXT[action]}" for action in order if counts[action])
    else:
        changes = "no changes"
    verb = "apply" if command == "install" else command
    if blocked:
        print(f"{verb} {app}: incomplete (config left unchanged)")
        print(f"  changes: {changes}")
        print("  note: see stderr for the pasteable integration snippet")
        return
    if select or not kept:
        if app in PAIR_APPS:
            dark, light = theme_pair(theme, catalog)
            selection = f"{dark.metadata.selectable_id} + {light.metadata.selectable_id} (follows host appearance)"
        else:
            selection = theme.metadata.selectable_id
        print(f"{verb} {app}: {selection}")
    else:
        print(f"{verb} {app}: existing selection kept")
    print(f"  changes: {changes}")
    note = _RELOAD_NOTES.get(app)
    if note:
        print(f"  note: {note}")


def _mutate(command: str, ns: argparse.Namespace, parser: argparse.ArgumentParser) -> int:
    if command == "install":
        print("warning: install is deprecated; use apply", file=sys.stderr)
    if ns.adopt and ns.app not in ADOPT_APPS:
        parser.error(f"{ns.app} does not support --adopt")
    catalog = load_catalog()
    theme = default_theme(catalog) if ns.theme is None else get_theme(ns.theme, catalog)
    config_dir = ns.config_dir.expanduser() if ns.config_dir is not None else None
    # Every mutation command runs one complete plan: assets plus integration.
    # apply (and install) always (re)select the resolved theme; setup keeps a
    # managed selection unless --theme names one.
    select = command != "setup" or ns.theme is not None
    kept = False
    if not select:
        kept = _selection_exists(ns.app, config_dir)
        if kept and ns.app in RESOLVED_PRESERVE_APPS:
            try:
                theme = installed_theme(_current_id(ns.app, config_dir), catalog)
            except ThemeError:
                kept = False
    manual: str | None = None
    adoptable = False
    match ns.app:
        case "wezterm":
            results, lua = setup_wezterm(
                theme,
                catalog,
                config_dir=config_dir,
                dry_run=ns.dry_run,
                follow_symlinks=ns.follow_symlinks,
                adopt=ns.adopt,
                replace_pointer=select,
            )
            manual = lua.snippet
            adoptable = lua.adoptable
        case "nvim":
            results = setup_nvim(
                theme,
                catalog,
                config_dir=config_dir,
                dry_run=ns.dry_run,
                follow_symlinks=ns.follow_symlinks,
                replace_pointer=select,
            )
        case "codex":
            results = setup_codex(
                theme,
                catalog,
                config_dir=config_dir,
                dry_run=ns.dry_run,
                follow_symlinks=ns.follow_symlinks,
                replace_theme=select,
            )
        case "lazygit":
            results = setup_lazygit(
                theme,
                catalog,
                config_dir=config_dir,
                dry_run=ns.dry_run,
                follow_symlinks=ns.follow_symlinks,
                adopt=ns.adopt,
            )
        case "starship":
            results = _starship_results(theme, ns, config_dir)
        case "claude":
            results = setup_claude(
                theme,
                catalog,
                config_dir=config_dir,
                dry_run=ns.dry_run,
                follow_symlinks=ns.follow_symlinks,
                replace_theme=select,
            )
        case "herdr":
            results = [
                apply_herdr(
                    theme,
                    catalog,
                    config_dir=config_dir,
                    dry_run=ns.dry_run,
                    follow_symlinks=ns.follow_symlinks,
                    adopt=ns.adopt,
                )
            ]
        case unreachable:
            raise CliError(f"unsupported app: {unreachable}; choose from {', '.join(APP_NAMES)}")
    _report(results, verbose=ns.verbose)
    _summarize(command, ns.app, select, theme, catalog, results, kept=kept, blocked=manual is not None)
    if ns.app == "starship":
        print(
            f"Source the zsh highlight snippet after zsh-syntax-highlighting:\n  {SOURCE_HINT}\n",
            file=sys.stderr,
        )
    if manual is not None:
        if command == "setup":
            reason = (
                "it already selects a color scheme"
                if adoptable
                else "its Lua shape is not one sf2-themes can safely edit"
            )
            hint = "Pass --adopt to replace it, or add" if adoptable else "Add"
            print(
                f"WezTerm config was left unchanged because {reason}.\n{hint} this integration to wezterm.lua:\n",
                file=sys.stderr,
            )
        else:
            hint = "Pass --adopt to replace its color scheme, or add" if adoptable else "Add"
            print(
                "apply incomplete: WezTerm config was left unchanged, so nothing loads the theme yet.\n"
                f"{hint} this integration to wezterm.lua:\n",
                file=sys.stderr,
            )
        print(manual, end="", file=sys.stderr)
        return 0 if command == "setup" else 1
    return 0


def _current_id(app: str, config_dir: Path | None) -> str:
    match app:
        case "wezterm":
            return wezterm_current()
        case "herdr":
            return herdr_current(config_dir)
        case "nvim":
            return nvim_current(config_dir)
        case "codex":
            return codex_current(config_dir)
        case "lazygit":
            return lazygit_current(config_dir)
        case "starship":
            return starship_current(config_dir)
        case "claude":
            return claude_current(config_dir)
        case unreachable:
            raise CliError(f"unsupported app: {unreachable}; choose from {', '.join(APP_NAMES)}")


def _selection_exists(app: str, config_dir: Path | None) -> bool:
    """True when the app already holds a selection that setup will keep."""
    match app:
        case "wezterm":
            return wezterm_pointer_path().is_file()
        case "nvim":
            return nvim_pointer_path(config_dir).is_file()
        case _:
            try:
                _current_id(app, config_dir)
            except ThemeError:
                return False
            return True


def dispatch(arguments: list[str]) -> int:
    """Run one CLI command. Returns a process exit code."""
    parser, mutation = _build_parser()
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
                print(_current_id(ns.app, ns.config_dir.expanduser() if ns.config_dir else None))
            case "setup" | "apply" | "install":
                return _mutate(ns.command, ns, mutation[ns.command])
    except SystemExit as exit_:
        return int(exit_.code) if isinstance(exit_.code, int) else 0
    except (ThemeError, OSError, tomllib.TOMLDecodeError, json.JSONDecodeError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 1
    return 0


def main(arguments: list[str] | None = None) -> int:
    """CLI entry point."""
    return dispatch(sys.argv[1:] if arguments is None else arguments)
