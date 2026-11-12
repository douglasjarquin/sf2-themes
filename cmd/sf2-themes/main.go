// Command sf2-themes applies the SF2 theme catalog to terminal and editor configs.
package main

import (
	"os"

	"github.com/douglasjarquin/sf2-themes/v2/internal/cli"
)

func main() {
	// The TOML library switches to TOML 1.1 when this variable is set. User configs must be
	// read as TOML 1.0, the way the Python CLI read them, whatever the user's environment holds.
	_ = os.Unsetenv("BURNTSUSHI_TOML_110")
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
