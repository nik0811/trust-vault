package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestIntegration_Audit(t *testing.T) {
	_, _, token := setupTestData(t)

	// Get audit trail
	trailResp := makeRequest(t, "GET", "/api/v1/audit/trail", nil, token)
	if trailResp.Code != http.StatusOK {
		t.Errorf("Get audit trail failed: %d", trailResp.Code)
	}

	// Get AI usage
	usageResp := makeRequest(t, "GET", "/api/v1/audit/datasets/test-dataset/ai-usage", nil, token)
	if usageResp.Code != http.StatusOK {
		t.Errorf("Get AI usage failed: %d", usageResp.Code)
	}

	// Get compliance report
	reportResp := makeRequest(t, "GET", "/api/v1/audit/compliance-report", nil, token)
	if reportResp.Code != http.StatusOK {
		t.Errorf("Get compliance report failed: %d", reportResp.Code)
	}

	// Get lineage
	lineageResp := makeRequest(t, "GET", "/api/v1/audit/lineage/test-dataset", nil, token)
	if lineageResp.Code != http.StatusOK {
		t.Errorf("Get lineage failed: %d", lineageResp.Code)
	}
}

func TestIntegration_Observability(t *testing.T) {
	_, _, token := setupTestData(t)

	// Get system health
	healthResp := makeRequest(t, "GET", "/api/v1/observability/health", nil, token)
	if healthResp.Code != http.StatusOK {
		t.Errorf("Get system health failed: %d", healthResp.Code)
	}

	var health map[string]any
	json.Unmarshal(healthResp.Body.Bytes(), &health)
	if health["status"] != "healthy" {
		t.Errorf("Expected healthy status, got %v", health["status"])
	}

	// Get metrics
	metricsResp := makeRequest(t, "GET", "/api/v1/observability/metrics", nil, token)
	if metricsResp.Code != http.StatusOK {
		t.Errorf("Get metrics failed: %d", metricsResp.Code)
	}

	// Get alerts
	alertsResp := makeRequest(t, "GET", "/api/v1/observability/alerts", nil, token)
	if alertsResp.Code != http.StatusOK {
		t.Errorf("Get alerts failed: %d", alertsResp.Code)
	}

	// Create alert rule
	ruleResp := makeRequest(t, "POST", "/api/v1/observability/alerts/rules", map[string]any{
		"name":      "High error rate",
		"condition": "error_rate > 0.1",
		"severity":  "critical",
	}, token)
	if ruleResp.Code != http.StatusCreated {
		t.Errorf("Create alert rule failed: %d", ruleResp.Code)
	}
}

func TestIntegration_AIGovernance(t *testing.T) {
	_, _, token := setupTestData(t)

	// Create AI policy - use valid type: ai
	createResp := makeRequest(t, "POST", "/api/v1/governance/policies", map[string]any{
		"name": "No PHI for Training",
		"type": "ai",
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Logf("Create AI policy response: %s", createResp.Body.String())
		t.Errorf("Create AI policy failed: %d", createResp.Code)
	}

	// List AI policies
	listResp := makeRequest(t, "GET", "/api/v1/ai-governance/policies", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List AI policies failed: %d", listResp.Code)
	}

	// Evaluate AI eligibility
	evalResp := makeRequest(t, "POST", "/api/v1/ai-governance/evaluate", map[string]any{
		"dataset_id": "test-dataset",
		"purpose":    "training",
	}, token)
	if evalResp.Code != http.StatusOK {
		t.Errorf("Evaluate AI eligibility failed: %d", evalResp.Code)
	}

	// Get eligibility status
	eligResp := makeRequest(t, "GET", "/api/v1/ai-governance/eligible/test-dataset", nil, token)
	if eligResp.Code != http.StatusOK {
		t.Errorf("Get eligibility failed: %d", eligResp.Code)
	}

	// Get model lineage
	lineageResp := makeRequest(t, "GET", "/api/v1/ai-governance/lineage/model-123", nil, token)
	if lineageResp.Code != http.StatusOK {
		t.Errorf("Get model lineage failed: %d", lineageResp.Code)
	}

	// Generate model card
	cardResp := makeRequest(t, "POST", "/api/v1/ai-governance/model-card", map[string]any{
		"model_id": "model-123",
	}, token)
	if cardResp.Code != http.StatusOK {
		t.Errorf("Generate model card failed: %d", cardResp.Code)
	}
}

func TestIntegration_Notifications(t *testing.T) {
	_, _, token := setupTestData(t)

	// List notifications
	listResp := makeRequest(t, "GET", "/api/v1/notifications", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List notifications failed: %d", listResp.Code)
	}

	// Create webhook - must have valid URL
	webhookResp := makeRequest(t, "POST", "/api/v1/notifications/webhooks", map[string]any{
		"url":    "https://example.com/webhook",
		"events": []string{"policy.violated", "classification.completed"},
	}, token)
	if webhookResp.Code != http.StatusCreated {
		t.Logf("Create webhook response: %s", webhookResp.Body.String())
		t.Errorf("Create webhook failed: %d", webhookResp.Code)
		return
	}

	var webhook map[string]any
	json.Unmarshal(webhookResp.Body.Bytes(), &webhook)
	if webhook["id"] == nil {
		t.Error("Expected webhook id in response")
		return
	}
	webhookID := webhook["id"].(string)

	// List webhooks
	listWebhooksResp := makeRequest(t, "GET", "/api/v1/notifications/webhooks", nil, token)
	if listWebhooksResp.Code != http.StatusOK {
		t.Errorf("List webhooks failed: %d", listWebhooksResp.Code)
	}

	// Delete webhook
	deleteResp := makeRequest(t, "DELETE", "/api/v1/notifications/webhooks/"+webhookID, nil, token)
	if deleteResp.Code != http.StatusOK {
		t.Errorf("Delete webhook failed: %d", deleteResp.Code)
	}
}
