package cli

import "github.com/douglasjarquin/sf2-themes/v2/internal/adapters/claude"

func init() { register("claude", claude.App) }
