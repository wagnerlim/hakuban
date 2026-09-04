package task

import "testing"

// Config: missing file → sane defaults; and a partial file only overrides the
// keys present, keeping the rest at the default (the merge is the critical point).
func TestConfigDefaultsAndPartialMerge(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	cfg := s.LoadConfig() // file does not exist
	if cfg.Theme != "omni" || cfg.DateFormat != "2006-01-02" || !cfg.ConfirmDelete {
		t.Fatalf("defaults errados: %+v", cfg)
	}

	// write only a subset (like a config.yml hand-edited without confirm_delete)
	cfg.PreviewPane = true
	cfg.DateFormat = "02/01/2006"
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	got := s.LoadConfig()
	if !got.PreviewPane || got.DateFormat != "02/01/2006" {
		t.Fatalf("valores salvos não voltaram: %+v", got)
	}
	if !got.ConfirmDelete || got.Theme != "omni" {
		t.Fatalf("defaults não preservados no merge: %+v", got)
	}
}
