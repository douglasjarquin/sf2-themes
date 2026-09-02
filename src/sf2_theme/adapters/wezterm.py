"""WezTerm color-scheme files, path discovery, and apply/setup."""

import os
from collections.abc import Sequence
from pathlib import Path

from sf2_theme.adapters.wezterm_lua import LuaSetup, setup_lua
from sf2_theme.catalog import installed_theme, theme_pair
from sf2_theme.errors import ThemeError
from sf2_theme.filesystem import WriteResult, write_file
from sf2_theme.model import Theme, project_adapter_colors, selectable_id

LEGACY_SCHEME_FILE_PREFIX = "street-fighter-ii"


def config_root() -> Path:
    """Return the XDG-style configuration root."""
    configured = os.environ.get("XDG_CONFIG_HOME")
    return Path(configured).expanduser() if configured else Path.home() / ".config"


def wezterm_dir(config_dir: Path | None) -> Path:
    """Return the WezTerm configuration directory."""
    if config_dir is not None:
        return config_dir
    configured = os.environ.get("WEZTERM_CONFIG_DIR")
    if configured:
        return Path(configured).expanduser()
    return config_root() / "wezterm"


def wezterm_lua_path(config_dir: Path | None) -> Path:
    """Return the wezterm.lua path to read or create.

    `XDG_CONFIG_HOME` and `WEZTERM_CONFIG_FILE` pin the write target even when
    the file does not exist yet, so tests and custom prefixes never fall through
    to the real home directory.
    """
    if config_dir is not None:
        return config_dir / "wezterm.lua"
    configured = os.environ.get("WEZTERM_CONFIG_FILE")
    if configured:
        return Path(configured).expanduser()
    xdg = os.environ.get("XDG_CONFIG_HOME")
    if xdg:
        return Path(xdg).expanduser() / "wezterm" / "wezterm.lua"
    default = Path.home() / ".config" / "wezterm" / "wezterm.lua"
    home_lua = Path.home() / ".wezterm.lua"
    if default.exists() or not home_lua.exists():
        return default
    return home_lua


def colors_dir(config_dir: Path | None) -> Path:
    """Return the directory for custom WezTerm scheme files."""
    return wezterm_dir(config_dir) / "colors"


def current_pointer_path() -> Path:
    """Return the managed color-scheme pointer file."""
    return config_root() / "sf2-theme" / "wezterm-current.lua"


def scheme_filename(theme: Theme) -> str:
    """Return the on-disk scheme file name."""
    return f"{theme.metadata.selectable_id}.toml"


def render_scheme(theme: Theme) -> str:
    """Render a WezTerm color scheme TOML file.

    Covers every key Catppuccin-style schemes ship in TOML: core chrome, ANSI,
    indexed extras, compose/bell cues, and retro tab_bar. Fancy-tab window_frame
    colors are Lua-only and stay out of scheme files.
    """
    ui = theme.ui
    adapter = project_adapter_colors(ui)
    semantic = theme.semantic
    ansi = ", ".join(f'"{color}"' for color in theme.ansi_normal.as_tuple())
    brights = ", ".join(f'"{color}"' for color in theme.ansi_bright.as_tuple())
    lines = [
        "[colors]",
        f'foreground = "{ui.foreground}"',
        f'background = "{ui.background}"',
        f'cursor_bg = "{adapter.cursor_bg}"',
        f'cursor_fg = "{adapter.cursor_fg}"',
        f'cursor_border = "{adapter.cursor_bg}"',
        f'selection_bg = "{adapter.selection_bg}"',
        f'selection_fg = "{adapter.selection_fg}"',
        f'split = "{adapter.surface1}"',
        # Muted reads as a real thumb; border is often identical to the track.
        f'scrollbar_thumb = "{ui.muted}"',
        f'compose_cursor = "{ui.accent_secondary}"',
        f'visual_bell = "{ui.overlay}"',
        f"ansi = [{ansi}]",
        f"brights = [{brights}]",
        "",
        "[colors.indexed]",
        f'16 = "{semantic.orange}"',
        f'17 = "{ui.accent}"',
        "",
        "[colors.tab_bar]",
        f'background = "{ui.background}"',
        f'inactive_tab_edge = "{ui.border}"',
        "",
        "[colors.tab_bar.active_tab]",
        f'bg_color = "{ui.accent}"',
        f'fg_color = "{ui.cursor_text}"',
        'intensity = "Normal"',
        "italic = false",
        "strikethrough = false",
        'underline = "None"',
        "",
        "[colors.tab_bar.inactive_tab]",
        f'bg_color = "{ui.surface}"',
        f'fg_color = "{ui.muted}"',
        'intensity = "Normal"',
        "italic = false",
        "strikethrough = false",
        'underline = "None"',
        "",
        "[colors.tab_bar.inactive_tab_hover]",
        f'bg_color = "{ui.overlay}"',
        f'fg_color = "{ui.foreground}"',
        'intensity = "Normal"',
        "italic = false",
        "strikethrough = false",
        'underline = "None"',
        "",
        "[colors.tab_bar.new_tab]",
        f'bg_color = "{ui.surface}"',
        f'fg_color = "{ui.muted}"',
        'intensity = "Normal"',
        "italic = false",
        "strikethrough = false",
        'underline = "None"',
        "",
        "[colors.tab_bar.new_tab_hover]",
        f'bg_color = "{ui.overlay}"',
        f'fg_color = "{ui.foreground}"',
        'intensity = "Normal"',
        "italic = false",
        "strikethrough = false",
        'underline = "None"',
        "",
        "[metadata]",
        f'name = "{theme.metadata.selectable_id}"',
    ]
    aliases = list(theme.metadata.aliases)
    if theme.metadata.id == "main" and "street-fighter-2" not in aliases:
        aliases.insert(0, "street-fighter-2")
    rendered = ", ".join(f'"{selectable_id(alias)}"' for alias in aliases)
    lines.append(f"aliases = [{rendered}]")
    lines.append("")
    return "\n".join(lines)


def render_pointer(dark: Theme, light: Theme) -> str:
    """Render the managed Lua pointer that follows host appearance."""
    return "\n".join(
        (
            f"-- sf2-themes: {dark.metadata.selectable_id}",
            'local wezterm = require("wezterm")',
            'local appearance = wezterm.gui and wezterm.gui.get_appearance() or "Dark"',
            'if appearance:find("Dark") then',
            f'  return "{dark.metadata.selectable_id}"',
            "end",
            f'return "{light.metadata.selectable_id}"',
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
    """Write every scheme file into WezTerm's colors directory."""
    target = colors_dir(config_dir)
    if not dry_run:
        for theme in themes:
            (target / f"{LEGACY_SCHEME_FILE_PREFIX}-{theme.metadata.id}.toml").unlink(missing_ok=True)
    results: list[WriteResult] = []
    for theme in themes:
        results.append(
            write_file(
                target / scheme_filename(theme),
                render_scheme(theme),
                dry_run=dry_run,
                follow_symlinks=follow_symlinks,
            )
        )
    return results


def write_pointer(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    dry_run: bool,
    follow_symlinks: bool,
) -> WriteResult:
    """Write the managed current-scheme pointer for a dark/light pair."""
    dark, light = theme_pair(theme, themes)
    return write_file(
        current_pointer_path(),
        render_pointer(dark, light),
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
    )


def apply_wezterm(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
) -> list[WriteResult]:
    """Write schemes and the current pointer. Does not edit user Lua."""
    results = write_schemes(themes, config_dir=config_dir, dry_run=dry_run, follow_symlinks=follow_symlinks)
    results.append(write_pointer(theme, themes, dry_run=dry_run, follow_symlinks=follow_symlinks))
    return results


def setup_wezterm(
    theme: Theme,
    themes: Sequence[Theme],
    *,
    config_dir: Path | None,
    dry_run: bool,
    follow_symlinks: bool,
    adopt: bool = False,
    replace_pointer: bool = True,
) -> tuple[list[WriteResult], LuaSetup]:
    """Install schemes, pointer, and optionally a known-safe Lua integration."""
    results = write_schemes(
        themes,
        config_dir=config_dir,
        dry_run=dry_run,
        follow_symlinks=follow_symlinks,
    )
    pointer = current_pointer_path()
    if replace_pointer or not pointer.is_file():
        results.append(write_pointer(theme, themes, dry_run=dry_run, follow_symlinks=follow_symlinks))
    else:
        try:
            current = installed_theme(read_current_id(), themes)
        except ThemeError:
            current = None
        if current is not None:
            results.append(write_pointer(current, themes, dry_run=dry_run, follow_symlinks=follow_symlinks))
    lua_path = wezterm_lua_path(config_dir)
    existing = lua_path.read_text(encoding="utf-8") if lua_path.exists() else ""
    lua = setup_lua(existing, current_pointer_path(), adopt=adopt)
    if lua.mutated:
        results.append(write_file(lua_path, lua.content, dry_run=dry_run, follow_symlinks=follow_symlinks))
    return results, lua


def read_current_id() -> str:
    """Read the theme id stored in the managed pointer."""
    path = current_pointer_path()
    if not path.is_file():
        raise ThemeError(f"no WezTerm theme applied yet ({path} missing)")
    for line in path.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if stripped.startswith("-- sf2-themes:"):
            return stripped.split(":", 1)[1].strip()
    raise ThemeError(f"could not read theme id from {path}")
