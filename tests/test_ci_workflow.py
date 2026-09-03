import subprocess
from pathlib import Path

from _workflow_yaml import load as _load_yaml

WORKFLOW_DIR = Path(__file__).parents[1] / ".github" / "workflows"


def _load(name: str) -> dict:
    return _load_yaml((WORKFLOW_DIR / name).read_text())


def test_verify_workflow_remains_the_authoritative_pull_request_workflow() -> None:
    assert (WORKFLOW_DIR / "verify.yml").is_file()
    assert not (WORKFLOW_DIR / "ci.yml").exists()


def test_gate_job_requires_every_selected_check() -> None:
    workflow = _load("verify.yml")
    gate = workflow["jobs"]["gate"]

    assert set(gate["needs"]) == {"changes", "actionlint", "test", "toolchain-checks", "web"}


def test_gate_job_script_fails_when_any_selected_job_did_not_succeed() -> None:
    workflow = _load("verify.yml")
    gate = workflow["jobs"]["gate"]
    script = gate["steps"][0]["run"]

    def run_gate(changes: str, actionlint: str, python: str, toolchain_checks: str, web: str) -> int:
        result = subprocess.run(
            ["bash", "-c", script],
            env={
                "CHANGES": changes,
                "ACTIONLINT": actionlint,
                "PYTHON": python,
                "TOOLCHAIN_CHECKS": toolchain_checks,
                "WEB": web,
            },
            capture_output=True,
            text=True,
        )
        return result.returncode

    assert run_gate("success", "success", "success", "success", "success") == 0
    assert run_gate("success", "skipped", "success", "skipped", "skipped") == 0
    assert run_gate("failure", "success", "success", "success", "success") == 1
    assert run_gate("success", "failure", "success", "success", "success") == 1
    assert run_gate("success", "success", "failure", "success", "success") == 1
    assert run_gate("success", "success", "success", "failure", "success") == 1
    assert run_gate("success", "success", "success", "success", "cancelled") == 1


def test_deploy_and_verify_web_jobs_install_through_the_same_containerized_task() -> None:
    verify_steps = {step["name"]: step["run"] for step in _load("verify.yml")["jobs"]["web"]["steps"] if "run" in step}
    deploy_steps = {
        step["name"]: step["run"] for step in _load("deploy.yml")["jobs"]["build"]["steps"] if "run" in step
    }

    install_command = "scripts/ci/run-in-dev-container.sh mise run web:install:npm"
    assert deploy_steps["Install dependencies"] == verify_steps["Install dependencies"] == install_command


def test_deploy_build_step_binds_dist_to_the_runner_filesystem_for_the_pages_artifact() -> None:
    deploy_steps = {
        step["name"]: step["run"] for step in _load("deploy.yml")["jobs"]["build"]["steps"] if "run" in step
    }
    verify_steps = {step["name"]: step["run"] for step in _load("verify.yml")["jobs"]["web"]["steps"] if "run" in step}

    # deploy's Pages artifact upload runs on the bare runner and needs dist on
    # its real filesystem; verify's web job never reads dist outside the
    # container, so it keeps using the isolated named volume instead.
    assert deploy_steps["Build"] == "scripts/ci/run-in-dev-container.sh --bind-dist mise run web:build:npm"
    assert verify_steps["Build"] == "scripts/ci/run-in-dev-container.sh mise run web:build:npm"


def test_toolchain_checks_and_web_jobs_gc_their_overlay_volumes() -> None:
    for job_name in ("toolchain-checks", "web"):
        job = _load("verify.yml")["jobs"][job_name]
        gc_steps = [step for step in job["steps"] if step.get("run") == "scripts/ci/gc-job-overlay-volumes.sh"]
        assert len(gc_steps) == 1
        assert gc_steps[0].get("if") == "always()"
