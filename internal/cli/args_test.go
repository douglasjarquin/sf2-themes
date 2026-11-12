package cli

import (
	"reflect"
	"testing"
)

func mustParse(t *testing.T, args ...string) Request {
	t.Helper()
	result, err := parseArgs(args)
	if err != nil {
		t.Fatalf("parseArgs(%v): %s", args, err.message)
	}
	if result.outcome != runCommand {
		t.Fatalf("parseArgs(%v) outcome %d", args, result.outcome)
	}
	return result.request
}

func TestFlagsParseTheSameInAnyPositionAndSpelling(t *testing.T) {
	want := mustParse(t, "apply", "wezterm", "--theme", "vega")
	for _, args := range [][]string{
		{"apply", "--theme=vega", "wezterm"},
		{"apply", "--theme", "vega", "wezterm"},
		{"apply", "wezterm", "--theme=vega"},
	} {
		if got := mustParse(t, args...); !reflect.DeepEqual(got, want) {
			t.Errorf("%v parsed to %+v, want %+v", args, got, want)
		}
	}
	if want.App != "wezterm" || want.Theme == nil || *want.Theme != "vega" {
		t.Errorf("request = %+v", want)
	}
}

func TestMutationFlags(t *testing.T) {
	got := mustParse(t, "setup", "-v", "nvim", "--config-dir", "/tmp/x", "--dry-run", "--follow-symlinks", "--adopt")
	if got.App != "nvim" || !got.Verbose || !got.DryRun || !got.FollowSymlinks || !got.Adopt ||
		got.ConfigDir == nil || *got.ConfigDir != "/tmp/x" || got.Theme != nil {
		t.Errorf("request = %+v", got)
	}
	if got := mustParse(t, "apply", "wezterm", "--verbose", "--theme="); got.Theme == nil || *got.Theme != "" || !got.Verbose {
		t.Errorf("an empty --theme= value is still a value: %+v", got)
	}
}

func TestDoubleDashEndsFlags(t *testing.T) {
	if got := mustParse(t, "show", "--", "-x"); !reflect.DeepEqual(got.Names, []string{"-x"}) {
		t.Errorf("show -- -x names = %v", got.Names)
	}
	if got := mustParse(t, "validate", "ryu", "--", "--all"); got.All || !reflect.DeepEqual(got.Names, []string{"ryu", "--all"}) {
		t.Errorf("validate ryu -- --all = %+v", got)
	}
}

func TestValueThatLooksLikeANegativeNumberIsAValue(t *testing.T) {
	if got := mustParse(t, "apply", "--theme", "-1", "wezterm"); got.Theme == nil || *got.Theme != "-1" {
		t.Errorf("request = %+v", got)
	}
}

func TestValidateTakesOneRunOfNames(t *testing.T) {
	got := mustParse(t, "validate", "ryu", "ken", "--all")
	if !got.All || !reflect.DeepEqual(got.Names, []string{"ryu", "ken"}) {
		t.Errorf("request = %+v", got)
	}
	if _, err := parseArgs([]string{"validate", "ryu", "--all", "ken"}); err == nil || err.message != "unrecognized arguments: ken" {
		t.Errorf("names after an option = %v, want unrecognized arguments: ken", err)
	}
}

func TestCombinedShortSwitches(t *testing.T) {
	if got := mustParse(t, "apply", "wezterm", "-vv"); !got.Verbose {
		t.Errorf("-vv = %+v", got)
	}
	if _, err := parseArgs([]string{"apply", "wezterm", "-vx"}); err == nil || err.message != "unrecognized arguments: -x" {
		t.Errorf("-vx = %v, want unrecognized arguments: -x", err)
	}
}
