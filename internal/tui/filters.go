package tui

import (
	"encoding/json"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/wagnerlim/hakuban/internal/task"
)

// filterOption is one selectable option of a filter (label shown, value = query fragment).
type filterOption struct{ Label, Value string }

// filterHasOptions reports whether a filter is option-based (static or dynamic) vs a simple
// on/off toggle — without running any options_cmd (checks the definition only).
func filterHasOptions(s *task.Store, board, name string) bool {
	def, ok := s.FilterDef(board, name)
	return ok && (def.OptionsCmd != "" || len(def.Options) > 0)
}

// filterOptions returns a filter's selectable options in display order: for a dynamic
// filter it runs OptionsCmd (which controls the order, e.g. "Atual" first); for static it
// returns Options sorted by label. Empty for a simple filter (it has no options).
func filterOptions(s *task.Store, board, name string) []filterOption {
	def, ok := s.FilterDef(board, name)
	if !ok {
		return nil
	}
	switch {
	case def.OptionsCmd != "":
		return runOptionsCmd(def.OptionsCmd, s.Dir())
	case len(def.Options) > 0:
		labels := make([]string, 0, len(def.Options))
		for l := range def.Options {
			labels = append(labels, l)
		}
		sort.Strings(labels)
		out := make([]filterOption, 0, len(labels))
		for _, l := range labels {
			out = append(out, filterOption{Label: l, Value: def.Options[l]})
		}
		return out
	}
	return nil
}

// resolveSyncFilters turns a column's selection (use_filters: filter → option labels) into
// the values the sync hook receives, GROUPED by filter. Simple filter → its Value; option
// filter → the selected labels' values (static or via OptionsCmd). The core never joins the
// values — the hook ORs a filter's options and ANDs across filters. nil if nothing.
func resolveSyncFilters(s *task.Store, board, col string) map[string][]string {
	use := s.ColumnUseFilters(board, col)
	if len(use) == 0 {
		return nil
	}
	out := map[string][]string{}
	for name, labels := range use {
		def, ok := s.FilterDef(board, name)
		if !ok {
			continue
		}
		var vals []string
		if def.OptionsCmd != "" || len(def.Options) > 0 {
			byLabel := map[string]string{}
			for _, o := range filterOptions(s, board, name) {
				byLabel[o.Label] = o.Value
			}
			for _, l := range labels {
				if v := byLabel[l]; v != "" {
					vals = append(vals, v)
				}
			}
		} else if def.Value != "" { // simple filter: toggled on → its single value
			vals = append(vals, def.Value)
		}
		if len(vals) > 0 {
			out[name] = vals
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// runOptionsCmd runs a filter's dynamic options hook (`sh -c cmd`, cwd = data dir) and
// parses its stdout as ordered `label<TAB>value` lines. A failure yields nil (the filter
// contributes nothing) — best-effort, never blocks.
func runOptionsCmd(command, dir string) []filterOption {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	stdout, err := cmd.Output()
	if err != nil {
		return nil
	}
	var out []filterOption
	for _, line := range strings.Split(string(stdout), "\n") {
		if label, value, ok := strings.Cut(line, "\t"); ok {
			out = append(out, filterOption{Label: strings.TrimSpace(label), Value: strings.TrimSpace(value)})
		}
	}
	return out
}

// filtersJSON encodes the grouped filters for the HAKUBAN_FILTERS env var (empty → "{}").
// The sync hook parses it (an object of filter → [values]) and joins as it wishes.
func filtersJSON(grouped map[string][]string) string {
	if len(grouped) == 0 {
		return "{}"
	}
	b, err := json.Marshal(grouped)
	if err != nil {
		return "{}"
	}
	return string(b)
}
