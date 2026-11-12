package cli

import "github.com/douglasjarquin/sf2-themes/v2/internal/adapters/wezterm"

func init() { register("wezterm", wezterm.App) }
