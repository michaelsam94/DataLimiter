//go:build !windows

package datalimiter

import "errors"

type UnsupportedFirewall struct{}

func (OSDeps) Firewall() Firewall {
	return UnsupportedFirewall{}
}

func (UnsupportedFirewall) ActiveProfiles() ([]string, error) {
	return nil, errors.New("Windows Firewall is only available on Windows")
}
func (UnsupportedFirewall) ProfilePolicy(string) (string, error) {
	return "", errors.New("Windows Firewall is only available on Windows")
}
func (UnsupportedFirewall) SetProfilePolicy(string, string) error {
	return errors.New("Windows Firewall is only available on Windows")
}
func (UnsupportedFirewall) DeleteDataLimiterRules() error {
	return errors.New("Windows Firewall is only available on Windows")
}
func (UnsupportedFirewall) AddRule(FirewallRule) error {
	return errors.New("Windows Firewall is only available on Windows")
}
func (UnsupportedFirewall) DataLimiterRulesPresent() (bool, error) {
	return false, errors.New("Windows Firewall is only available on Windows")
}
func (UnsupportedFirewall) EnabledOutboundAllowRules() ([]FirewallRuleIdentity, error) {
	return nil, errors.New("Windows Firewall is only available on Windows")
}
func (UnsupportedFirewall) DisableRules([]FirewallRuleIdentity) error {
	return errors.New("Windows Firewall is only available on Windows")
}
func (UnsupportedFirewall) EnableRules([]FirewallRuleIdentity) error {
	return errors.New("Windows Firewall is only available on Windows")
}
