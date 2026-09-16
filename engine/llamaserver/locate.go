package llamaserver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// EnvBinary overrides where llama-server is found, e.g. to test a custom build.
const EnvBinary = "SPOTFILE_LLAMA_SERVER"

// Locate finds llama-server, checking in order: the EnvBinary override, a copy
// bundled next to the running executable, then PATH (development installs such
// as Homebrew).
func Locate() (string, error) {
	return locate(os.Executable)
}

func locate(executable func() (string, error)) (string, error) {
	if p := os.Getenv(EnvBinary); p != "" {
		if !isFile(p) {
			return "", fmt.Errorf("%s=%q is not a file", EnvBinary, p)
		}
		return p, nil
	}
	if exe, err := executable(); err == nil && exe != "" {
		if bundled := filepath.Join(filepath.Dir(exe), binaryName()); isFile(bundled) {
			return bundled, nil
		}
	}
	if p, err := exec.LookPath(binaryName()); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("%s not found: bundle it next to the app, install llama.cpp, or set %s", binaryName(), EnvBinary)
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "llama-server.exe"
	}
	return "llama-server"
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
