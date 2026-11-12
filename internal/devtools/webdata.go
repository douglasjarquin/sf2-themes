package devtools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// RenderWebData returns the browser theme data document for the catalog: two-space indented JSON with
// non-ASCII characters written as themselves, ending in a newline.
func RenderWebData(themes []catalog.Theme) string {
	entries := make([]any, 0, len(themes))
	for _, theme := range themes {
		entries = append(entries, browserTheme(theme))
	}
	payload := pycompat.NewObject()
	payload.Set("schema_version", 1)
	payload.Set("themes", entries)
	return pycompat.DumpsUnicode(payload) + "\n"
}

func browserTheme(theme catalog.Theme) *pycompat.Object {
	m := theme.Metadata
	meta := pycompat.NewObject()
	meta.Set("id", m.ID)
	meta.Set("display_name", m.DisplayName)
	meta.Set("kind", string(m.Kind))
	meta.Set("introduced_in", string(m.IntroducedIn))
	if m.Character != nil {
		meta.Set("character", *m.Character)
	}
	aliases := make([]any, 0, len(m.Aliases))
	for _, alias := range m.Aliases {
		aliases = append(aliases, alias)
	}
	meta.Set("aliases", aliases)
	meta.Set("name", m.Name)
	meta.Set("variant", string(m.Variant))
	meta.Set("family", m.Family)
	meta.Set("stage", m.Stage)

	ui := colorObject(catalog.UIColorFields, theme.UI.Colors())
	a := catalog.ProjectAdapterColors(theme.UI)
	for _, role := range []struct {
		name  string
		color catalog.HexColor
	}{
		{"cursor_bg", a.CursorBg},
		{"cursor_fg", a.CursorFg},
		{"selection_bg", a.SelectionBg},
		{"selection_fg", a.SelectionFg},
		{"panel_bg", a.PanelBg},
		{"sidebar_bg", a.SidebarBg},
		{"active_row_bg", a.ActiveRowBg},
		{"navigate_row_bg", a.NavigateRowBg},
		{"surface_dim", a.SurfaceDim},
		{"surface0", a.Surface0},
		{"surface1", a.Surface1},
		{"overlay0", a.Overlay0},
		{"overlay1", a.Overlay1},
		{"subtext", a.Subtext},
	} {
		ui.Set(role.name, string(role.color))
	}

	ansi := pycompat.NewObject()
	ansi.Set("normal", colorObject(catalog.ANSIOrder, theme.AnsiNormal.Colors()))
	ansi.Set("bright", colorObject(catalog.ANSIOrder, theme.AnsiBright.Colors()))

	entry := pycompat.NewObject()
	entry.Set("schema_version", 1)
	entry.Set("meta", meta)
	entry.Set("ui", ui)
	entry.Set("semantic", colorObject(catalog.SemanticFields, theme.Semantic.Colors()))
	entry.Set("ansi", ansi)
	return entry
}

func colorObject(names []string, colors []catalog.HexColor) *pycompat.Object {
	object := pycompat.NewObject()
	for i, name := range names {
		object.Set(name, string(colors[i]))
	}
	return object
}

// WebData renders the validated catalog in the directory source. With check it compares the result to output
// and reports it verified (exit 0) or stale (exit 1); without check it writes output atomically.
// A catalog or file error prints its bare message on stderr and returns exit status 1.
func WebData(source, output string, check bool, stdout, stderr io.Writer) int {
	themes, err := catalog.LoadFS(os.DirFS(source))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	content := RenderWebData(themes)
	if check {
		if !isFile(output) {
			_, _ = fmt.Fprintf(stderr, "stale generated theme data: %s\n", output)
			return 1
		}
		current, err := pycompat.ReadText(output)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		if current != content {
			_, _ = fmt.Fprintf(stderr, "stale generated theme data: %s\n", output)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "verified generated theme data: %s\n", output)
		return 0
	}
	if err := writeAtomic(output, content); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "wrote %s\n", output)
	return 0
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// writeAtomic stages content in a temporary file beside path and renames it into place.
func writeAtomic(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return err
	}
	staged, err := os.CreateTemp(dir, "tmp")
	if err != nil {
		return err
	}
	if _, err := staged.WriteString(content); err != nil {
		_ = staged.Close()
		_ = os.Remove(staged.Name())
		return err
	}
	if err := staged.Close(); err != nil {
		_ = os.Remove(staged.Name())
		return err
	}
	if err := os.Rename(staged.Name(), path); err != nil {
		_ = os.Remove(staged.Name())
		return err
	}
	return nil
}
