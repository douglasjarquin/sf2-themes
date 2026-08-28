"""Command-line interface for the Street Fighter II theme pack."""

import sys
import tomllib
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path

from sf2_theme import __version__
from sf2_theme.adapters.codex import apply_codex, setup_codex
from sf2_theme.adapters.codex import read_current_id as codex_current
from sf2_theme.adapters.herdr import apply_herdr
from sf2_theme.adapters.herdr import read_current_id as herdr_current
from sf2_theme.adapters.nvim import apply_nvim, setup_nvim
from sf2_theme.adapters.nvim import read_current_id as nvim_current
from sf2_theme.adapters.starship import apply_starship
from sf2_theme.adapters.starship import read_current_id as starship_current
from sf2_theme.adapters.wezterm import apply_wezterm, setup_wezterm
from sf2_theme.adapters.wezterm import read_current_id as wezterm_current
from sf2_theme.adapters.zsh_syntax import SOURCE_HINT, apply_zsh_syntax
from sf2_theme.catalog import (
    catalog_issues,
    default_theme,
    get_theme,
    load_catalog,
    parse_catalog,
    show_theme,
)
from sf2_theme.errors import CliError, ThemeError
from sf2_theme.filesystem import WriteResult
from sf2_theme.model import Theme
from sf2_theme.validation import validate_theme

APP_NAMES = ("wezterm", "herdr", "nvim", "codex", "starship")
HELP_TEXT = """Street Fighter II theme pack

Usage:
  sf2-themes apps
  sf2-themes themes
  sf2-themes show THEME
  sf2-themes validate [THEME | --all]
  sf2-themes setup APP [--config-dir PATH] [--dry-run] [--follow-symlinks] [--adopt]
  sf2-themes apply APP [--theme THEME] [--config-dir PATH] [--dry-run] [--follow-symlinks]
  sf2-themes current APP [--config-dir PATH]
  sf2-themes install APP ...

setup performs one-time application integration.
apply writes or selects the active theme (default: main).
install is a deprecated alias for apply.
starship also refreshes ~/.config/sf2-theme/zsh-syntax-highlighting.zsh.
"""


@dataclass(frozen=True, slots=True)
class Options:
    """Shared flags parsed from a command's trailing arguments."""

    theme: str | None = None
    config_dir: Path | None = None
    dry_run: bool = False
    follow_symlinks: bool = False
    adopt: bool = False
    all_themes: bool = False
    rest: tuple[str, ...] = ()


def parse_options(arguments: Sequence[str]) -> Options:
    """Parse shared flags. Remaining positional args stay in `rest`."""
    theme: str | None = None
    config_dir: Path | None = None
    dry_run = False
    follow_symlinks = False
    adopt = False
    all_themes = False
    rest: list[str] = []
    index = 0
    while index < len(arguments):
        item = arguments[index]
        match item:
            case "--theme":
                theme = _need_value(arguments, index, item)
                index += 2
            case "--config-dir":
                config_dir = Path(_need_value(arguments, index, item)).expanduser()
                index += 2
            case "--dry-run":
                dry_run = True
                index += 1
            case "--follow-symlinks":
                follow_symlinks = True
                index += 1
            case "--adopt":
                adopt = True
                index += 1
            case "--all":
                all_themes = True
                index += 1
            case flag if flag.startswith("-"):
                raise CliError(f"unknown option: {flag}")
            case _:
                rest.append(item)
                index += 1
    return Options(
        theme=theme,
        config_dir=config_dir,
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
        adopt=adopt,
        all_themes=all_themes,
        rest=tuple(rest),
    )


def _need_value(arguments: Sequence[str], index: int, flag: str) -> str:
    if index + 1 >= len(arguments):
        raise CliError(f"{flag} requires a value")
    return arguments[index + 1]


def _app(name: str) -> str:
    if name not in APP_NAMES:
        raise CliError(f"unsupported app: {name}; choose {', '.join(APP_NAMES)}")
    return name


def _report(results: Sequence[WriteResult]) -> None:
    for result in results:
        print(f"{result.action.value}: {result.path}")
        if result.diff and result.action.value.startswith("would_"):
            print(result.diff, end="")


def _apply_starship(theme: Theme, options: Options) -> list[WriteResult]:
    return [
        apply_starship(
            theme,
            config_dir=options.config_dir,
            dry_run=options.dry_run,
            follow_symlinks=options.follow_symlinks,
        ),
        apply_zsh_syntax(
            theme,
            dry_run=options.dry_run,
            follow_symlinks=options.follow_symlinks,
        ),
    ]


def _setup(app: str, options: Options) -> None:
    catalog = load_catalog()
    theme = default_theme(catalog) if options.theme is None else get_theme(options.theme, catalog)
    match app:
        case "wezterm":
            results, lua = setup_wezterm(
                theme,
                catalog,
                config_dir=options.config_dir,
                dry_run=options.dry_run,
                follow_symlinks=options.follow_symlinks,
                adopt=options.adopt,
                replace_pointer=options.theme is not None,
            )
            _report(results)
            if lua.snippet is not None:
                print(
                    "WezTerm config was left unchanged because it already selects a color scheme.\n"
                    "Pass --adopt to replace it, or add this integration:\n",
                    file=sys.stderr,
                )
                print(lua.snippet, end="", file=sys.stderr)
        case "herdr":
            result = apply_herdr(
                theme,
                config_dir=options.config_dir,
                dry_run=options.dry_run,
                follow_symlinks=options.follow_symlinks,
                adopt=options.adopt,
            )
            _report((result,))
        case "nvim":
            _report(
                setup_nvim(
                    theme,
                    catalog,
                    config_dir=options.config_dir,
                    dry_run=options.dry_run,
                    follow_symlinks=options.follow_symlinks,
                    replace_pointer=options.theme is not None,
                )
            )
        case "codex":
            _report(
                setup_codex(
                    theme,
                    catalog,
                    config_dir=options.config_dir,
                    dry_run=options.dry_run,
                    follow_symlinks=options.follow_symlinks,
                    replace_theme=options.theme is not None,
                )
            )
        case "starship":
            _report(_apply_starship(theme, options))
            print(
                "Source the zsh highlight snippet after zsh-syntax-highlighting:\n"
                f"  {SOURCE_HINT}\n",
                file=sys.stderr,
            )
        case unreachable:
            raise CliError(f"unsupported app: {unreachable}")


def _apply(app: str, options: Options) -> None:
    catalog = load_catalog()
    theme = default_theme(catalog) if options.theme is None else get_theme(options.theme, catalog)
    match app:
        case "wezterm":
            _report(
                apply_wezterm(
                    theme,
                    catalog,
                    config_dir=options.config_dir,
                    dry_run=options.dry_run,
                    follow_symlinks=options.follow_symlinks,
                )
            )
        case "herdr":
            _report(
                (
                    apply_herdr(
                        theme,
                        config_dir=options.config_dir,
                        dry_run=options.dry_run,
                        follow_symlinks=options.follow_symlinks,
                        adopt=options.adopt,
                    ),
                )
            )
        case "nvim":
            _report(
                apply_nvim(
                    theme,
                    catalog,
                    config_dir=options.config_dir,
                    dry_run=options.dry_run,
                    follow_symlinks=options.follow_symlinks,
                )
            )
        case "codex":
            _report(
                apply_codex(
                    theme,
                    catalog,
                    config_dir=options.config_dir,
                    dry_run=options.dry_run,
                    follow_symlinks=options.follow_symlinks,
                )
            )
        case "starship":
            _report(_apply_starship(theme, options))
        case unreachable:
            raise CliError(f"unsupported app: {unreachable}")


def _validate(options: Options) -> int:
    catalog = parse_catalog()
    if options.all_themes or not options.rest:
        themes = catalog
        lines = catalog_issues(themes)
    else:
        themes = tuple(get_theme(name, catalog) for name in options.rest)
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


def _current(app: str, options: Options) -> None:
    match app:
        case "wezterm":
            print(wezterm_current())
        case "herdr":
            print(herdr_current(options.config_dir))
        case "nvim":
            print(nvim_current(options.config_dir))
        case "codex":
            print(codex_current(options.config_dir))
        case "starship":
            print(starship_current(options.config_dir))
        case unreachable:
            raise CliError(f"unsupported app: {unreachable}")


def dispatch(arguments: list[str]) -> int:
    """Run one CLI command. Returns a process exit code."""
    if not arguments or arguments in (["--help"], ["-h"]):
        print(HELP_TEXT, end="")
        return 0
    if arguments in (["--version"], ["-V"]):
        print(__version__)
        return 0
    command, *rest = arguments
    apps_hint = ", ".join(APP_NAMES)
    try:
        match command:
            case "apps":
                if rest:
                    raise CliError("apps does not take options")
                print("\n".join(APP_NAMES))
            case "themes":
                if rest:
                    raise CliError("themes does not take options")
                for theme in load_catalog():
                    print(f"{theme.metadata.id}\t{theme.metadata.display_name}")
            case "show":
                options = parse_options(rest)
                if len(options.rest) != 1:
                    raise CliError("show requires a theme id")
                print(show_theme(get_theme(options.rest[0])), end="")
            case "validate":
                return _validate(parse_options(rest))
            case "setup":
                options = parse_options(rest)
                if len(options.rest) != 1:
                    raise CliError(f"setup requires an app: {apps_hint}")
                _setup(_app(options.rest[0]), options)
            case "apply":
                options = parse_options(rest)
                if len(options.rest) != 1:
                    raise CliError(f"apply requires an app: {apps_hint}")
                _apply(_app(options.rest[0]), options)
            case "install":
                print("warning: install is deprecated; use apply", file=sys.stderr)
                options = parse_options(rest)
                if len(options.rest) != 1:
                    raise CliError(f"install requires an app: {apps_hint}")
                _apply(_app(options.rest[0]), options)
            case "current":
                options = parse_options(rest)
                if len(options.rest) != 1:
                    raise CliError(f"current requires an app: {apps_hint}")
                _current(_app(options.rest[0]), options)
            case _:
                raise CliError(f"unknown command: {command}; run sf2-themes --help")
    except (ThemeError, OSError, tomllib.TOMLDecodeError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 1
    return 0

def main(arguments: list[str] | None = None) -> int:
    """CLI entry point."""
    return dispatch(sys.argv[1:] if arguments is None else arguments)
