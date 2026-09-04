import json
import tomllib
from pathlib import Path

from _workflow_yaml import load as _load_yaml

ROOT = Path(__file__).parents[1]


def _environment() -> dict:
    return json.loads((ROOT / ".cursor" / "environment.json").read_text())


def test_cursor_environment_points_at_the_dev_dockerfile() -> None:
    config = _environment()
    dockerfile = (ROOT / ".cursor" / config["build"]["dockerfile"]).resolve()

    assert config["build"]["context"] == ".."
    assert dockerfile == (ROOT / "docker" / "dev" / "Dockerfile").resolve()
    assert dockerfile.is_file()


def test_cursor_environment_install_refreshes_locked_tools_and_deps() -> None:
    config = _environment()

    assert config["user"] == "dev"
    assert config["install"] == "mise install --locked && mise run deps"
    assert config["terminals"][0]["name"] == "web"
    assert config["terminals"][0]["command"] == "mise run web:dev:container"


def test_dev_dockerfile_defaults_to_the_same_ubuntu_pin_as_the_toolchain() -> None:
    toolchain = (ROOT / "docker" / "toolchain" / "Dockerfile").read_text()
    dev = (ROOT / "docker" / "dev" / "Dockerfile").read_text()
    ubuntu = next(line.split("=", 1)[1] for line in toolchain.splitlines() if line.startswith("ARG UBUNTU_IMAGE="))

    assert f"ARG TOOLCHAIN_IMAGE={ubuntu}" in dev
    assert "ARG TOOLCHAIN_IMAGE=sf2-themes-toolchain" not in dev
    assert 'COPY docker/toolchain/apt-packages.txt' in dev


def test_deps_task_installs_python_aube_and_playwright() -> None:
    mise = tomllib.loads((ROOT / "mise.toml").read_text())
    run = mise["tasks"]["deps"]["run"]

    assert "uv sync --all-extras" in run
    assert "aube -C web install" in run
    assert "playwright install chromium" in run


def test_verify_python_job_runs_when_cursor_environment_changes() -> None:
    workflow = _load_yaml((ROOT / ".github" / "workflows" / "verify.yml").read_text())
    steps = workflow["jobs"]["changes"]["steps"]
    filters = next(step["with"]["filters"] for step in steps if "filters" in step.get("with", {}))

    assert ".cursor/**" in filters
