package tui

import (
	"reflect"
	"testing"
)

// Every language in langNames exists and has ALL fields filled in (catches the field
// forgotten in a hand-typed catalog). Reflect avoids listing 30 fields here.
func TestCatalogsComplete(t *testing.T) {
	for _, name := range langNames {
		m, ok := langs[name]
		if !ok {
			t.Fatalf("idioma %q em langNames não tem catálogo", name)
		}
		v := reflect.ValueOf(m)
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).Kind() == reflect.String && v.Field(i).String() == "" {
				t.Errorf("idioma %q: campo %s vazio", name, v.Type().Field(i).Name)
			}
		}
		// keyLabels needs a label for every rebindable action (otherwise the modal
		// shows a blank line)
		for _, a := range keyActions {
			if m.keyLabels[a.id] == "" {
				t.Errorf("idioma %q: keyLabels sem rótulo pra ação %q", name, a.id)
			}
		}
	}
}

// applyLang swaps the active catalog; an unknown one falls back to the first (pt-BR).
func TestApplyLangAndLabels(t *testing.T) {
	applyLang("en-US")
	if statusLabel("done") != "done" || priorityLabel("high") != "high" {
		t.Fatalf("en-US errado: %q / %q", statusLabel("done"), priorityLabel("high"))
	}
	applyLang("zh-Hans")
	if statusLabel("doing") != "进行中" || priorityLabel("low") != "低" {
		t.Fatalf("zh-Hans errado: %q / %q", statusLabel("doing"), priorityLabel("low"))
	}
	applyLang("klingon") // unknown → fallback
	if msg.stDone != langs[langNames[0]].stDone {
		t.Fatal("idioma desconhecido devia cair no default")
	}
	applyLang(langNames[0]) // restore for the other tests
}
