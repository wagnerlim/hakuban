package tui

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"v0.1.2", "0.1.1", true},
		{"v0.2.0", "v0.1.9", true},
		{"v1.0.0", "0.10.10", true},
		{"v0.1.1", "0.1.1", false},
		{"v0.1.0", "0.1.1", false},
		{"v0.1.2", "v0.1.2-0.20261001-abc", false}, // go install pseudo-version of the same X.Y.Z
		{"garbage", "0.1.1", false},
		{"v0.1.2", "dev", false},
	} {
		if got := newer(c.a, c.b); got != c.want {
			t.Errorf("newer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// The first call hits the API and writes the cache; within the TTL the cache answers
// even with the server gone; past the TTL and offline, the stale tag still comes back.
func TestLatestReleaseCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v9.9.9"}`))
	}))
	old := latestURL
	latestURL = srv.URL
	defer func() { latestURL = old }()

	dir := t.TempDir()
	now := time.Now()
	if got := latestRelease(dir, now); got != "v9.9.9" {
		t.Fatalf("from API: got %q", got)
	}
	srv.Close()
	if got := latestRelease(dir, now.Add(time.Hour)); got != "v9.9.9" {
		t.Fatalf("from fresh cache: got %q", got)
	}
	if got := latestRelease(dir, now.Add(2*updateTTL)); got != "v9.9.9" {
		t.Fatalf("stale cache while offline: got %q", got)
	}
}
