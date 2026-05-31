package datalimiter

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrChromeNotFound = errors.New("chrome not found")

func ChromeCandidatePaths() []string {
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		candidates = append(candidates, filepath.Join(localAppData, "Google", "Chrome", "Application", "chrome.exe"))
	}
	return candidates
}

func FindChromeIn(paths []string) (string, error) {
	for _, path := range paths {
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", ErrChromeNotFound
}
