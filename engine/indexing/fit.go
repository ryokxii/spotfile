package indexing

import (
	"strings"
)

// fitToTokens splits text into pieces of at most maxTokens as measured by
// count, keeping words in order. Oversized text is halved by words until each
// half fits; a single word that is still too long is cut short.
func fitToTokens(text string, maxTokens int, count func(string) (int, error)) ([]string, error) {
	n, err := count(text)
	if err != nil {
		return nil, err
	}
	if n <= maxTokens {
		return []string{text}, nil
	}

	words := strings.Fields(text)
	if len(words) <= 1 {
		cut, err := truncateToTokens(text, n, maxTokens, count)
		if err != nil {
			return nil, err
		}
		return []string{cut}, nil
	}

	mid := len(words) / 2
	left, err := fitToTokens(strings.Join(words[:mid], " "), maxTokens, count)
	if err != nil {
		return nil, err
	}
	right, err := fitToTokens(strings.Join(words[mid:], " "), maxTokens, count)
	if err != nil {
		return nil, err
	}
	return append(left, right...), nil
}

// truncateToTokens shortens text (currently n tokens) by runes until it fits.
func truncateToTokens(text string, n, maxTokens int, count func(string) (int, error)) (string, error) {
	runes := []rune(text)
	for n > maxTokens && len(runes) > 1 {
		keep := len(runes) * maxTokens / n
		if keep >= len(runes) {
			keep = len(runes) - 1
		}
		runes = runes[:max(keep, 1)]
		var err error
		if n, err = count(string(runes)); err != nil {
			return "", err
		}
	}
	return string(runes), nil
}
