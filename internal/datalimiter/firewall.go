package datalimiter

import (
	"fmt"
	"sort"
)

const RulePrefix = "DataLimiter -"

type Firewall interface {
	ActiveProfiles() ([]string, error)
	ProfilePolicy(profile string) (string, error)
	SetProfilePolicy(profile, policy string) error
	DeleteDataLimiterRules() error
	AddRule(rule FirewallRule) error
	DataLimiterRulesPresent() (bool, error)
}

type FirewallRule struct {
	Name      string
	Program   string
	Protocol  string
	LocalPort string
	RemotePort string
}

type PlanStep struct {
	Description string
	Apply       func(Firewall) error
}

func EnablePlan(profiles []string, chromePath string) []PlanStep {
	policies := map[string]string{}
	for _, profile := range profiles {
		policies[profile] = "blockinbound,allowoutbound"
	}
	return EnablePlanWithPolicies(policies, chromePath)
}

func EnablePlanWithPolicies(savedPolicies map[string]string, chromePath string) []PlanStep {
	steps := []PlanStep{
		{
			Description: "remove existing DataLimiter rules",
			Apply: func(fw Firewall) error {
				return fw.DeleteDataLimiterRules()
			},
		},
	}

	profiles := make([]string, 0, len(savedPolicies))
	for profile := range savedPolicies {
		profiles = append(profiles, profile)
	}
	sort.Strings(profiles)

	for _, profile := range profiles {
		savedPolicy := savedPolicies[profile]
		profile, savedPolicy := profile, savedPolicy
		steps = append(steps, PlanStep{
			Description: fmt.Sprintf("set %s outbound policy to block", profile),
			Apply: func(fw Firewall) error {
				return fw.SetProfilePolicy(profile, withBlockedOutbound(savedPolicy))
			},
		})
	}

	for _, rule := range ExpectedRules(chromePath) {
		rule := rule
		steps = append(steps, PlanStep{
			Description: "add firewall rule " + rule.Name,
			Apply: func(fw Firewall) error {
				return fw.AddRule(rule)
			},
		})
	}

	return steps
}

func withBlockedOutbound(policy string) string {
	for i, r := range policy {
		if r == ',' {
			return policy[:i+1] + "blockoutbound"
		}
	}
	return "blockinbound,blockoutbound"
}

func ExpectedRules(chromePath string) []FirewallRule {
	return []FirewallRule{
		{Name: RulePrefix + " Allow Chrome", Program: chromePath},
		{Name: RulePrefix + " Allow DNS UDP", Protocol: "UDP", RemotePort: "53"},
		{Name: RulePrefix + " Allow DNS TCP", Protocol: "TCP", RemotePort: "53"},
		{Name: RulePrefix + " Allow DHCP Client", Protocol: "UDP", LocalPort: "68", RemotePort: "67"},
		{Name: RulePrefix + " Allow DHCP Server Replies", Protocol: "UDP", LocalPort: "67", RemotePort: "68"},
	}
}

func ExecutePlan(fw Firewall, steps []PlanStep) error {
	for _, step := range steps {
		if err := step.Apply(fw); err != nil {
			return fmt.Errorf("%s: %w", step.Description, err)
		}
	}
	return nil
}
