package engine

import "strings"

// Chunk splits text into overlapping word windows. maxWords is the window size;
// overlap is the number of words shared between consecutive chunks.
func Chunk(text string, maxWords, overlap int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	if len(words) <= maxWords {
		return []string{text}
	}

	step := maxWords - overlap
	if step < 1 {
		step = 1
	}

	var chunks []string
	for i := 0; i < len(words); i += step {
		end := i + maxWords
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[i:end], " "))
		if end == len(words) {
			break
		}
	}
	return chunks
}
