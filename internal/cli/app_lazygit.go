package cli

import "github.com/douglasjarquin/sf2-themes/v2/internal/adapters/lazygit"

func init() { register("lazygit", lazygit.App) }
