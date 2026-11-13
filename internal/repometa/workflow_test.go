// Package repometa holds contract tests over the repository's CI workflows, dev container, docs,
// and Cursor Cloud environment. It has no runtime code.
package repometa

import (
	"bufio"
	"errors"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// root is the repository root, relative to this package's directory.
const root = "../.."

// workflowStep is the part of a GitHub Actions step these tests read.
type workflowStep struct {
	Name string            `yaml:"name"`
	ID   string            `yaml:"id"`
	If   string            `yaml:"if"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	With map[string]string `yaml:"with"`
	Env  map[string]string `yaml:"env"`
	// ContinueOnError lets the job go on when the step fails.
	ContinueOnError bool `yaml:"continue-on-error"`
}

// workflowJob is the part of a GitHub Actions job these tests read.
type workflowJob struct {
	Needs   stringList        `yaml:"needs"`
	If      string            `yaml:"if"`
	Outputs map[string]string `yaml:"outputs"`
	Steps   []workflowStep    `yaml:"steps"`
}

type workflow struct {
	Jobs map[string]workflowJob `yaml:"jobs"`
}

// stringList reads `needs`, which is a single job name or a list of them.
type stringList []string

func (l *stringList) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		*l = []string{node.Value}
		return nil
	}
	var list []string
	if err := node.Decode(&list); err != nil {
		return err
	}
	*l = list
	return nil
}

func readFile(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func loadWorkflow(t *testing.T, name string) workflow {
	t.Helper()
	var w workflow
	if err := yaml.Unmarshal([]byte(readFile(t, ".github/workflows/"+name)), &w); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return w
}

func job(t *testing.T, w workflow, name string) workflowJob {
	t.Helper()
	j, ok := w.Jobs[name]
	if !ok {
		t.Fatalf("no %s job", name)
	}
	return j
}

// runSteps maps each named step that runs a command to that command.
func runSteps(j workflowJob) map[string]string {
	steps := map[string]string{}
	for _, s := range j.Steps {
		if s.Run != "" {
			steps[s.Name] = s.Run
		}
	}
	return steps
}

func step(t *testing.T, j workflowJob, id string) workflowStep {
	t.Helper()
	for _, s := range j.Steps {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no step with id %s", id)
	return workflowStep{}
}

// pathFilters parses the dorny/paths-filter `filters` block of the changes job.
func pathFilters(t *testing.T) map[string][]string {
	t.Helper()
	filter := step(t, job(t, loadWorkflow(t, "verify.yml"), "changes"), "filter")
	var filters map[string][]string
	if err := yaml.Unmarshal([]byte(filter.With["filters"]), &filters); err != nil {
		t.Fatalf("filters: %v", err)
	}
	return filters
}

// runBash runs script with bash in a cleared environment and returns its exit code.
func runBash(t *testing.T, script string, env map[string]string) int {
	t.Helper()
	command := exec.Command("bash", "-c", script)
	command.Env = []string{"PATH=" + os.Getenv("PATH")}
	for key, value := range env {
		command.Env = append(command.Env, key+"="+value)
	}
	out, err := command.CombinedOutput()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exitErr):
		return exitErr.ExitCode()
	}
	t.Fatalf("bash: %v\n%s", err, out)
	return -1
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}

func TestVerifyWorkflowRemainsTheAuthoritativePullRequestWorkflow(t *testing.T) {
	if _, err := os.Stat(filepath.Join(root, ".github/workflows/verify.yml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".github/workflows/ci.yml")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ci.yml exists beside verify.yml: %v", err)
	}
}

func TestVerifyWorkflowRunsTheGoAndToolingJobSet(t *testing.T) {
	w := loadWorkflow(t, "verify.yml")
	want := []string{"actionlint", "changes", "gate", "go", "release-rehearsal", "toolchain-checks", "web"}
	if got := sortedKeys(w.Jobs); !slices.Equal(got, want) {
		t.Errorf("jobs = %v, want %v", got, want)
	}
	for name, j := range w.Jobs {
		for _, s := range j.Steps {
			if strings.Contains(strings.ToLower(s.Uses+s.Run), "python") {
				t.Errorf("%s job step %q still uses Python: %s%s", name, s.Name, s.Uses, s.Run)
			}
		}
	}
}

func TestChangesJobSelectsEveryFilterAndFailsOpen(t *testing.T) {
	filters := pathFilters(t)
	names := []string{"go", "tooling", "web", "workflows"}
	if got := sortedKeys(filters); !slices.Equal(got, names) {
		t.Fatalf("path filters = %v, want %v", got, names)
	}
	changes := job(t, loadWorkflow(t, "verify.yml"), "changes")
	if got := sortedKeys(changes.Outputs); !slices.Equal(got, names) {
		t.Errorf("changes outputs = %v, want %v", got, names)
	}
	for _, name := range names {
		if want := "${{ steps.select.outputs." + name + " }}"; changes.Outputs[name] != want {
			t.Errorf("changes output %s = %q, want %q", name, changes.Outputs[name], want)
		}
	}
	filter := step(t, changes, "filter")
	if !strings.HasPrefix(filter.Uses, "dorny/paths-filter@") {
		t.Errorf("filter step uses %q", filter.Uses)
	}
	// A failed filter step must not fail the job: the select step then runs every check.
	if !filter.ContinueOnError {
		t.Errorf("the filter step stops the changes job when it fails")
	}

	selectStep := step(t, changes, "select")
	if selectStep.If != "always()" {
		t.Errorf("select step if = %q, want always()", selectStep.If)
	}
	selected := func(env map[string]string) map[string]string {
		t.Helper()
		out := filepath.Join(t.TempDir(), "github-output")
		if err := os.WriteFile(out, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		env["GITHUB_OUTPUT"] = out
		if code := runBash(t, selectStep.Run, env); code != 0 {
			t.Fatalf("select step exited %d", code)
		}
		file, err := os.Open(out)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = file.Close() }()
		got := map[string]string{}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			key, value, _ := strings.Cut(scanner.Text(), "=")
			got[key] = value
		}
		return got
	}
	filterValues := map[string]string{
		"FILTER_WORKFLOWS": "false", "FILTER_TOOLING": "true", "FILTER_GO": "false", "FILTER_WEB": "true",
	}
	withFilters := func(env map[string]string) map[string]string {
		for key, value := range filterValues {
			env[key] = value
		}
		return env
	}
	everything := map[string]string{"workflows": "true", "tooling": "true", "go": "true", "web": "true"}
	for _, tc := range []struct {
		name string
		env  map[string]string
		want map[string]string
	}{
		{"filter step failed", withFilters(map[string]string{"FILTER_OUTCOME": "failure", "EVENT_NAME": "pull_request", "BASE_SHA": "abc"}), everything},
		{"filter step skipped its outputs", map[string]string{"FILTER_OUTCOME": "failure", "EVENT_NAME": "pull_request", "BASE_SHA": "abc"}, everything},
		{"push with no before commit", withFilters(map[string]string{"FILTER_OUTCOME": "success", "EVENT_NAME": "push", "EVENT_BEFORE": ""}), everything},
		{"push creating the branch", withFilters(map[string]string{"FILTER_OUTCOME": "success", "EVENT_NAME": "push", "EVENT_BEFORE": strings.Repeat("0", 40)}), everything},
		{"pull request with no base", withFilters(map[string]string{"FILTER_OUTCOME": "success", "EVENT_NAME": "pull_request", "BASE_SHA": ""}), everything},
		{"pull request with filter data", withFilters(map[string]string{"FILTER_OUTCOME": "success", "EVENT_NAME": "pull_request", "BASE_SHA": "abc"}),
			map[string]string{"workflows": "false", "tooling": "true", "go": "false", "web": "true"}},
		{"push with filter data", withFilters(map[string]string{"FILTER_OUTCOME": "success", "EVENT_NAME": "push", "EVENT_BEFORE": "abc"}),
			map[string]string{"workflows": "false", "tooling": "true", "go": "false", "web": "true"}},
		{"successful filter missing one output", map[string]string{"FILTER_OUTCOME": "success", "EVENT_NAME": "pull_request", "BASE_SHA": "abc",
			"FILTER_WORKFLOWS": "false", "FILTER_TOOLING": "false", "FILTER_WEB": "false"}, everything},
		{"successful filter with a non-boolean output", map[string]string{"FILTER_OUTCOME": "success", "EVENT_NAME": "pull_request", "BASE_SHA": "abc",
			"FILTER_WORKFLOWS": "false", "FILTER_TOOLING": "false", "FILTER_GO": "maybe", "FILTER_WEB": "false"}, everything},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := selected(tc.env); !maps.Equal(got, tc.want) {
				t.Errorf("selected %v, want %v", got, tc.want)
			}
		})
	}
	for _, name := range names {
		if want := "${{ steps.filter.outputs." + name + " }}"; selectStep.Env["FILTER_"+strings.ToUpper(name)] != want {
			t.Errorf("select env FILTER_%s = %q, want %q", strings.ToUpper(name), selectStep.Env["FILTER_"+strings.ToUpper(name)], want)
		}
	}
}

func TestEachJobRunsOnItsOwnFilter(t *testing.T) {
	w := loadWorkflow(t, "verify.yml")
	for jobName, filter := range map[string]string{
		"actionlint": "workflows", "toolchain-checks": "tooling", "go": "go", "release-rehearsal": "go", "web": "web",
	} {
		j := job(t, w, jobName)
		if want := "needs.changes.outputs." + filter + " == 'true'"; j.If != want {
			t.Errorf("%s if = %q, want %q", jobName, j.If, want)
		}
		if !slices.Equal(j.Needs, []string{"changes"}) {
			t.Errorf("%s needs = %v, want [changes]", jobName, j.Needs)
		}
	}
}

func TestContractTestsRunWhenTheFilesTheyReadChange(t *testing.T) {
	// The go job runs these contract tests, so it must run when a file they read changes.
	filters := pathFilters(t)
	for _, path := range []string{".github/workflows/**", ".cursor/**", "docker/**", "scripts/ci/**", "mise.toml", "internal/**"} {
		if !slices.Contains(filters["go"], path) {
			t.Errorf("the go filter misses %s", path)
		}
	}
	for _, path := range []string{".cursor/**", "docker/**", "scripts/**", "mise.toml", "mise.lock"} {
		if !slices.Contains(filters["tooling"], path) {
			t.Errorf("the tooling filter misses %s", path)
		}
	}
	// The web build runs the Go generator over the catalog.
	for _, path := range []string{"web/**", "themes/**", "cmd/sf2-dev/**", "internal/**", "go.mod", "go.sum"} {
		if !slices.Contains(filters["web"], path) {
			t.Errorf("the web filter misses %s", path)
		}
	}
}

func TestGateJobRequiresEverySelectedCheck(t *testing.T) {
	gate := job(t, loadWorkflow(t, "verify.yml"), "gate")
	got := slices.Clone([]string(gate.Needs))
	sort.Strings(got)
	if want := []string{"actionlint", "changes", "go", "release-rehearsal", "toolchain-checks", "web"}; !slices.Equal(got, want) {
		t.Errorf("gate needs = %v, want %v", got, want)
	}
	if gate.If != "always()" {
		t.Errorf("gate if = %q, want always()", gate.If)
	}
}

func TestGateJobScriptRequiresSelectedJobsToSucceed(t *testing.T) {
	gate := job(t, loadWorkflow(t, "verify.yml"), "gate")
	script := gate.Steps[0].Run
	results := map[string]string{
		"CHANGES": "changes", "ACTIONLINT": "actionlint", "TOOLCHAIN_CHECKS": "toolchain-checks",
		"GO": "go", "RELEASE_REHEARSAL": "release-rehearsal", "WEB": "web",
	}
	selections := map[string]string{
		"SELECTED_WORKFLOWS": "workflows", "SELECTED_TOOLING": "tooling", "SELECTED_GO": "go", "SELECTED_WEB": "web",
	}
	want := slices.Sorted(maps.Keys(results))
	want = slices.Sorted(slices.Values(append(want, slices.Collect(maps.Keys(selections))...)))
	if got := sortedKeys(gate.Steps[0].Env); !slices.Equal(got, want) {
		t.Errorf("gate env = %v, want %v", got, want)
	}
	for name, need := range results {
		if want := "${{ needs." + need + ".result }}"; gate.Steps[0].Env[name] != want {
			t.Errorf("gate env %s = %q, want %q", name, gate.Steps[0].Env[name], want)
		}
	}
	for name, output := range selections {
		if want := "${{ needs.changes.outputs." + output + " }}"; gate.Steps[0].Env[name] != want {
			t.Errorf("gate env %s = %q, want %q", name, gate.Steps[0].Env[name], want)
		}
	}
	all := map[string]string{"SELECTED_WORKFLOWS": "true", "SELECTED_TOOLING": "true", "SELECTED_GO": "true", "SELECTED_WEB": "true"}
	none := map[string]string{"SELECTED_WORKFLOWS": "false", "SELECTED_TOOLING": "false", "SELECTED_GO": "false", "SELECTED_WEB": "false"}
	only := func(output string) map[string]string {
		selection := maps.Clone(none)
		selection["SELECTED_"+strings.ToUpper(output)] = "true"
		return selection
	}
	succeeded := map[string]string{"CHANGES": "success", "ACTIONLINT": "success", "TOOLCHAIN_CHECKS": "success", "GO": "success", "RELEASE_REHEARSAL": "success", "WEB": "success"}
	skipped := map[string]string{"CHANGES": "success", "ACTIONLINT": "skipped", "TOOLCHAIN_CHECKS": "skipped", "GO": "skipped", "RELEASE_REHEARSAL": "skipped", "WEB": "skipped"}
	with := func(base map[string]string, name, result string) map[string]string {
		changed := maps.Clone(base)
		changed[name] = result
		return changed
	}
	for _, tc := range []struct {
		name      string
		selection map[string]string
		results   map[string]string
		want      int
	}{
		{"every selected job succeeded", all, succeeded, 0},
		{"nothing selected and everything skipped", none, skipped, 0},
		{"only the selected job ran", only("go"), with(with(skipped, "GO", "success"), "RELEASE_REHEARSAL", "success"), 0},
		{"a selected job was skipped", all, with(succeeded, "WEB", "skipped"), 1},
		{"the selected go filter skipped the release rehearsal", only("go"), with(skipped, "GO", "success"), 1},
		{"an unselected job failed", only("web"), with(with(skipped, "WEB", "success"), "ACTIONLINT", "failure"), 1},
		{"an unselected job was cancelled", none, with(skipped, "TOOLCHAIN_CHECKS", "cancelled"), 1},
		{"the changes job failed", all, with(succeeded, "CHANGES", "failure"), 1},
		{"the changes job was skipped", all, with(succeeded, "CHANGES", "skipped"), 1},
		{"a selected job failed", all, with(succeeded, "TOOLCHAIN_CHECKS", "failure"), 1},
		{"a selected job was cancelled", all, with(succeeded, "WEB", "cancelled"), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := maps.Clone(tc.results)
			maps.Copy(env, tc.selection)
			if got := runBash(t, script, env); got != tc.want {
				t.Errorf("gate exited %d, want %d", got, tc.want)
			}
		})
	}
}

func TestGoJobBuildsTestsAndLintsThroughMiseTasks(t *testing.T) {
	steps := runSteps(job(t, loadWorkflow(t, "verify.yml"), "go"))
	for name, want := range map[string]string{"Build": "mise run build", "Test": "mise run test", "Lint": "mise run lint"} {
		if steps[name] != want {
			t.Errorf("go job %s step = %q, want %q", name, steps[name], want)
		}
	}
	if len(steps) != 3 {
		t.Errorf("go job runs %v, want only Build, Test, and Lint", sortedKeys(steps))
	}
}

func TestToolchainChecksRunTheCatalogAndShellChecksInTheDevContainer(t *testing.T) {
	steps := runSteps(job(t, loadWorkflow(t, "verify.yml"), "toolchain-checks"))
	for name, want := range map[string]string{
		"Validate catalog": "scripts/ci/run-in-dev-container.sh mise run validate-catalog",
		"ShellCheck":       "scripts/ci/run-in-dev-container.sh mise run shellcheck",
	} {
		if steps[name] != want {
			t.Errorf("toolchain-checks %s step = %q, want %q", name, steps[name], want)
		}
	}
}

func TestDeployAndVerifyWebJobsInstallThroughTheSameContainerizedTask(t *testing.T) {
	verify := runSteps(job(t, loadWorkflow(t, "verify.yml"), "web"))
	deploy := runSteps(job(t, loadWorkflow(t, "deploy.yml"), "build"))
	const install = "scripts/ci/run-in-dev-container.sh mise run web:install"
	if verify["Install dependencies"] != install || deploy["Install dependencies"] != install {
		t.Errorf("install steps: verify %q, deploy %q, want %q", verify["Install dependencies"], deploy["Install dependencies"], install)
	}
}

func TestDeployBuildStepBindsDistToTheRunnerFilesystemForThePagesArtifact(t *testing.T) {
	// deploy's Pages artifact upload runs on the bare runner and needs dist on its real
	// filesystem; verify's web job never reads dist outside the container, so it keeps
	// using the isolated named volume instead.
	verify := runSteps(job(t, loadWorkflow(t, "verify.yml"), "web"))
	deploy := runSteps(job(t, loadWorkflow(t, "deploy.yml"), "build"))
	if want := "scripts/ci/run-in-dev-container.sh --bind-dist mise run web:build"; deploy["Build"] != want {
		t.Errorf("deploy Build = %q, want %q", deploy["Build"], want)
	}
	if want := "scripts/ci/run-in-dev-container.sh mise run web:build"; verify["Build"] != want {
		t.Errorf("verify Build = %q, want %q", verify["Build"], want)
	}
}

func TestVerifyAndDeployWebJobsUseAubeInsteadOfNpm(t *testing.T) {
	verify := runSteps(job(t, loadWorkflow(t, "verify.yml"), "web"))
	deploy := runSteps(job(t, loadWorkflow(t, "deploy.yml"), "build"))
	for _, steps := range []map[string]string{verify, deploy} {
		for name, script := range steps {
			for _, banned := range []string{"npm ", "npx ", ":npm"} {
				if strings.Contains(script, banned) {
					t.Errorf("step %q runs %q: %s", name, banned, script)
				}
			}
		}
	}
	if want := "scripts/ci/run-in-dev-container.sh mise run web:check"; verify["Check"] != want {
		t.Errorf("verify Check = %q, want %q", verify["Check"], want)
	}
	if want := "scripts/ci/run-in-dev-container.sh mise run web:test"; verify["Test"] != want {
		t.Errorf("verify Test = %q, want %q", verify["Test"], want)
	}
}

func TestToolchainChecksAndWebJobsGCTheirOverlayVolumes(t *testing.T) {
	w := loadWorkflow(t, "verify.yml")
	for _, name := range []string{"toolchain-checks", "web"} {
		var gc []workflowStep
		for _, s := range job(t, w, name).Steps {
			if s.Run == "scripts/ci/gc-job-overlay-volumes.sh" {
				gc = append(gc, s)
			}
		}
		if len(gc) != 1 || gc[0].If != "always()" {
			t.Errorf("%s has %d overlay GC steps, want one that runs always(): %+v", name, len(gc), gc)
		}
	}
}

func TestEveryContainerizedJobAuthenticatesToGHCRBeforeItsFirstContainerStep(t *testing.T) {
	for _, name := range []string{"verify.yml", "deploy.yml"} {
		w := loadWorkflow(t, name)
		for jobName, j := range w.Jobs {
			firstContainer, firstLogin := -1, -1
			for i, s := range j.Steps {
				if firstContainer < 0 && strings.HasPrefix(s.Run, "scripts/ci/run-in-dev-container.sh") {
					firstContainer = i
				}
				if firstLogin < 0 && strings.HasPrefix(s.Uses, "docker/login-action") && s.With["registry"] == "ghcr.io" {
					firstLogin = i
				}
			}
			switch {
			case firstContainer < 0:
			case firstLogin < 0:
				t.Errorf("%s:%s runs a containerized step but never logs into ghcr.io", name, jobName)
			case firstLogin > firstContainer:
				t.Errorf("%s:%s logs into ghcr.io after its first containerized step", name, jobName)
			}
		}
	}
}
