package llamaserver

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ErrNotFound is returned (wrapped) when no llama-server executable exists.
var ErrNotFound = errors.New("llama-server not found")

// EnvBinary overrides where llama-server is found, e.g. to test a custom build.
const EnvBinary = "SPOTFILE_LLAMA_SERVER"

// Locate finds llama-server, checking in order: the EnvBinary override, a copy
// bundled next to the running executable, PATH, then common install
// directories. The last step matters for GUI apps: macOS launches apps from
// Finder with PATH=/usr/bin:/bin:/usr/sbin:/sbin, which omits Homebrew.
func Locate() (string, error) {
	return locate(os.Executable, installDirs())
}

// installDirs lists where package managers put llama-server outside PATH.
func installDirs() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/opt/homebrew/bin", "/usr/local/bin"} // Homebrew on Apple Silicon, Intel
	case "linux":
		return []string{"/home/linuxbrew/.linuxbrew/bin", "/usr/local/bin"}
	default:
		return nil
	}
}

func locate(executable func() (string, error), dirs []string) (string, error) {
	if p := os.Getenv(EnvBinary); p != "" {
		if !isFile(p) {
			return "", fmt.Errorf("%w: %s=%q is not a file", ErrNotFound, EnvBinary, p)
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
	for _, dir := range dirs {
		if p := filepath.Join(dir, binaryName()); isFile(p) {
			return p, nil
		}
	}
	return "", fmt.Errorf("%w: bundle %s next to the app, install llama.cpp, or set %s", ErrNotFound, binaryName(), EnvBinary)
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
