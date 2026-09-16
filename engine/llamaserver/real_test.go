package llamaserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRealEmbeddingServer starts the installed llama-server with bge-small and
// requests one embedding. Skipped when llama.cpp or the model is absent, unless
// SPOTFILE_REQUIRE_LLAMA=1 (set in CI), which makes that a failure.
func TestRealEmbeddingServer(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a real llama-server")
	}
	// embeddertest cannot be imported here (it depends on this package).
	unavailable := t.Skipf
	if os.Getenv("SPOTFILE_REQUIRE_LLAMA") == "1" {
		unavailable = t.Fatalf
	}
	bin, err := Locate()
	if err != nil {
		unavailable("llama-server not found: %v", err)
	}
	home, _ := os.UserHomeDir()
	model := filepath.Join(home, ".spotfile", "models", "bge-small-en-v1.5-f16.gguf")
	if _, err := os.Stat(model); err != nil {
		unavailable("model missing: %s", model)
	}

	s := New(Config{
		Binary:       bin,
		Args:         []string{"-m", model, "--embeddings", "--pooling", "cls", "-ngl", "99", "-np", "2", "-c", "1024", "-ub", "1024"},
		StartTimeout: time.Minute,
	})
	defer s.Close()

	l, err := s.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer l.Release()

	body, _ := json.Marshal(map[string]any{"input": []string{"the quick brown fox"}})
	req, _ := http.NewRequest(http.MethodPost, l.BaseURL+"/v1/embeddings", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+l.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("embeddings request: %v", err)
	}
	defer resp.Body.Close()
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("embeddings response: status %d, decode err %v", resp.StatusCode, err)
	}
	if len(out.Data) != 1 || len(out.Data[0].Embedding) != 384 {
		t.Fatalf("unexpected embedding shape: %+v", out)
	}
}
