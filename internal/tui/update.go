package tui

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Version is the running binary's version, set by cmd/hakuban before the TUI starts.
// "dev" (a local build) never checks for updates: there is nothing to compare.
var Version = "dev"

// latestURL is the GitHub API endpoint for the newest release (a var so the test can
// point it at an httptest server).
var latestURL = "https://api.github.com/repos/wagnerlim/hakuban/releases/latest"

// releasesPage is what the footer points at. The notice does not guess the install
// channel (brew/scoop/go install): a link to the release cannot be wrong.
const releasesPage = "github.com/wagnerlim/hakuban/releases/latest"

// updateTTL is how long a check is reused, so opening the TUI does not hit the
// network every time.
const updateTTL = 24 * time.Hour

// updateMsg carries the newest release tag ("" = nothing newer, or the check failed).
type updateMsg string

// updateCache is <dir>/update.json: the last tag seen and when it was fetched.
type updateCache struct {
	Latest  string    `json:"latest"`
	Checked time.Time `json:"checked"`
}

// checkUpdateCmd runs the check off the UI goroutine. Off when update_check is false
// in config.yml or the binary is a dev build. Every failure is silent: no network must
// never become an error on screen.
func (m *Model) checkUpdateCmd() tea.Cmd {
	if !m.cfg.UpdateCheck || Version == "dev" {
		return nil
	}
	dir := m.store.Dir()
	return func() tea.Msg {
		if latest := latestRelease(dir, time.Now()); newer(latest, Version) {
			return updateMsg(latest)
		}
		return updateMsg("")
	}
}

// latestRelease returns the newest release tag: from the cache while it is fresh,
// otherwise from the GitHub API (and refreshes the cache). "" when it can't tell.
func latestRelease(dir string, now time.Time) string {
	path := filepath.Join(dir, "update.json")
	var c updateCache
	if data, err := os.ReadFile(path); err == nil && json.Unmarshal(data, &c) == nil && now.Sub(c.Checked) < updateTTL {
		return c.Latest
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(latestURL)
	if err != nil {
		return c.Latest // offline: a stale answer beats none
	}
	defer resp.Body.Close()
	var r struct {
		TagName string `json:"tag_name"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&r) != nil || r.TagName == "" {
		return c.Latest
	}
	if data, err := json.Marshal(updateCache{Latest: r.TagName, Checked: now}); err == nil {
		tmp := path + ".tmp" // atomic like every other write in the data dir
		if os.WriteFile(tmp, data, 0o644) == nil {
			os.Rename(tmp, path)
		}
	}
	return r.TagName
}

// newer reports whether release tag a is a higher X.Y.Z than b. The "v" prefix and any
// pre-release suffix are ignored; anything unparseable is never newer.
// ponytail: plain X.Y.Z compare, swap for golang.org/x/mod/semver if pre-releases ever ship.
func newer(a, b string) bool {
	pa, okA := parseVersion(a)
	pb, okB := parseVersion(b)
	if !okA || !okB {
		return false
	}
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parseVersion(s string) ([3]int, bool) {
	var v [3]int
	s = strings.TrimPrefix(s, "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return v, false
		}
		v[i] = n
	}
	return v, true
}
