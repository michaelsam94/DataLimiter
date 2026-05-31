package datalimiter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveExecutableFromPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.exe")
	if err := os.WriteFile(path, []byte("fake"), 0600); err != nil {
		t.Fatal(err)
	}

	app, err := ResolveExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	if app.Name != "sample" || app.Path == "" {
		t.Fatalf("app = %#v", app)
	}
}

func TestResolveExecutableMissing(t *testing.T) {
	_, err := ResolveExecutable(filepath.Join(t.TempDir(), "missing.exe"))
	if err != ErrExecutableNotFound {
		t.Fatalf("err = %v, want ErrExecutableNotFound", err)
	}
}
