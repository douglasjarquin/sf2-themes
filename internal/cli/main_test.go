package cli

import (
	"fmt"
	"os"
	"testing"
)

// TestMain points XDG_STATE_HOME at a temporary directory so no test records ownership in the real manifest.
func TestMain(m *testing.M) {
	state, err := os.MkdirTemp("", "sf2-cli-state-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.Setenv("XDG_STATE_HOME", state); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(state)
	os.Exit(code)
}
