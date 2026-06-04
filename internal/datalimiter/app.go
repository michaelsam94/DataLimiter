package datalimiter

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

const Version = "0.3.1"

type App struct {
	deps Deps
}

type Deps interface {
	IsAdmin() bool
	FindChrome() (string, error)
	ResolveApp(input string) (AllowedApp, error)
	StateStore() StateStore
	Firewall() Firewall
}

func NewApp(deps Deps) App {
	return App{deps: deps}
}

func (a App) Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	var err error
	switch args[0] {
	case "enable":
		if len(args) != 1 {
			printUsage(stderr)
			return 2
		}
		err = a.enable(stdout)
	case "disable":
		if len(args) != 1 {
			printUsage(stderr)
			return 2
		}
		err = a.disable(stdout)
	case "status":
		if len(args) != 1 {
			printUsage(stderr)
			return 2
		}
		err = a.status(stdout)
	case "repair":
		if len(args) != 1 {
			printUsage(stderr)
			return 2
		}
		err = a.repair(stdout)
	case "app":
		err = a.appCommand(args[1:], stdout, stderr)
	case "strict":
		err = a.strictCommand(args[1:], stdout, stderr)
	default:
		printUsage(stderr)
		return 2
	}

	if err != nil {
		if errors.Is(err, errUsage) {
			return 2
		}
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: datalimiter <enable|disable|status|repair>")
	fmt.Fprintln(w, "       datalimiter app add <name-or-path>")
	fmt.Fprintln(w, "       datalimiter app remove <name-or-path>")
	fmt.Fprintln(w, "       datalimiter strict <enable|disable>")
}

func (a App) enable(stdout io.Writer) error {
	if err := a.requireAdmin(); err != nil {
		return err
	}

	chromePath, err := a.deps.FindChrome()
	if err != nil {
		return fmt.Errorf("Chrome was not found in a supported install location; no firewall changes were made")
	}

	fw := a.deps.Firewall()
	profiles, err := fw.ActiveProfiles()
	if err != nil {
		return fmt.Errorf("read active firewall profiles: %w", err)
	}
	if len(profiles) == 0 {
		return errors.New("no active Windows Firewall profiles were reported; no firewall changes were made")
	}

	saved := map[string]string{}
	for _, profile := range profiles {
		policy, err := fw.ProfilePolicy(profile)
		if err != nil {
			return fmt.Errorf("snapshot %s profile firewall policy: %w", profile, err)
		}
		saved[profile] = policy
	}

	state := State{
		Active:        true,
		SavedPolicies: saved,
		ChromePath:    chromePath,
		Version:       Version,
	}
	if err := a.deps.StateStore().Save(state); err != nil {
		return fmt.Errorf("save rollback state before firewall changes: %w", err)
	}

	plan := EnablePlanWithPolicies(saved, chromePath, nil)
	if err := ExecutePlan(fw, plan); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "DataLimiter enabled.")
	fmt.Fprintln(stdout, "Chrome:", chromePath)
	return nil
}

func (a App) disable(stdout io.Writer) error {
	if err := a.requireAdmin(); err != nil {
		return err
	}

	store := a.deps.StateStore()
	state, stateErr := store.Load()
	fw := a.deps.Firewall()

	if err := fw.DeleteDataLimiterRules(); err != nil {
		return fmt.Errorf("remove DataLimiter firewall rules: %w", err)
	}

	if stateErr == nil {
		if state.StrictMode {
			if err := fw.EnableRules(state.DisabledRules); err != nil {
				return fmt.Errorf("restore firewall rules disabled by strict mode: %w", err)
			}
		}
		for profile, policy := range state.SavedPolicies {
			if err := fw.SetProfilePolicy(profile, policy); err != nil {
				return fmt.Errorf("restore %s profile firewall policy: %w", profile, err)
			}
		}
	} else if !errors.Is(stateErr, ErrStateNotFound) {
		return fmt.Errorf("read saved state after removing rules: %w", stateErr)
	}

	if err := store.Delete(); err != nil && !errors.Is(err, ErrStateNotFound) {
		return fmt.Errorf("delete state file: %w", err)
	}

	fmt.Fprintln(stdout, "DataLimiter disabled.")
	if errors.Is(stateErr, ErrStateNotFound) {
		fmt.Fprintln(stdout, "No state file was found, so outbound firewall policy could not be restored automatically.")
	}
	return nil
}

func (a App) status(stdout io.Writer) error {
	store := a.deps.StateStore()
	state, stateErr := store.Load()
	rules, rulesErr := a.deps.Firewall().DataLimiterRulesPresent()
	if rulesErr != nil {
		return fmt.Errorf("read DataLimiter firewall rules: %w", rulesErr)
	}

	switch {
	case stateErr == nil && state.Active && rules:
		fmt.Fprintln(stdout, "Status: active")
		printStrictMode(stdout, state)
		printAllowedApps(stdout, state)
	case stateErr == nil && state.Active && !rules:
		fmt.Fprintln(stdout, "Status: inconsistent")
		fmt.Fprintln(stdout, "State says DataLimiter is active, but expected firewall rules are missing. Run: datalimiter repair")
		printStrictMode(stdout, state)
		printAllowedApps(stdout, state)
	case errors.Is(stateErr, ErrStateNotFound) && rules:
		fmt.Fprintln(stdout, "Status: inconsistent")
		fmt.Fprintln(stdout, "DataLimiter firewall rules exist, but the state file is missing. Run datalimiter disable to remove rules, then restore outbound policy manually if needed.")
	case errors.Is(stateErr, ErrStateNotFound):
		fmt.Fprintln(stdout, "Status: inactive")
	case stateErr != nil:
		return fmt.Errorf("read state file: %w", stateErr)
	default:
		fmt.Fprintln(stdout, "Status: inactive")
	}
	return nil
}

func (a App) repair(stdout io.Writer) error {
	if err := a.requireAdmin(); err != nil {
		return err
	}

	state, err := a.deps.StateStore().Load()
	if err != nil {
		return fmt.Errorf("repair requires an active state file: %w", err)
	}
	if !state.Active {
		return errors.New("DataLimiter is not active; run datalimiter enable first")
	}

	if err := ExecutePlan(a.deps.Firewall(), EnablePlanWithPolicies(state.SavedPolicies, state.ChromePath, state.AllowedApps)); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "DataLimiter firewall rules repaired.")
	return nil
}

func (a App) appCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) != 2 {
		printUsage(stderr)
		return errUsage
	}

	switch args[0] {
	case "add":
		return a.addApp(args[1], stdout)
	case "remove", "rm":
		return a.removeApp(args[1], stdout)
	default:
		printUsage(stderr)
		return errUsage
	}
}

func (a App) strictCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) != 1 {
		printUsage(stderr)
		return errUsage
	}

	switch args[0] {
	case "enable":
		return a.enableStrict(stdout)
	case "disable":
		return a.disableStrict(stdout)
	default:
		printUsage(stderr)
		return errUsage
	}
}

func (a App) enableStrict(stdout io.Writer) error {
	if err := a.requireAdmin(); err != nil {
		return err
	}

	store := a.deps.StateStore()
	state, err := a.loadActiveState()
	if err != nil {
		return err
	}
	if state.StrictMode {
		fmt.Fprintln(stdout, "Strict mode already enabled.")
		return nil
	}

	fw := a.deps.Firewall()
	fmt.Fprintln(stdout, "Strict mode: scanning existing outbound allow rules...")
	rules, err := fw.EnabledOutboundAllowRules()
	if err != nil {
		return fmt.Errorf("snapshot enabled outbound allow rules: %w", err)
	}
	fmt.Fprintf(stdout, "Strict mode: captured %d outbound allow rules to temporarily disable.\n", len(rules))

	state.StrictMode = true
	state.DisabledRules = rules
	fmt.Fprintln(stdout, "Strict mode: saving restore state...")
	if err := store.Save(state); err != nil {
		return fmt.Errorf("save strict mode state before firewall changes: %w", err)
	}
	fmt.Fprintln(stdout, "Strict mode: disabling captured outbound allow rules...")
	if err := fw.DisableRules(rules); err != nil {
		return fmt.Errorf("disable existing outbound allow rules for strict mode: %w", err)
	}
	fmt.Fprintln(stdout, "Strict mode: reapplying DataLimiter allow rules...")
	if err := ExecutePlan(fw, EnablePlanWithPolicies(state.SavedPolicies, state.ChromePath, state.AllowedApps)); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "Strict mode enabled.")
	fmt.Fprintf(stdout, "Disabled outbound allow rules: %d\n", len(rules))
	return nil
}

func (a App) disableStrict(stdout io.Writer) error {
	if err := a.requireAdmin(); err != nil {
		return err
	}

	store := a.deps.StateStore()
	state, err := a.loadActiveState()
	if err != nil {
		return err
	}
	if !state.StrictMode {
		fmt.Fprintln(stdout, "Strict mode is not enabled.")
		return nil
	}

	fmt.Fprintf(stdout, "Strict mode: restoring %d outbound allow rules...\n", len(state.DisabledRules))
	if err := a.deps.Firewall().EnableRules(state.DisabledRules); err != nil {
		return fmt.Errorf("restore firewall rules disabled by strict mode: %w", err)
	}

	restored := len(state.DisabledRules)
	state.StrictMode = false
	state.DisabledRules = nil
	fmt.Fprintln(stdout, "Strict mode: saving restore state...")
	if err := store.Save(state); err != nil {
		return fmt.Errorf("save strict mode state: %w", err)
	}

	fmt.Fprintln(stdout, "Strict mode disabled.")
	fmt.Fprintf(stdout, "Restored outbound allow rules: %d\n", restored)
	return nil
}

func (a App) addApp(input string, stdout io.Writer) error {
	if err := a.requireAdmin(); err != nil {
		return err
	}

	app, err := a.deps.ResolveApp(input)
	if err != nil {
		return fmt.Errorf("resolve executable %q: %w", input, err)
	}

	state, err := a.loadActiveState()
	if err != nil {
		return err
	}
	if hasAllowedApp(state.AllowedApps, app) {
		fmt.Fprintln(stdout, "App already allowed:", app.Name, "("+app.Path+")")
		return nil
	}

	state.AllowedApps = append(state.AllowedApps, app)
	if err := a.deps.StateStore().Save(state); err != nil {
		return fmt.Errorf("save allowed app state: %w", err)
	}
	if err := ExecutePlan(a.deps.Firewall(), EnablePlanWithPolicies(state.SavedPolicies, state.ChromePath, state.AllowedApps)); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "Allowed app added:", app.Name)
	fmt.Fprintln(stdout, "Path:", app.Path)
	return nil
}

func (a App) removeApp(input string, stdout io.Writer) error {
	if err := a.requireAdmin(); err != nil {
		return err
	}

	state, err := a.loadActiveState()
	if err != nil {
		return err
	}

	remaining, removed := removeAllowedApp(state.AllowedApps, input)
	if !removed {
		return fmt.Errorf("app %q is not in the allowed app list", input)
	}

	state.AllowedApps = remaining
	if err := a.deps.StateStore().Save(state); err != nil {
		return fmt.Errorf("save allowed app state: %w", err)
	}
	if err := ExecutePlan(a.deps.Firewall(), EnablePlanWithPolicies(state.SavedPolicies, state.ChromePath, state.AllowedApps)); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "Allowed app removed:", input)
	return nil
}

func (a App) loadActiveState() (State, error) {
	state, err := a.deps.StateStore().Load()
	if err != nil {
		return State{}, fmt.Errorf("this command requires DataLimiter to be active: %w", err)
	}
	if !state.Active {
		return State{}, errors.New("DataLimiter is not active; run datalimiter enable first")
	}
	return state, nil
}

func printAllowedApps(stdout io.Writer, state State) {
	fmt.Fprintln(stdout, "Allowed apps:")
	fmt.Fprintln(stdout, "  Chrome:", state.ChromePath)
	if len(state.AllowedApps) == 0 {
		fmt.Fprintln(stdout, "  Extra apps: none")
		return
	}
	fmt.Fprintln(stdout, "  Extra apps:")
	for _, app := range state.AllowedApps {
		fmt.Fprintln(stdout, "   -", app.Name+":", app.Path)
	}
}

func printStrictMode(stdout io.Writer, state State) {
	if state.StrictMode {
		fmt.Fprintf(stdout, "Strict mode: enabled (%d outbound allow rules disabled)\n", len(state.DisabledRules))
		return
	}
	fmt.Fprintln(stdout, "Strict mode: disabled")
}

func hasAllowedApp(apps []AllowedApp, app AllowedApp) bool {
	for _, existing := range apps {
		if strings.EqualFold(existing.Path, app.Path) || strings.EqualFold(existing.Name, app.Name) {
			return true
		}
	}
	return false
}

func removeAllowedApp(apps []AllowedApp, input string) ([]AllowedApp, bool) {
	needle := strings.TrimSuffix(strings.ToLower(input), ".exe")
	remaining := make([]AllowedApp, 0, len(apps))
	removed := false
	for _, app := range apps {
		appName := strings.TrimSuffix(strings.ToLower(app.Name), ".exe")
		appBase := strings.TrimSuffix(strings.ToLower(baseName(app.Path)), ".exe")
		if strings.EqualFold(app.Path, input) || appName == needle || appBase == needle {
			removed = true
			continue
		}
		remaining = append(remaining, app)
	}
	return remaining, removed
}

var errUsage = errors.New("invalid command")

func (a App) requireAdmin() error {
	if !a.deps.IsAdmin() {
		return errors.New("Administrator privileges are required for this command. Re-run the terminal as Administrator.")
	}
	return nil
}
