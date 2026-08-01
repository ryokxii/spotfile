package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testLibraryPath returns a loadable onnxruntime shared library path, or skips
// the test if none is installed. The env-init bug only manifests once the
// library loads and InitializeEnvironment succeeds.
func testLibraryPath(t *testing.T) string {
	t.Helper()
	for _, p := range []string{
		"/opt/homebrew/lib/libonnxruntime.dylib",
		"/usr/local/lib/libonnxruntime.dylib",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("onnxruntime shared library not found")
	return ""
}

// TestEnvInitIdempotent verifies that repeated New calls never surface the
// "already been initialized" error from the global ORT environment.
//
// The ORT environment is a process-global singleton. A previous New that
// initialized the env but then failed at a later step (missing model) must not
// poison subsequent New calls with a misleading "already been initialized"
// error — the second call should report the real underlying failure instead.
func TestEnvInitIdempotent(t *testing.T) {
	vocabPath := filepath.Join(t.TempDir(), "vocab.txt")
	if err := createTestVocab(vocabPath); err != nil {
		t.Fatalf("create test vocab: %v", err)
	}

	cfg := Config{
		LibraryPath: testLibraryPath(t),
		ModelPath:   filepath.Join(t.TempDir(), "missing-model.onnx"),
		VocabPath:   vocabPath,
		Workers:     1,
	}

	// First call initializes the global env, then fails creating the session
	// because the model file is absent.
	if _, err := New(cfg); err == nil {
		t.Fatal("expected first New to fail on missing model")
	}

	// Second call must surface the real session error, not the leaked-env one.
	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected second New to fail on missing model")
	}
	if strings.Contains(err.Error(), "already been initialized") {
		t.Fatalf("ORT env init is not idempotent: %v", err)
	}
}
