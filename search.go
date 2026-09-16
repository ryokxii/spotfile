package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"spotfile/engine/llm"
	"spotfile/engine/vectorstore"
)

// Search embeds query and returns the topK nearest stored chunks.
func (a *App) Search(query string, topK int) ([]vectorstore.SearchResult, error) {
	started := time.Now()
	if a.embedder == nil {
		log.Printf("search: rejected because engine is not initialized")
		return nil, fmt.Errorf("engine not initialised — call InitEngine first")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		log.Printf("search: rejected empty query")
		return nil, fmt.Errorf("search query cannot be empty")
	}
	if topK <= 0 {
		topK = 5
	}
	log.Printf("search: started query=%q topK=%d indexedChunks=%d", query, topK, a.store.Len())
	if a.store.Len() == 0 {
		log.Printf("search: completed results=0 duration=%s (no indexed chunks)", time.Since(started).Round(time.Millisecond))
		return []vectorstore.SearchResult{}, nil
	}

	// Embed the query with the locally loaded BGE ONNX model. No query text or
	// document content leaves this process.
	vec, err := a.embedder.EmbedQuery(a.ctx, query)
	if err != nil {
		log.Printf("search: embedding failed query=%q duration=%s error=%v", query, time.Since(started).Round(time.Millisecond), err)
		return nil, fmt.Errorf("embed local search query: %w", err)
	}
	results := a.store.Search(vec, topK)
	log.Printf("search: completed results=%d duration=%s", len(results), time.Since(started).Round(time.Millisecond))
	for i, result := range results {
		log.Printf(
			"search: result=%d score=%.4f path=%q chunk=%d page=%d content=%q",
			i+1,
			result.Score,
			result.DocPath,
			result.ChunkIdx,
			result.PageNum,
			result.Text,
		)
	}
	return results, nil
}

// GenerateAnswer searches for relevant chunks and uses LLM to generate an answer.
func (a *App) GenerateAnswer(query string, topK int) (string, error) {
	if a.embedder == nil {
		return "", fmt.Errorf("engine not initialised — call InitEngine first")
	}
	if a.llm == nil {
		return "", fmt.Errorf("LLM not initialized")
	}

	results, err := a.Search(query, topK)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}
	if len(results) == 0 {
		return "No relevant documents found to generate an answer from.", nil
	}

	context := llm.FormatContext(results, 2000)
	systemPrompt := "You are a helpful search assistant. Answer based on the provided documents. If information is not in the documents, say so clearly."
	answer, err := a.llm.Generate(a.ctx, systemPrompt, query, context)
	if err != nil {
		return "", fmt.Errorf("generation failed: %w", err)
	}
	return answer, nil
}

// StoreSize returns the number of indexed chunks.
func (a *App) StoreSize() int {
	return a.store.Len()
}
