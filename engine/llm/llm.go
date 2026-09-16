// Package llm is a placeholder for local answer generation over search results.
package llm

import (
	"context"
	"fmt"
	"strings"

	"spotfile/engine/vectorstore"
)

// Config holds configuration for the LLM engine.
type Config struct {
	ModelPath   string // Path to GGUF model file
	ModelType   string // "llama" or "gemma"
	ContextSize int    // Context window size (default 2048)
	MaxTokens   int    // Max generation tokens (default 512)
	Temperature float32
	TopP        float32
}

// Model provides local inference via a quantized language model.
type Model struct {
	cfg    Config
	loaded bool
	// In real implementation, would hold model pointer from llama.go
	// For now, we have a graceful no-op implementation
}

// New initializes the model with the given config.
// Returns error if model file not found (allows graceful degradation).
func New(cfg Config) (*Model, error) {
	if cfg.ContextSize == 0 {
		cfg.ContextSize = 2048
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 512
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}
	if cfg.TopP == 0 {
		cfg.TopP = 0.9
	}

	m := &Model{cfg: cfg, loaded: false}

	// TODO: Load actual GGUF model via llama.go when available
	// For now, this is a placeholder that allows search to work
	// LLM generation will return an error asking user to download model

	return m, nil
}

// Generate creates a response based on system prompt, query, and context.
// Context is formatted search results; query is user's question.
// Returns error if model not loaded with helpful message.
func (l *Model) Generate(ctx context.Context, systemPrompt, query, context string) (string, error) {
	if !l.loaded {
		return "", fmt.Errorf(
			"LLM model not loaded. To enable generative search:\n"+
				"1. Download a quantized model (Llama 3.2 3B or Gemma 3 4B in GGUF format)\n"+
				"2. Place it at: %s\n"+
				"3. Restart the app\n"+
				"For now, semantic search is available without generative answers", l.cfg.ModelPath)
	}

	// TODO: Implement actual LLM inference when llama.go is integrated
	// This would:
	// 1. Format prompt: systemPrompt + context + query
	// 2. Tokenize input
	// 3. Run inference with temperature/topP
	// 4. Decode output tokens
	// 5. Stream/return response

	// Placeholder: return formatted message
	answer := fmt.Sprintf(
		"[LLM Response]\n\n"+
			"Question: %s\n\n"+
			"Based on the provided documents:\n%s",
		query, context)
	return answer, nil
}

// Close releases model resources.
func (l *Model) Close() error {
	// TODO: Cleanup model resources
	return nil
}

// IsLoaded returns whether a model is currently loaded.
func (l *Model) IsLoaded() bool {
	return l.loaded
}

// FormatContext formats search results for use as LLM context.
func FormatContext(results []vectorstore.SearchResult, maxLength int) string {
	var sb strings.Builder
	sb.WriteString("## Relevant Documents\n\n")

	for _, result := range results {
		sb.WriteString(fmt.Sprintf("**Source: %s (Chunk %d, Score: %.1f%%)**\n",
			result.DocPath, result.ChunkIdx, result.Score*100))
		sb.WriteString(fmt.Sprintf("%s\n\n", result.Text))

		// Stop if context exceeds max length
		if sb.Len() > maxLength {
			sb.WriteString("[... truncated ...]\n")
			break
		}
	}

	return sb.String()
}
