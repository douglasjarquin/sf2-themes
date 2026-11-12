package cli

import "github.com/douglasjarquin/sf2-themes/v2/internal/adapters/herdrapp"

func init() { register("herdr-app", herdrapp.App) }
