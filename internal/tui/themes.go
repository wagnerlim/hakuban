package tui

import "charm.land/lipgloss/v2"

// palette holds the semantic colors of a theme (F13). The token vocabulary —
// base/surface0/surface1/overlay0/overlay1/subtext0 plus the named hues (mauve,
// peach, teal…) — is Catppuccin's (github.com/catppuccin/catppuccin, MIT); the
// same shape is used by herdr (github.com/ogulcancelik/herdr, Apache-2.0), where
// this UI's semantic mapping was first prototyped. isDark decides the Glamour
// style.
//
// Every palette below carries the hex values published by its own upstream, each
// permissively licensed (Catppuccin, Tokyo Night, Dracula, Nord, Gruvbox, One,
// Solarized, Kanagawa, Rosé Pine, Vesper). See CREDITS.md for the list.
//
// Single TUI instance → I keep the active palette in a global var (pal) and
// rebuild the styles in applyPalette on every theme switch.
// ponytail: mutable global fine in a single-window app; if it ever runs 2
// Models, this becomes a Model field.
type palette struct {
	name   string
	isDark bool

	// hex "#rrggbb" (or ANSI index / "" = Reset in the terminal theme); become
	// lipgloss.Color in applyPalette.
	accent     string // highlight, active borders
	panelBg    string // base — text over hues/accent
	surface0   string // selected item background (tab/shadow)
	surface1   string // inactive border, scroll track
	surfaceDim string // separator / shadow
	overlay0   string // muted text (secondary, ids)
	overlay1   string // lighter muted text
	text       string // main text
	subtext0   string // soft text (inactive tab)
	mauve      string
	green      string
	yellow     string
	red        string
	blue       string
	teal       string
	peach      string

	// optional border override for the NON-selected column; "" = uses surface1
	// (subtle border). omni uses pink, like in the user's VSCode.
	colBorder string
}

// themes are the available themes, in the order they cycle in the menu. The
// first (Omni) is the default and is the author's own (Rocketseat-flavoured);
// the rest map each upstream theme's published palette onto the tokens above.
var themes = []palette{
	{name: "omni", isDark: true, accent: "#67e480", panelBg: "#191622", surface0: "#201b2d", surface1: "#2a2438", surfaceDim: "#15121f", overlay0: "#5a4b81", overlay1: "#988bc7", text: "#e1e1e6", subtext0: "#988bc7", mauve: "#988bc7", green: "#67e480", yellow: "#e7de79", red: "#ed4556", blue: "#78d1e1", teal: "#78d1e1", peach: "#ff79c6", colBorder: "#ff79c6"},
	{name: "catppuccin", isDark: true, accent: "#89b4fa", panelBg: "#181825", surface0: "#313244", surface1: "#45475a", surfaceDim: "#1e1e2e", overlay0: "#6c7086", overlay1: "#7f849c", text: "#cdd6f4", subtext0: "#a6adc8", mauve: "#cba6f7", green: "#a6e3a1", yellow: "#f9e2af", red: "#f38ba8", blue: "#89b4fa", teal: "#94e2d5", peach: "#fab387"},
	{name: "catppuccin-latte", isDark: false, accent: "#1e66f5", panelBg: "#eff1f5", surface0: "#ccd0da", surface1: "#bcc0cc", surfaceDim: "#e6e9ef", overlay0: "#9ca0b0", overlay1: "#8c8fa1", text: "#4c4f69", subtext0: "#6c6f85", mauve: "#8839ef", green: "#40a02b", yellow: "#df8e1d", red: "#d20f39", blue: "#1e66f5", teal: "#179299", peach: "#fe640b"},
	{name: "tokyo-night", isDark: true, accent: "#7aa2f7", panelBg: "#1a1b26", surface0: "#24283b", surface1: "#414868", surfaceDim: "#1a1b26", overlay0: "#565f89", overlay1: "#697196", text: "#c0caf5", subtext0: "#a9b1d6", mauve: "#bb9af7", green: "#9ece6a", yellow: "#e0af68", red: "#f7768e", blue: "#7aa2f7", teal: "#7dcfff", peach: "#ff9e64"},
	{name: "tokyo-night-day", isDark: false, accent: "#2e7de9", panelBg: "#e1e2e7", surface0: "#c4c8da", surface1: "#a8aecb", surfaceDim: "#d2d3da", overlay0: "#8990b3", overlay1: "#68709a", text: "#3760bf", subtext0: "#6172b0", mauve: "#7847bd", green: "#587539", yellow: "#8c6c3e", red: "#f52a65", blue: "#2e7de9", teal: "#118c74", peach: "#b15c00"},
	{name: "dracula", isDark: true, accent: "#bd93f9", panelBg: "#282a36", surface0: "#44475a", surface1: "#6272a4", surfaceDim: "#282a36", overlay0: "#6272a4", overlay1: "#828cb4", text: "#f8f8f2", subtext0: "#d2d2dc", mauve: "#ff79c6", green: "#50fa7b", yellow: "#f1fa8c", red: "#ff5555", blue: "#8be9fd", teal: "#8be9fd", peach: "#ffb86c"},
	{name: "nord", isDark: true, accent: "#88c0d0", panelBg: "#2e3440", surface0: "#3b4252", surface1: "#434c5e", surfaceDim: "#2e3440", overlay0: "#4c566a", overlay1: "#646e82", text: "#eceff4", subtext0: "#d8dee9", mauve: "#b48ead", green: "#a3be8c", yellow: "#ebcb8b", red: "#bf616a", blue: "#81a1c1", teal: "#8fbcbb", peach: "#d08770"},
	{name: "gruvbox", isDark: true, accent: "#d79921", panelBg: "#282828", surface0: "#3c3836", surface1: "#504945", surfaceDim: "#282828", overlay0: "#928374", overlay1: "#a89984", text: "#ebdbb2", subtext0: "#d5c4a1", mauve: "#d3869b", green: "#b8bb26", yellow: "#fabd2f", red: "#fb4934", blue: "#83a598", teal: "#8ec07c", peach: "#fe8019"},
	{name: "gruvbox-light", isDark: false, accent: "#076678", panelBg: "#fbf1c7", surface0: "#ebdbb2", surface1: "#d5c4a1", surfaceDim: "#f2e5bc", overlay0: "#928374", overlay1: "#7c6f64", text: "#3c3836", subtext0: "#504945", mauve: "#8f3f71", green: "#79740e", yellow: "#b57614", red: "#9d0006", blue: "#076678", teal: "#427b58", peach: "#af3a03"},
	{name: "one-dark", isDark: true, accent: "#61afef", panelBg: "#282c34", surface0: "#2c313a", surface1: "#3e4451", surfaceDim: "#282c34", overlay0: "#5c6370", overlay1: "#737a87", text: "#abb2bf", subtext0: "#969ca8", mauve: "#c678dd", green: "#98c379", yellow: "#e5c07b", red: "#e06c75", blue: "#61afef", teal: "#56b6c2", peach: "#d19a66"},
	{name: "one-light", isDark: false, accent: "#4078f2", panelBg: "#fafafa", surface0: "#f0f0f1", surface1: "#e5e5e6", surfaceDim: "#f5f5f6", overlay0: "#a0a1a7", overlay1: "#686b77", text: "#383a42", subtext0: "#686b77", mauve: "#a626a4", green: "#50a14f", yellow: "#c18401", red: "#e45649", blue: "#4078f2", teal: "#0184bc", peach: "#986801"},
	{name: "solarized", isDark: true, accent: "#268bd2", panelBg: "#002b36", surface0: "#073642", surface1: "#586e75", surfaceDim: "#002b36", overlay0: "#586e75", overlay1: "#657b83", text: "#93a1a1", subtext0: "#839496", mauve: "#d33682", green: "#859900", yellow: "#b58900", red: "#dc322f", blue: "#268bd2", teal: "#2aa198", peach: "#cb4b16"},
	{name: "solarized-light", isDark: false, accent: "#268bd2", panelBg: "#fdf6e3", surface0: "#eee8d5", surface1: "#93a1a1", surfaceDim: "#eee8d5", overlay0: "#93a1a1", overlay1: "#586e75", text: "#657b83", subtext0: "#839496", mauve: "#d33682", green: "#859900", yellow: "#b58900", red: "#dc322f", blue: "#268bd2", teal: "#2aa198", peach: "#cb4b16"},
	{name: "kanagawa", isDark: true, accent: "#7e9cd8", panelBg: "#1f1f28", surface0: "#2a2a37", surface1: "#363646", surfaceDim: "#1f1f28", overlay0: "#727169", overlay1: "#87867d", text: "#dcd7ba", subtext0: "#c8c3aa", mauve: "#957fb8", green: "#76946a", yellow: "#c0a36e", red: "#c34043", blue: "#7e9cd8", teal: "#7fb4ca", peach: "#ffa066"},
	{name: "kanagawa-lotus", isDark: false, accent: "#4d699b", panelBg: "#f2ecbc", surface0: "#dcd5ac", surface1: "#c9cbd1", surfaceDim: "#d5cea3", overlay0: "#a09cac", overlay1: "#8a8980", text: "#545464", subtext0: "#43436c", mauve: "#624c83", green: "#6f894e", yellow: "#77713f", red: "#c84053", blue: "#4d699b", teal: "#4e8ca2", peach: "#cc6d00"},
	{name: "rose-pine", isDark: true, accent: "#c4a7e7", panelBg: "#191724", surface0: "#1f1d2e", surface1: "#26233a", surfaceDim: "#191724", overlay0: "#6e6a86", overlay1: "#908caa", text: "#e0def4", subtext0: "#c8c5dc", mauve: "#c4a7e7", green: "#31748f", yellow: "#f6c177", red: "#eb6f92", blue: "#31748f", teal: "#9ccfd8", peach: "#ea9a97"},
	{name: "rose-pine-dawn", isDark: false, accent: "#907aa9", panelBg: "#faf4ed", surface0: "#f2e9e1", surface1: "#fffaf3", surfaceDim: "#f2e9e1", overlay0: "#9893a5", overlay1: "#797593", text: "#464261", subtext0: "#797593", mauve: "#907aa9", green: "#286983", yellow: "#ea9d34", red: "#b4637a", blue: "#286983", teal: "#56949f", peach: "#d7827e"},
	{name: "vesper", isDark: true, accent: "#ffc799", panelBg: "#1a1a1a", surface0: "#232323", surface1: "#282828", surfaceDim: "#101010", overlay0: "#5c5c5c", overlay1: "#7e7e7e", text: "#ffffff", subtext0: "#a0a0a0", mauve: "#ffd1a8", green: "#99ffe4", yellow: "#ffc799", red: "#ff8080", blue: "#b0b0b0", teal: "#66ddcc", peach: "#ffc799"},
	// 16-color fallback: uses ANSI and "" (Reset) → inherits the terminal's bg/fg.
	{name: "terminal", isDark: true, accent: "4", panelBg: "", surface0: "", surface1: "8", surfaceDim: "8", overlay0: "7", overlay1: "15", text: "", subtext0: "7", mauve: "7", green: "2", yellow: "3", red: "9", blue: "4", teal: "6", peach: "3"},
}

// themeNames is the cycling order in the menu.
var themeNames = func() []string {
	out := make([]string, len(themes))
	for i, t := range themes {
		out[i] = t.name
	}
	return out
}()

// themeByName resolves a theme by name; unknown falls back to the default (the first).
func themeByName(name string) palette {
	for _, t := range themes {
		if t.name == name {
			return t
		}
	}
	return themes[0]
}

// pal is the active palette; hues/onAccent are read straight from it in the badge functions.
var pal palette

// Styles rebuilt by applyPalette from the active theme.
var (
	colStyle, colSelStyle, colDropStyle         lipgloss.Style
	titleStyle, helpStyle                       lipgloss.Style
	miniBox, miniBoxSel                         lipgloss.Style
	cardBox, paneBox, cardTitle, metaKey, faint lipgloss.Style
	paneSep                                     lipgloss.Style // preview pane divider
	sbThumb, sbTrack                            lipgloss.Style
	tabActive, tabInactive, tabPlus             lipgloss.Style
	dotRed, dotOff                              lipgloss.Style
	shadowStyle                                 lipgloss.Style
	prefixBadge, kmGroup                        lipgloss.Style
	okStyle, failStyle                          lipgloss.Style // action loader: success/in-progress vs failure
)

func init() { applyPalette(themes[0]) }

// applyPalette makes p the active palette and rebuilds every style from the
// tokens. Called in New() and on every theme switch in the menu.
func applyPalette(p palette) {
	pal = p
	accent := lipgloss.Color(p.accent)
	surface1 := lipgloss.Color(p.surface1)
	overlay0 := lipgloss.Color(p.overlay0)
	rounded := func() lipgloss.Style { return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()) }

	// inactive column border: theme override, else surface1 (subtle)
	colBorder := surface1
	if p.colBorder != "" {
		colBorder = lipgloss.Color(p.colBorder)
	}
	colStyle = rounded().Padding(0, 1).BorderForeground(colBorder)
	colSelStyle = colStyle.BorderForeground(accent)
	colDropStyle = colStyle.BorderForeground(lipgloss.Color(p.green))
	titleStyle = lipgloss.NewStyle().Bold(true)
	helpStyle = lipgloss.NewStyle().Foreground(overlay0)

	miniBox = rounded().BorderForeground(colBorder).Padding(0, 1) // inactive card: same color as the inactive column
	miniBoxSel = rounded().BorderForeground(accent).Padding(0, 1)

	cardBox = rounded().BorderForeground(accent).Padding(1, 2)
	paneBox = rounded().BorderForeground(accent).Padding(0, 1) // highlight border, same as the selected column
	paneSep = lipgloss.NewStyle().Foreground(colBorder)        // divider in the same color as the inactive column border
	cardTitle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	metaKey = lipgloss.NewStyle().Foreground(overlay0).Width(9)
	faint = lipgloss.NewStyle().Foreground(overlay0)

	sbThumb = lipgloss.NewStyle().Foreground(accent)
	sbTrack = lipgloss.NewStyle().Foreground(surface1)

	tabActive = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(p.panelBg)).Background(accent)
	tabInactive = lipgloss.NewStyle().Foreground(lipgloss.Color(p.subtext0)).Background(lipgloss.Color(p.surface0))
	tabPlus = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(p.green))

	// prefix/keybinds: green badge with dark text; section in green.
	prefixBadge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(p.panelBg)).Background(lipgloss.Color(p.green))
	kmGroup = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(p.green))
	okStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.green))
	failStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.red))

	dotRed = lipgloss.NewStyle().Foreground(lipgloss.Color(p.red))
	dotOff = lipgloss.NewStyle().Foreground(overlay0)

	shadowStyle = lipgloss.NewStyle().Background(lipgloss.Color(p.surfaceDim))
}
