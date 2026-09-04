package task

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Store holds all tasks in memory (map by id). Each command/session opens
// a Store, operates and exits — the disk is the source of truth, not this map.
type Store struct {
	dir    string // e.g. ~/.hakuban
	tasks  map[string]*Task
	boards map[string]*Board
}

func tasksDir(dir string) string   { return filepath.Join(dir, "tasks") }
func archiveDir(dir string) string { return filepath.Join(dir, "archive") }

// jiraDir holds the read-only mirrors materialized by the sync (F16). It is a
// derived cache (gitignored): the truth of those cards is Jira, not the disk.
func jiraDir(dir string) string { return filepath.Join(dir, "jira") }

// taskFile is a task's on-disk path: mirrors go to jira/, everything else to
// tasks/. A single point so Save/Delete don't write a mirror in the wrong place.
func taskFile(dir string, t *Task) string {
	if t.Mirror() {
		return filepath.Join(jiraDir(dir), t.ID+".md")
	}
	return filepath.Join(tasksDir(dir), t.ID+".md")
}

// DefaultDir is the default data dir (~/.hakuban), overridable by
// $HAKUBAN_TASK_DIR. ponytail: no XDG until someone asks for it.
func DefaultDir() string {
	if d := os.Getenv("HAKUBAN_TASK_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".hakuban"
	}
	return filepath.Join(home, ".hakuban")
}

// Open loads all tasks from <dir>/tasks/*.md into memory. Creates the directory
// if it does not exist. Each task's id comes from the file name.
func Open(dir string) (*Store, error) {
	s := &Store{dir: dir, tasks: map[string]*Task{}, boards: map[string]*Board{}}
	if err := os.MkdirAll(tasksDir(dir), 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(tasksDir(dir))
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".md")
		content, err := os.ReadFile(filepath.Join(tasksDir(dir), e.Name()))
		if err != nil {
			return nil, err
		}
		t, err := unmarshal(id, string(content))
		if err != nil {
			return nil, err
		}
		s.tasks[id] = t
	}
	if err := s.loadMirrors(); err != nil {
		return nil, err
	}
	if err := s.loadBoards(); err != nil {
		return nil, err
	}
	return s, nil
}

// loadMirrors reads jira/*.md into memory as read-only mirrors (F16). A missing
// directory is not an error. A file in jira/ is always a mirror (forces Source), even
// if the frontmatter omits the field; a local task with the same id (unlikely collision) wins.
func (s *Store) loadMirrors() error {
	entries, err := os.ReadDir(jiraDir(s.dir))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".md")
		if _, ok := s.tasks[id]; ok {
			continue
		}
		content, err := os.ReadFile(filepath.Join(jiraDir(s.dir), e.Name()))
		if err != nil {
			return err
		}
		t, err := unmarshal(id, string(content))
		if err != nil {
			return err
		}
		t.Source = SourceJira
		s.tasks[id] = t
	}
	return nil
}

func now() time.Time { return time.Now().UTC().Truncate(time.Second) }

// Save persists the task (atomic write: tmp + rename). If the ID is empty,
// it generates a new collision-free id and marks it as a creation. Always updates Modified.
func (s *Store) Save(t *Task) error {
	if t.ID == "" {
		board := t.Project
		if board == "" && t.Parent != "" { // a child inherits the board (and the key) from the root
			board = s.EffectiveProject(t.Parent)
		}
		t.ID = s.newID(board) // board key (KEY-NN); no key → random id
		t.Created = now()
	}
	if t.Created.IsZero() {
		t.Created = now()
	}
	if t.Status == "" {
		t.Status = StatusBacklog
	}
	t.Modified = now()

	data, err := t.marshal()
	if err != nil {
		return err
	}
	path := taskFile(s.dir, t) // mirror → jira/, everything else → tasks/
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := atomicWrite(path, data); err != nil {
		return err
	}
	s.tasks[t.ID] = t
	return nil
}

// atomicWrite writes to a .tmp and renames — rename is atomic on the same fs, so
// an external reader never sees a half-written file. Non-negotiable: two
// writers (app + $EDITOR).
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		os.Remove(tmp) // write failed mid-way → don't leave a partial .tmp behind
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// Delete removes the task from disk and from memory. It does NOT touch the children — cascade vs
// promote is a front-end decision (F2), which orchestrates by calling Delete/Save.
func (s *Store) Delete(id string) error {
	path := filepath.Join(tasksDir(s.dir), id+".md")
	if t := s.tasks[id]; t != nil {
		path = taskFile(s.dir, t) // a mirror lives in jira/
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(s.tasks, id)
	return nil
}

// LinkToJira converts a local card into a Jira mirror after the on_enter script
// created the issue (F16): it recreates the card with id = the issue key — so the
// sync matches by id and does NOT duplicate — marks source=jira and gets rid of the local file.
// Idempotent when the id is already the key. ponytail: children that pointed to the old
// id become roots (tolerated by the graph); re-parenting is left for when it hurts.
func (s *Store) LinkToJira(oldID, key, status string) error {
	old := s.tasks[oldID]
	if old == nil || key == "" {
		return nil
	}
	if status == "" {
		status = old.Status
	}
	if oldID == key { // already at the right id: just mark it and re-save
		old.Jira, old.Source, old.Status = key, SourceJira, status
		return s.Save(old)
	}
	mirror := *old // copies the card's fields (does not mutate `old` → Delete finds tasks/)
	mirror.ID, mirror.Jira, mirror.Source, mirror.Status = key, key, SourceJira, status
	if err := s.Save(&mirror); err != nil { // writes jira/key.md
		return err
	}
	return s.Delete(oldID) // removes tasks/oldID.md (old is still local)
}

// MirrorSpec describes a mirror to materialize. The agent only FETCHED the issues (via
// MCP); the one that writes the .md is the core — so the disk stays in the core and the agent
// never touches a file (F16, Slice B). Status already comes mapped to a COLUMN by the caller.
type MirrorSpec struct {
	Key, Title, Status, Priority string
}

// ReconcileMirrors materializes a board's mirrors from what the sync brought
// and REMOVES the board's mirrors that did not come (an issue left the tracked set —
// e.g. someone finished it and it no longer matches any JQL). The position is re-asserted
// by the tracker's truth: the status comes from the caller already mapped to a column. Returns
// how many mirrors remained. A mirror is ONLY written/removed in jira/ (via taskFile).
func (s *Store) ReconcileMirrors(board string, specs []MirrorSpec) (int, error) {
	// An empty sync does NOT delete anything: the agent sometimes returns [] due to variance (gives up,
	// timeout), and "0 issues" is too ambiguous to justify deleting the mirrors —
	// that was what made the card vanish. It only adds/updates; it never empties by mistake.
	if len(specs) == 0 {
		return 0, nil
	}
	keep := make(map[string]bool, len(specs))
	for _, sp := range specs {
		if sp.Key == "" {
			continue
		}
		keep[sp.Key] = true
		prio := sp.Priority
		if prio == "" {
			prio = "normal"
		}
		m := &Task{ID: sp.Key, Title: sp.Title, Status: sp.Status, Priority: prio,
			Project: board, Jira: sp.Key, Source: SourceJira}
		if err := s.Save(m); err != nil {
			return 0, err
		}
	}
	for id, t := range s.tasks { // stale: a board mirror the sync did not list → goes away
		if t.Mirror() && t.Project == board && !keep[id] {
			if err := s.Delete(id); err != nil {
				return 0, err
			}
		}
	}
	return len(keep), nil
}

// Archive moves the task to <dir>/archive/<id>.md and takes it out of the active list.
func (s *Store) Archive(id string) error {
	if err := os.MkdirAll(archiveDir(s.dir), 0o755); err != nil {
		return err
	}
	from := filepath.Join(tasksDir(s.dir), id+".md")
	to := filepath.Join(archiveDir(s.dir), id+".md")
	if err := os.Rename(from, to); err != nil {
		return err
	}
	delete(s.tasks, id)
	return nil
}

// Dir is this store's data dir.
func (s *Store) Dir() string { return s.dir }

// Get returns the task by id (nil if it does not exist).
func (s *Store) Get(id string) *Task { return s.tasks[id] }

// All returns all active tasks, sorted by creation (then id) for
// deterministic output.
func (s *Store) All() []*Task {
	out := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	sortTasks(out)
	return out
}

// Children returns the direct children of id — derived from the graph (nobody stores
// a list of children; the edge lives in the child's `parent`).
func (s *Store) Children(id string) []*Task {
	var out []*Task
	for _, t := range s.tasks {
		if t.Parent == id {
			out = append(out, t)
		}
	}
	sortTasks(out)
	return out
}

func sortTasks(ts []*Task) {
	sort.Slice(ts, func(i, j int) bool {
		if !ts[i].Created.Equal(ts[j].Created) {
			return ts[i].Created.Before(ts[j].Created)
		}
		return ts[i].ID < ts[j].ID
	})
}

// root walks up the `parent` edges to the root of the tree. Tolerates a dangling
// parent (treats the node as root) and cycles (stops on revisit).
func (s *Store) root(id string) *Task {
	seen := map[string]bool{}
	t := s.tasks[id]
	for t != nil && t.Parent != "" && !seen[t.ID] {
		seen[t.ID] = true
		p := s.tasks[t.Parent]
		if p == nil {
			break
		}
		t = p
	}
	return t
}

// EffectiveProject is the project inherited from the root of the tree (walk-up). A child does not
// store its own project — the value is always the root's.
func (s *Store) EffectiveProject(id string) string {
	if r := s.root(id); r != nil {
		return r.Project
	}
	return ""
}

// EffectiveTags is the additive union of the own tags + those of all ancestors,
// sorted. Dynamic inheritance: reflects the ancestors' current state at load time.
func (s *Store) EffectiveTags(id string) []string {
	set := map[string]bool{}
	seen := map[string]bool{}
	for t := s.tasks[id]; t != nil && !seen[t.ID]; t = s.tasks[t.Parent] {
		seen[t.ID] = true
		for _, tag := range t.Tags {
			set[tag] = true
		}
		if t.Parent == "" {
			break
		}
	}
	out := make([]string, 0, len(set))
	for tag := range set {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

// newID generates the id of a new card. If the board has a key (KEY), it uses an
// incremental KEY-NN; otherwise it falls back to the legacy random id. Guarantees no collision.
func (s *Store) newID(boardID string) string {
	key := ""
	if b := s.boards[boardID]; b != nil {
		key = b.Key
	}
	if key == "" {
		return s.randomID()
	}
	for n := s.nextSeq(key); ; n++ {
		if id := fmt.Sprintf("%s-%02d", key, n); !s.idExists(id) {
			return id
		}
	}
}

// nextSeq is the key's next number: max(suffix of the existing KEY-NN) + 1.
// Derived (no saved counter) — survives a card created by hand.
func (s *Store) nextSeq(key string) int {
	prefix := key + "-"
	max := 0
	for id := range s.tasks {
		if strings.HasPrefix(id, prefix) {
			if n, err := strconv.Atoi(id[len(prefix):]); err == nil && n > max {
				max = n
			}
		}
	}
	return max + 1
}

// idExists checks for a collision in memory and on disk.
func (s *Store) idExists(id string) bool {
	if _, ok := s.tasks[id]; ok {
		return true
	}
	_, err := os.Stat(filepath.Join(tasksDir(s.dir), id+".md"))
	return err == nil
}

// randomID is the legacy id (base32, 6 chars) for boards with no key.
func (s *Store) randomID() string {
	for {
		if id := genID(); !s.idExists(id) {
			return id
		}
	}
}

func genID() string {
	b := make([]byte, 5) // 40 bits → 8 base32 chars; we use 6 (~1e9 space)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand falhou: %v", err)) // should not happen
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(b))[:6]
}
