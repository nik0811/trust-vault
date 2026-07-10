package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestIntegration_Classification(t *testing.T) {
	_, _, token := setupTestData(t)

	// Classify text
	classifyResp := makeRequest(t, "POST", "/api/v1/classify/text", map[string]any{
		"text": "Contact john@example.com or call 555-123-4567",
	}, token)

	if classifyResp.Code != http.StatusOK {
		t.Errorf("Classify text failed: %d - %s", classifyResp.Code, classifyResp.Body.String())
	}

	var result map[string]any
	json.Unmarshal(classifyResp.Body.Bytes(), &result)
	entities := result["entities"].([]any)
	if len(entities) == 0 {
		t.Error("Expected entities to be detected")
	}
}

func TestIntegration_ClassifyDataset(t *testing.T) {
	_, _, token := setupTestData(t)

	resp := makeRequest(t, "POST", "/api/v1/classify/dataset", map[string]any{
		"dataset_id": "test-dataset-123",
		"async":      true,
	}, token)

	if resp.Code != http.StatusOK {
		t.Errorf("Classify dataset failed: %d", resp.Code)
	}

	var result map[string]any
	json.Unmarshal(resp.Body.Bytes(), &result)
	if result["status"] != "queued" {
		t.Errorf("Expected status=queued, got %v", result["status"])
	}
}

func TestIntegration_ListModels(t *testing.T) {
	_, _, token := setupTestData(t)

	resp := makeRequest(t, "GET", "/api/v1/classify/models", nil, token)

	if resp.Code != http.StatusOK {
		t.Errorf("List models failed: %d", resp.Code)
	}

	var models []map[string]any
	json.Unmarshal(resp.Body.Bytes(), &models)
	if len(models) == 0 {
		t.Error("Expected at least one model")
	}
}

func TestIntegration_Gate_Query(t *testing.T) {
	_, _, token := setupTestData(t)

	resp := makeRequest(t, "POST", "/api/v1/gate/query", map[string]any{
		"query":     "What is the company policy?",
		"max_chunks": 3,
	}, token)

	if resp.Code != http.StatusOK {
		t.Errorf("Gate query failed: %d - %s", resp.Code, resp.Body.String())
	}

	var result map[string]any
	json.Unmarshal(resp.Body.Bytes(), &result)
	if result["decision"] == nil {
		t.Error("Expected decision in response")
	}
}

func TestIntegration_Gate_Retrieve(t *testing.T) {
	_, _, token := setupTestData(t)

	resp := makeRequest(t, "POST", "/api/v1/gate/retrieve", map[string]any{
		"query":     "test query",
		"max_chunks": 5,
	}, token)

	if resp.Code != http.StatusOK {
		t.Errorf("Gate retrieve failed: %d", resp.Code)
	}
}

func TestIntegration_Gate_Validate(t *testing.T) {
	_, _, token := setupTestData(t)

	resp := makeRequest(t, "POST", "/api/v1/gate/validate", map[string]any{
		"response": "This is a test LLM response",
	}, token)

	if resp.Code != http.StatusOK {
		t.Errorf("Gate validate failed: %d", resp.Code)
	}
}

func TestIntegration_Gate_Stats(t *testing.T) {
	_, _, token := setupTestData(t)

	resp := makeRequest(t, "GET", "/api/v1/gate/stats", nil, token)

	if resp.Code != http.StatusOK {
		t.Errorf("Gate stats failed: %d", resp.Code)
	}

	var stats map[string]any
	json.Unmarshal(resp.Body.Bytes(), &stats)
	if stats["total_queries"] == nil {
		t.Error("Expected total_queries in stats")
	}
}
