package task

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// State is the UI state persisted between sessions: which tabs are open and
// which one is active. Lives in <dir>/state.yml (derived/disposable, not the source
// of truth — if it disappears, the app reopens all boards).
type State struct {
	Open   []string `yaml:"open"`
	Active int      `yaml:"active"`
}

func statePath(dir string) string { return filepath.Join(dir, "state.yml") }

// LoadState reads the state; absence/error returns the zero state (the TUI then opens everything).
func (s *Store) LoadState() State {
	var st State
	if data, err := os.ReadFile(statePath(s.dir)); err == nil {
		yaml.Unmarshal(data, &st)
	}
	return st
}

// SaveState persists the state (atomic write).
func (s *Store) SaveState(st State) error {
	data, err := yaml.Marshal(st)
	if err != nil {
		return err
	}
	return atomicWrite(statePath(s.dir), data)
}
