package engine

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
)

const (
	clsID = 101
	sepID = 102
	unkID = 100
	padID = 0
)

type BertTokenizer struct {
	vocab  map[string]int64
	maxLen int
}

func LoadTokenizer(vocabPath string, maxLen int) (*BertTokenizer, error) {
	f, err := os.Open(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("open vocab: %w", err)
	}
	defer f.Close()

	vocab := make(map[string]int64)
	var id int64
	s := bufio.NewScanner(f)
	for s.Scan() {
		vocab[s.Text()] = id
		id++
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("read vocab: %w", err)
	}
	return &BertTokenizer{vocab: vocab, maxLen: maxLen}, nil
}

type Encoding struct {
	InputIDs      []int64
	AttentionMask []int64
	TokenTypeIDs  []int64
}

func (t *BertTokenizer) Encode(text string) Encoding {
	tokens := t.tokenize(text)

	maxContent := t.maxLen - 2
	if len(tokens) > maxContent {
		tokens = tokens[:maxContent]
	}

	seqLen := len(tokens) + 2
	inputIDs := make([]int64, t.maxLen)
	mask := make([]int64, t.maxLen)
	typeIDs := make([]int64, t.maxLen)

	inputIDs[0] = clsID
	mask[0] = 1
	for i, tok := range tokens {
		inputIDs[i+1] = t.id(tok)
		mask[i+1] = 1
	}
	inputIDs[seqLen-1] = sepID
	mask[seqLen-1] = 1

	return Encoding{InputIDs: inputIDs, AttentionMask: mask, TokenTypeIDs: typeIDs}
}

func (t *BertTokenizer) tokenize(text string) []string {
	words := basicTokenize(text)
	var out []string
	for _, w := range words {
		out = append(out, t.wordpiece(w)...)
	}
	return out
}

func (t *BertTokenizer) wordpiece(word string) []string {
	if _, ok := t.vocab[word]; ok {
		return []string{word}
	}
	runes := []rune(word)
	var tokens []string
	start := 0
	for start < len(runes) {
		end := len(runes)
		var found string
		for start < end {
			sub := string(runes[start:end])
			if start > 0 {
				sub = "##" + sub
			}
			if _, ok := t.vocab[sub]; ok {
				found = sub
				break
			}
			end--
		}
		if found == "" {
			return []string{"[UNK]"}
		}
		tokens = append(tokens, found)
		start = end
	}
	return tokens
}

func (t *BertTokenizer) id(token string) int64 {
	if id, ok := t.vocab[token]; ok {
		return id
	}
	return unkID
}

func basicTokenize(text string) []string {
	text = strings.ToLower(text)
	var words []string
	var cur strings.Builder
	for _, r := range text {
		switch {
		case unicode.IsSpace(r):
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
		case unicode.IsPunct(r):
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
			words = append(words, string(r))
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}
