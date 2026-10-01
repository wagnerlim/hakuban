package tui

import "testing"

// Guards the theme table (hand-typed): unique, non-empty names, main tokens
// filled in (except "terminal", which uses "" = Reset on purpose),
// and the themeByName fallback.
func TestThemes(t *testing.T) {
	seen := map[string]bool{}
	for _, th := range themes {
		if th.name == "" {
			t.Fatal("theme with no name")
		}
		if seen[th.name] {
			t.Fatalf("nome de tema duplicado: %q", th.name)
		}
		seen[th.name] = true
		if th.name == "terminal" {
			continue // uses "" (Reset) and ANSI on purpose
		}
		for label, v := range map[string]string{
			"accent": th.accent, "surface1": th.surface1, "overlay0": th.overlay0,
			"red": th.red, "green": th.green, "blue": th.blue,
		} {
			if v == "" {
				t.Errorf("tema %q: token %s vazio", th.name, label)
			}
		}
	}

	if themeByName("catppuccin").name != "catppuccin" {
		t.Fatal("themeByName did not find catppuccin")
	}
	if themeByName("inexistente").name != themes[0].name {
		t.Fatal("desconhecido devia cair no default (themes[0])")
	}
	if len(themeNames) != len(themes) {
		t.Fatalf("themeNames (%d) != themes (%d)", len(themeNames), len(themes))
	}
}

// Muted text (help line, hints, meta labels) has to stay readable in every theme: the
// hand-typed overlay0 alone fails WCAG AA everywhere (omni was 2.3:1).
func TestMutedTextContrast(t *testing.T) {
	for _, th := range themes {
		if th.name == "terminal" {
			continue
		}
		m, _ := parseHex(readable(th.overlay0, th.text, th.panelBg, minTextContrast))
		bg, _ := parseHex(th.panelBg)
		if c := contrast(m, bg); c < minTextContrast {
			t.Errorf("tema %q: texto apagado com contraste %.1f:1 (< %.1f)", th.name, c, minTextContrast)
		}
	}
	if got := readable("7", "15", "", minTextContrast); got != "7" {
		t.Fatalf("ANSI color should pass through, got %q", got)
	}
}
