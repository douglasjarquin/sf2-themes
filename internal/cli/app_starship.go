package cli

import "github.com/douglasjarquin/sf2-themes/v2/internal/adapters/starship"

func init() { register("starship", starship.App) }
