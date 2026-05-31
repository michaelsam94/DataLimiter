package datalimiter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindChromeInReturnsFirstExistingFile(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "missing.exe")
	second := filepath.Join(dir, "chrome.exe")
	if err := os.WriteFile(second, []byte("fake"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := FindChromeIn([]string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if got != second {
		t.Fatalf("got %q, want %q", got, second)
	}
}

func TestFindChromeInReturnsNotFound(t *testing.T) {
	_, err := FindChromeIn([]string{filepath.Join(t.TempDir(), "chrome.exe")})
	if err != ErrChromeNotFound {
		t.Fatalf("err = %v, want ErrChromeNotFound", err)
	}
}
