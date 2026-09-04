package tui

import (
	"testing"

	"charm.land/lipgloss/v2"
)

// scanURLs has to find the URL in the *rendered* line (ANSI and all) and hand back the
// x range of the URL text only — that's what makes the click land on the link and not
// on the words around it.
func TestScanURLs(t *testing.T) {
	m := &Model{}
	styled := lipgloss.NewStyle().Bold(true).Render("veja")
	m.scanURLs([]string{
		"",
		styled + " https://acme.atlassian.net/browse/ACME-1775.",
		"(https://example.com/a) e https://example.com/b",
	})

	if len(m.detailLinks) != 3 {
		t.Fatalf("esperava 3 links, veio %d: %+v", len(m.detailLinks), m.detailLinks)
	}
	// trailing "." must not be part of the URL; x0 skips "veja " even though it's styled.
	got := m.detailLinks[0]
	if got.url != "https://acme.atlassian.net/browse/ACME-1775" {
		t.Errorf("url = %q", got.url)
	}
	if got.line != 3 { // index 1 + 2 (border + padding)
		t.Errorf("line = %d, esperava 3", got.line)
	}
	if got.x0 != 3+len("veja ") {
		t.Errorf("x0 = %d, esperava %d", got.x0, 3+len("veja "))
	}
	if got.x1-got.x0 != len(got.url) {
		t.Errorf("largura = %d, esperava %d", got.x1-got.x0, len(got.url))
	}
	// the closing ")" of "(url)" stays out of the URL
	if m.detailLinks[1].url != "https://example.com/a" {
		t.Errorf("url entre parênteses = %q", m.detailLinks[1].url)
	}
}
