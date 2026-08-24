#!/usr/bin/env python3
"""Author fully resolved character TOMLs from identity seeds. Not used at runtime."""

from __future__ import annotations

import sys
from dataclasses import dataclass
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from sf2_theme.model import HexColor, parse_hex, theme_to_toml  # noqa: E402
from sf2_theme.parse import parse_theme  # noqa: E402
from sf2_theme.validation import (  # noqa: E402
    NONTEXT_CONTRAST,
    TEXT_CONTRAST,
    contrast_ratio,
    require_valid,
)

CREAM = parse_hex("#fff4d6")
GOLD = parse_hex("#f2b134")
TEAL = parse_hex("#35c4c2")
MAIN_SEMANTIC = {
    "red": parse_hex("#e8565f"),
    "green": parse_hex("#6ecb78"),
    "yellow": parse_hex("#f2b134"),
    "blue": parse_hex("#4aa5ff"),
    "magenta": parse_hex("#c783d9"),
    "cyan": parse_hex("#35c4c2"),
    "orange": parse_hex("#dc8d55"),
}
MAIN_BRIGHT = {
    "red": parse_hex("#ff7880"),
    "green": parse_hex("#8ee695"),
    "yellow": parse_hex("#ffd166"),
    "blue": parse_hex("#74bdff"),
    "magenta": parse_hex("#e1a5ef"),
    "cyan": parse_hex("#62dedb"),
    "white": parse_hex("#fffaf0"),
}


@dataclass(frozen=True, slots=True)
class Seed:
    theme_id: str
    character: str
    era: str
    aliases: tuple[str, ...]
    background: str
    accent: str
    secondary: str


SEEDS = (
    Seed("ryu", "Ryu", "world-warrior", (), "#101522", "#d83a3a", "#d8cab0"),
    Seed("ken", "Ken", "world-warrior", (), "#1a1014", "#e23b3b", "#f2b134"),
    Seed("chun-li", "Chun-Li", "world-warrior", (), "#0e1530", "#2f5bd6", "#f2b134"),
    Seed("e-honda", "E. Honda", "world-warrior", (), "#16182a", "#d63c2e", "#3a3f78"),
    Seed("blanka", "Blanka", "world-warrior", (), "#0c1810", "#2fa84a", "#f08a20"),
    Seed("zangief", "Zangief", "world-warrior", (), "#1a1014", "#c41e3a", "#d4a017"),
    Seed("guile", "Guile", "world-warrior", (), "#101820", "#4a6b3a", "#4aa5ff"),
    Seed("dhalsim", "Dhalsim", "world-warrior", (), "#1a120c", "#e07020", "#c41e1e"),
    Seed("balrog", "Balrog", "champion-edition", ("boxer",), "#101428", "#2a4a9a", "#d62939"),
    Seed("vega", "Vega", "champion-edition", ("claw",), "#16121c", "#8a3cb8", "#d62939"),
    Seed("sagat", "Sagat", "champion-edition", (), "#121018", "#2a4a9a", "#f2b134"),
    Seed("m-bison", "M. Bison", "champion-edition", ("dictator",), "#140a12", "#c41e3a", "#7a3cb8"),
    Seed("cammy", "Cammy", "super", (), "#10180e", "#3d6b2f", "#d62939"),
    Seed("t-hawk", "T. Hawk", "super", (), "#14120c", "#2a5a8a", "#2aa8a0"),
    Seed("fei-long", "Fei Long", "super", (), "#12100e", "#d62939", "#f2b134"),
    Seed("dee-jay", "Dee Jay", "super", (), "#0c1424", "#d62939", "#f2b134"),
    Seed("akuma", "Akuma", "super-turbo", (), "#0c0a12", "#e04020", "#6b4423"),
)


def _channels(color: HexColor) -> tuple[int, int, int]:
    value = color.removeprefix("#")
    return int(value[0:2], 16), int(value[2:4], 16), int(value[4:6], 16)


def mix(base: HexColor, overlay: HexColor, amount: float) -> HexColor:
    start = _channels(base)
    end = _channels(overlay)
    mixed = tuple(round(left + (right - left) * amount) for left, right in zip(start, end, strict=True))
    return parse_hex(f"#{mixed[0]:02x}{mixed[1]:02x}{mixed[2]:02x}")


def lighten(color: HexColor, amount: float) -> HexColor:
    return mix(color, parse_hex("#ffffff"), amount)


def darken(color: HexColor, amount: float) -> HexColor:
    return mix(color, parse_hex("#000000"), amount)


def cream_safe(background: HexColor, overlay: HexColor, amount: float) -> HexColor:
    candidate = mix(background, overlay, amount)
    while contrast_ratio(CREAM, candidate) < TEXT_CONTRAST and amount > 0.04:
        amount -= 0.04
        candidate = mix(background, overlay, amount)
    if contrast_ratio(CREAM, candidate) < TEXT_CONTRAST:
        candidate = mix(background, overlay, 0.12)
    return candidate


def pick_selection(background: HexColor, secondary: HexColor) -> tuple[HexColor, HexColor]:
    if contrast_ratio(background, secondary) >= TEXT_CONTRAST:
        return secondary, background
    if contrast_ratio(background, GOLD) >= TEXT_CONTRAST:
        return GOLD, background
    return CREAM, background


def readable_accent(background: HexColor, accent: HexColor) -> HexColor:
    candidate = accent
    amount = 0.0
    while contrast_ratio(candidate, background) < NONTEXT_CONTRAST and amount < 0.6:
        amount += 0.05
        candidate = lighten(accent, amount)
    return candidate


def pick_cursor(background: HexColor, accent: HexColor, secondary: HexColor) -> tuple[HexColor, HexColor]:
    for candidate in (GOLD, secondary, lighten(accent, 0.25)):
        if contrast_ratio(background, candidate) >= TEXT_CONTRAST:
            return candidate, background
    return GOLD, background


def build(seed: Seed) -> str:
    background = parse_hex(seed.background)
    accent = readable_accent(background, parse_hex(seed.accent))
    secondary = parse_hex(seed.secondary)
    cursor_bg, cursor_fg = pick_cursor(background, accent, secondary)
    selection_bg, selection_fg = pick_selection(background, secondary)
    panel = mix(background, CREAM, 0.04)
    surface0 = mix(background, CREAM, 0.10)
    surface1 = mix(background, CREAM, 0.16)
    overlay0 = mix(background, CREAM, 0.42)
    overlay1 = mix(background, CREAM, 0.55)
    subtext = mix(CREAM, background, 0.30)
    active = cream_safe(background, TEAL, 0.28)
    navigate = cream_safe(background, GOLD, 0.22)
    ansi_black = background
    bright_black = mix(background, CREAM, 0.38)
    if bright_black == ansi_black:
        bright_black = lighten(ansi_black, 0.25)
    white = mix(CREAM, background, 0.12)
    if white == CREAM:
        white = mix(CREAM, background, 0.18)
    green = MAIN_SEMANTIC["green"]
    if seed.theme_id == "blanka":
        green = parse_hex("#3dcc6a")
    if seed.theme_id == "cammy":
        green = parse_hex("#5cb86a")
    semantic = dict(MAIN_SEMANTIC)
    semantic["green"] = green
    if seed.theme_id in {"ken", "zangief", "m-bison", "akuma", "ryu"}:
        semantic["red"] = mix(MAIN_SEMANTIC["red"], accent, 0.35)
    if seed.theme_id in {"chun-li", "guile", "balrog", "sagat"}:
        semantic["blue"] = mix(MAIN_SEMANTIC["blue"], accent, 0.40)
    if seed.theme_id == "dhalsim":
        semantic["orange"] = mix(MAIN_SEMANTIC["orange"], secondary, 0.35)
        semantic["yellow"] = mix(MAIN_SEMANTIC["yellow"], parse_hex(seed.accent), 0.25)
    if seed.theme_id == "vega":
        semantic["magenta"] = mix(MAIN_SEMANTIC["magenta"], accent, 0.45)
    if seed.theme_id == "akuma":
        semantic["orange"] = parse_hex("#e07038")
        semantic["red"] = parse_hex("#e8565f")
    bright_red = lighten(semantic["red"], 0.18)
    if bright_red == semantic["red"]:
        bright_red = lighten(semantic["red"], 0.28)
    aliases = ", ".join(f'"{alias}"' for alias in seed.aliases)
    raw = f"""
schema_version = 1

[meta]
id = "{seed.theme_id}"
display_name = "Street Fighter II - {seed.character}"
kind = "character"
character = "{seed.character}"
introduced_in = "{seed.era}"
aliases = [{aliases}]

[ui]
background = "{background}"
foreground = "{CREAM}"
cursor_bg = "{cursor_bg}"
cursor_fg = "{cursor_fg}"
selection_bg = "{selection_bg}"
selection_fg = "{selection_fg}"
panel_bg = "{panel}"
sidebar_bg = "{background}"
active_row_bg = "{active}"
navigate_row_bg = "{navigate}"
surface_dim = "{background}"
surface0 = "{surface0}"
surface1 = "{surface1}"
overlay0 = "{overlay0}"
overlay1 = "{overlay1}"
subtext = "{subtext}"
accent = "{accent}"

[semantic]
red = "{semantic['red']}"
green = "{semantic['green']}"
yellow = "{semantic['yellow']}"
blue = "{semantic['blue']}"
magenta = "{semantic['magenta']}"
cyan = "{semantic['cyan']}"
orange = "{semantic['orange']}"

[ansi.normal]
black = "{ansi_black}"
red = "{semantic['red']}"
green = "{semantic['green']}"
yellow = "{semantic['yellow']}"
blue = "{semantic['blue']}"
magenta = "{semantic['magenta']}"
cyan = "{semantic['cyan']}"
white = "{white}"

[ansi.bright]
black = "{bright_black}"
red = "{bright_red}"
green = "{lighten(semantic['green'], 0.18)}"
yellow = "{lighten(semantic['yellow'], 0.18)}"
blue = "{lighten(semantic['blue'], 0.18)}"
magenta = "{lighten(semantic['magenta'], 0.18)}"
cyan = "{lighten(semantic['cyan'], 0.18)}"
white = "{MAIN_BRIGHT['white']}"
"""
    theme = parse_theme(__import__("tomllib").loads(raw), source=seed.theme_id)
    require_valid(theme)
    return theme_to_toml(theme)


def main() -> int:
    root = Path(__file__).resolve().parents[1] / "themes" / "characters"
    root.mkdir(parents=True, exist_ok=True)
    for seed in SEEDS:
        path = root / f"{seed.theme_id}.toml"
        path.write_text(build(seed), encoding="utf-8")
        print(f"wrote {path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
