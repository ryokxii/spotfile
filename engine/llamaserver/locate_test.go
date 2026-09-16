package llamaserver

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeExecutable(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, binaryName())
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLocate(t *testing.T) {
	envBin := writeExecutable(t, t.TempDir())
	bundledDir := t.TempDir()
	bundledBin := writeExecutable(t, bundledDir)
	pathDir := t.TempDir()
	pathBin := writeExecutable(t, pathDir)

	tests := []struct {
		name       string
		env        string
		executable string
		path       string
		want       string
		wantErr    bool
	}{
		{"env override wins", envBin, filepath.Join(bundledDir, "spotfile"), pathDir, envBin, false},
		{"bundled next to app", "", filepath.Join(bundledDir, "spotfile"), pathDir, bundledBin, false},
		{"falls back to PATH", "", filepath.Join(t.TempDir(), "spotfile"), pathDir, pathBin, false},
		{"missing env target is an error", filepath.Join(t.TempDir(), "nope"), "", pathDir, "", true},
		{"nothing found", "", filepath.Join(t.TempDir(), "spotfile"), t.TempDir(), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(EnvBinary, tt.env)
			t.Setenv("PATH", tt.path)
			got, err := locate(func() (string, error) { return tt.executable, nil })
			if tt.wantErr {
				if !errors.Is(err, ErrNotFound) {
					t.Fatalf("locate = %q, %v; want ErrNotFound", got, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("locate: %v", err)
			}
			if got != tt.want {
				t.Fatalf("locate = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBinaryName(t *testing.T) {
	want := "llama-server"
	if runtime.GOOS == "windows" {
		want = "llama-server.exe"
	}
	if got := binaryName(); got != want {
		t.Fatalf("binaryName = %q, want %q", got, want)
	}
}
