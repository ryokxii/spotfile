// Package embeddertest provides helpers for tests that need the real embedding
// model. Helpers skip the test when llama.cpp or the model is not installed, so
// suites never fail on machines without them — unless EnvRequire is set.
package embeddertest

import (
	"os"
	"path/filepath"
	"testing"

	"spotfile/engine/embedder"
	"spotfile/engine/llamaserver"
)

// EnvRequire, when "1", turns a missing llama.cpp or model into a test failure
// instead of a skip. CI sets it so integration tests cannot pass without running.
const EnvRequire = "SPOTFILE_REQUIRE_LLAMA"

// Unavailable skips the test, or fails it when EnvRequire is set.
func Unavailable(tb testing.TB, format string, args ...any) {
	tb.Helper()
	if os.Getenv(EnvRequire) == "1" {
		tb.Fatalf(format+" (required: %s=1)", append(args, EnvRequire)...)
	}
	tb.Skipf(format, args...)
}

// ModelPath returns the installed model path (~/.spotfile/models), or skips.
func ModelPath(tb testing.TB) string {
	tb.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		Unavailable(tb, "no home dir: %v", err)
	}
	path := filepath.Join(home, ".spotfile", "models", embedder.ModelFile)
	if _, err := os.Stat(path); err != nil {
		Unavailable(tb, "embedding model missing: %s", path)
	}
	return path
}

// RealModel starts bge-small-en-v1.5 on llama-server and closes it when the
// test ends, or skips when llama.cpp or the model is absent.
func RealModel(tb testing.TB) *embedder.Model {
	tb.Helper()
	if _, err := llamaserver.Locate(); err != nil {
		Unavailable(tb, "llama-server not available: %v", err)
	}
	m, err := embedder.New(embedder.Config{ModelPath: ModelPath(tb)})
	if err != nil {
		tb.Fatalf("embedder.New: %v", err)
	}
	tb.Cleanup(m.Close)
	return m
}
