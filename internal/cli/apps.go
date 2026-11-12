package cli

import (
	"fmt"
	"slices"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
)

// apps holds the file-writing adapters by app name. Each adapter registers itself from its own file,
// app_<name>.go, with one line: `func init() { register("<name>", <name>.App) }`.
// Slack has no config file and is handled by the commands themselves.
var apps = map[string]adapters.App{}

// register adds an adapter under one of the names the parser accepts.
func register(name string, app adapters.App) {
	if name == "slack" || !slices.Contains(appNames, name) {
		panic(fmt.Sprintf("register: %q is not a file-writing app name", name))
	}
	if _, taken := apps[name]; taken {
		panic(fmt.Sprintf("register: %q is registered twice", name))
	}
	apps[name] = app
}

// lookupApp returns the adapter for name, or an error while that adapter is not ported yet.
func lookupApp(name string) (adapters.App, error) {
	app, ok := apps[name]
	if !ok {
		return adapters.App{}, fmt.Errorf("%s: adapter is not ported to this build yet", name)
	}
	return app, nil
}
