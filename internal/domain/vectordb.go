package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// VectorDB defines the interface for vector database operations
type VectorDB interface {
	Search(ctx context.Context, tenantID string, vector []float32, topK int, filters map[string]any) ([]VectorSearchResult, error)
	Upsert(ctx context.Context, tenantID string, documents []VectorDocument) error
	Delete(ctx context.Context, tenantID string, ids []string) error
	IsHealthy(ctx context.Context) bool
}

// VectorDocument represents a document to be stored in the vector DB
type VectorDocument struct {
	ID         string         `json:"id"`
	Content    string         `json:"content"`
	Vector     []float32      `json:"vector,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Source     string         `json:"source,omitempty"`
	Sensitivity string        `json:"sensitivity,omitempty"`
}

// VectorSearchResult represents a search result from the vector DB
type VectorSearchResult struct {
	ID          string         `json:"id"`
	Content     string         `json:"content"`
	Score       float32        `json:"score"`
	Source      string         `json:"source"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Sensitivity string         `json:"sensitivity,omitempty"`
}

// VectorDBConfig holds configuration for vector DB connections
type VectorDBConfig struct {
	Type       string `json:"type"` // qdrant, pinecone, weaviate, chroma, custom
	URL        string `json:"url"`
	APIKey     string `json:"api_key,omitempty"`
	Collection string `json:"collection,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Index      string `json:"index,omitempty"`
	Dimension  int    `json:"dimension,omitempty"`
}

// NewVectorDB creates a vector DB adapter based on configuration
func NewVectorDB(cfg VectorDBConfig) (VectorDB, error) {
	switch strings.ToLower(cfg.Type) {
	case "qdrant":
		return NewQdrantAdapter(cfg), nil
	case "pinecone":
		return NewPineconeAdapter(cfg), nil
	case "weaviate":
		return NewWeaviateAdapter(cfg), nil
	case "chroma":
		return NewChromaAdapter(cfg), nil
	case "custom", "http":
		return NewCustomHTTPAdapter(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported vector DB type: %s", cfg.Type)
	}
}

// BaseAdapter provides common HTTP client functionality
type BaseAdapter struct {
	client     *http.Client
	url        string
	apiKey     string
	collection string
}

func newBaseAdapter(url, apiKey, collection string) BaseAdapter {
	return BaseAdapter{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		url:        strings.TrimSuffix(url, "/"),
		apiKey:     apiKey,
		collection: collection,
	}
}

func (b *BaseAdapter) doRequest(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, b.url+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if b.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.apiKey)
		req.Header.Set("Api-Key", b.apiKey)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// QdrantAdapter implements VectorDB for Qdrant
type QdrantAdapter struct {
	BaseAdapter
}

func NewQdrantAdapter(cfg VectorDBConfig) *QdrantAdapter {
	url := cfg.URL
	if url == "" {
		url = os.Getenv("QDRANT_URL")
		if url == "" {
			url = "http://localhost:6333"
		}
	}
	collection := cfg.Collection
	if collection == "" {
		collection = "securelens"
	}
	return &QdrantAdapter{
		BaseAdapter: newBaseAdapter(url, cfg.APIKey, collection),
	}
}

func (q *QdrantAdapter) Search(ctx context.Context, tenantID string, vector []float32, topK int, filters map[string]any) ([]VectorSearchResult, error) {
	filter := map[string]any{
		"must": []map[string]any{
			{"key": "tenant_id", "match": map[string]any{"value": tenantID}},
		},
	}

	if filters != nil {
		for k, v := range filters {
			filter["must"] = append(filter["must"].([]map[string]any), map[string]any{
				"key": k, "match": map[string]any{"value": v},
			})
		}
	}

	reqBody := map[string]any{
		"vector":       vector,
		"limit":        topK,
		"with_payload": true,
		"filter":       filter,
	}

	var resp struct {
		Result []struct {
			ID      string         `json:"id"`
			Score   float32        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}

	path := fmt.Sprintf("/collections/%s/points/search", q.collection)
	if err := q.doRequest(ctx, "POST", path, reqBody, &resp); err != nil {
		return nil, err
	}

	results := make([]VectorSearchResult, 0, len(resp.Result))
	for _, r := range resp.Result {
		content, _ := r.Payload["content"].(string)
		source, _ := r.Payload["source"].(string)
		sensitivity, _ := r.Payload["sensitivity"].(string)
		results = append(results, VectorSearchResult{
			ID:          r.ID,
			Content:     content,
			Score:       r.Score,
			Source:      source,
			Metadata:    r.Payload,
			Sensitivity: sensitivity,
		})
	}

	return results, nil
}

func (q *QdrantAdapter) Upsert(ctx context.Context, tenantID string, documents []VectorDocument) error {
	points := make([]map[string]any, 0, len(documents))
	for _, doc := range documents {
		payload := doc.Metadata
		if payload == nil {
			payload = make(map[string]any)
		}
		payload["tenant_id"] = tenantID
		payload["content"] = doc.Content
		payload["source"] = doc.Source
		payload["sensitivity"] = doc.Sensitivity

		points = append(points, map[string]any{
			"id":      doc.ID,
			"vector":  doc.Vector,
			"payload": payload,
		})
	}

	path := fmt.Sprintf("/collections/%s/points", q.collection)
	return q.doRequest(ctx, "PUT", path, map[string]any{"points": points}, nil)
}

func (q *QdrantAdapter) Delete(ctx context.Context, tenantID string, ids []string) error {
	reqBody := map[string]any{
		"points": ids,
		"filter": map[string]any{
			"must": []map[string]any{
				{"key": "tenant_id", "match": map[string]any{"value": tenantID}},
			},
		},
	}

	path := fmt.Sprintf("/collections/%s/points/delete", q.collection)
	return q.doRequest(ctx, "POST", path, reqBody, nil)
}

func (q *QdrantAdapter) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", q.url+"/healthz", nil)
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

// PineconeAdapter implements VectorDB for Pinecone
type PineconeAdapter struct {
	BaseAdapter
	namespace string
}

func NewPineconeAdapter(cfg VectorDBConfig) *PineconeAdapter {
	url := cfg.URL
	if url == "" {
		url = os.Getenv("PINECONE_URL")
	}
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("PINECONE_API_KEY")
	}
	namespace := cfg.Namespace
	if namespace == "" {
		namespace = "default"
	}
	return &PineconeAdapter{
		BaseAdapter: newBaseAdapter(url, apiKey, cfg.Index),
		namespace:   namespace,
	}
}

func (p *PineconeAdapter) Search(ctx context.Context, tenantID string, vector []float32, topK int, filters map[string]any) ([]VectorSearchResult, error) {
	filter := map[string]any{
		"tenant_id": map[string]any{"$eq": tenantID},
	}
	if filters != nil {
		for k, v := range filters {
			filter[k] = map[string]any{"$eq": v}
		}
	}

	reqBody := map[string]any{
		"vector":          vector,
		"topK":            topK,
		"includeMetadata": true,
		"namespace":       p.namespace,
		"filter":          filter,
	}

	var resp struct {
		Matches []struct {
			ID       string         `json:"id"`
			Score    float32        `json:"score"`
			Metadata map[string]any `json:"metadata"`
		} `json:"matches"`
	}

	if err := p.doRequest(ctx, "POST", "/query", reqBody, &resp); err != nil {
		return nil, err
	}

	results := make([]VectorSearchResult, 0, len(resp.Matches))
	for _, m := range resp.Matches {
		content, _ := m.Metadata["content"].(string)
		source, _ := m.Metadata["source"].(string)
		sensitivity, _ := m.Metadata["sensitivity"].(string)
		results = append(results, VectorSearchResult{
			ID:          m.ID,
			Content:     content,
			Score:       m.Score,
			Source:      source,
			Metadata:    m.Metadata,
			Sensitivity: sensitivity,
		})
	}

	return results, nil
}

func (p *PineconeAdapter) Upsert(ctx context.Context, tenantID string, documents []VectorDocument) error {
	vectors := make([]map[string]any, 0, len(documents))
	for _, doc := range documents {
		metadata := doc.Metadata
		if metadata == nil {
			metadata = make(map[string]any)
		}
		metadata["tenant_id"] = tenantID
		metadata["content"] = doc.Content
		metadata["source"] = doc.Source
		metadata["sensitivity"] = doc.Sensitivity

		vectors = append(vectors, map[string]any{
			"id":       doc.ID,
			"values":   doc.Vector,
			"metadata": metadata,
		})
	}

	reqBody := map[string]any{
		"vectors":   vectors,
		"namespace": p.namespace,
	}

	return p.doRequest(ctx, "POST", "/vectors/upsert", reqBody, nil)
}

func (p *PineconeAdapter) Delete(ctx context.Context, tenantID string, ids []string) error {
	reqBody := map[string]any{
		"ids":       ids,
		"namespace": p.namespace,
		"filter": map[string]any{
			"tenant_id": map[string]any{"$eq": tenantID},
		},
	}

	return p.doRequest(ctx, "POST", "/vectors/delete", reqBody, nil)
}

func (p *PineconeAdapter) IsHealthy(ctx context.Context) bool {
	var resp struct {
		Status string `json:"status"`
	}
	if err := p.doRequest(ctx, "GET", "/describe_index_stats", nil, &resp); err != nil {
		return false
	}
	return true
}

// WeaviateAdapter implements VectorDB for Weaviate
type WeaviateAdapter struct {
	BaseAdapter
	className string
}

func NewWeaviateAdapter(cfg VectorDBConfig) *WeaviateAdapter {
	url := cfg.URL
	if url == "" {
		url = os.Getenv("WEAVIATE_URL")
		if url == "" {
			url = "http://localhost:8080"
		}
	}
	className := cfg.Collection
	if className == "" {
		className = "Document"
	}
	return &WeaviateAdapter{
		BaseAdapter: newBaseAdapter(url, cfg.APIKey, className),
		className:   className,
	}
}

func (w *WeaviateAdapter) Search(ctx context.Context, tenantID string, vector []float32, topK int, filters map[string]any) ([]VectorSearchResult, error) {
	whereFilter := map[string]any{
		"path":     []string{"tenant_id"},
		"operator": "Equal",
		"valueText": tenantID,
	}

	query := fmt.Sprintf(`{
		Get {
			%s(
				nearVector: {vector: %v}
				limit: %d
				where: %s
			) {
				_additional {
					id
					distance
				}
				content
				source
				sensitivity
				tenant_id
			}
		}
	}`, w.className, vector, topK, mustJSON(whereFilter))

	var resp struct {
		Data struct {
			Get map[string][]struct {
				Additional struct {
					ID       string  `json:"id"`
					Distance float32 `json:"distance"`
				} `json:"_additional"`
				Content     string `json:"content"`
				Source      string `json:"source"`
				Sensitivity string `json:"sensitivity"`
			} `json:"Document"`
		} `json:"data"`
	}

	if err := w.doRequest(ctx, "POST", "/v1/graphql", map[string]string{"query": query}, &resp); err != nil {
		return nil, err
	}

	results := make([]VectorSearchResult, 0)
	if docs, ok := resp.Data.Get[w.className]; ok {
		for _, doc := range docs {
			results = append(results, VectorSearchResult{
				ID:          doc.Additional.ID,
				Content:     doc.Content,
				Score:       1 - doc.Additional.Distance,
				Source:      doc.Source,
				Sensitivity: doc.Sensitivity,
			})
		}
	}

	return results, nil
}

func (w *WeaviateAdapter) Upsert(ctx context.Context, tenantID string, documents []VectorDocument) error {
	for _, doc := range documents {
		obj := map[string]any{
			"class": w.className,
			"id":    doc.ID,
			"properties": map[string]any{
				"content":     doc.Content,
				"source":      doc.Source,
				"sensitivity": doc.Sensitivity,
				"tenant_id":   tenantID,
			},
			"vector": doc.Vector,
		}

		if err := w.doRequest(ctx, "POST", "/v1/objects", obj, nil); err != nil {
			log.Warn().Err(err).Str("id", doc.ID).Msg("Failed to upsert document to Weaviate")
		}
	}
	return nil
}

func (w *WeaviateAdapter) Delete(ctx context.Context, tenantID string, ids []string) error {
	for _, id := range ids {
		path := fmt.Sprintf("/v1/objects/%s/%s", w.className, id)
		if err := w.doRequest(ctx, "DELETE", path, nil, nil); err != nil {
			log.Warn().Err(err).Str("id", id).Msg("Failed to delete document from Weaviate")
		}
	}
	return nil
}

func (w *WeaviateAdapter) IsHealthy(ctx context.Context) bool {
	var resp struct {
		Status string `json:"status"`
	}
	if err := w.doRequest(ctx, "GET", "/v1/.well-known/ready", nil, &resp); err != nil {
		return false
	}
	return true
}

// ChromaAdapter implements VectorDB for Chroma
type ChromaAdapter struct {
	BaseAdapter
}

func NewChromaAdapter(cfg VectorDBConfig) *ChromaAdapter {
	url := cfg.URL
	if url == "" {
		url = os.Getenv("CHROMA_URL")
		if url == "" {
			url = "http://localhost:8000"
		}
	}
	collection := cfg.Collection
	if collection == "" {
		collection = "securelens"
	}
	return &ChromaAdapter{
		BaseAdapter: newBaseAdapter(url, cfg.APIKey, collection),
	}
}

func (c *ChromaAdapter) Search(ctx context.Context, tenantID string, vector []float32, topK int, filters map[string]any) ([]VectorSearchResult, error) {
	where := map[string]any{
		"tenant_id": tenantID,
	}
	if filters != nil {
		for k, v := range filters {
			where[k] = v
		}
	}

	reqBody := map[string]any{
		"query_embeddings": [][]float32{vector},
		"n_results":        topK,
		"where":            where,
		"include":          []string{"documents", "metadatas", "distances"},
	}

	var resp struct {
		IDs       [][]string           `json:"ids"`
		Documents [][]string           `json:"documents"`
		Metadatas [][]map[string]any   `json:"metadatas"`
		Distances [][]float32          `json:"distances"`
	}

	path := fmt.Sprintf("/api/v1/collections/%s/query", c.collection)
	if err := c.doRequest(ctx, "POST", path, reqBody, &resp); err != nil {
		return nil, err
	}

	results := make([]VectorSearchResult, 0)
	if len(resp.IDs) > 0 {
		for i, id := range resp.IDs[0] {
			var content, source, sensitivity string
			var metadata map[string]any

			if len(resp.Documents) > 0 && len(resp.Documents[0]) > i {
				content = resp.Documents[0][i]
			}
			if len(resp.Metadatas) > 0 && len(resp.Metadatas[0]) > i {
				metadata = resp.Metadatas[0][i]
				source, _ = metadata["source"].(string)
				sensitivity, _ = metadata["sensitivity"].(string)
			}

			var score float32 = 1.0
			if len(resp.Distances) > 0 && len(resp.Distances[0]) > i {
				score = 1 - resp.Distances[0][i]
			}

			results = append(results, VectorSearchResult{
				ID:          id,
				Content:     content,
				Score:       score,
				Source:      source,
				Metadata:    metadata,
				Sensitivity: sensitivity,
			})
		}
	}

	return results, nil
}

func (c *ChromaAdapter) Upsert(ctx context.Context, tenantID string, documents []VectorDocument) error {
	ids := make([]string, 0, len(documents))
	embeddings := make([][]float32, 0, len(documents))
	docs := make([]string, 0, len(documents))
	metadatas := make([]map[string]any, 0, len(documents))

	for _, doc := range documents {
		ids = append(ids, doc.ID)
		embeddings = append(embeddings, doc.Vector)
		docs = append(docs, doc.Content)

		metadata := doc.Metadata
		if metadata == nil {
			metadata = make(map[string]any)
		}
		metadata["tenant_id"] = tenantID
		metadata["source"] = doc.Source
		metadata["sensitivity"] = doc.Sensitivity
		metadatas = append(metadatas, metadata)
	}

	reqBody := map[string]any{
		"ids":        ids,
		"embeddings": embeddings,
		"documents":  docs,
		"metadatas":  metadatas,
	}

	path := fmt.Sprintf("/api/v1/collections/%s/upsert", c.collection)
	return c.doRequest(ctx, "POST", path, reqBody, nil)
}

func (c *ChromaAdapter) Delete(ctx context.Context, tenantID string, ids []string) error {
	reqBody := map[string]any{
		"ids": ids,
		"where": map[string]any{
			"tenant_id": tenantID,
		},
	}

	path := fmt.Sprintf("/api/v1/collections/%s/delete", c.collection)
	return c.doRequest(ctx, "POST", path, reqBody, nil)
}

func (c *ChromaAdapter) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.url+"/api/v1/heartbeat", nil)
	if err != nil {
		return false
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// CustomHTTPAdapter implements VectorDB for custom HTTP endpoints
type CustomHTTPAdapter struct {
	BaseAdapter
}

func NewCustomHTTPAdapter(cfg VectorDBConfig) *CustomHTTPAdapter {
	return &CustomHTTPAdapter{
		BaseAdapter: newBaseAdapter(cfg.URL, cfg.APIKey, cfg.Collection),
	}
}

func (c *CustomHTTPAdapter) Search(ctx context.Context, tenantID string, vector []float32, topK int, filters map[string]any) ([]VectorSearchResult, error) {
	reqBody := map[string]any{
		"tenant_id": tenantID,
		"vector":    vector,
		"top_k":     topK,
		"filters":   filters,
	}

	var resp struct {
		Results []VectorSearchResult `json:"results"`
	}

	if err := c.doRequest(ctx, "POST", "/search", reqBody, &resp); err != nil {
		return nil, err
	}

	return resp.Results, nil
}

func (c *CustomHTTPAdapter) Upsert(ctx context.Context, tenantID string, documents []VectorDocument) error {
	reqBody := map[string]any{
		"tenant_id": tenantID,
		"documents": documents,
	}

	return c.doRequest(ctx, "POST", "/upsert", reqBody, nil)
}

func (c *CustomHTTPAdapter) Delete(ctx context.Context, tenantID string, ids []string) error {
	reqBody := map[string]any{
		"tenant_id": tenantID,
		"ids":       ids,
	}

	return c.doRequest(ctx, "POST", "/delete", reqBody, nil)
}

func (c *CustomHTTPAdapter) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.url+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
