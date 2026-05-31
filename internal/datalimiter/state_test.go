package datalimiter

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	want := State{
		Active: true,
		SavedPolicies: map[string]string{
			"publicprofile": "blockinbound,allowoutbound",
		},
		ChromePath: `C:\Chrome\chrome.exe`,
		AllowedApps: []AllowedApp{
			{Name: "slack", Path: `C:\Apps\slack.exe`},
		},
		StrictMode: true,
		DisabledRules: []FirewallRuleIdentity{
			{Name: "rule-id", DisplayName: "Existing App"},
		},
		Version: Version,
	}
	if err := SaveState(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadState(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.ChromePath != want.ChromePath || got.SavedPolicies["publicprofile"] != want.SavedPolicies["publicprofile"] || !got.StrictMode || len(got.AllowedApps) != 1 || len(got.DisabledRules) != 1 {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestLoadStateMissing(t *testing.T) {
	_, err := LoadState(filepath.Join(t.TempDir(), "missing.json"))
	if !errors.Is(err, ErrStateNotFound) {
		t.Fatalf("err = %v, want ErrStateNotFound", err)
	}
}
