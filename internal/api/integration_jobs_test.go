package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestIntegration_Jobs(t *testing.T) {
	_, _, token := setupTestData(t)

	// Create job
	createResp := makeRequest(t, "POST", "/api/v1/jobs", map[string]any{
		"name":     "Daily Scan",
		"type":     "scan",
		"schedule": "0 0 * * *",
		"config": map[string]any{
			"datasource_id": "ds-123",
		},
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("Create job failed: %d - %s", createResp.Code, createResp.Body.String())
	}

	var job map[string]any
	json.Unmarshal(createResp.Body.Bytes(), &job)
	jobID := job["id"].(string)

	// List jobs
	listResp := makeRequest(t, "GET", "/api/v1/jobs", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List jobs failed: %d", listResp.Code)
	}

	// Get job
	getResp := makeRequest(t, "GET", "/api/v1/jobs/"+jobID, nil, token)
	if getResp.Code != http.StatusOK {
		t.Errorf("Get job failed: %d", getResp.Code)
	}

	// Run job now
	runResp := makeRequest(t, "POST", "/api/v1/jobs/"+jobID+"/run-now", nil, token)
	if runResp.Code != http.StatusOK {
		t.Errorf("Run job failed: %d", runResp.Code)
	}

	// Delete job
	deleteResp := makeRequest(t, "DELETE", "/api/v1/jobs/"+jobID, nil, token)
	if deleteResp.Code != http.StatusOK {
		t.Errorf("Delete job failed: %d", deleteResp.Code)
	}
}

func TestIntegration_Remediation(t *testing.T) {
	_, _, token := setupTestData(t)

	// List actions
	listResp := makeRequest(t, "GET", "/api/v1/remediation/actions", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List remediation actions failed: %d", listResp.Code)
	}

	// Create action
	createResp := makeRequest(t, "POST", "/api/v1/remediation/actions", map[string]any{
		"type":       "redact",
		"dataset_id": "test-dataset",
		"reason":     "PII detected",
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Errorf("Create remediation action failed: %d", createResp.Code)
	}

	var action map[string]any
	json.Unmarshal(createResp.Body.Bytes(), &action)
	actionID := action["id"].(string)

	// Approve action
	approveResp := makeRequest(t, "POST", "/api/v1/remediation/actions/"+actionID+"/approve", nil, token)
	if approveResp.Code != http.StatusOK {
		t.Errorf("Approve action failed: %d", approveResp.Code)
	}

	// Execute action
	execResp := makeRequest(t, "POST", "/api/v1/remediation/actions/"+actionID+"/execute", nil, token)
	if execResp.Code != http.StatusOK {
		t.Errorf("Execute action failed: %d", execResp.Code)
	}

	// Get history
	historyResp := makeRequest(t, "GET", "/api/v1/remediation/history", nil, token)
	if historyResp.Code != http.StatusOK {
		t.Errorf("Get remediation history failed: %d", historyResp.Code)
	}
}

func TestIntegration_Reports(t *testing.T) {
	_, _, token := setupTestData(t)

	// Generate report
	genResp := makeRequest(t, "POST", "/api/v1/reports/generate", map[string]any{
		"type":      "compliance",
		"date_from": "2026-01-01",
		"date_to":   "2026-12-31",
	}, token)
	if genResp.Code != http.StatusOK {
		t.Errorf("Generate report failed: %d", genResp.Code)
	}

	// List reports
	listResp := makeRequest(t, "GET", "/api/v1/reports", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List reports failed: %d", listResp.Code)
	}

	// Get analytics summary
	summaryResp := makeRequest(t, "GET", "/api/v1/analytics/summary", nil, token)
	if summaryResp.Code != http.StatusOK {
		t.Errorf("Get analytics summary failed: %d", summaryResp.Code)
	}

	// Get analytics trends
	trendsResp := makeRequest(t, "GET", "/api/v1/analytics/trends", nil, token)
	if trendsResp.Code != http.StatusOK {
		t.Errorf("Get analytics trends failed: %d", trendsResp.Code)
	}
}

func TestIntegration_Labels(t *testing.T) {
	_, _, token := setupTestData(t)

	// Get dataset label
	getResp := makeRequest(t, "GET", "/api/v1/labels/datasets/test-dataset", nil, token)
	if getResp.Code != http.StatusOK {
		t.Errorf("Get label failed: %d", getResp.Code)
	}

	// Assign label
	assignResp := makeRequest(t, "POST", "/api/v1/labels/assign", map[string]any{
		"dataset_id": "test-dataset",
		"label":      "CONFIDENTIAL",
	}, token)
	if assignResp.Code != http.StatusCreated {
		t.Errorf("Assign label failed: %d", assignResp.Code)
	}

	// Get label rules
	rulesResp := makeRequest(t, "GET", "/api/v1/labels/rules", nil, token)
	if rulesResp.Code != http.StatusOK {
		t.Errorf("Get label rules failed: %d", rulesResp.Code)
	}

	// Create label rule
	createRuleResp := makeRequest(t, "POST", "/api/v1/labels/rules", map[string]any{
		"classification": "PHI",
		"label":          "RESTRICTED",
	}, token)
	if createRuleResp.Code != http.StatusCreated {
		t.Errorf("Create label rule failed: %d", createRuleResp.Code)
	}

	// Get label summary
	summaryResp := makeRequest(t, "GET", "/api/v1/labels/summary", nil, token)
	if summaryResp.Code != http.StatusOK {
		t.Errorf("Get label summary failed: %d", summaryResp.Code)
	}
}

func TestIntegration_Feedback(t *testing.T) {
	_, _, token := setupTestData(t)

	// Submit correction
	correctionResp := makeRequest(t, "POST", "/api/v1/feedback/correction", map[string]any{
		"classification_id": "class-123",
		"corrected_label":   "PERSON_NAME",
	}, token)
	if correctionResp.Code != http.StatusCreated {
		t.Errorf("Submit correction failed: %d", correctionResp.Code)
	}

	// Submit confirmation
	confirmResp := makeRequest(t, "POST", "/api/v1/feedback/confirmation", map[string]any{
		"classification_id": "class-456",
	}, token)
	if confirmResp.Code != http.StatusCreated {
		t.Errorf("Submit confirmation failed: %d", confirmResp.Code)
	}

	// Get feedback stats
	statsResp := makeRequest(t, "GET", "/api/v1/feedback/stats", nil, token)
	if statsResp.Code != http.StatusOK {
		t.Errorf("Get feedback stats failed: %d", statsResp.Code)
	}

	// Create custom entity
	entityResp := makeRequest(t, "POST", "/api/v1/feedback/custom-entity", map[string]any{
		"name":    "PRODUCT_CODE",
		"pattern": "[A-Z]{3}-[0-9]{4}",
	}, token)
	if entityResp.Code != http.StatusCreated {
		t.Errorf("Create custom entity failed: %d", entityResp.Code)
	}

	// Get knowledge cache
	cacheResp := makeRequest(t, "GET", "/api/v1/feedback/knowledge-cache", nil, token)
	if cacheResp.Code != http.StatusOK {
		t.Errorf("Get knowledge cache failed: %d", cacheResp.Code)
	}
}
