package embedder_test

import (
	"path/filepath"
	"strings"
	"testing"

	"spotfile/engine/embedder"
	"spotfile/engine/embedder/embeddertest"
)

// TestEnvInitIdempotent verifies that repeated embedder.New calls never surface the
// "already been initialized" error from the global ORT environment.
//
// The ORT environment is a process-global singleton. A previous New that
// initialized the env but then failed at a later step (missing model) must not
// poison subsequent New calls with a misleading "already been initialized"
// error — the second call should report the real underlying failure instead.
func TestEnvInitIdempotent(t *testing.T) {
	cfg := embedder.Config{
		LibraryPath: embeddertest.LibraryPath(t),
		ModelPath:   filepath.Join(t.TempDir(), "missing-model.onnx"),
		VocabPath:   embeddertest.WriteVocab(t),
		Workers:     1,
	}

	// First call initializes the global env, then fails creating the session
	// because the model file is absent.
	if _, err := embedder.New(cfg); err == nil {
		t.Fatal("expected first New to fail on missing model")
	}

	// Second call must surface the real session error, not the leaked-env one.
	_, err := embedder.New(cfg)
	if err == nil {
		t.Fatal("expected second New to fail on missing model")
	}
	if strings.Contains(err.Error(), "already been initialized") {
		t.Fatalf("ORT env init is not idempotent: %v", err)
	}
}
