package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestIntegration_Advisor(t *testing.T) {
	_, _, token := setupTestData(t)

	// Get recommendations
	recsResp := makeRequest(t, "GET", "/api/v1/advisor/recommendations", nil, token)
	if recsResp.Code != http.StatusOK {
		t.Errorf("Get recommendations failed: %d", recsResp.Code)
	}

	var recs []map[string]any
	json.Unmarshal(recsResp.Body.Bytes(), &recs)
	if len(recs) == 0 {
		t.Error("Expected at least one recommendation")
	}
	// Verify frontend-expected fields
	if len(recs) > 0 {
		if recs[0]["priority"] == nil {
			t.Error("Expected priority field in recommendation")
		}
		if recs[0]["type"] == nil {
			t.Error("Expected type field in recommendation")
		}
	}

	// Get compliance gaps
	gapsResp := makeRequest(t, "GET", "/api/v1/advisor/gaps", nil, token)
	if gapsResp.Code != http.StatusOK {
		t.Errorf("Get compliance gaps failed: %d", gapsResp.Code)
	}
	// Verify gaps is an array (frontend expects array format)
	var gaps []map[string]any
	json.Unmarshal(gapsResp.Body.Bytes(), &gaps)
	// gaps can be empty if no compliance issues

	// Generate defense docket
	docketResp := makeRequest(t, "POST", "/api/v1/advisor/defense-docket", map[string]any{
		"regulations": []string{"GDPR", "CCPA"},
		"date_from":   "2026-01-01",
		"date_to":     "2026-12-31",
	}, token)
	if docketResp.Code != http.StatusOK {
		t.Errorf("Generate defense docket failed: %d", docketResp.Code)
	}

	// Get playbook
	playbookResp := makeRequest(t, "GET", "/api/v1/advisor/playbook/missing_retention", nil, token)
	if playbookResp.Code != http.StatusOK {
		t.Errorf("Get playbook failed: %d", playbookResp.Code)
	}

	// Get risk score
	riskResp := makeRequest(t, "GET", "/api/v1/advisor/risk-score", nil, token)
	if riskResp.Code != http.StatusOK {
		t.Errorf("Get risk score failed: %d", riskResp.Code)
	}

	var risk map[string]any
	json.Unmarshal(riskResp.Body.Bytes(), &risk)
	if risk["overall_score"] == nil {
		t.Error("Expected overall_score in risk response")
	}
}

func TestIntegration_ROT(t *testing.T) {
	_, _, token := setupTestData(t)

	// Get ROT summary
	summaryResp := makeRequest(t, "GET", "/api/v1/rot/summary", nil, token)
	if summaryResp.Code != http.StatusOK {
		t.Errorf("Get ROT summary failed: %d", summaryResp.Code)
	}

	// Get ROT datasets
	datasetsResp := makeRequest(t, "GET", "/api/v1/rot/datasets", nil, token)
	if datasetsResp.Code != http.StatusOK {
		t.Errorf("Get ROT datasets failed: %d", datasetsResp.Code)
	}

	// Get duplicates
	dupsResp := makeRequest(t, "GET", "/api/v1/rot/duplicates", nil, token)
	if dupsResp.Code != http.StatusOK {
		t.Errorf("Get duplicates failed: %d", dupsResp.Code)
	}

	// Trigger ROT scan
	scanResp := makeRequest(t, "POST", "/api/v1/rot/scan", nil, token)
	if scanResp.Code != http.StatusOK {
		t.Errorf("Trigger ROT scan failed: %d", scanResp.Code)
	}

	// Remediate ROT
	remediateResp := makeRequest(t, "POST", "/api/v1/rot/remediate", map[string]any{
		"dataset_ids": []string{"ds-1", "ds-2"},
		"action":      "archive",
	}, token)
	if remediateResp.Code != http.StatusOK {
		t.Errorf("Remediate ROT failed: %d", remediateResp.Code)
	}
}

func TestIntegration_Integrations(t *testing.T) {
	_, _, token := setupTestData(t)

	// Create integration - must match validation rules
	createResp := makeRequest(t, "POST", "/api/v1/integrations", map[string]any{
		"name":     "Slack Alerts",
		"type":     "communication",
		"provider": "slack",
		"config": map[string]string{
			"webhook_url": "https://hooks.slack.com/xxx",
		},
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Logf("Create integration response: %s", createResp.Body.String())
		t.Fatalf("Create integration failed: %d", createResp.Code)
	}

	var integration map[string]any
	json.Unmarshal(createResp.Body.Bytes(), &integration)
	integrationID := integration["id"].(string)

	// List integrations
	listResp := makeRequest(t, "GET", "/api/v1/integrations", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List integrations failed: %d", listResp.Code)
	}

	// Get integration
	getResp := makeRequest(t, "GET", "/api/v1/integrations/"+integrationID, nil, token)
	if getResp.Code != http.StatusOK {
		t.Errorf("Get integration failed: %d", getResp.Code)
	}

	// Update integration
	updateResp := makeRequest(t, "PUT", "/api/v1/integrations/"+integrationID, map[string]any{
		"name":     "Updated Slack",
		"type":     "communication",
		"sync_freq": "hourly",
	}, token)
	if updateResp.Code != http.StatusOK {
		t.Errorf("Update integration failed: %d", updateResp.Code)
	}

	// Test integration
	testResp := makeRequest(t, "POST", "/api/v1/integrations/"+integrationID+"/test", nil, token)
	if testResp.Code != http.StatusOK {
		t.Errorf("Test integration failed: %d", testResp.Code)
	}

	// Sync integration
	syncResp := makeRequest(t, "POST", "/api/v1/integrations/"+integrationID+"/sync", nil, token)
	if syncResp.Code != http.StatusOK {
		t.Errorf("Sync integration failed: %d", syncResp.Code)
	}

	// Get integration logs
	logsResp := makeRequest(t, "GET", "/api/v1/integrations/"+integrationID+"/logs", nil, token)
	if logsResp.Code != http.StatusOK {
		t.Errorf("Get integration logs failed: %d", logsResp.Code)
	}

	// Delete integration
	deleteResp := makeRequest(t, "DELETE", "/api/v1/integrations/"+integrationID, nil, token)
	if deleteResp.Code != http.StatusOK {
		t.Errorf("Delete integration failed: %d", deleteResp.Code)
	}
}

func TestIntegration_DataMap(t *testing.T) {
	_, _, token := setupTestData(t)

	// Get data map
	mapResp := makeRequest(t, "GET", "/api/v1/datamap", nil, token)
	if mapResp.Code != http.StatusOK {
		t.Errorf("Get data map failed: %d", mapResp.Code)
	}

	var dataMap map[string]any
	json.Unmarshal(mapResp.Body.Bytes(), &dataMap)
	if dataMap["nodes"] == nil {
		t.Error("Expected nodes in data map")
	}

	// Get sources
	sourcesResp := makeRequest(t, "GET", "/api/v1/datamap/sources", nil, token)
	if sourcesResp.Code != http.StatusOK {
		t.Errorf("Get data map sources failed: %d", sourcesResp.Code)
	}

	// Get flows
	flowsResp := makeRequest(t, "GET", "/api/v1/datamap/flows", nil, token)
	if flowsResp.Code != http.StatusOK {
		t.Errorf("Get data flows failed: %d", flowsResp.Code)
	}

	// Get coverage
	coverageResp := makeRequest(t, "GET", "/api/v1/datamap/coverage", nil, token)
	if coverageResp.Code != http.StatusOK {
		t.Errorf("Get coverage failed: %d", coverageResp.Code)
	}

	// Get geography
	geoResp := makeRequest(t, "GET", "/api/v1/datamap/geography", nil, token)
	if geoResp.Code != http.StatusOK {
		t.Errorf("Get geography failed: %d", geoResp.Code)
	}

	// Get dark data
	darkResp := makeRequest(t, "GET", "/api/v1/datamap/dark-data", nil, token)
	if darkResp.Code != http.StatusOK {
		t.Errorf("Get dark data failed: %d", darkResp.Code)
	}
}

func TestIntegration_Documents(t *testing.T) {
	_, _, token := setupTestData(t)

	// Extract document
	extractResp := makeRequest(t, "POST", "/api/v1/documents/extract", map[string]any{
		"content": "base64-encoded-content",
		"format":  "pdf",
	}, token)
	if extractResp.Code != http.StatusOK {
		t.Errorf("Extract document failed: %d", extractResp.Code)
	}

	// Classify document
	classifyResp := makeRequest(t, "POST", "/api/v1/documents/classify", map[string]any{
		"content": "This document contains john@example.com",
	}, token)
	if classifyResp.Code != http.StatusOK {
		t.Errorf("Classify document failed: %d", classifyResp.Code)
	}

	// Get review queue
	queueResp := makeRequest(t, "GET", "/api/v1/documents/review-queue", nil, token)
	if queueResp.Code != http.StatusOK {
		t.Errorf("Get review queue failed: %d", queueResp.Code)
	}
}
