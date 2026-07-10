package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Qdrant struct {
	baseURL    string
	client     *http.Client
	collection string
	embedder   *Embedder
}

func NewQdrant(baseURL, collection string) *Qdrant {
	if baseURL == "" {
		baseURL = envOr("QDRANT_URL", "http://localhost:6333")
	}
	if collection == "" {
		collection = "trustvault"
	}
	return &Qdrant{
		baseURL:    baseURL,
		client:     &http.Client{Timeout: 30 * time.Second},
		collection: collection,
		embedder:   NewEmbedder("", "", "", 0),
	}
}

// SetEmbedder allows setting a custom embedder
func (q *Qdrant) SetEmbedder(e *Embedder) {
	q.embedder = e
}

// Embedder returns the configured embedder
func (q *Qdrant) Embedder() *Embedder {
	return q.embedder
}

type Point struct {
	ID      string         `json:"id"`
	Vector  []float32      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

type SearchRequest struct {
	Vector      []float32      `json:"vector"`
	Limit       int            `json:"limit"`
	Filter      map[string]any `json:"filter,omitempty"`
	WithPayload bool           `json:"with_payload"`
}

type SearchResult struct {
	ID      string         `json:"id"`
	Score   float32        `json:"score"`
	Payload map[string]any `json:"payload"`
}

func (q *Qdrant) Upsert(ctx context.Context, tenantID string, points []Point) error {
	for i := range points {
		if points[i].ID == "" {
			points[i].ID = uuid.New().String()
		}
		if points[i].Payload == nil {
			points[i].Payload = make(map[string]any)
		}
		points[i].Payload["tenant_id"] = tenantID
	}

	body, _ := json.Marshal(map[string]any{"points": points})
	url := fmt.Sprintf("%s/collections/%s/points", q.baseURL, q.collection)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("qdrant upsert failed: %d", resp.StatusCode)
	}
	return nil
}

func (q *Qdrant) Search(ctx context.Context, tenantID string, vector []float32, limit int) ([]SearchResult, error) {
	body, _ := json.Marshal(SearchRequest{
		Vector:      vector,
		Limit:       limit,
		WithPayload: true,
		Filter: map[string]any{
			"must": []map[string]any{
				{"key": "tenant_id", "match": map[string]any{"value": tenantID}},
			},
		},
	})

	url := fmt.Sprintf("%s/collections/%s/points/search", q.baseURL, q.collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Result []SearchResult `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Result, nil
}

func (q *Qdrant) Delete(ctx context.Context, tenantID string, ids []string) error {
	body, _ := json.Marshal(map[string]any{
		"points": ids,
		"filter": map[string]any{
			"must": []map[string]any{
				{"key": "tenant_id", "match": map[string]any{"value": tenantID}},
			},
		},
	})

	url := fmt.Sprintf("%s/collections/%s/points/delete", q.baseURL, q.collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("qdrant delete failed: %d", resp.StatusCode)
	}
	return nil
}

func (q *Qdrant) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", q.baseURL+"/healthz", nil)
	if err != nil {
		return false
	}
	resp, err := q.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// UpsertText indexes text content by generating embeddings automatically
func (q *Qdrant) UpsertText(ctx context.Context, tenantID string, id string, content string, metadata map[string]any) error {
	embedding, err := q.embedder.Embed(ctx, content)
	if err != nil {
		return fmt.Errorf("generate embedding: %w", err)
	}

	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["content"] = content

	return q.Upsert(ctx, tenantID, []Point{{
		ID:      id,
		Vector:  embedding,
		Payload: metadata,
	}})
}

// SearchText searches for similar content using text query
func (q *Qdrant) SearchText(ctx context.Context, tenantID string, query string, limit int) ([]SearchResult, error) {
	embedding, err := q.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}
	return q.Search(ctx, tenantID, embedding, limit)
}
