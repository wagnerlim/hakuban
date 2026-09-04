package task

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the user's preferences, persisted in <dir>/config.yml (same
// pattern as state.go). Sane defaults: the file may not exist and everything works.
// Hand-editable — the TUI re-reads it on load. Does NOT store the data dir (it would
// live inside itself); that still comes via env HAKUBAN_TASK_DIR + flag.
type Config struct {
	Theme         string   `yaml:"theme"`          // theme name (see internal/tui/themes.go) (live)
	Lang          string   `yaml:"lang"`           // pt-BR | en-US | zh-Hans (F21)
	PreviewPane   bool     `yaml:"preview_pane"`   // .md pane in the footer (live)
	DateFormat    string   `yaml:"date_format"`    // Go layout for due    (live)
	Editor        string   `yaml:"editor"`         // "" = $EDITOR         (F3)
	ConfirmDelete bool     `yaml:"confirm_delete"` //                      (F2)
	Priorities    []string `yaml:"priorities"`     // levels               (F9)
	// Keys are shortcut overrides (action→key). Only what differs from the default
	// lives here — the TUI applies the defaults on top. Action and keys are a TUI
	// concept (internal/tui); here it's just an opaque map, hand-editable.
	Keys map[string]string `yaml:"keys,omitempty"`
	// Tags is the tag catalog (F8): color + description per tag. It's a suggestion,
	// not a lock — a tag outside the catalog renders with the default color. Hand-editable.
	Tags []TagDef `yaml:"tags,omitempty"`
}

// TagDef is an entry in the tag catalog. Color is a named hue from the theme's
// palette (mauve|blue|green|yellow|peach|red|teal) — resolved at render time, so it
// adapts the color to the active theme; "" falls back to the default color. Desc is a short reminder.
type TagDef struct {
	Name  string `yaml:"name"`
	Color string `yaml:"color,omitempty"`
	Desc  string `yaml:"desc,omitempty"`
}

// DefaultConfig holds the sane defaults applied when the file is missing a field.
func DefaultConfig() Config {
	return Config{
		Theme:         "omni",
		Lang:          "pt-BR",
		DateFormat:    "2006-01-02",
		ConfirmDelete: true,
		Priorities:    []string{"low", "normal", "high"},
	}
}

func configPath(dir string) string { return filepath.Join(dir, "config.yml") }

// LoadConfig reads the config on top of the defaults — missing fields / a missing
// file keep the sane default. Only keys present in the file override.
func (s *Store) LoadConfig() Config {
	cfg := DefaultConfig()
	if data, err := os.ReadFile(configPath(s.dir)); err == nil {
		yaml.Unmarshal(data, &cfg)
	}
	return cfg
}

// SaveConfig persists the config (atomic write).
func (s *Store) SaveConfig(cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return atomicWrite(configPath(s.dir), data)
}
