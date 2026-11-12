// Package devtools holds the maintainer generators behind sf2-dev: the SVG previews and the web theme data.
package devtools

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// RenderPreview returns the SVG swatch for one theme: its name, six UI chips, and the sixteen ANSI cells.
func RenderPreview(theme catalog.Theme) string {
	ui := theme.UI
	lines := []string{
		`<svg xmlns="http://www.w3.org/2000/svg" width="320" height="176" viewBox="0 0 320 176">`,
		fmt.Sprintf(`<rect width="320" height="176" fill="%s"/>`, ui.Background),
		fmt.Sprintf(`<text x="16" y="22" font-family="ui-monospace, monospace" font-size="12" fill="%s">%s</text>`,
			ui.Foreground, theme.Metadata.DisplayName),
	}
	chips := []struct {
		label string
		color catalog.HexColor
	}{
		{"background", ui.Background},
		{"foreground", ui.Foreground},
		{"accent", ui.Accent},
		{"active row", ui.ActiveRowBg()},
		{"navigate", ui.NavigateRowBg()},
		{"selection", ui.SelectionBg()},
	}
	for index, chip := range chips {
		x := 16 + index*52
		lines = append(lines, fmt.Sprintf(
			`<rect x="%d" y="36" width="48" height="40" fill="%s"/>`+
				`<text x="%d" y="88" text-anchor="middle" font-size="7" fill="%s">%s</text>`,
			x, chip.color, x+24, ui.Foreground, chip.label))
	}
	ansi := append(theme.AnsiNormal.Colors(), theme.AnsiBright.Colors()...)
	for index, color := range ansi {
		x := 16 + (index%8)*36
		y := 92 + (index/8)*36
		lines = append(lines, fmt.Sprintf(`<rect x="%d" y="%d" width="32" height="32" fill="%s"/>`, x, y, color))
	}
	lines = append(lines, "</svg>", "")
	return strings.Join(lines, "\n")
}

// WritePreviews writes <id>.svg for every theme into dir, creating dir when needed,
// and prints a "wrote <path>" line for each file.
func WritePreviews(themes []catalog.Theme, dir string, stdout io.Writer) error {
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return err
	}
	for _, theme := range themes {
		path := pycompat.Path(dir, theme.Metadata.ID+".svg")
		if err := os.WriteFile(path, []byte(RenderPreview(theme)), 0o666); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "wrote %s\n", path)
	}
	return nil
}

// Previews renders every theme in the catalog directory source, unvalidated, into dir.
// A failure prints its bare message on stderr and returns exit status 1.
func Previews(source, dir string, stdout, stderr io.Writer) int {
	themes, err := catalog.ParseFS(os.DirFS(source))
	if err == nil {
		err = WritePreviews(themes, dir, stdout)
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
