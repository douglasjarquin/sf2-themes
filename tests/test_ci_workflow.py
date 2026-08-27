import subprocess
from pathlib import Path

import yaml

WORKFLOW_DIR = Path(__file__).parents[1] / ".github" / "workflows"


def _load(name: str) -> dict:
    return yaml.safe_load((WORKFLOW_DIR / name).read_text())


def test_verify_workflow_remains_the_authoritative_pull_request_workflow() -> None:
    assert (WORKFLOW_DIR / "verify.yml").is_file()
    assert not (WORKFLOW_DIR / "ci.yml").exists()


def test_gate_job_requires_every_selected_check() -> None:
    workflow = _load("verify.yml")
    gate = workflow["jobs"]["gate"]

    assert set(gate["needs"]) == {"changes", "actionlint", "test", "web"}


def test_gate_job_script_fails_when_any_selected_job_did_not_succeed() -> None:
    workflow = _load("verify.yml")
    gate = workflow["jobs"]["gate"]
    script = gate["steps"][0]["run"]

    def run_gate(changes: str, actionlint: str, python: str, web: str) -> int:
        result = subprocess.run(
            ["bash", "-c", script],
            env={"CHANGES": changes, "ACTIONLINT": actionlint, "PYTHON": python, "WEB": web},
            capture_output=True,
            text=True,
        )
        return result.returncode

    assert run_gate("success", "success", "success", "success") == 0
    assert run_gate("success", "skipped", "success", "skipped") == 0
    assert run_gate("failure", "success", "success", "success") == 1
    assert run_gate("success", "failure", "success", "success") == 1
    assert run_gate("success", "success", "failure", "success") == 1
    assert run_gate("success", "success", "success", "cancelled") == 1


def test_deploy_and_verify_web_jobs_use_the_same_lockfile_backed_install_and_build() -> None:
    verify_steps = {step["name"]: step["run"] for step in _load("verify.yml")["jobs"]["web"]["steps"] if "run" in step}
    deploy_steps = {
        step["name"]: step["run"] for step in _load("deploy.yml")["jobs"]["build"]["steps"] if "run" in step
    }

    assert deploy_steps["Install dependencies"] == verify_steps["Install dependencies"] == "npm --prefix web ci"
    assert deploy_steps["Build"] == verify_steps["Build"] == "npm --prefix web run build"
