package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestIntegration_Quality(t *testing.T) {
	_, _, token := setupTestData(t)

	// Get quality score (should return default for non-existent)
	resp := makeRequest(t, "GET", "/api/v1/quality/datasets/test-dataset", nil, token)
	if resp.Code != http.StatusOK {
		t.Errorf("Get quality failed: %d", resp.Code)
	}

	// Assess quality
	assessResp := makeRequest(t, "POST", "/api/v1/quality/assess", map[string]any{
		"dataset_id": "test-dataset",
	}, token)
	if assessResp.Code != http.StatusOK {
		t.Errorf("Assess quality failed: %d", assessResp.Code)
	}

	// Get trends
	trendsResp := makeRequest(t, "GET", "/api/v1/quality/trends", nil, token)
	if trendsResp.Code != http.StatusOK {
		t.Errorf("Get trends failed: %d", trendsResp.Code)
	}

	// Set thresholds
	thresholdResp := makeRequest(t, "POST", "/api/v1/quality/thresholds", map[string]any{
		"dimension": "completeness",
		"minimum":   0.8,
		"severity":  "high",
	}, token)
	if thresholdResp.Code != http.StatusOK {
		t.Errorf("Set thresholds failed: %d", thresholdResp.Code)
	}
}

func TestIntegration_Privacy_DSAR(t *testing.T) {
	_, _, token := setupTestData(t)

	// Create DSAR
	createResp := makeRequest(t, "POST", "/api/v1/privacy/dsar", map[string]any{
		"subject_id": "user-123",
		"type":       "access",
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("Create DSAR failed: %d - %s", createResp.Code, createResp.Body.String())
	}

	var created map[string]any
	json.Unmarshal(createResp.Body.Bytes(), &created)
	dsarID := created["id"].(string)

	// List DSARs
	listResp := makeRequest(t, "GET", "/api/v1/privacy/dsar", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List DSARs failed: %d", listResp.Code)
	}

	// Get DSAR
	getResp := makeRequest(t, "GET", "/api/v1/privacy/dsar/"+dsarID, nil, token)
	if getResp.Code != http.StatusOK {
		t.Errorf("Get DSAR failed: %d", getResp.Code)
	}

	// Get DSAR package
	packageResp := makeRequest(t, "GET", "/api/v1/privacy/dsar/"+dsarID+"/package", nil, token)
	if packageResp.Code != http.StatusOK {
		t.Errorf("Get DSAR package failed: %d", packageResp.Code)
	}
}

func TestIntegration_Privacy_PIA(t *testing.T) {
	_, _, token := setupTestData(t)

	// Generate PIA
	genResp := makeRequest(t, "POST", "/api/v1/privacy/pia", map[string]any{
		"dataset_id": "test-dataset",
	}, token)
	if genResp.Code != http.StatusOK {
		t.Errorf("Generate PIA failed: %d", genResp.Code)
	}

	var pia map[string]any
	json.Unmarshal(genResp.Body.Bytes(), &pia)
	if pia["risk_score"] == nil {
		t.Error("Expected risk_score in PIA")
	}

	// Get PIA
	getResp := makeRequest(t, "GET", "/api/v1/privacy/pia/test-dataset", nil, token)
	if getResp.Code != http.StatusOK {
		t.Errorf("Get PIA failed: %d", getResp.Code)
	}
}

func TestIntegration_Privacy_RoPA(t *testing.T) {
	_, _, token := setupTestData(t)

	// List RoPA
	listResp := makeRequest(t, "GET", "/api/v1/privacy/ropa", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List RoPA failed: %d", listResp.Code)
	}

	// Create RoPA
	createResp := makeRequest(t, "POST", "/api/v1/privacy/ropa", map[string]any{
		"activity": "Data processing",
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Errorf("Create RoPA failed: %d", createResp.Code)
	}
}

func TestIntegration_Privacy_Consent(t *testing.T) {
	_, _, token := setupTestData(t)

	// Record consent
	recordResp := makeRequest(t, "POST", "/api/v1/privacy/consent", map[string]any{
		"subject_id": "user-123",
		"purpose":    "marketing",
	}, token)
	if recordResp.Code != http.StatusOK {
		t.Errorf("Record consent failed: %d", recordResp.Code)
	}

	// Withdraw consent
	withdrawResp := makeRequest(t, "DELETE", "/api/v1/privacy/consent/user-123", nil, token)
	if withdrawResp.Code != http.StatusOK {
		t.Errorf("Withdraw consent failed: %d", withdrawResp.Code)
	}
}

func TestIntegration_Privacy_Retention(t *testing.T) {
	_, _, token := setupTestData(t)

	// Get violations
	violationsResp := makeRequest(t, "GET", "/api/v1/privacy/retention/violations", nil, token)
	if violationsResp.Code != http.StatusOK {
		t.Errorf("Get retention violations failed: %d", violationsResp.Code)
	}

	// Set policy
	policyResp := makeRequest(t, "POST", "/api/v1/privacy/retention/policies", map[string]any{
		"classification": "PII",
		"retention_days": 365,
	}, token)
	if policyResp.Code != http.StatusOK {
		t.Errorf("Set retention policy failed: %d", policyResp.Code)
	}
}
