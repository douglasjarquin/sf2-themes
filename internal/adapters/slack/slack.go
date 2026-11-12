// Package slack renders the Slack custom-theme import line. Slack stores the theme in the client, not a file, so
// it stays outside the inspect, plan, and commit path.
package slack

import (
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
)

// RenderTheme returns the four-color Import line for one catalog theme.
func RenderTheme(theme catalog.Theme) string {
	colors := []catalog.HexColor{theme.UI.Background, theme.UI.Accent, theme.Semantic.Green, theme.Semantic.Red}
	parts := make([]string, len(colors))
	for i, color := range colors {
		parts[i] = strings.ToLower(string(color))
	}
	return strings.Join(parts, ",")
}
