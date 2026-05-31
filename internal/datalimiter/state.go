package datalimiter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrStateNotFound = errors.New("state file not found")

type State struct {
	Active        bool              `json:"active"`
	SavedPolicies map[string]string `json:"savedPolicies"`
	ChromePath    string            `json:"chromePath"`
	AllowedApps   []AllowedApp      `json:"allowedApps,omitempty"`
	Version       string            `json:"version"`
}

type AllowedApp struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type StateStore interface {
	Load() (State, error)
	Save(State) error
	Delete() error
}

type ProgramDataStateStore struct{}

func (ProgramDataStateStore) Load() (State, error) {
	return LoadState(StatePath())
}

func (ProgramDataStateStore) Save(state State) error {
	return SaveState(StatePath(), state)
}

func (ProgramDataStateStore) Delete() error {
	err := os.Remove(StatePath())
	if errors.Is(err, os.ErrNotExist) {
		return ErrStateNotFound
	}
	return err
}

func StatePath() string {
	base := os.Getenv("ProgramData")
	if base == "" {
		base = `C:\ProgramData`
	}
	return filepath.Join(base, "DataLimiter", "state.json")
}

func LoadState(path string) (State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, ErrStateNotFound
	}
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse state file: %w", err)
	}
	return state, nil
}

func SaveState(path string, state State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0600)
}
