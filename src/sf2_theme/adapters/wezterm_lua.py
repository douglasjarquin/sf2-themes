"""Conservative WezTerm Lua integration. Guessing is a failure."""

import re
from dataclasses import dataclass
from pathlib import Path

BUILDER = re.compile(r"^local\s+(\w+)\s*=\s*wezterm\.config_builder\(\)\s*$", re.MULTILINE)
RETURN_NAME = re.compile(r"^return\s+(\w+)\s*$", re.MULTILINE)
COLOR_SCHEME_LINE = re.compile(r"^\s*(?:\w+\.)?color_scheme\s*=")
SF2_SCHEME_VALUE = re.compile(r'"(?:sf2-[^"]*|street-fighter-2|Street Fighter II - [^"]*|street-fighter-ii-[^"]*)"')
TERM_THEME_GUARD = "TERM_THEME"


@dataclass(frozen=True, slots=True)
class LuaSetup:
    """Result of attempting to integrate the managed pointer into wezterm.lua."""

    content: str
    mutated: bool
    snippet: str | None


def lua_path_literal(path: Path) -> str:
    """Escape a filesystem path for a Lua double-quoted string."""
    return str(path).replace("\\", "\\\\").replace('"', '\\"')


def integration_snippet(pointer: Path, builder: str = "config") -> str:
    """Return the exact Lua block a human should paste.

    TERM_THEME follows the selected sf2 scheme so Cursor Agent and similar TUIs
    do not guess light chrome against a dark scheme (or the reverse).
    """
    literal = lua_path_literal(pointer)
    return "\n".join(
        (
            f'local sf2_current = "{literal}"',
            "wezterm.add_to_config_reload_watch_list(sf2_current)",
            "local sf2_scheme = dofile(sf2_current)",
            f"{builder}.color_scheme = sf2_scheme",
            f"local sf2_env = {builder}.set_environment_variables or {{}}",
            'sf2_env.TERM_THEME = (type(sf2_scheme) == "string" and sf2_scheme:find("-light", 1, true)) and "light" or "dark"',
            f"{builder}.set_environment_variables = sf2_env",
            "",
        )
    )


def starter_config(pointer: Path) -> str:
    """Return a complete starter wezterm.lua that selects the managed scheme."""
    return "\n".join(
        (
            'local wezterm = require("wezterm")',
            "local config = wezterm.config_builder()",
            integration_snippet(pointer).rstrip(),
            "return config",
            "",
        )
    )


def already_integrated(existing: str, pointer: Path) -> bool:
    """True when the file already dofiles the managed pointer with TERM_THEME."""
    marker = str(pointer)
    has_pointer = "wezterm-current.lua" in existing and (marker in existing or "dofile" in existing)
    return has_pointer and TERM_THEME_GUARD in existing


def needs_term_theme_upgrade(existing: str, pointer: Path) -> bool:
    """True when sf2 integration exists but the Cursor Agent TERM_THEME guard does not."""
    marker = str(pointer)
    has_pointer = "wezterm-current.lua" in existing and (marker in existing or "dofile" in existing)
    return has_pointer and TERM_THEME_GUARD not in existing


def _upgrade_term_theme_guard(existing: str, pointer: Path) -> str | None:
    """Insert TERM_THEME wiring after the managed color_scheme assignment when possible."""
    builder = BUILDER.search(existing)
    if builder is None:
        return None
    name = builder.group(1)
    lines = existing.splitlines(keepends=True)
    scheme_idx = next(
        (
            index
            for index, line in enumerate(lines)
            if "dofile(sf2_current)" in line and "color_scheme" in line
        ),
        None,
    )
    if scheme_idx is None:
        return None
    indent = re.match(r"^(\s*)", lines[scheme_idx]).group(1)
    replacement = [
        f"{indent}local sf2_scheme = dofile(sf2_current)\n",
        f"{indent}{name}.color_scheme = sf2_scheme\n",
        f"{indent}local sf2_env = {name}.set_environment_variables or {{}}\n",
        f'{indent}sf2_env.TERM_THEME = (type(sf2_scheme) == "string" and sf2_scheme:find("-light", 1, true)) and "light" or "dark"\n',
        f"{indent}{name}.set_environment_variables = sf2_env\n",
    ]
    # Drop a bare `local sf2_current` watch/assignment block's old one-line scheme set.
    new_lines = lines[:scheme_idx] + replacement + lines[scheme_idx + 1 :]
    return "".join(new_lines)


def _scheme_kind(line: str) -> str | None:
    if COLOR_SCHEME_LINE.match(line) is None:
        return None
    if SF2_SCHEME_VALUE.search(line) is not None:
        return "sf2"
    return "other"


def _strip_scheme_lines(existing: str, *, kinds: frozenset[str]) -> str:
    kept = [line for line in existing.splitlines() if _scheme_kind(line) not in kinds]
    newline = "\n" if existing.endswith("\n") else ""
    return "\n".join(kept) + newline


def _has_scheme_kind(existing: str, kind: str) -> bool:
    return any(_scheme_kind(line) == kind for line in existing.splitlines())


def setup_lua(existing: str, pointer: Path, *, adopt: bool = False) -> LuaSetup:
    """Mutate only empty files and known-safe config_builder configs."""
    snippet = integration_snippet(pointer)
    if not existing.strip():
        return LuaSetup(content=starter_config(pointer), mutated=True, snippet=None)
    if already_integrated(existing, pointer):
        return LuaSetup(content=existing, mutated=False, snippet=None)
    if needs_term_theme_upgrade(existing, pointer):
        upgraded = _upgrade_term_theme_guard(existing, pointer)
        if upgraded is not None:
            return LuaSetup(content=upgraded, mutated=True, snippet=None)
        return LuaSetup(content=existing, mutated=False, snippet=snippet)
    builder = BUILDER.search(existing)
    returned = RETURN_NAME.search(existing)
    if builder is None or returned is None or builder.group(1) != returned.group(1):
        return LuaSetup(content=existing, mutated=False, snippet=snippet)
    has_foreign = _has_scheme_kind(existing, "other")
    has_sf2 = _has_scheme_kind(existing, "sf2")
    if has_foreign and not adopt:
        return LuaSetup(content=existing, mutated=False, snippet=snippet)
    strip_kinds = {"sf2"}
    if adopt:
        strip_kinds.add("other")
    cleaned = _strip_scheme_lines(existing, kinds=frozenset(strip_kinds)) if (has_sf2 or adopt) else existing
    returned = RETURN_NAME.search(cleaned)
    if returned is None:
        return LuaSetup(content=existing, mutated=False, snippet=snippet)
    name = builder.group(1)
    assignment = integration_snippet(pointer, name).rstrip()
    prefix = cleaned[: returned.start()].rstrip()
    suffix = cleaned[returned.start() :]
    return LuaSetup(content=f"{prefix}\n{assignment}\n{suffix}", mutated=True, snippet=None)
