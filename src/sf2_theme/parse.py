"""Parse theme TOML into typed Theme values."""

from collections.abc import Mapping

from sf2_theme.errors import ThemeError
from sf2_theme.model import (
    ANSI_ORDER,
    SEMANTIC_FIELDS,
    UI_COLOR_FIELDS,
    AnsiColors,
    HexColor,
    IntroducedIn,
    SemanticColors,
    Theme,
    ThemeKind,
    ThemeMetadata,
    ThemeVariant,
    UiColors,
    parse_hex,
)


def _require_table(raw: object, name: str) -> Mapping[str, object]:  # object: TOML boundary
    match raw:
        case dict() as table:
            return table
        case _:
            raise ThemeError(f"{name} must be a table")


def _require_str(raw: object, name: str) -> str:  # object: TOML boundary
    match raw:
        case str() as text:
            return text
        case None:
            raise ThemeError(f"missing {name}")
        case _:
            raise ThemeError(f"{name} must be a string")


def _require_int(raw: object, name: str) -> int:  # object: TOML boundary
    match raw:
        case bool():
            raise ThemeError(f"{name} must be an integer")
        case int() as number:
            return number
        case _:
            raise ThemeError(f"{name} must be an integer")


def _require_str_list(raw: object, name: str) -> tuple[str, ...]:  # object: TOML boundary
    match raw:
        case None:
            return ()
        case list() as items:
            values: list[str] = []
            for index, item in enumerate(items):
                match item:
                    case str() as text:
                        values.append(text)
                    case _:
                        raise ThemeError(f"{name}[{index}] must be a string")
            return tuple(values)
        case _:
            raise ThemeError(f"{name} must be an array of strings")


def _hex_fields(table: Mapping[str, object], fields: tuple[str, ...], prefix: str) -> dict[str, HexColor]:
    unknown = sorted(set(table) - set(fields))
    if unknown:
        raise ThemeError(f"unknown {prefix} keys: {', '.join(unknown)}")
    return {name: parse_hex(_require_str(table.get(name), f"{prefix}.{name}")) for name in fields}


def _parse_kind(raw: str, source: str) -> ThemeKind:
    match raw:
        case ThemeKind.MAIN.value:
            return ThemeKind.MAIN
        case ThemeKind.CHARACTER.value:
            return ThemeKind.CHARACTER
        case _:
            raise ThemeError(f"{source}: kind must be main or character")


def _parse_era(raw: str, source: str) -> IntroducedIn:
    match raw:
        case IntroducedIn.MAIN.value:
            return IntroducedIn.MAIN
        case IntroducedIn.WORLD_WARRIOR.value:
            return IntroducedIn.WORLD_WARRIOR
        case IntroducedIn.CHAMPION_EDITION.value:
            return IntroducedIn.CHAMPION_EDITION
        case IntroducedIn.SUPER.value:
            return IntroducedIn.SUPER
        case IntroducedIn.SUPER_TURBO.value:
            return IntroducedIn.SUPER_TURBO
        case _:
            raise ThemeError(f"{source}: unknown introduced_in {raw!r}")


def _parse_variant(raw: str, source: str) -> ThemeVariant:
    match raw:
        case ThemeVariant.DARK.value:
            return ThemeVariant.DARK
        case ThemeVariant.LIGHT.value:
            return ThemeVariant.LIGHT
        case _:
            raise ThemeError(f"{source}: variant must be dark or light")


def parse_theme(raw: object, *, source: str) -> Theme:  # object: TOML boundary
    """Parse a complete theme table. `source` is used in error messages."""
    table = _require_table(raw, source)
    extra_top = sorted(set(table) - {"schema_version", "meta", "ui", "semantic", "ansi"})
    if extra_top:
        raise ThemeError(f"{source}: unknown top-level keys: {', '.join(extra_top)}")
    version = _require_int(table.get("schema_version"), f"{source}.schema_version")
    if version != 1:
        raise ThemeError(f"{source}: unsupported schema_version {version}")
    meta_table = _require_table(table.get("meta"), f"{source}.meta")
    kind = _parse_kind(_require_str(meta_table.get("kind"), f"{source}.meta.kind"), source)
    character_raw = meta_table.get("character")
    character: str | None
    match character_raw:
        case None if kind is ThemeKind.MAIN:
            character = None
        case str() as name if kind is ThemeKind.CHARACTER:
            character = name
        case _:
            raise ThemeError(f"{source}: character is required for character themes and forbidden for main")
    allowed_meta = {
        "id",
        "display_name",
        "kind",
        "introduced_in",
        "character",
        "aliases",
        "name",
        "variant",
        "family",
        "stage",
    }
    extra_meta = sorted(set(meta_table) - allowed_meta)
    if extra_meta:
        raise ThemeError(f"{source}: unknown meta keys: {', '.join(extra_meta)}")
    metadata = ThemeMetadata(
        id=_require_str(meta_table.get("id"), f"{source}.meta.id"),
        display_name=_require_str(meta_table.get("display_name"), f"{source}.meta.display_name"),
        kind=kind,
        introduced_in=_parse_era(
            _require_str(meta_table.get("introduced_in"), f"{source}.meta.introduced_in"),
            source,
        ),
        character=character,
        aliases=_require_str_list(meta_table.get("aliases"), f"{source}.meta.aliases"),
        name=_require_str(meta_table.get("name"), f"{source}.meta.name"),
        variant=_parse_variant(
            _require_str(meta_table.get("variant"), f"{source}.meta.variant"),
            source,
        ),
        family=_require_str(meta_table.get("family"), f"{source}.meta.family"),
        stage=_require_str(meta_table.get("stage"), f"{source}.meta.stage"),
    )
    ui = UiColors(**_hex_fields(_require_table(table.get("ui"), f"{source}.ui"), UI_COLOR_FIELDS, f"{source}.ui"))
    semantic = SemanticColors(
        **_hex_fields(
            _require_table(table.get("semantic"), f"{source}.semantic"),
            SEMANTIC_FIELDS,
            f"{source}.semantic",
        )
    )
    ansi_table = _require_table(table.get("ansi"), f"{source}.ansi")
    extra_ansi = sorted(set(ansi_table) - {"normal", "bright"})
    if extra_ansi:
        raise ThemeError(f"{source}: unknown ansi keys: {', '.join(extra_ansi)}")
    normal = AnsiColors(
        **_hex_fields(
            _require_table(ansi_table.get("normal"), f"{source}.ansi.normal"),
            ANSI_ORDER,
            f"{source}.ansi.normal",
        )
    )
    bright = AnsiColors(
        **_hex_fields(
            _require_table(ansi_table.get("bright"), f"{source}.ansi.bright"),
            ANSI_ORDER,
            f"{source}.ansi.bright",
        )
    )
    return Theme(metadata=metadata, ui=ui, semantic=semantic, ansi_normal=normal, ansi_bright=bright)
