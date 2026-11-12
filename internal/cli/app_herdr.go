package cli

import "github.com/douglasjarquin/sf2-themes/v2/internal/adapters/herdr"

func init() { register("herdr", herdr.App) }
