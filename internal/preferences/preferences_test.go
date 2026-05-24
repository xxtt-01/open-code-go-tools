package preferences

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingPreferencesUsesDefaults(t *testing.T) {
	prefs, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if prefs.CloseBehavior != DefaultCloseBehavior {
		t.Fatalf("expected default close behavior %q, got %q", DefaultCloseBehavior, prefs.CloseBehavior)
	}
}

func TestSaveAndLoadPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	want := Preferences{CloseBehavior: "minimize"}
	if err := want.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.CloseBehavior != want.CloseBehavior {
		t.Fatalf("expected close behavior %q, got %q", want.CloseBehavior, got.CloseBehavior)
	}
}

func TestRejectInvalidCloseBehavior(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preferences.json")
	if err := os.WriteFile(path, []byte(`{"close_behavior":"unknown"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid close behavior error")
	}
}
