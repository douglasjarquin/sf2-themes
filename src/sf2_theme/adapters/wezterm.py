"""WezTerm color-scheme files, path discovery, and the planned integration."""

import os
import re
from collections.abc import Sequence
from dataclasses import replace
from pathlib import Path

from sf2_theme.adapters.wezterm_lua import classify, integrate, integration_snippet
from sf2_theme.catalog import theme_pair
from sf2_theme.model import Theme, project_adapter_colors, selectable_id
from sf2_theme.plan import Inspection, Ownership, Remove, Step, Write, read_existing, recorded, recorded_id

LEGACY_SCHEME_FILE_PREFIX = "street-fighter-ii"
ID_COMMENT = re.compile(r"^-- sf2-themes:\s*(\S+)\s*$")


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


def inspect_wezterm(themes: Sequence[Theme], config_dir: Path | None) -> Inspection:
    """Ownership comes from wezterm.lua; the selection from the sf2-owned pointer."""
    pointer = current_pointer_path()
    lua_path = wezterm_lua_path(config_dir)
    ownership = classify(read_existing(lua_path), pointer)
    if pointer.is_file():
        selection = recorded(ownership, recorded_id(pointer.read_text(encoding="utf-8"), ID_COMMENT), themes)
    else:
        selection = Inspection(ownership)
    if ownership is Ownership.UNKNOWN_SHAPE:
        return replace(selection, detail=f"Add this integration to {lua_path}:\n\n{integration_snippet(pointer)}")
    return selection


def plan_wezterm(theme: Theme, themes: Sequence[Theme], config_dir: Path | None, slot: bool) -> list[Step]:
    """Plan schemes and the pointer, and the wezterm.lua integration when `slot` allows it."""
    target = colors_dir(config_dir)
    steps: list[Step] = [Remove(target / f"{LEGACY_SCHEME_FILE_PREFIX}-{entry.metadata.id}.toml") for entry in themes]
    steps.extend(Write(target / scheme_filename(entry), render_scheme(entry)) for entry in themes)
    dark, light = theme_pair(theme, themes)
    steps.append(Write(current_pointer_path(), render_pointer(dark, light)))
    if slot:
        lua_path = wezterm_lua_path(config_dir)
        existing = read_existing(lua_path)
        steps.append(Write(lua_path, integrate(existing, current_pointer_path())))
    return steps
