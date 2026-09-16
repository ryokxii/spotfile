package embedder_test

import (
	"context"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"spotfile/engine/embedder"
	"spotfile/engine/embedder/embeddertest"
)

func TestNewRejectsMissingModel(t *testing.T) {
	_, err := embedder.New(embedder.Config{ModelPath: filepath.Join(t.TempDir(), "missing.gguf")})
	if err == nil {
		t.Fatal("New succeeded with a missing model file")
	}
}

func TestEmbedReturnsNormalizedVector(t *testing.T) {
	m := embeddertest.RealModel(t)

	vec, err := m.Embed(context.Background(), "the quick brown fox")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != embedder.Dimensions {
		t.Fatalf("dimension = %d, want %d", len(vec), embedder.Dimensions)
	}
	if n := norm(vec); math.Abs(n-1) > 1e-3 {
		t.Fatalf("vector norm = %.5f, want 1", n)
	}
}

// TestBatchEmbedMatchesSingle guards ordering: each batch result must belong to
// the text at the same index.
func TestBatchEmbedMatchesSingle(t *testing.T) {
	m := embeddertest.RealModel(t)
	ctx := context.Background()
	texts := []string{
		"the quick brown fox",
		strings.Repeat("semantic search over local documents ", 40),
		"a",
		"when is the midterm exam",
	}

	batch, err := m.BatchEmbed(ctx, texts)
	if err != nil {
		t.Fatalf("BatchEmbed: %v", err)
	}
	if len(batch) != len(texts) {
		t.Fatalf("got %d vectors for %d texts", len(batch), len(texts))
	}
	for i, text := range texts {
		single, err := m.Embed(ctx, text)
		if err != nil {
			t.Fatalf("Embed(%d): %v", i, err)
		}
		if sim := dot(batch[i], single); sim < 0.999 {
			t.Errorf("text %d: cosine(batch, single) = %.5f, want >= 0.999", i, sim)
		}
	}
}

func TestBatchEmbedEmptyInput(t *testing.T) {
	m := embeddertest.RealModel(t)
	vecs, err := m.BatchEmbed(context.Background(), nil)
	if err != nil || len(vecs) != 0 {
		t.Fatalf("BatchEmbed(nil) = %d vectors, %v; want 0, nil", len(vecs), err)
	}
}

// TestEmbedQueryUsesInstruction checks the bge retrieval instruction is applied
// to queries but not to passages.
func TestEmbedQueryUsesInstruction(t *testing.T) {
	m := embeddertest.RealModel(t)
	ctx := context.Background()
	const q = "how do I install spotfile"

	query, err := m.EmbedQuery(ctx, q)
	if err != nil {
		t.Fatalf("EmbedQuery: %v", err)
	}
	passage, err := m.Embed(ctx, q)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if sim := dot(query, passage); sim > 0.999 {
		t.Fatalf("query and passage embeddings are identical (cosine %.5f); instruction not applied", sim)
	}
}

func TestCountTokens(t *testing.T) {
	m := embeddertest.RealModel(t)
	ctx := context.Background()

	short, err := m.CountTokens(ctx, "hello world")
	if err != nil {
		t.Fatalf("CountTokens: %v", err)
	}
	if short != 2 {
		t.Fatalf("CountTokens(hello world) = %d, want 2", short)
	}
	long, err := m.CountTokens(ctx, strings.Repeat("hello ", 600))
	if err != nil {
		t.Fatalf("CountTokens: %v", err)
	}
	if long <= embedder.MaxTokens {
		t.Fatalf("CountTokens(600 words) = %d, want > %d", long, embedder.MaxTokens)
	}
}

// TestBatchEmbedReportsOversizedInput documents the contract callers rely on:
// inputs over MaxTokens are rejected with an error, not silently truncated.
func TestBatchEmbedReportsOversizedInput(t *testing.T) {
	m := embeddertest.RealModel(t)
	_, err := m.BatchEmbed(context.Background(), []string{strings.Repeat("hello ", 600)})
	if err == nil {
		t.Fatal("BatchEmbed accepted an input longer than the model window")
	}
}

func dot(a, b []float32) float64 {
	var s float64
	for i := range a {
		s += float64(a[i]) * float64(b[i])
	}
	return s
}

func norm(v []float32) float64 { return math.Sqrt(dot(v, v)) }
