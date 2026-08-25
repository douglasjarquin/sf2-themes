import os
import re
from collections.abc import Sequence
from pathlib import Path
from typing import Final

from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import WriteResult, write_file
from sf2_theme.model import Theme

SCHEME_FILE_PREFIX: Final = "street-fighter-ii"
CURRENT_DIR_NAME: Final = "sf2-theme"
CURRENT_FILE_NAME: Final = "current.lua"
PLUGIN_FILE_NAME: Final = "sf2-theme.lua"
ID_COMMENT: Final = re.compile(r"^-- sf2-theme:\s+(\S+)\s*$")


def config_root() -> Path:
    configured = os.environ.get("XDG_CONFIG_HOME")
    return Path(configured).expanduser() if configured else Path.home() / ".config"


def nvim_dir(config_dir: Path | None) -> Path:
    if config_dir is not None:
        return config_dir
    configured = os.environ.get("NVIM_CONFIG_DIR")
    if configured:
        return Path(configured).expanduser()
    return config_root() / "nvim"


def colors_dir(config_dir: Path | None) -> Path:
    return nvim_dir(config_dir) / "colors"


def managed_dir(config_dir: Path | None) -> Path:
    return nvim_dir(config_dir) / CURRENT_DIR_NAME


def current_pointer_path(config_dir: Path | None = None) -> Path:
    return managed_dir(config_dir) / CURRENT_FILE_NAME


def plugin_path(config_dir: Path | None) -> Path:
    return nvim_dir(config_dir) / "plugin" / PLUGIN_FILE_NAME


def scheme_filename(theme: Theme) -> str:
    return f"{SCHEME_FILE_PREFIX}-{theme.metadata.id}.lua"


def scheme_name(theme: Theme) -> str:
    return f"{SCHEME_FILE_PREFIX}-{theme.metadata.id}"


def render_scheme(theme: Theme) -> str:
    ui = theme.ui
    ansi_normal = theme.ansi_normal
    ansi_bright = theme.ansi_bright
    colors = (
        ("background", ui.background),
        ("foreground", ui.foreground),
        ("cursor_bg", ui.cursor_bg),
        ("cursor_fg", ui.cursor_fg),
        ("selection_bg", ui.selection_bg),
        ("selection_fg", ui.selection_fg),
        ("panel_bg", ui.panel_bg),
        ("sidebar_bg", ui.sidebar_bg),
        ("active_row_bg", ui.active_row_bg),
        ("navigate_row_bg", ui.navigate_row_bg),
        ("surface_dim", ui.surface_dim),
        ("surface0", ui.surface0),
        ("surface1", ui.surface1),
        ("overlay0", ui.overlay0),
        ("overlay1", ui.overlay1),
        ("subtext", ui.subtext),
        ("accent", ui.accent),
        ("red", theme.semantic.red),
        ("green", theme.semantic.green),
        ("yellow", theme.semantic.yellow),
        ("blue", theme.semantic.blue),
        ("magenta", theme.semantic.magenta),
        ("cyan", theme.semantic.cyan),
        ("orange", theme.semantic.orange),
        ("normal_black", ansi_normal.black),
        ("normal_red", ansi_normal.red),
        ("normal_green", ansi_normal.green),
        ("normal_yellow", ansi_normal.yellow),
        ("normal_blue", ansi_normal.blue),
        ("normal_magenta", ansi_normal.magenta),
        ("normal_cyan", ansi_normal.cyan),
        ("normal_white", ansi_normal.white),
        ("bright_black", ansi_bright.black),
        ("bright_red", ansi_bright.red),
        ("bright_green", ansi_bright.green),
        ("bright_yellow", ansi_bright.yellow),
        ("bright_blue", ansi_bright.blue),
        ("bright_magenta", ansi_bright.magenta),
        ("bright_cyan", ansi_bright.cyan),
        ("bright_white", ansi_bright.white),
    )
    lines = [
        'vim.cmd("highlight clear")',
        'if vim.fn.exists("syntax_on") == 1 then vim.cmd("syntax reset") end',
        f'vim.o.background = "{"light" if theme.metadata.id.endswith("-light") else "dark"}"',
        f'vim.g.colors_name = "{scheme_name(theme)}"',
        "local colors = {",
    ]
    lines.extend(f'  {name} = "{color}",' for name, color in colors)
    lines.extend(
        (
            "}",
            "local highlights = {",
            "  Normal = { fg = colors.foreground, bg = colors.background },",
            "  NormalFloat = { fg = colors.foreground, bg = colors.panel_bg },",
            "  Cursor = { fg = colors.background, bg = colors.cursor_bg },",
            "  CursorLine = { bg = colors.surface0 },",
            "  Visual = { bg = colors.selection_bg, fg = colors.selection_fg },",
            "  Search = { bg = colors.yellow, fg = colors.background },",
            "  IncSearch = { bg = colors.orange, fg = colors.background },",
            "  StatusLine = { bg = colors.surface1, fg = colors.foreground },",
            "  StatusLineNC = { bg = colors.surface0, fg = colors.subtext },",
            "  WinSeparator = { fg = colors.surface1 },",
            "  Pmenu = { bg = colors.panel_bg, fg = colors.foreground },",
            "  PmenuSel = { bg = colors.selection_bg, fg = colors.selection_fg },",
            "  LineNr = { fg = colors.overlay1 },",
            "  CursorLineNr = { fg = colors.accent, bold = true },",
            "  Comment = { fg = colors.subtext, italic = true },",
            "  Constant = { fg = colors.cyan },",
            "  String = { fg = colors.green },",
            "  Number = { fg = colors.orange },",
            "  Boolean = { fg = colors.orange },",
            "  Identifier = { fg = colors.foreground },",
            "  Function = { fg = colors.blue },",
            "  Statement = { fg = colors.magenta },",
            "  Keyword = { fg = colors.red },",
            "  Type = { fg = colors.yellow },",
            "  Special = { fg = colors.orange },",
            "  Error = { fg = colors.red, bold = true },",
            "  Todo = { fg = colors.background, bg = colors.yellow, bold = true },",
            "  DiagnosticError = { fg = colors.red },",
            "  DiagnosticWarn = { fg = colors.yellow },",
            "  DiagnosticInfo = { fg = colors.blue },",
            "  DiagnosticHint = { fg = colors.cyan },",
            "  DiffAdd = { bg = colors.green, fg = colors.background },",
            "  DiffChange = { bg = colors.blue, fg = colors.background },",
            "  DiffDelete = { bg = colors.red, fg = colors.background },",
            "  Terminal = { fg = colors.normal_white, bg = colors.background },",
            "}",
            "for group, opts in pairs(highlights) do",
            "  vim.api.nvim_set_hl(0, group, opts)",
            "end",
            "",
        )
    )
    return "\n".join(lines)


def render_pointer(theme: Theme) -> str:
    return f"-- sf2-theme: {theme.metadata.id}\nvim.cmd(\"colorscheme {scheme_name(theme)}\")\n"


def render_loader() -> str:
    return "\n".join(
        (
            'local current = vim.fn.stdpath("config") .. "/sf2-theme/current.lua"',
            "if vim.fn.filereadable(current) == 1 then",
            "  dofile(current)",
            "end",
            "",
        )
    )


def write_schemes(
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
) -> list[WriteResult]:
    target = colors_dir(config_dir)
    return [
        write_file(
            target / scheme_filename(theme),
            render_scheme(theme),
            dry_run=dry_run,
            follow_symlinks=follow_symlinks,
        )
        for theme in themes
    ]


def write_pointer(
    theme: Theme,
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
) -> WriteResult:
    return write_file(
        current_pointer_path(config_dir),
        render_pointer(theme),
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
    )


def apply_nvim(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
) -> list[WriteResult]:
    results = write_schemes(
        themes,
        config_dir=config_dir,
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
    )
    results.append(
        write_pointer(
            theme,
            config_dir=config_dir,
            dry_run=dry_run,
            follow_symlinks=follow_symlinks,
        )
    )
    return results


def setup_nvim(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
    replace_pointer: bool = True,
) -> list[WriteResult]:
    results = write_schemes(
        themes,
        config_dir=config_dir,
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
    )
    pointer = current_pointer_path(config_dir)
    if replace_pointer or not pointer.is_file():
        results.append(
            write_pointer(
                theme,
                config_dir=config_dir,
                dry_run=dry_run,
                follow_symlinks=follow_symlinks,
            )
        )
    results.append(
        write_file(
            plugin_path(config_dir),
            render_loader(),
            dry_run=dry_run,
            follow_symlinks=follow_symlinks,
        )
    )
    return results


def read_current_id(config_dir: Path | None = None) -> str:
    path = current_pointer_path(config_dir)
    if not path.is_file():
        raise ThemeError(f"no Neovim theme applied yet ({path} missing)")
    for line in path.read_text(encoding="utf-8").splitlines():
        match = ID_COMMENT.match(line.strip())
        if match:
            return match.group(1)
    raise ThemeError(f"could not read theme id from {path}")
