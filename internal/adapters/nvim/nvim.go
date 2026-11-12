// Package nvim writes every catalog colorscheme for Neovim, the pointer that follows TERM_THEME or 'background', and
// the startup loader. The adapter owns these files outright, so nothing in them is ever foreign.
package nvim

import (
	"os"
	"regexp"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// App is the Neovim adapter.
var App = adapters.App{
	Label:   "Neovim",
	Inspect: Inspect,
	Plan:    Plan,
	Target:  nvimDir,
	Pair:    true,
	Reload:  "restart Neovim to load the new colorscheme",
}

const (
	legacySchemeFilePrefix = "street-fighter-ii"
	currentDirName         = "sf2-theme"
	currentFileName        = "current.lua"
	pluginFileName         = "sf2-theme.lua"
	colorsDirName          = "colors"
	pluginDirName          = "plugin"
)

var idComment = regexp.MustCompile(`^-- sf2-themes:` + pycompat.Space + `+(` + pycompat.NotSpace + `+)` + pycompat.Space + `*$`)

func configRoot() (string, error) {
	if configured := os.Getenv("XDG_CONFIG_HOME"); configured != "" {
		return pycompat.ExpandUser(configured)
	}
	home, err := pycompat.Home()
	return pycompat.Path(home, ".config"), err
}

func nvimDir(configDir string) (string, error) {
	if configDir != "" {
		return configDir, nil
	}
	if configured := os.Getenv("NVIM_CONFIG_DIR"); configured != "" {
		return pycompat.ExpandUser(configured)
	}
	root, err := configRoot()
	return pycompat.Path(root, "nvim"), err
}

func schemeFilename(theme catalog.Theme) string {
	return theme.Metadata.SelectableID() + ".lua"
}

func renderScheme(theme catalog.Theme) string {
	ui := theme.UI
	adapter := catalog.ProjectAdapterColors(ui)
	normal := theme.AnsiNormal
	bright := theme.AnsiBright
	semantic := theme.Semantic
	colors := []struct {
		name  string
		color catalog.HexColor
	}{
		{"background", ui.Background},
		{"foreground", ui.Foreground},
		{"cursor_bg", adapter.CursorBg},
		{"cursor_fg", adapter.CursorFg},
		{"selection_bg", adapter.SelectionBg},
		{"selection_fg", adapter.SelectionFg},
		{"panel_bg", adapter.PanelBg},
		{"sidebar_bg", adapter.SidebarBg},
		{"active_row_bg", adapter.ActiveRowBg},
		{"navigate_row_bg", adapter.NavigateRowBg},
		{"surface_dim", adapter.SurfaceDim},
		{"surface0", adapter.Surface0},
		{"surface1", adapter.Surface1},
		{"overlay0", adapter.Overlay0},
		{"overlay1", adapter.Overlay1},
		{"subtext", adapter.Subtext},
		{"accent", ui.Accent},
		{"red", semantic.Red},
		{"green", semantic.Green},
		{"yellow", semantic.Yellow},
		{"blue", semantic.Blue},
		{"magenta", semantic.Magenta},
		{"cyan", semantic.Cyan},
		{"orange", semantic.Orange},
		{"normal_black", normal.Black},
		{"normal_red", normal.Red},
		{"normal_green", normal.Green},
		{"normal_yellow", normal.Yellow},
		{"normal_blue", normal.Blue},
		{"normal_magenta", normal.Magenta},
		{"normal_cyan", normal.Cyan},
		{"normal_white", normal.White},
		{"bright_black", bright.Black},
		{"bright_red", bright.Red},
		{"bright_green", bright.Green},
		{"bright_yellow", bright.Yellow},
		{"bright_blue", bright.Blue},
		{"bright_magenta", bright.Magenta},
		{"bright_cyan", bright.Cyan},
		{"bright_white", bright.White},
	}
	background := "dark"
	if theme.Metadata.Variant == catalog.VariantLight {
		background = "light"
	}
	lines := []string{
		`vim.cmd("highlight clear")`,
		`if vim.fn.exists("syntax_on") == 1 then vim.cmd("syntax reset") end`,
		`vim.o.background = "` + background + `"`,
		`vim.g.colors_name = "` + theme.Metadata.SelectableID() + `"`,
		"local colors = {",
	}
	for _, entry := range colors {
		lines = append(lines, "  "+entry.name+` = "`+string(entry.color)+`",`)
	}
	lines = append(lines,
		"}",
		"local highlights = {",
		"  Normal = { fg = colors.foreground, bg = colors.background },",
		"  NormalFloat = { fg = colors.foreground, bg = colors.panel_bg },",
		"  Cursor = { fg = colors.background, bg = colors.cursor_bg },",
		"  CursorLine = { bg = colors.surface0 },",
		"  Visual = { bg = colors.selection_bg, fg = colors.selection_fg },",
		"  Search = { bg = colors.yellow, fg = colors.background },",
		"  IncSearch = { bg = colors.orange, fg = colors.background },",
		"  StatusLine = { bg = colors.surface1, fg = colors.foreground },",
		"  StatusLineNC = { bg = colors.surface0, fg = colors.subtext },",
		"  WinSeparator = { fg = colors.surface1 },",
		"  Pmenu = { bg = colors.panel_bg, fg = colors.foreground },",
		"  PmenuSel = { bg = colors.selection_bg, fg = colors.selection_fg },",
		"  LineNr = { fg = colors.overlay1 },",
		"  CursorLineNr = { fg = colors.accent, bold = true },",
		"  Comment = { fg = colors.subtext, italic = true },",
		"  Constant = { fg = colors.cyan },",
		"  String = { fg = colors.green },",
		"  Number = { fg = colors.orange },",
		"  Boolean = { fg = colors.orange },",
		"  Identifier = { fg = colors.foreground },",
		"  Function = { fg = colors.blue },",
		"  Statement = { fg = colors.magenta },",
		"  Keyword = { fg = colors.red },",
		"  Type = { fg = colors.yellow },",
		"  Special = { fg = colors.orange },",
		"  Error = { fg = colors.red, bold = true },",
		"  Todo = { fg = colors.background, bg = colors.yellow, bold = true },",
		"  DiagnosticError = { fg = colors.red },",
		"  DiagnosticWarn = { fg = colors.yellow },",
		"  DiagnosticInfo = { fg = colors.blue },",
		"  DiagnosticHint = { fg = colors.cyan },",
		"  DiffAdd = { bg = colors.green, fg = colors.background },",
		"  DiffChange = { bg = colors.blue, fg = colors.background },",
		"  DiffDelete = { bg = colors.red, fg = colors.background },",
		"  Terminal = { fg = colors.normal_white, bg = colors.background },",
		"}",
		"for group, opts in pairs(highlights) do",
		"  vim.api.nvim_set_hl(0, group, opts)",
		"end",
		"",
	)
	return strings.Join(lines, "\n")
}

func renderPointer(dark, light catalog.Theme) string {
	darkID := dark.Metadata.SelectableID()
	lightID := light.Metadata.SelectableID()
	return strings.Join([]string{
		"-- sf2-themes: " + darkID,
		`local light_mode = vim.env.TERM_THEME == "light" or (vim.env.TERM_THEME ~= "dark" and vim.o.background == "light")`,
		"if light_mode then",
		`  vim.cmd("colorscheme ` + lightID + `")`,
		"else",
		`  vim.cmd("colorscheme ` + darkID + `")`,
		"end",
		"",
	}, "\n")
}

// renderLoader renders the startup plugin. TERM_THEME freezes at process start, so a host appearance change made
// after launch never reaches an already-running Neovim; the loader re-derives it from the live host appearance
// instead of trusting the frozen variable.
func renderLoader() string {
	return strings.Join([]string{
		`local current = vim.fn.stdpath("config") .. "/sf2-theme/current.lua"`,
		"local function sf2_apply_current()",
		"  if vim.fn.filereadable(current) == 1 then",
		"    dofile(current)",
		"  end",
		"end",
		"sf2_apply_current()",
		"",
		`if vim.fn.has("mac") == 1 then`,
		"  local function sf2_sync_term_theme()",
		`    vim.fn.system({ "defaults", "read", "-g", "AppleInterfaceStyle" })`,
		`    local want = vim.v.shell_error == 0 and "dark" or "light"`,
		"    if vim.env.TERM_THEME ~= want then",
		"      vim.env.TERM_THEME = want",
		"      sf2_apply_current()",
		"    end",
		"  end",
		"  sf2_sync_term_theme()",
		`  local group = vim.api.nvim_create_augroup("sf2-theme-appearance", { clear = true })`,
		`  vim.api.nvim_create_autocmd({ "FocusGained", "VimResume" }, {`,
		"    group = group,",
		"    callback = sf2_sync_term_theme,",
		"  })",
		"  local uv = vim.uv or vim.loop",
		"  local timer = uv.new_timer()",
		"  if timer then",
		"    timer:start(15000, 15000, vim.schedule_wrap(sf2_sync_term_theme))",
		"  end",
		"end",
		"",
	}, "\n")
}

// Inspect reads the pointer alone, because the adapter owns its files outright.
func Inspect(themes []catalog.Theme, configDir string) (plan.Inspection, error) {
	dir, err := nvimDir(configDir)
	if err != nil {
		return plan.Inspection{}, err
	}
	pointer := pycompat.Path(dir, currentDirName, currentFileName)
	found, err := fsys.IsFile(pointer)
	if err != nil || !found {
		return plan.Inspection{Ownership: plan.Absent}, err
	}
	text, err := pycompat.ReadText(pointer)
	if err != nil {
		return plan.Inspection{}, err
	}
	id, recorded := plan.RecordedID(text, idComment)
	return plan.Recorded(plan.Managed, id, recorded, themes), nil
}

// Plan returns every colorscheme, the pointer, and the loader; Neovim has no user-owned setting for slot to keep.
func Plan(theme catalog.Theme, themes []catalog.Theme, configDir string, _ bool) ([]plan.Step, error) {
	dir, err := nvimDir(configDir)
	if err != nil {
		return nil, err
	}
	target := pycompat.Path(dir, colorsDirName)
	steps := make([]plan.Step, 0, 2*len(themes)+2)
	for _, entry := range themes {
		steps = append(steps, plan.Remove(pycompat.Path(target, legacySchemeFilePrefix+"-"+entry.Metadata.ID+".lua")))
	}
	for _, entry := range themes {
		steps = append(steps, plan.Write(pycompat.Path(target, schemeFilename(entry)), renderScheme(entry)))
	}
	dark, light, err := catalog.ThemePair(theme, themes)
	if err != nil {
		return nil, err
	}
	steps = append(steps,
		plan.Write(pycompat.Path(dir, currentDirName, currentFileName), renderPointer(dark, light)),
		plan.Write(pycompat.Path(dir, pluginDirName, pluginFileName), renderLoader()),
	)
	return steps, nil
}
