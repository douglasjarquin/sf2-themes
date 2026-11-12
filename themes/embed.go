// Package themes embeds the authoritative TOML theme catalog so the binary needs no sibling files.
package themes

import "embed"

// FS holds main.toml, main-light.toml, and characters/*.toml at their catalog-relative paths.
//
//go:embed *.toml characters/*.toml
var FS embed.FS
