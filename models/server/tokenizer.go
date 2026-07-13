package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Tokenizer handles text tokenization for the GLiNER model
type Tokenizer struct {
	vocab         map[string]int
	reverseVocab  map[int]string
	specialTokens map[string]int
	maxLength     int
	unkTokenID    int
	padTokenID    int
	clsTokenID    int
	sepTokenID    int
}

// TokenizerConfig represents the tokenizer.json structure
type TokenizerConfig struct {
	Model struct {
		Vocab map[string]int `json:"vocab"`
		Type  string         `json:"type"`
	} `json:"model"`
	AddedTokens []struct {
		ID      int    `json:"id"`
		Content string `json:"content"`
		Special bool   `json:"special"`
	} `json:"added_tokens"`
	Truncation struct {
		MaxLength int `json:"max_length"`
	} `json:"truncation"`
	PreTokenizer struct {
		Type string `json:"type"`
	} `json:"pre_tokenizer"`
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
		unkTokenID:    0,
		padTokenID:    0,
		clsTokenID:    1,
		sepTokenID:    2,
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
		t.specialTokens[st.Content] = st.ID
		t.reverseVocab[st.ID] = st.Content
		
		// Identify common special tokens
		switch st.Content {
		case "[UNK]", "<unk>":
			t.unkTokenID = st.ID
		case "[PAD]", "<pad>":
			t.padTokenID = st.ID
		case "[CLS]", "<s>":
			t.clsTokenID = st.ID
		case "[SEP]", "</s>":
			t.sepTokenID = st.ID
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
	ids := make([]int, 0, len(tokens))

	for _, token := range tokens {
		if id, ok := t.vocab[token]; ok {
			ids = append(ids, id)
		} else if id, ok := t.vocab[strings.ToLower(token)]; ok {
			ids = append(ids, id)
		} else {
			// Try subword tokenization
			subIDs := t.tokenizeSubword(token)
			if len(subIDs) > 0 {
				ids = append(ids, subIDs...)
			} else {
				// Unknown token
				ids = append(ids, t.unkTokenID)
			}
		}
	}

	// Truncate if necessary
	if len(ids) > t.maxLength {
		ids = ids[:t.maxLength]
	}

	return ids, nil
}

// EncodeWithSpecialTokens adds CLS and SEP tokens
func (t *Tokenizer) EncodeWithSpecialTokens(text string) ([]int, error) {
	ids, err := t.Encode(text)
	if err != nil {
		return nil, err
	}

	// Add [CLS] at start and [SEP] at end
	result := make([]int, 0, len(ids)+2)
	result = append(result, t.clsTokenID)
	result = append(result, ids...)
	result = append(result, t.sepTokenID)

	return result, nil
}

// tokenize splits text into tokens
func (t *Tokenizer) tokenize(text string) []string {
	tokens := make([]string, 0)
	words := strings.Fields(text)

	for _, word := range words {
		// Check if word is in vocab directly
		if _, ok := t.vocab[word]; ok {
			tokens = append(tokens, word)
			continue
		}
		if _, ok := t.vocab[strings.ToLower(word)]; ok {
			tokens = append(tokens, strings.ToLower(word))
			continue
		}

		// Apply WordPiece/BPE tokenization
		subTokens := t.wordPieceTokenize(word)
		tokens = append(tokens, subTokens...)
	}

	return tokens
}

// tokenizeSubword attempts to break a word into subwords
func (t *Tokenizer) tokenizeSubword(word string) []int {
	ids := make([]int, 0)
	remaining := word

	for len(remaining) > 0 {
		found := false
		for end := len(remaining); end > 0; end-- {
			subword := remaining[:end]
			if len(ids) > 0 {
				subword = "##" + subword
			}

			if id, ok := t.vocab[subword]; ok {
				ids = append(ids, id)
				remaining = remaining[end:]
				found = true
				break
			}
			if id, ok := t.vocab[strings.ToLower(subword)]; ok {
				ids = append(ids, id)
				remaining = remaining[end:]
				found = true
				break
			}
		}

		if !found {
			// Skip one character
			remaining = remaining[1:]
			if len(remaining) == 0 {
				ids = append(ids, t.unkTokenID)
			}
		}
	}

	return ids
}

// wordPieceTokenize applies WordPiece tokenization to a single word
func (t *Tokenizer) wordPieceTokenize(word string) []string {
	if len(word) == 0 {
		return nil
	}

	tokens := make([]string, 0)
	start := 0
	wordRunes := []rune(word)
	wordLen := len(wordRunes)

	for start < wordLen {
		end := wordLen
		found := false

		for end > start {
			substr := string(wordRunes[start:end])
			if start > 0 {
				substr = "##" + substr
			}

			if _, ok := t.vocab[substr]; ok {
				tokens = append(tokens, substr)
				found = true
				break
			}
			if _, ok := t.vocab[strings.ToLower(substr)]; ok {
				tokens = append(tokens, strings.ToLower(substr))
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

// GetVocabSize returns the vocabulary size
func (t *Tokenizer) GetVocabSize() int {
	return len(t.vocab)
}

// GetSpecialTokenID returns the ID for a special token
func (t *Tokenizer) GetSpecialTokenID(token string) (int, bool) {
	id, ok := t.specialTokens[token]
	return id, ok
}

// TokenizeWords tokenizes text and returns word boundaries
func (t *Tokenizer) TokenizeWords(text string) ([]int, []int, []string) {
	words := strings.Fields(text)
	var allIDs []int
	var wordsMask []int
	
	for _, word := range words {
		wordTokens := t.wordPieceTokenize(word)
		for i, token := range wordTokens {
			if id, ok := t.vocab[token]; ok {
				allIDs = append(allIDs, id)
			} else {
				allIDs = append(allIDs, t.unkTokenID)
			}
			
			// First subword of each word gets mask = 1
			if i == 0 {
				wordsMask = append(wordsMask, 1)
			} else {
				wordsMask = append(wordsMask, 0)
			}
		}
	}
	
	return allIDs, wordsMask, words
}

// PadSequence pads a sequence to the specified length
func (t *Tokenizer) PadSequence(ids []int, length int) []int {
	if len(ids) >= length {
		return ids[:length]
	}
	
	padded := make([]int, length)
	copy(padded, ids)
	for i := len(ids); i < length; i++ {
		padded[i] = t.padTokenID
	}
	return padded
}

// CreateAttentionMask creates an attention mask for the given IDs
func (t *Tokenizer) CreateAttentionMask(ids []int, paddedLength int) []int {
	mask := make([]int, paddedLength)
	for i := 0; i < len(ids) && i < paddedLength; i++ {
		if ids[i] != t.padTokenID {
			mask[i] = 1
		}
	}
	return mask
}
