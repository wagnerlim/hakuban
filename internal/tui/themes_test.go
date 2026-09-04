package tui

import "testing"

// Guards the theme table (hand-typed): unique, non-empty names, main tokens
// filled in (except "terminal", which uses "" = Reset on purpose),
// and the themeByName fallback.
func TestThemes(t *testing.T) {
	seen := map[string]bool{}
	for _, th := range themes {
		if th.name == "" {
			t.Fatal("tema sem nome")
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
		t.Fatal("themeByName não achou catppuccin")
	}
	if themeByName("inexistente").name != themes[0].name {
		t.Fatal("desconhecido devia cair no default (themes[0])")
	}
	if len(themeNames) != len(themes) {
		t.Fatalf("themeNames (%d) != themes (%d)", len(themeNames), len(themes))
	}
}
