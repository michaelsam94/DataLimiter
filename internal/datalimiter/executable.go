package datalimiter

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrExecutableNotFound = errors.New("executable not found")

func ResolveExecutable(input string) (AllowedApp, error) {
	cleaned := strings.TrimSpace(input)
	if cleaned == "" {
		return AllowedApp{}, ErrExecutableNotFound
	}

	if looksLikePath(cleaned) {
		return allowedAppFromPath(cleaned)
	}

	name := cleaned
	if filepath.Ext(name) == "" {
		name += ".exe"
	}

	resolved, err := exec.LookPath(name)
	if err != nil {
		return AllowedApp{}, ErrExecutableNotFound
	}
	return allowedAppFromPath(resolved)
}

func allowedAppFromPath(path string) (AllowedApp, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return AllowedApp{}, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AllowedApp{}, ErrExecutableNotFound
		}
		return AllowedApp{}, err
	}
	if info.IsDir() {
		return AllowedApp{}, ErrExecutableNotFound
	}

	base := baseName(absPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return AllowedApp{Name: name, Path: absPath}, nil
}

func looksLikePath(input string) bool {
	return filepath.IsAbs(input) || strings.ContainsAny(input, `\/`)
}

func baseName(path string) string {
	return filepath.Base(path)
}
