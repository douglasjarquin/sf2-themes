import os
import sys
from pathlib import Path

from sf2_theme.cli import dispatch

REAL_HOME = Path.home()

_EXACT_FILES = (
    ".config/wezterm/wezterm.lua",
    ".wezterm.lua",
    ".config/sf2-theme/wezterm-current.lua",
    ".config/sf2-theme/zsh-syntax-highlighting.zsh",
    ".config/starship.toml",
    ".config/herdr/config.toml",
    ".codex/config.toml",
    ".claude/settings.json",
    "Library/Application Support/lazygit/config.yml",
    ".config/lazygit/config.yml",
)

_MANAGED_GLOBS = (
    (".config/wezterm/colors", "sf2-*.toml"),
    (".config/nvim/colors", "sf2-*.lua"),
    (".config/nvim/sf2-theme", "*"),
    (".config/nvim/plugin", "sf2-theme.lua"),
    (".codex/themes", "sf2-*.tmTheme"),
    (".claude/themes", "sf2-*.json"),
    ("Library/Application Support/lazygit/themes", "sf2-*.yml"),
    (".config/lazygit/themes", "sf2-*.yml"),
)


def _snapshot_home(home: Path) -> dict[Path, bytes]:
    snapshot: dict[Path, bytes] = {}
    for relative in _EXACT_FILES:
        path = home / relative
        if path.is_file():
            snapshot[path] = path.read_bytes()
    for relative, pattern in _MANAGED_GLOBS:
        directory = home / relative
        if directory.is_dir():
            snapshot.update({path: path.read_bytes() for path in directory.glob(pattern) if path.is_file()})
    return snapshot


def test_default_apply_writes_only_inside_sandboxed_home() -> None:
    before = _snapshot_home(REAL_HOME)

    for app in ("wezterm", "herdr", "nvim", "codex", "starship", "lazygit", "claude"):
        assert dispatch(["apply", app]) == 0, app

    assert _snapshot_home(REAL_HOME) == before

    sandbox = Path(os.environ["HOME"])
    assert sandbox != REAL_HOME
    expected = [
        sandbox / ".config/wezterm/wezterm.lua",
        sandbox / ".config/wezterm/colors/sf2-main.toml",
        sandbox / ".config/sf2-theme/wezterm-current.lua",
        sandbox / ".config/sf2-theme/zsh-syntax-highlighting.zsh",
        sandbox / ".config/starship.toml",
        sandbox / ".config/herdr/config.toml",
        sandbox / ".config/nvim/plugin/sf2-theme.lua",
        sandbox / ".config/nvim/sf2-theme/current.lua",
        sandbox / ".codex/config.toml",
        sandbox / ".claude/settings.json",
    ]
    if sys.platform == "darwin":
        expected.append(sandbox / "Library/Application Support/lazygit/config.yml")
    else:
        expected.append(sandbox / ".config/lazygit/config.yml")
    for path in expected:
        assert path.is_file(), f"expected sandboxed write: {path}"
