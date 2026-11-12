"""Conservative WezTerm Lua integration. Guessing is a failure."""

import re
from pathlib import Path

from sf2_theme.plan import Ownership

BUILDER = re.compile(r"^local\s+(\w+)\s*=\s*wezterm\.config_builder\(\)\s*$", re.MULTILINE)
RETURN_NAME = re.compile(r"^return\s+(\w+)\s*$", re.MULTILINE)
COLOR_SCHEME_LINE = re.compile(r"^\s*(?:\w+\.)?color_scheme\s*=")
SF2_SCHEME_VALUE = re.compile(r'"(?:sf2-[^"]*|street-fighter-2|Street Fighter II - [^"]*|street-fighter-ii-[^"]*)"')
TERM_THEME_GUARD = "TERM_THEME"
TERM_THEME_ASSIGN = (
    'sf2_env.TERM_THEME = (type(sf2_scheme) == "string" and sf2_scheme:find("-light", 1, true)) and "light" or "dark"'
)


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
            TERM_THEME_ASSIGN,
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


def _code(existing: str) -> str:
    """Drop Lua comment lines so commented-out integration never counts."""
    return "\n".join(line for line in existing.splitlines() if not line.lstrip().startswith("--"))


def _references_pointer(code: str, pointer: Path) -> bool:
    return "wezterm-current.lua" in code and (str(pointer) in code or "dofile" in code)


def _upgrade_term_theme_guard(existing: str, pointer: Path) -> str | None:
    """Insert TERM_THEME wiring after the managed color_scheme assignment when possible."""
    builder = BUILDER.search(existing)
    if builder is None:
        return None
    name = builder.group(1)
    lines = existing.splitlines(keepends=True)
    scheme_idx = next(
        (index for index, line in enumerate(lines) if "dofile(sf2_current)" in line and "color_scheme" in line),
        None,
    )
    if scheme_idx is None:
        return None
    indent = re.match(r"^(\s*)", lines[scheme_idx]).group(1)
    replacement = [
        f"{indent}local sf2_scheme = dofile(sf2_current)\n",
        f"{indent}{name}.color_scheme = sf2_scheme\n",
        f"{indent}local sf2_env = {name}.set_environment_variables or {{}}\n",
        f"{indent}{TERM_THEME_ASSIGN}\n",
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


def _builder_name(code: str) -> str | None:
    builder = BUILDER.search(code)
    returned = RETURN_NAME.search(code)
    if builder is None or returned is None or builder.group(1) != returned.group(1):
        return None
    return builder.group(1)


def classify(existing: str, pointer: Path) -> Ownership:
    """Classify who owns the color scheme in wezterm.lua.

    Pointer integration is recognised before the config shape, so a pasted
    snippet in a config sf2-themes cannot edit still counts as managed.
    """
    if not existing.strip():
        return Ownership.ABSENT
    code = _code(existing)
    if _references_pointer(code, pointer):
        if TERM_THEME_GUARD in code:
            return Ownership.MANAGED
        if _upgrade_term_theme_guard(existing, pointer) is not None:
            return Ownership.LEGACY
        return Ownership.UNKNOWN_SHAPE
    if _builder_name(code) is None:
        return Ownership.UNKNOWN_SHAPE
    if _has_scheme_kind(existing, "other"):
        return Ownership.FOREIGN
    if _has_scheme_kind(existing, "sf2"):
        return Ownership.LEGACY
    return Ownership.ABSENT


def integrate(existing: str, pointer: Path) -> str:
    """Return wezterm.lua with the managed pointer integration.

    The caller has already classified the config and approved replacing any
    color-scheme assignment; a shape `classify` rejects is returned unchanged.
    """
    if not existing.strip():
        return starter_config(pointer)
    code = _code(existing)
    if _references_pointer(code, pointer):
        if TERM_THEME_GUARD in code:
            return existing
        return _upgrade_term_theme_guard(existing, pointer) or existing
    name = _builder_name(code)
    cleaned = _strip_scheme_lines(existing, kinds=frozenset({"sf2", "other"}))
    returned = RETURN_NAME.search(cleaned)
    if name is None or returned is None:
        return existing
    assignment = integration_snippet(pointer, name).rstrip()
    prefix = cleaned[: returned.start()].rstrip()
    suffix = cleaned[returned.start() :]
    return f"{prefix}\n{assignment}\n{suffix}"
