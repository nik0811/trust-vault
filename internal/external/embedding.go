package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Embedder generates vector embeddings for text
type Embedder struct {
	endpoint  string
	apiKey    string
	model     string
	dimension int
	client    *http.Client
}

// EmbeddingRequest is the OpenAI-compatible embedding request
type EmbeddingRequest struct {
	Input          any    `json:"input"`
	Model          string `json:"model"`
	EncodingFormat string `json:"encoding_format,omitempty"`
}

// EmbeddingResponse is the OpenAI-compatible embedding response
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// NewEmbedder creates a new embedding client
// Supports OpenAI, Cohere, and any OpenAI-compatible API (Ollama, vLLM, etc.)
func NewEmbedder(endpoint, apiKey, model string, dimension int) *Embedder {
	if endpoint == "" {
		endpoint = envOr("EMBEDDING_API_URL", "http://localhost:11434/v1")
	}
	if apiKey == "" {
		apiKey = os.Getenv("EMBEDDING_API_KEY")
	}
	if model == "" {
		model = envOr("EMBEDDING_MODEL", "nomic-embed-text")
	}
	if dimension == 0 {
		dimension = envOrInt("EMBEDDING_DIMENSION", 768)
	}

	return &Embedder{
		endpoint:  endpoint,
		apiKey:    apiKey,
		model:     model,
		dimension: dimension,
		client:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Embed generates embeddings for a single text
func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *Embedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body, _ := json.Marshal(EmbeddingRequest{
		Input: texts,
		Model: e.model,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", e.endpoint+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	embeddings := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}

	return embeddings, nil
}

// Dimension returns the configured embedding dimension
func (e *Embedder) Dimension() int {
	return e.dimension
}

// IsHealthy checks if the embedding service is available
func (e *Embedder) IsHealthy(ctx context.Context) bool {
	// Try to get a simple embedding
	_, err := e.Embed(ctx, "test")
	return err == nil
}

func envOrInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return fallback
}
