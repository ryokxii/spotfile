// Package embedder turns text into normalized embedding vectors by running
// bge-small-en-v1.5 on a llama.cpp llama-server child process.
package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"spotfile/engine/llamaserver"
)

const (
	// ModelFile is the GGUF file expected in Spotfile's models directory.
	ModelFile = "bge-small-en-v1.5-f16.gguf"
	// ID identifies how stored vectors were produced. Changing the model,
	// quantization or pooling must change ID so existing indexes are rebuilt.
	ID = "bge-small-en-v1.5-f16/cls"
	// Dimensions is the embedding vector length.
	Dimensions = 384
	// MaxTokens is the longest input the model accepts: its 512-token window
	// minus the [CLS] and [SEP] tokens the server adds.
	MaxTokens = 510
	// ParallelRequests is how many requests the server processes at once.
	// Callers gain nothing from sending more concurrently.
	ParallelRequests = 2

	// queryInstruction is bge-small-en-v1.5's recommended prefix for short
	// search queries matched against passages. Passages are embedded as-is.
	queryInstruction = "Represent this sentence for searching relevant passages: "

	startTimeout = time.Minute
	errBodyLimit = 1 << 12
)

// Config configures the embedding model.
type Config struct {
	// ModelPath is the bge-small-en-v1.5 GGUF file. Required.
	ModelPath string
	// ServerPath is the llama-server executable. Empty means llamaserver.Locate.
	ServerPath string
}

// Model embeds text through a llama-server running in embedding mode. It is
// safe for concurrent use.
type Model struct {
	srv  *llamaserver.Server
	http *http.Client
}

// New validates the configuration. The server process starts on first use or
// on Start.
func New(cfg Config) (*Model, error) {
	if _, err := os.Stat(cfg.ModelPath); err != nil {
		return nil, fmt.Errorf("embedding model: %w", err)
	}
	perSlot := MaxTokens + 2
	ctxSize := strconv.Itoa(perSlot * ParallelRequests)
	srv := llamaserver.New(llamaserver.Config{
		Binary: cfg.ServerPath,
		Args: []string{
			"-m", cfg.ModelPath,
			"--embeddings", "--pooling", "cls",
			"-ngl", "99", // all layers on the GPU when one is available
			"-np", strconv.Itoa(ParallelRequests),
			"-c", ctxSize, "-b", ctxSize,
			// Encoder models need a whole input in one physical batch.
			"-ub", ctxSize,
			"--no-webui",
		},
		StartTimeout: startTimeout,
	})
	return &Model{srv: srv, http: &http.Client{}}, nil
}

// Start launches the server and waits until the model is loaded, so problems
// such as a missing llama-server surface immediately rather than on first use.
func (m *Model) Start(ctx context.Context) error {
	lease, err := m.srv.Acquire(ctx)
	if err != nil {
		return err
	}
	lease.Release()
	return nil
}

// Close stops the server.
func (m *Model) Close() {
	_ = m.srv.Close()
}

// Embed returns the normalized embedding of a passage.
func (m *Model) Embed(ctx context.Context, text string) ([]float32, error) {
	vecs, err := m.BatchEmbed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// EmbedQuery returns the normalized embedding of a search query, with the
// retrieval instruction bge expects for queries.
func (m *Model) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	return m.Embed(ctx, queryInstruction+query)
}

// BatchEmbed returns one normalized embedding per text, in order. Every text
// must be at most MaxTokens long (see CountTokens); longer inputs are rejected
// by the server rather than truncated.
func (m *Model) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	var resp struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := m.post(ctx, "/v1/embeddings", map[string]any{"input": texts}, &resp); err != nil {
		return nil, fmt.Errorf("embed %d texts: %w", len(texts), err)
	}
	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("embed: got %d vectors for %d texts", len(resp.Data), len(texts))
	}
	vecs := make([][]float32, len(texts))
	for _, d := range resp.Data {
		if d.Index < 0 || d.Index >= len(texts) || len(d.Embedding) != Dimensions {
			return nil, fmt.Errorf("embed: bad vector at index %d (length %d)", d.Index, len(d.Embedding))
		}
		vecs[d.Index] = l2Normalize(d.Embedding)
	}
	return vecs, nil
}

// CountTokens returns how many model tokens text occupies, excluding the
// special tokens added at embedding time. Compare against MaxTokens.
func (m *Model) CountTokens(ctx context.Context, text string) (int, error) {
	var resp struct {
		Tokens []int `json:"tokens"`
	}
	if err := m.post(ctx, "/tokenize", map[string]any{"content": text}, &resp); err != nil {
		return 0, fmt.Errorf("tokenize: %w", err)
	}
	return len(resp.Tokens), nil
}

// post sends a JSON request to the server and decodes a JSON response.
func (m *Model) post(ctx context.Context, path string, body, out any) error {
	lease, err := m.srv.Acquire(ctx)
	if err != nil {
		return err
	}
	defer lease.Release()

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, lease.BaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+lease.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, errBodyLimit))
		return fmt.Errorf("llama-server %s: %s: %s", path, resp.Status, bytes.TrimSpace(msg))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func l2Normalize(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return v
	}
	n := math.Sqrt(sum)
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) / n)
	}
	return out
}
