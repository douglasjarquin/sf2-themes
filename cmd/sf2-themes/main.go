// Command sf2-themes applies the SF2 theme catalog to terminal and editor configs.
package main

import (
	"fmt"
	"os"

	"github.com/douglasjarquin/sf2-themes/v2/internal/version"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(version.Version)
		return
	}
	fmt.Fprintln(os.Stderr, "sf2-themes: not implemented")
	os.Exit(2)
}
