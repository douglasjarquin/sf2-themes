from pathlib import Path


def test_verify_workflow_remains_the_authoritative_pull_request_workflow() -> None:
    workflow_dir = Path(__file__).parents[1] / ".github" / "workflows"

    assert (workflow_dir / "verify.yml").is_file()
    assert not (workflow_dir / "ci.yml").exists()
