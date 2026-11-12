package cli

import "github.com/douglasjarquin/sf2-themes/v2/internal/adapters/nvim"

func init() { register("nvim", nvim.App) }
