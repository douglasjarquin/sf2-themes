"""Minimal, dependency-free YAML subset loader for GitHub Actions workflow files.

Supports exactly the constructs used in .github/workflows/*.yml: block
mappings, block sequences of mappings, quoted/unquoted scalars, and literal
block scalars ("|"). It is not a general-purpose YAML parser.
"""

from __future__ import annotations


def load(text: str) -> dict:
    lines = text.splitlines()
    value, _ = _parse_block(lines, 0, 0)
    return value if value is not None else {}


def _indent_of(line: str) -> int:
    return len(line) - len(line.lstrip(" "))


def _is_blank_or_comment(line: str) -> bool:
    stripped = line.strip()
    return stripped == "" or stripped.startswith("#")


def _skip_noise(lines: list[str], i: int) -> int:
    while i < len(lines) and _is_blank_or_comment(lines[i]):
        i += 1
    return i


def _parse_scalar(raw: str):
    text = raw.strip()
    if len(text) >= 2 and text[0] == text[-1] and text[0] in "\"'":
        return text[1:-1]
    if text.startswith("[") and text.endswith("]"):
        inner = text[1:-1].strip()
        return [] if inner == "" else [_parse_scalar(item) for item in inner.split(",")]
    if text in ("", "null", "~"):
        return None
    if text == "true":
        return True
    if text == "false":
        return False
    return text


def _parse_block(lines: list[str], i: int, indent: int):
    i = _skip_noise(lines, i)
    if i >= len(lines):
        return None, i
    first_indent = _indent_of(lines[i])
    if first_indent < indent:
        return None, i
    content = lines[i][first_indent:]
    if content == "-" or content.startswith("- "):
        return _parse_sequence(lines, i, first_indent)
    return _parse_mapping(lines, i, first_indent)


def _parse_sequence(lines: list[str], i: int, indent: int):
    items = []
    while True:
        i = _skip_noise(lines, i)
        if i >= len(lines) or _indent_of(lines[i]) != indent:
            break
        content = lines[i][indent:]
        if not (content == "-" or content.startswith("- ")):
            break
        rest = content[2:] if content.startswith("- ") else ""
        item_indent = indent + 2
        if rest.strip() == "":
            value, i = _parse_block(lines, i + 1, item_indent)
        else:
            synthetic = lines[:i] + [(" " * item_indent) + rest] + lines[i + 1 :]
            value, i = _parse_mapping(synthetic, i, item_indent)
        items.append(value)
    return items, i


def _parse_block_scalar(lines: list[str], i: int, key_indent: int):
    content_lines = []
    base_indent = None
    while i < len(lines):
        line = lines[i]
        if line.strip() == "":
            content_lines.append("")
            i += 1
            continue
        cur_indent = _indent_of(line)
        if cur_indent <= key_indent:
            break
        if base_indent is None:
            base_indent = cur_indent
        content_lines.append(line[base_indent:])
        i += 1
    while content_lines and content_lines[-1] == "":
        content_lines.pop()
    return "\n".join(content_lines), i


def _parse_mapping(lines: list[str], i: int, indent: int):
    result: dict = {}
    while True:
        i = _skip_noise(lines, i)
        if i >= len(lines) or _indent_of(lines[i]) != indent:
            break
        content = lines[i][indent:]
        if content == "-" or content.startswith("- "):
            break
        key, sep, rest = content.partition(":")
        if not sep:
            break
        key = _parse_scalar(key)
        rest = rest.strip()
        if rest == "":
            nxt = _skip_noise(lines, i + 1)
            if nxt < len(lines) and _indent_of(lines[nxt]) > indent:
                value, i = _parse_block(lines, i + 1, _indent_of(lines[nxt]))
            else:
                value = None
                i += 1
        elif rest in ("|", "|-", "|+", ">", ">-"):
            value, i = _parse_block_scalar(lines, i + 1, indent)
        else:
            value = _parse_scalar(rest)
            i += 1
        result[key] = value
    return result, i
