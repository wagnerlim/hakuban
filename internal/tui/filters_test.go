package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wagnerlim/hakuban/internal/task"
)

// filtersStore writes a board with a filters registry + a column using them, and returns
// the store. The dynamic filter's options_cmd is a shell script written into the data dir.
func filtersStore(t *testing.T) *task.Store {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "boards"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	// dynamic options hook: prints two `label<TAB>value` lines.
	opts := "printf 'Atual\\tsprint in openSprints()\\nSprint 3\\tsprint = 1570\\n'\n"
	if err := os.WriteFile(filepath.Join(dir, "hooks", "opts.sh"), []byte(opts), 0o755); err != nil {
		t.Fatal(err)
	}
	board := `---
name: B
key: B
filters:
    contas: 'assignee = me'
    sprint:
        options_cmd: sh hooks/opts.sh
actions:
    Col:
        jql: 'project = X'
        use_filters:
            contas: []
            sprint: [Atual, "Sprint 3"]
    Simples:
        use_filters:
            contas: []
---
`
	if err := os.WriteFile(filepath.Join(dir, "boards", "b.md"), []byte(board), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Filter YAML: scalar → Value (contas); mapping → OptionsCmd (sprint).
func TestFilterUnmarshal(t *testing.T) {
	s := filtersStore(t)
	if f, ok := s.FilterDef("b", "contas"); !ok || f.Value != "assignee = me" || f.OptionsCmd != "" {
		t.Fatalf("contas devia ser simples (Value), veio %+v", f)
	}
	if f, ok := s.FilterDef("b", "sprint"); !ok || f.OptionsCmd == "" || f.Value != "" {
		t.Fatalf("sprint devia ter options_cmd, veio %+v", f)
	}
}

// resolveSyncFilters: simple filter → its Value; dynamic filter → runs options_cmd and maps
// the selected labels to their values (OR handled later by the hook). Grouped by filter.
func TestResolveSyncFilters(t *testing.T) {
	s := filtersStore(t)
	got := resolveSyncFilters(s, "b", "Col")
	want := map[string][]string{
		"contas": {"assignee = me"},
		"sprint": {"sprint in openSprints()", "sprint = 1570"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveSyncFilters:\n got  %v\n want %v", got, want)
	}
	// a column with only a simple filter selected
	if got := resolveSyncFilters(s, "b", "Simples"); !reflect.DeepEqual(got, map[string][]string{"contas": {"assignee = me"}}) {
		t.Fatalf("coluna Simples: veio %v", got)
	}
}

// filtersJSON encodes the grouped map (empty → "{}").
func TestFiltersJSON(t *testing.T) {
	if got := filtersJSON(nil); got != "{}" {
		t.Fatalf("vazio devia dar {}, veio %q", got)
	}
	if got := filtersJSON(map[string][]string{"a": {"x"}}); got != `{"a":["x"]}` {
		t.Fatalf("json veio %q", got)
	}
}
