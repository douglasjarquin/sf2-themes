// Package version holds the release version stamped into the binary at link time.
package version

// Version is replaced by the release build through
// -ldflags "-X github.com/douglasjarquin/sf2-themes/v2/internal/version.Version=<semver>".
var Version = "dev"
