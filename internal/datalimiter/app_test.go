package datalimiter

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunUnknownCommand(t *testing.T) {
	app := NewApp(fakeDeps{})
	var stderr bytes.Buffer
	code := app.Run([]string{"wat"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("stderr = %q, want usage", stderr.String())
	}
}

func TestEnableRequiresAdminBeforeChromeLookup(t *testing.T) {
	deps := fakeDeps{admin: false, chromePath: `C:\Chrome\chrome.exe`}
	app := NewApp(deps)
	var stderr bytes.Buffer
	code := app.Run([]string{"enable"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "Administrator privileges") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestEnableFailsWithoutChromeBeforeFirewallChanges(t *testing.T) {
	fw := &fakeFirewall{}
	deps := fakeDeps{admin: true, chromeErr: ErrChromeNotFound, fw: fw}
	app := NewApp(deps)
	var stderr bytes.Buffer
	code := app.Run([]string{"enable"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if len(fw.calls) != 0 {
		t.Fatalf("firewall calls = %v, want none", fw.calls)
	}
}

func TestEnableSavesStateAndAddsExpectedRules(t *testing.T) {
	store := &memoryStore{}
	fw := &fakeFirewall{
		activeProfiles: []string{"publicprofile"},
		policies:      map[string]string{"publicprofile": "blockinbound,allowoutbound"},
	}
	deps := fakeDeps{admin: true, chromePath: `C:\Chrome\chrome.exe`, fw: fw, store: store}
	app := NewApp(deps)
	code := app.Run([]string{"enable"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !store.state.Active || store.state.ChromePath != deps.chromePath {
		t.Fatalf("state = %#v", store.state)
	}
	if got := len(fw.added); got != len(ExpectedRules(deps.chromePath)) {
		t.Fatalf("added rules = %d, want %d", got, len(ExpectedRules(deps.chromePath)))
	}
}

func TestStatusInconsistentWhenStateActiveButRulesMissing(t *testing.T) {
	store := &memoryStore{state: State{Active: true, ChromePath: `C:\Chrome\chrome.exe`}, exists: true}
	fw := &fakeFirewall{rulesPresent: false}
	app := NewApp(fakeDeps{fw: fw, store: store})
	var stdout bytes.Buffer
	code := app.Run([]string{"status"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "inconsistent") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestDisableDeletesRulesEvenWhenStateMissing(t *testing.T) {
	store := &memoryStore{}
	fw := &fakeFirewall{}
	app := NewApp(fakeDeps{admin: true, fw: fw, store: store})
	code := app.Run([]string{"disable"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !contains(fw.calls, "delete-rules") {
		t.Fatalf("calls = %v, want delete-rules", fw.calls)
	}
}

type fakeDeps struct {
	admin      bool
	chromePath string
	chromeErr  error
	fw         *fakeFirewall
	store      *memoryStore
}

func (d fakeDeps) IsAdmin() bool { return d.admin }
func (d fakeDeps) FindChrome() (string, error) {
	if d.chromeErr != nil {
		return "", d.chromeErr
	}
	return d.chromePath, nil
}
func (d fakeDeps) Firewall() Firewall {
	if d.fw != nil {
		return d.fw
	}
	return &fakeFirewall{}
}
func (d fakeDeps) StateStore() StateStore {
	if d.store != nil {
		return d.store
	}
	return &memoryStore{}
}

type fakeFirewall struct {
	activeProfiles []string
	policies      map[string]string
	rulesPresent  bool
	calls         []string
	added         []FirewallRule
}

func (f *fakeFirewall) ActiveProfiles() ([]string, error) {
	f.calls = append(f.calls, "active-profiles")
	if f.activeProfiles == nil {
		return []string{"publicprofile"}, nil
	}
	return f.activeProfiles, nil
}
func (f *fakeFirewall) ProfilePolicy(profile string) (string, error) {
	f.calls = append(f.calls, "profile-policy:"+profile)
	if f.policies == nil {
		return "blockinbound,allowoutbound", nil
	}
	return f.policies[profile], nil
}
func (f *fakeFirewall) SetProfilePolicy(profile, policy string) error {
	f.calls = append(f.calls, "set-policy:"+profile+":"+policy)
	return nil
}
func (f *fakeFirewall) DeleteDataLimiterRules() error {
	f.calls = append(f.calls, "delete-rules")
	return nil
}
func (f *fakeFirewall) AddRule(rule FirewallRule) error {
	f.calls = append(f.calls, "add-rule:"+rule.Name)
	f.added = append(f.added, rule)
	return nil
}
func (f *fakeFirewall) DataLimiterRulesPresent() (bool, error) {
	return f.rulesPresent, nil
}

type memoryStore struct {
	state  State
	exists bool
}

func (s *memoryStore) Load() (State, error) {
	if !s.exists {
		return State{}, ErrStateNotFound
	}
	return s.state, nil
}
func (s *memoryStore) Save(state State) error {
	s.state = state
	s.exists = true
	return nil
}
func (s *memoryStore) Delete() error {
	if !s.exists {
		return ErrStateNotFound
	}
	s.exists = false
	return nil
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
