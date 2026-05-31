//go:build windows

package datalimiter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type NetshFirewall struct{}

func (OSDeps) Firewall() Firewall {
	return NetshFirewall{}
}

func (NetshFirewall) ActiveProfiles() ([]string, error) {
	out, err := runNetsh("advfirewall", "show", "currentprofile")
	if err != nil {
		return nil, err
	}
	lower := strings.ToLower(out)
	profiles := []string{}
	for _, profile := range []string{"domainprofile", "privateprofile", "publicprofile"} {
		shortName := strings.TrimSuffix(profile, "profile")
		if strings.Contains(lower, shortName+" profile") {
			profiles = append(profiles, profile)
		}
	}
	if len(profiles) == 0 {
		return []string{"domainprofile", "privateprofile", "publicprofile"}, nil
	}
	return profiles, nil
}

func (NetshFirewall) ProfilePolicy(profile string) (string, error) {
	out, err := runNetsh("advfirewall", "show", profile)
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`(?mi)^\s*Firewall Policy\s+(.+?)\s*$`)
	match := re.FindStringSubmatch(out)
	if len(match) != 2 {
		return "", errors.New("firewall policy line not found")
	}
	return strings.ReplaceAll(strings.ToLower(match[1]), " ", ""), nil
}

func (NetshFirewall) SetProfilePolicy(profile, policy string) error {
	_, err := runNetsh("advfirewall", "set", profile, "firewallpolicy", policy)
	return err
}

func (NetshFirewall) DeleteDataLimiterRules() error {
	_, err := runPowerShell(`Get-NetFirewallRule -DisplayName 'DataLimiter -*' -ErrorAction SilentlyContinue | Remove-NetFirewallRule`)
	return err
}

func (NetshFirewall) AddRule(rule FirewallRule) error {
	psArgs := []string{
		"New-NetFirewallRule",
		"-DisplayName", quotePS(rule.Name),
		"-Direction", "Outbound",
		"-Action", "Allow",
		"-Enabled", "True",
		"-Profile", "Any",
	}
	if rule.Program != "" {
		psArgs = append(psArgs, "-Program", quotePS(rule.Program))
	}
	if rule.Protocol != "" {
		psArgs = append(psArgs, "-Protocol", rule.Protocol)
	}
	if rule.LocalPort != "" {
		psArgs = append(psArgs, "-LocalPort", rule.LocalPort)
	}
	if rule.RemotePort != "" {
		psArgs = append(psArgs, "-RemotePort", rule.RemotePort)
	}
	_, err := runPowerShell(strings.Join(psArgs, " "))
	return err
}

func (NetshFirewall) DataLimiterRulesPresent() (bool, error) {
	out, err := runPowerShell(`if (Get-NetFirewallRule -DisplayName 'DataLimiter -*' -ErrorAction SilentlyContinue) { 'present' }`)
	if err != nil {
		return false, err
	}
	return strings.Contains(out, "present"), nil
}

func (NetshFirewall) EnabledOutboundAllowRules() ([]FirewallRuleIdentity, error) {
	command := `
$rules = Get-NetFirewallRule -Direction Outbound -Action Allow -Enabled True |
  Where-Object {
    $_.DisplayName -notlike 'DataLimiter -*' -and
    $_.Name -notlike 'CoreNet-*' -and
    $_.Name -notlike 'NETDIS-*' -and
    $_.Name -notlike 'Microsoft-Windows-*' -and
    $_.Name -notlike 'MicrosoftWindows.Client.*' -and
    $_.Name -notlike 'Microsoft.Windows.*' -and
    $_.Name -notlike 'MicrosoftWindows.*' -and
    $_.Name -notlike 'Microsoft.*_cw5n1h2txyewy-*' -and
    $_.Name -notlike 'Microsoft.*_8wekyb3d8bbwe-*' -and
    $_.Name -notlike 'Windows.*_cw5n1h2txyewy-*'
  } |
  Select-Object -Property Name,DisplayName
if ($rules) { $rules | ConvertTo-Json -Compress }
`
	out, err := runPowerShell(command)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	var rules []FirewallRuleIdentity
	if strings.HasPrefix(out, "[") {
		if err := json.Unmarshal([]byte(out), &rules); err != nil {
			return nil, err
		}
		return rules, nil
	}

	var rule FirewallRuleIdentity
	if err := json.Unmarshal([]byte(out), &rule); err != nil {
		return nil, err
	}
	return []FirewallRuleIdentity{rule}, nil
}

func (NetshFirewall) DisableRules(rules []FirewallRuleIdentity) error {
	return setRulesEnabled(rules, false)
}

func (NetshFirewall) EnableRules(rules []FirewallRuleIdentity) error {
	return setRulesEnabled(rules, true)
}

func runNetsh(args ...string) (string, error) {
	cmd := exec.Command("netsh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("netsh %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func runPowerShell(command string) (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("PowerShell firewall command failed: %s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func quotePS(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func setRulesEnabled(rules []FirewallRuleIdentity, enabled bool) error {
	if len(rules) == 0 {
		return nil
	}

	names := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule.Name != "" {
			names = append(names, quotePS(rule.Name))
		}
	}
	if len(names) == 0 {
		return nil
	}

	cmdlet := "Disable-NetFirewallRule"
	if enabled {
		cmdlet = "Enable-NetFirewallRule"
	}
	for _, chunk := range chunkStrings(names, 100) {
		command := "$names = @(" + strings.Join(chunk, ",") + "); Get-NetFirewallRule -Name $names -ErrorAction SilentlyContinue | " + cmdlet
		if _, err := runPowerShell(command); err != nil {
			return err
		}
	}
	return nil
}

func chunkStrings(items []string, size int) [][]string {
	if size <= 0 || len(items) == 0 {
		return nil
	}
	chunks := make([][]string, 0, (len(items)+size-1)/size)
	for start := 0; start < len(items); start += size {
		end := start + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[start:end])
	}
	return chunks
}
