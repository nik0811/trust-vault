package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// Tokenizer handles text tokenization for the GLiNER model
type Tokenizer struct {
	vocab       map[string]int
	reverseVocab map[int]string
	specialTokens map[string]int
	maxLength   int
}

// TokenizerConfig represents the tokenizer.json structure
type TokenizerConfig struct {
	Model struct {
		Vocab map[string]int `json:"vocab"`
	} `json:"model"`
	AddedTokens []struct {
		ID      int    `json:"id"`
		Content string `json:"content"`
		Special bool   `json:"special"`
	} `json:"added_tokens"`
	Truncation struct {
		MaxLength int `json:"max_length"`
	} `json:"truncation"`
}

// NewTokenizer loads a tokenizer from a JSON file
func NewTokenizer(path string) (*Tokenizer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tokenizer file: %w", err)
	}

	var config TokenizerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse tokenizer config: %w", err)
	}

	t := &Tokenizer{
		vocab:         config.Model.Vocab,
		reverseVocab:  make(map[int]string),
		specialTokens: make(map[string]int),
		maxLength:     512,
	}

	if config.Truncation.MaxLength > 0 {
		t.maxLength = config.Truncation.MaxLength
	}

	// Build reverse vocab
	for token, id := range t.vocab {
		t.reverseVocab[id] = token
	}

	// Add special tokens
	for _, st := range config.AddedTokens {
		if st.Special {
			t.specialTokens[st.Content] = st.ID
		}
	}

	return t, nil
}

// Encode converts text to token IDs
func (t *Tokenizer) Encode(text string) ([]int, error) {
	if t.vocab == nil || len(t.vocab) == 0 {
		// Fallback: simple whitespace tokenization with character-level fallback
		return t.simpleEncode(text), nil
	}

	tokens := t.tokenize(text)
	ids := make([]int, 0, len(tokens)+2)

	// Add [CLS] token if exists
	if clsID, ok := t.specialTokens["[CLS]"]; ok {
		ids = append(ids, clsID)
	}

	for _, token := range tokens {
		if id, ok := t.vocab[token]; ok {
			ids = append(ids, id)
		} else if id, ok := t.vocab[strings.ToLower(token)]; ok {
			ids = append(ids, id)
		} else {
			// Unknown token - use [UNK] or skip
			if unkID, ok := t.specialTokens["[UNK]"]; ok {
				ids = append(ids, unkID)
			}
		}
	}

	// Add [SEP] token if exists
	if sepID, ok := t.specialTokens["[SEP]"]; ok {
		ids = append(ids, sepID)
	}

	// Truncate if necessary
	if len(ids) > t.maxLength {
		ids = ids[:t.maxLength]
	}

	return ids, nil
}

// tokenize splits text into tokens using WordPiece-like algorithm
func (t *Tokenizer) tokenize(text string) []string {
	tokens := make([]string, 0)
	words := strings.Fields(text)

	for _, word := range words {
		subTokens := t.wordPieceTokenize(word)
		tokens = append(tokens, subTokens...)
	}

	return tokens
}

// wordPieceTokenize applies WordPiece tokenization to a single word
func (t *Tokenizer) wordPieceTokenize(word string) []string {
	if len(word) == 0 {
		return nil
	}

	tokens := make([]string, 0)
	start := 0
	wordLen := utf8.RuneCountInString(word)

	for start < wordLen {
		end := wordLen
		found := false

		for end > start {
			substr := string([]rune(word)[start:end])
			if start > 0 {
				substr = "##" + substr
			}

			if _, ok := t.vocab[substr]; ok {
				tokens = append(tokens, substr)
				found = true
				break
			}
			end--
		}

		if !found {
			// Character not in vocab, add as unknown
			tokens = append(tokens, "[UNK]")
			start++
		} else {
			start = end
		}
	}

	return tokens
}

// simpleEncode provides a fallback encoding when vocab is not available
func (t *Tokenizer) simpleEncode(text string) []int {
	// Simple character-level encoding
	ids := make([]int, 0, len(text))
	for _, r := range text {
		ids = append(ids, int(r))
	}
	if len(ids) > t.maxLength {
		ids = ids[:t.maxLength]
	}
	return ids
}

// Decode converts token IDs back to text
func (t *Tokenizer) Decode(ids []int) string {
	tokens := make([]string, 0, len(ids))
	for _, id := range ids {
		if token, ok := t.reverseVocab[id]; ok {
			// Skip special tokens
			if _, isSpecial := t.specialTokens[token]; !isSpecial {
				tokens = append(tokens, token)
			}
		}
	}

	// Join tokens, handling WordPiece continuation markers
	var result strings.Builder
	for i, token := range tokens {
		if strings.HasPrefix(token, "##") {
			result.WriteString(token[2:])
		} else {
			if i > 0 {
				result.WriteString(" ")
			}
			result.WriteString(token)
		}
	}

	return result.String()
}
