package indexing

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// wordCount treats each whitespace-separated word as one token, and each rune
// as a token for a single oversized "word", so splitting is easy to reason about.
func wordCount(text string) (int, error) {
	words := strings.Fields(text)
	if len(words) == 1 {
		return utf8.RuneCountInString(words[0]), nil
	}
	return len(words), nil
}

func words(n int) string {
	w := make([]string, n)
	for i := range w {
		w[i] = "w"
	}
	return strings.Join(w, " ")
}

func TestFitToTokens(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		max       int
		wantParts int
	}{
		{"fits unchanged", words(10), 10, 1},
		{"splits once", words(20), 10, 2},
		{"splits recursively", words(35), 10, 4},
		{"single oversized word is truncated", strings.Repeat("x", 50), 10, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, err := fitToTokens(tt.text, tt.max, wordCount)
			if err != nil {
				t.Fatalf("fitToTokens: %v", err)
			}
			if len(parts) != tt.wantParts {
				t.Fatalf("got %d parts, want %d: %q", len(parts), tt.wantParts, parts)
			}
			for i, p := range parts {
				n, _ := wordCount(p)
				if n > tt.max || strings.TrimSpace(p) == "" {
					t.Fatalf("part %d has %d tokens (max %d) or is empty: %q", i, n, tt.max, p)
				}
			}
		})
	}
}

func TestFitToTokensKeepsWordsInOrder(t *testing.T) {
	text := "a b c d e f g h i j k l m n o p"
	parts, err := fitToTokens(text, 5, wordCount)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(parts, " "); got != text {
		t.Fatalf("rejoined parts = %q, want %q", got, text)
	}
}
