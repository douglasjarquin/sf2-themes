from pathlib import Path

import pytest

_ADAPTER_ENV_VARS = (
    "XDG_CONFIG_HOME",
    "WEZTERM_CONFIG_DIR",
    "WEZTERM_CONFIG_FILE",
    "STARSHIP_CONFIG",
    "NVIM_CONFIG_DIR",
    "LAZYGIT_CONFIG_DIR",
    "HERDR_CONFIG_PATH",
    "CODEX_HOME",
    "CLAUDE_CONFIG_DIR",
    "SF2_THEME_DIR",
)


@pytest.fixture(autouse=True)
def _sandboxed_home(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    home = tmp_path / "home"
    home.mkdir()
    monkeypatch.setenv("HOME", str(home))
    monkeypatch.setenv("USERPROFILE", str(home))
    for variable in _ADAPTER_ENV_VARS:
        monkeypatch.delenv(variable, raising=False)
