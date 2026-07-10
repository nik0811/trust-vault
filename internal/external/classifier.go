package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// ClassifierClient is a client for the GLiNER classification service
type ClassifierClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewClassifierClient creates a new classifier client
func NewClassifierClient(baseURL string) *ClassifierClient {
	if baseURL == "" {
		baseURL = "http://localhost:8085"
	}
	return &ClassifierClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ClassifyRequest is the request to the classifier service
type ClassifyRequest struct {
	Text        string   `json:"text"`
	TenantID    string   `json:"tenant_id,omitempty"`
	EntityTypes []string `json:"entity_types,omitempty"`
	Threshold   float64  `json:"threshold,omitempty"`
}

// ClassifyResponse is the response from the classifier service
type ClassifyResponse struct {
	Entities     []ClassifiedEntity `json:"entities"`
	ProcessingMs int64              `json:"processing_ms"`
	ModelUsed    string             `json:"model_used"`
	CharCount    int                `json:"char_count"`
}

// ClassifiedEntity represents a detected entity
type ClassifiedEntity struct {
	Type       string  `json:"type"`
	Value      string  `json:"value"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Confidence float64 `json:"confidence"`
}

// Classify sends text to the classifier service for entity detection
func (c *ClassifierClient) Classify(ctx context.Context, text string, entityTypes []string, threshold float64) (*ClassifyResponse, error) {
	req := ClassifyRequest{
		Text:        text,
		EntityTypes: entityTypes,
		Threshold:   threshold,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/classify", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("classifier returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result ClassifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// BatchClassifyRequest is the request for batch classification
type BatchClassifyRequest struct {
	Items     []ClassifyRequest `json:"items"`
	TenantID  string            `json:"tenant_id,omitempty"`
	Threshold float64           `json:"threshold,omitempty"`
}

// BatchClassifyResponse is the response from batch classification
type BatchClassifyResponse struct {
	Results    []ClassifyResponse `json:"results"`
	TotalMs    int64              `json:"total_ms"`
	ItemCount  int                `json:"item_count"`
	TotalChars int                `json:"total_chars"`
}

// BatchClassify sends multiple texts for classification
func (c *ClassifierClient) BatchClassify(ctx context.Context, items []ClassifyRequest, threshold float64) (*BatchClassifyResponse, error) {
	req := BatchClassifyRequest{
		Items:     items,
		Threshold: threshold,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/classify/batch", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("classifier returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result BatchClassifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// IsHealthy checks if the classifier service is healthy
func (c *ClassifierClient) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Classifier health check failed")
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetInfo returns information about the classifier service
func (c *ClassifierClient) GetInfo(ctx context.Context) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/info", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
