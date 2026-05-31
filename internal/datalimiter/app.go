package datalimiter

import (
	"errors"
	"fmt"
	"io"
)

const Version = "0.1.0"

type App struct {
	deps Deps
}

type Deps interface {
	IsAdmin() bool
	FindChrome() (string, error)
	StateStore() StateStore
	Firewall() Firewall
}

func NewApp(deps Deps) App {
	return App{deps: deps}
}

func (a App) Run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		printUsage(stderr)
		return 2
	}

	var err error
	switch args[0] {
	case "enable":
		err = a.enable(stdout)
	case "disable":
		err = a.disable(stdout)
	case "status":
		err = a.status(stdout)
	case "repair":
		err = a.repair(stdout)
	default:
		printUsage(stderr)
		return 2
	}

	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: datalimiter <enable|disable|status|repair>")
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

	plan := EnablePlanWithPolicies(saved, chromePath)
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
		fmt.Fprintln(stdout, "Chrome:", state.ChromePath)
	case stateErr == nil && state.Active && !rules:
		fmt.Fprintln(stdout, "Status: inconsistent")
		fmt.Fprintln(stdout, "State says DataLimiter is active, but expected firewall rules are missing. Run: datalimiter repair")
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

	if err := ExecutePlan(a.deps.Firewall(), EnablePlanWithPolicies(state.SavedPolicies, state.ChromePath)); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "DataLimiter firewall rules repaired.")
	return nil
}

func (a App) requireAdmin() error {
	if !a.deps.IsAdmin() {
		return errors.New("Administrator privileges are required for this command. Re-run the terminal as Administrator.")
	}
	return nil
}
