package cli

import "github.com/douglasjarquin/sf2-themes/v2/internal/adapters/codex"

func init() { register("codex", codex.App) }
