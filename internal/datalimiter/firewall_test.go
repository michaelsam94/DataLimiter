package datalimiter

import "testing"

func TestExpectedRules(t *testing.T) {
	rules := ExpectedRules(`C:\Chrome\chrome.exe`, []AllowedApp{{Name: "slack", Path: `C:\Apps\slack.exe`}})
	if len(rules) != 6 {
		t.Fatalf("len = %d, want 6", len(rules))
	}
	for _, rule := range rules {
		if len(rule.Name) < len(RulePrefix) || rule.Name[:len(RulePrefix)] != RulePrefix {
			t.Fatalf("rule name %q does not use prefix %q", rule.Name, RulePrefix)
		}
	}
}

func TestEnablePlanSetsPoliciesBeforeAddingRules(t *testing.T) {
	fw := &fakeFirewall{}
	steps := EnablePlanWithPolicies(map[string]string{"publicprofile": "allowinbound,allowoutbound"}, `C:\Chrome\chrome.exe`, nil)
	if err := ExecutePlan(fw, steps); err != nil {
		t.Fatal(err)
	}
	wantPrefix := []string{"delete-rules", "set-policy:publicprofile:allowinbound,blockoutbound"}
	for i, want := range wantPrefix {
		if fw.calls[i] != want {
			t.Fatalf("call[%d] = %q, want %q; all calls: %v", i, fw.calls[i], want, fw.calls)
		}
	}
}
