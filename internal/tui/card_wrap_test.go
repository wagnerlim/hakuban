package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/wagnerlim/hakuban/internal/task"
)

// hyphen break in x/ansi Wordwrap used to overflow the card and double every line.
func TestRenderCardHyphenTitleFits(t *testing.T) {
	c := &task.Task{ID: "PMD-1887", Title: "[CORAHUB][DEV] - Front-end - Cadastro do Cliente + termo de consentimento"}
	for w := 10; w <= 40; w++ {
		out := renderCard(c, false, w, "", nil)
		for _, ln := range strings.Split(out, "\n") {
			if lipgloss.Width(ln) != w+4 {
				t.Fatalf("w=%d: line %q is %d wide, want %d", w, ln, lipgloss.Width(ln), w+4)
			}
		}
	}
}
