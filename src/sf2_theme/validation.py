"""Contrast, ANSI, and catalog uniqueness checks."""

from collections.abc import Sequence
from dataclasses import dataclass
from enum import StrEnum, unique

from sf2_theme.errors import ThemeError
from sf2_theme.model import ANSI_ORDER, HexColor, Theme

PRIMARY_CONTRAST: float = 7.0
TEXT_CONTRAST: float = 4.5
NONTEXT_CONTRAST: float = 3.0


@unique
class Severity(StrEnum):
    ERROR = "error"
    WARNING = "warning"


@dataclass(frozen=True, slots=True)
class Issue:
    """One validation finding."""

    severity: Severity
    message: str


def relative_luminance(color: HexColor) -> float:
    """WCAG relative luminance of a #rrggbb color."""
    value = color.removeprefix("#")
    red, green, blue = (int(value[i : i + 2], 16) for i in (0, 2, 4))

    def linear(channel: int) -> float:
        srgb = channel / 255.0
        if srgb <= 0.04045:
            return srgb / 12.92
        return ((srgb + 0.055) / 1.055) ** 2.4

    return 0.2126 * linear(red) + 0.7152 * linear(green) + 0.0722 * linear(blue)


def contrast_ratio(foreground: HexColor, background: HexColor) -> float:
    """WCAG contrast ratio between two colors."""
    first, second = relative_luminance(foreground), relative_luminance(background)
    lighter, darker = (first, second) if first >= second else (second, first)
    return (lighter + 0.05) / (darker + 0.05)


def _pair(theme_id: str, label: str, fg: HexColor, bg: HexColor, minimum: float) -> Issue | None:
    ratio = contrast_ratio(fg, bg)
    if ratio + 1e-9 >= minimum:
        return None
    return Issue(
        Severity.ERROR,
        f"{theme_id}: {label} contrast {ratio:.2f}:1 is below {minimum:.1f}:1 ({fg} on {bg})",
    )


def validate_theme(theme: Theme) -> tuple[Issue, ...]:
    """Return errors and warnings for one fully parsed theme."""
    ui = theme.ui
    theme_id = theme.metadata.id
    issues: list[Issue] = []
    required = (
        ("foreground/background", ui.foreground, ui.background, PRIMARY_CONTRAST),
        ("foreground/surface0", ui.foreground, ui.surface0, TEXT_CONTRAST),
        ("foreground/surface1", ui.foreground, ui.surface1, TEXT_CONTRAST),
        ("foreground/panel_bg", ui.foreground, ui.panel_bg, TEXT_CONTRAST),
        ("foreground/sidebar_bg", ui.foreground, ui.sidebar_bg, TEXT_CONTRAST),
        ("foreground/active_row_bg", ui.foreground, ui.active_row_bg, TEXT_CONTRAST),
        ("foreground/navigate_row_bg", ui.foreground, ui.navigate_row_bg, TEXT_CONTRAST),
        ("selection", ui.selection_fg, ui.selection_bg, TEXT_CONTRAST),
        ("cursor", ui.cursor_fg, ui.cursor_bg, TEXT_CONTRAST),
        ("accent/background", ui.accent, ui.background, NONTEXT_CONTRAST),
    )
    for label, fg, bg, minimum in required:
        issue = _pair(theme_id, label, fg, bg, minimum)
        if issue is not None:
            issues.append(issue)
    subtext_ratio = contrast_ratio(ui.subtext, ui.background)
    if subtext_ratio < TEXT_CONTRAST:
        issues.append(
            Issue(
                Severity.WARNING,
                f"{theme_id}: subtext on background is {subtext_ratio:.2f}:1 (secondary only)",
            )
        )
    issues.extend(_ansi_issues(theme))
    seen: dict[HexColor, list[str]] = {}
    for slot, color in _all_slots(theme):
        seen.setdefault(color, []).append(slot)
    for color, slots in seen.items():
        if len(slots) > 1:
            issues.append(
                Issue(Severity.WARNING, f"{theme_id}: {color} reused by {', '.join(slots)}")
            )
    return tuple(issues)


def _ansi_issues(theme: Theme) -> list[Issue]:
    issues: list[Issue] = []
    theme_id = theme.metadata.id
    normal = theme.ansi_normal.as_tuple()
    bright = theme.ansi_bright.as_tuple()
    if len(set(normal)) != 8:
        issues.append(Issue(Severity.ERROR, f"{theme_id}: ansi.normal has duplicate colors"))
    if len(set(bright)) != 8:
        issues.append(Issue(Severity.ERROR, f"{theme_id}: ansi.bright has duplicate colors"))
    for name, left, right in zip(ANSI_ORDER, normal, bright, strict=True):
        if left == right:
            issues.append(
                Issue(Severity.ERROR, f"{theme_id}: ansi.bright.{name} matches ansi.normal.{name}")
            )
    return issues


def _all_slots(theme: Theme) -> list[tuple[str, HexColor]]:
    slots: list[tuple[str, HexColor]] = []
    for name in theme.ui.__slots__:
        slots.append((f"ui.{name}", getattr(theme.ui, name)))
    for name in theme.semantic.__slots__:
        slots.append((f"semantic.{name}", getattr(theme.semantic, name)))
    for name in ANSI_ORDER:
        slots.append((f"ansi.normal.{name}", getattr(theme.ansi_normal, name)))
        slots.append((f"ansi.bright.{name}", getattr(theme.ansi_bright, name)))
    return slots


def require_valid(theme: Theme) -> None:
    """Raise ThemeError if the theme has any error-severity issues."""
    errors = [issue.message for issue in validate_theme(theme) if issue.severity is Severity.ERROR]
    if errors:
        raise ThemeError("\n".join(errors))


def validate_catalog(themes: Sequence[Theme]) -> tuple[Issue, ...]:
    """Check uniqueness of ids and aliases across the catalog."""
    issues: list[Issue] = []
    claimed: dict[str, str] = {}
    for theme in themes:
        for key in theme.lookup_keys():
            owner = claimed.get(key)
            if owner is not None:
                issues.append(
                    Issue(Severity.ERROR, f"duplicate lookup key {key!r} on {owner} and {theme.metadata.id}")
                )
            else:
                claimed[key] = theme.metadata.id
        if "bison" in theme.lookup_keys():
            issues.append(Issue(Severity.ERROR, f"{theme.metadata.id}: alias 'bison' is regionally ambiguous"))
    return issues
