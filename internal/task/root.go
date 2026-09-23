package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// The data dir is selectable (F13), but config.yml lives INSIDE it — so it can't
// store its own path (circular). So the path lives in a pointer OUTSIDE the
// data dir: <os.UserConfigDir>/hakuban/root.yml. Resolution at boot:
// env HAKUBAN_TASK_DIR > pointer > default ~/.hakuban.

type rootPointer struct {
	DataDir string `yaml:"datadir"`
}

func rootPointerPath() string {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cfg, "hakuban", "root.yml")
}

// homeDefault is the ~/.hakuban (final fallback).
func homeDefault() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".hakuban"
	}
	return filepath.Join(home, ".hakuban")
}

// ResolveDir decides the data dir at boot (env > pointer > default).
func ResolveDir() string {
	if d := os.Getenv("HAKUBAN_TASK_DIR"); d != "" {
		return d
	}
	if p := rootPointerPath(); p != "" {
		if data, err := os.ReadFile(p); err == nil {
			var r rootPointer
			if yaml.Unmarshal(data, &r) == nil && strings.TrimSpace(r.DataDir) != "" {
				return r.DataDir
			}
		}
	}
	return homeDefault()
}

// SaveDataDir writes the pointer to the chosen data dir.
func SaveDataDir(dir string) error {
	p := rootPointerPath()
	if p == "" {
		return errors.New("could not locate the user config directory")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(rootPointer{DataDir: dir})
	if err != nil {
		return err
	}
	return atomicWrite(p, data)
}

// dataEntries are the hakuban files/folders inside a data dir.
var dataEntries = []string{"tasks", "boards", "archive", "config.yml", "state.yml"}

// ErrTargetHasData: the target already contains hakuban data (never mixes).
var ErrTargetHasData = errors.New("a pasta de destino já tem dados do hakuban")

// HasData says whether dir already contains hakuban data (tasks/*.md or config.yml).
func HasData(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "config.yml")); err == nil {
		return true
	}
	entries, _ := os.ReadDir(tasksDir(dir))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			return true
		}
	}
	return false
}

// CountData counts the files MoveData would move (to confirm with the user).
func CountData(dir string) int {
	n := 0
	for _, sub := range []string{"tasks", "boards", "archive"} {
		entries, _ := os.ReadDir(filepath.Join(dir, sub))
		for _, e := range entries {
			if !e.IsDir() {
				n++
			}
		}
	}
	for _, f := range []string{"config.yml", "state.yml"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			n++
		}
	}
	return n
}

// MoveData moves the data from `from` to `to`. Aborts if `to` already has data (never
// overwrites/mixes). Moves via os.Rename — same filesystem only; on a
// different filesystem it returns a clear error instead of a half-move.
// ponytail: no cross-fs copy until someone asks; the error names the ceiling.
func MoveData(from, to string) error {
	if from == to {
		return nil
	}
	if HasData(to) {
		return ErrTargetHasData
	}
	if err := os.MkdirAll(to, 0o755); err != nil {
		return err
	}
	for _, name := range dataEntries {
		src := filepath.Join(from, name)
		if _, err := os.Stat(src); err != nil {
			continue // doesn't exist in this data dir, skip
		}
		if err := os.Rename(src, filepath.Join(to, name)); err != nil {
			return fmt.Errorf("movendo %s: %w (destino em outro filesystem?)", name, err)
		}
	}
	return nil
}
