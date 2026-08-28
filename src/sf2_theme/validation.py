"""Contrast, ANSI, and catalog uniqueness checks."""

from collections.abc import Sequence
from dataclasses import dataclass
from enum import StrEnum, unique

from sf2_theme.errors import ThemeError
from sf2_theme.model import ANSI_ORDER, HexColor, Theme, ThemeKind, ThemeVariant, project_adapter_colors

PRIMARY_CONTRAST: float = 7.0
TEXT_CONTRAST: float = 4.5
NONTEXT_CONTRAST: float = 3.0
EXPECTED_DARK_IDS: tuple[str, ...] = (
    "main",
    "akuma",
    "balrog",
    "blanka",
    "cammy",
    "chun-li",
    "dee-jay",
    "dhalsim",
    "e-honda",
    "fei-long",
    "guile",
    "ken",
    "m-bison",
    "ryu",
    "sagat",
    "t-hawk",
    "vega",
    "zangief",
)


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
        ("accent_secondary/background", ui.accent_secondary, ui.background, NONTEXT_CONTRAST),
    )
    for label, fg, bg, minimum in required:
        issue = _pair(theme_id, label, fg, bg, minimum)
        if issue is not None:
            issues.append(issue)
    for label, color in (("muted", ui.muted), ("subtle", ui.subtle)):
        ratio = contrast_ratio(color, ui.background)
        if ratio < TEXT_CONTRAST:
            issues.append(
                Issue(Severity.WARNING, f"{theme_id}: {label} on background is {ratio:.2f}:1 (secondary only)")
            )
    issues.extend(_projection_issues(theme))
    issues.extend(_semantic_issues(theme))
    issues.extend(_ansi_issues(theme))
    seen: dict[HexColor, list[str]] = {}
    for slot, color in _all_slots(theme):
        seen.setdefault(color, []).append(slot)
    for color, slots in seen.items():
        if len(slots) > 1:
            issues.append(Issue(Severity.WARNING, f"{theme_id}: {color} reused by {', '.join(slots)}"))
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
            issues.append(Issue(Severity.ERROR, f"{theme_id}: ansi.bright.{name} matches ansi.normal.{name}"))
    for row_name, row in (("normal", theme.ansi_normal), ("bright", theme.ansi_bright)):
        for name in ANSI_ORDER[1:7]:
            issue = _pair(
                theme_id,
                f"ansi.{row_name}.{name}/background",
                getattr(row, name),
                theme.ui.background,
                TEXT_CONTRAST,
            )
            if issue is not None:
                issues.append(issue)
    return issues


def _projection_issues(theme: Theme) -> list[Issue]:
    ui = theme.ui
    projection = project_adapter_colors(ui)
    expected = (
        ("cursor_bg", ui.cursor),
        ("cursor_fg", ui.cursor_text),
        ("selection_bg", ui.selection_background),
        ("selection_fg", ui.selection_foreground),
        ("panel_bg", ui.surface),
        ("sidebar_bg", ui.background),
        ("active_row_bg", ui.selection_background),
        ("navigate_row_bg", ui.overlay),
        ("surface_dim", ui.background),
        ("surface0", ui.surface),
        ("surface1", ui.overlay),
        ("overlay0", ui.border),
        ("overlay1", ui.muted),
        ("subtext", ui.subtle),
    )
    return [
        Issue(Severity.ERROR, f"{theme.metadata.id}: adapter projection {role} does not match canonical source")
        for role, color in expected
        if getattr(projection, role) != color
    ]


def _semantic_issues(theme: Theme) -> list[Issue]:
    return [
        Issue(Severity.ERROR, f"{theme.metadata.id}: semantic.{name} must match ansi.normal.{name}")
        for name in ANSI_ORDER[1:7]
        if getattr(theme.semantic, name) != getattr(theme.ansi_normal, name)
    ]


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
    """Check the exact roster, paired metadata, ids, and aliases."""
    issues: list[Issue] = []
    claimed: dict[str, str] = {}
    by_id = {theme.metadata.id: theme for theme in themes}
    expected_ids = {theme_id for dark_id in EXPECTED_DARK_IDS for theme_id in (dark_id, f"{dark_id}-light")}
    for theme_id in sorted(expected_ids - set(by_id)):
        issues.append(Issue(Severity.ERROR, f"missing expected theme {theme_id}"))
    for theme_id in sorted(set(by_id) - expected_ids):
        issues.append(Issue(Severity.ERROR, f"unexpected theme {theme_id}"))
    for theme in themes:
        meta = theme.metadata
        expected_variant = ThemeVariant.LIGHT if meta.id.endswith("-light") else ThemeVariant.DARK
        if meta.variant is not expected_variant:
            issues.append(Issue(Severity.ERROR, f"{meta.id}: variant must be {expected_variant.value}"))
        if meta.family != "sf2":
            issues.append(Issue(Severity.ERROR, f"{meta.id}: family must be sf2"))
        expected_kind = ThemeKind.MAIN if meta.id in {"main", "main-light"} else ThemeKind.CHARACTER
        if meta.kind is not expected_kind:
            issues.append(Issue(Severity.ERROR, f"{meta.id}: kind must be {expected_kind.value}"))
        for key in theme.lookup_keys():
            owner = claimed.get(key)
            if owner is not None:
                issues.append(Issue(Severity.ERROR, f"duplicate lookup key {key!r} on {owner} and {theme.metadata.id}"))
            else:
                claimed[key] = theme.metadata.id
        if "bison" in theme.lookup_keys():
            issues.append(Issue(Severity.ERROR, f"{theme.metadata.id}: alias 'bison' is regionally ambiguous"))
    for dark_id in EXPECTED_DARK_IDS:
        dark = by_id.get(dark_id)
        light = by_id.get(f"{dark_id}-light")
        if dark is not None and light is not None:
            issues.extend(_pairing_issues(dark, light))
    return issues


def _pairing_issues(dark: Theme, light: Theme) -> list[Issue]:
    dark_meta = dark.metadata
    light_meta = light.metadata
    expected = (
        ("display_name", f"{dark_meta.display_name} Light"),
        ("kind", dark_meta.kind),
        ("introduced_in", dark_meta.introduced_in),
        ("character", dark_meta.character),
        ("name", dark_meta.name),
        ("family", dark_meta.family),
        ("stage", dark_meta.stage),
    )
    issues = [
        Issue(Severity.ERROR, f"{light_meta.id}: {field} must match dark variant")
        for field, value in expected
        if getattr(light_meta, field) != value
    ]
    if light_meta.aliases:
        issues.append(Issue(Severity.ERROR, f"{light_meta.id}: light variant aliases must be empty"))
    if light.ui.background == dark.ui.background:
        issues.append(Issue(Severity.ERROR, f"{light_meta.id}: background must differ from dark variant"))
    if light.ui.foreground == dark.ui.foreground:
        issues.append(Issue(Severity.ERROR, f"{light_meta.id}: foreground must differ from dark variant"))
    return issues
