package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/trustvault/trustvault/internal/pkg"
)

func TestIntegration_Auth_Login(t *testing.T) {
	tenantID, _, _ := setupTestData(t)

	// Create a user for login test with proper UUID
	loginUserID := uuid.New().String()
	hash, _ := pkg.HashPassword("testpass123")
	_, err := testDB.Exec(`INSERT INTO users (id, tenant_id, email, password_hash, name, status) VALUES ($1, $2, $3, $4, $5, 'active')`,
		loginUserID, tenantID, "login@test.com", hash, "Login User")
	if err != nil {
		t.Fatalf("Failed to create login user: %v", err)
	}

	tests := []struct {
		name       string
		email      string
		password   string
		wantStatus int
	}{
		{"valid credentials", "login@test.com", "testpass123", http.StatusOK},
		{"wrong password", "login@test.com", "wrongpass", http.StatusUnauthorized},
		{"unknown user", "unknown@test.com", "testpass123", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := makeRequest(t, "POST", "/api/v1/auth/login", map[string]string{
				"email":    tt.email,
				"password": tt.password,
			}, "")

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d. Body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var resp map[string]any
				json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["access_token"] == nil {
					t.Error("Expected access_token in response")
				}
			}
		})
	}
}

func TestIntegration_DataSources_CRUD(t *testing.T) {
	_, _, token := setupTestData(t)

	// CREATE - use valid type from validation: postgres mysql s3 snowflake bigquery file
	createResp := makeRequest(t, "POST", "/api/v1/datasources", map[string]any{
		"name": "Test PostgreSQL",
		"type": "postgres",
	}, token)

	if createResp.Code != http.StatusCreated {
		t.Fatalf("Create failed: %d - %s", createResp.Code, createResp.Body.String())
	}

	var created map[string]any
	json.Unmarshal(createResp.Body.Bytes(), &created)
	dsID := created["id"].(string)

	// LIST
	listResp := makeRequest(t, "GET", "/api/v1/datasources", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List failed: %d", listResp.Code)
	}

	var list []map[string]any
	json.Unmarshal(listResp.Body.Bytes(), &list)
	if len(list) == 0 {
		t.Error("Expected at least one datasource")
	}

	// GET
	getResp := makeRequest(t, "GET", "/api/v1/datasources/"+dsID, nil, token)
	if getResp.Code != http.StatusOK {
		t.Errorf("Get failed: %d", getResp.Code)
	}

	// UPDATE
	updateResp := makeRequest(t, "PUT", "/api/v1/datasources/"+dsID, map[string]any{
		"name": "Updated PostgreSQL",
		"type": "postgres",
	}, token)
	if updateResp.Code != http.StatusOK {
		t.Errorf("Update failed: %d", updateResp.Code)
	}

	// DELETE
	deleteResp := makeRequest(t, "DELETE", "/api/v1/datasources/"+dsID, nil, token)
	if deleteResp.Code != http.StatusOK {
		t.Errorf("Delete failed: %d", deleteResp.Code)
	}

	// Verify deleted
	getAfterDelete := makeRequest(t, "GET", "/api/v1/datasources/"+dsID, nil, token)
	if getAfterDelete.Code != http.StatusNotFound {
		t.Errorf("Expected 404 after delete, got %d", getAfterDelete.Code)
	}
}

func TestIntegration_Policies_CRUD(t *testing.T) {
	_, _, token := setupTestData(t)

	// CREATE - use valid type from validation: access redaction ai retention
	createResp := makeRequest(t, "POST", "/api/v1/governance/policies", map[string]any{
		"name":        "Block PII to External LLM",
		"description": "Prevents PII from being sent to external LLMs",
		"type":        "access",
	}, token)

	if createResp.Code != http.StatusCreated {
		t.Fatalf("Create policy failed: %d - %s", createResp.Code, createResp.Body.String())
	}

	var created map[string]any
	json.Unmarshal(createResp.Body.Bytes(), &created)
	policyID := created["id"].(string)

	// LIST
	listResp := makeRequest(t, "GET", "/api/v1/governance/policies", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List policies failed: %d", listResp.Code)
	}

	// GET
	getResp := makeRequest(t, "GET", "/api/v1/governance/policies/"+policyID, nil, token)
	if getResp.Code != http.StatusOK {
		t.Errorf("Get policy failed: %d", getResp.Code)
	}

	// UPDATE
	updateResp := makeRequest(t, "PUT", "/api/v1/governance/policies/"+policyID, map[string]any{
		"name":   "Updated Policy",
		"type":   "access",
		"active": false,
	}, token)
	if updateResp.Code != http.StatusOK {
		t.Errorf("Update policy failed: %d", updateResp.Code)
	}

	// DELETE
	deleteResp := makeRequest(t, "DELETE", "/api/v1/governance/policies/"+policyID, nil, token)
	if deleteResp.Code != http.StatusOK {
		t.Errorf("Delete policy failed: %d", deleteResp.Code)
	}
}

func TestIntegration_Governance_Evaluate(t *testing.T) {
	_, _, token := setupTestData(t)

	// Create a policy first - use valid type
	makeRequest(t, "POST", "/api/v1/governance/policies", map[string]any{
		"name": "Test Policy",
		"type": "redaction",
	}, token)

	// Evaluate
	evalResp := makeRequest(t, "POST", "/api/v1/governance/evaluate", map[string]any{
		"data": "This contains test@email.com",
		"context": map[string]any{
			"user_role": "analyst",
		},
	}, token)

	if evalResp.Code != http.StatusOK {
		t.Errorf("Evaluate failed: %d - %s", evalResp.Code, evalResp.Body.String())
	}

	var result map[string]any
	json.Unmarshal(evalResp.Body.Bytes(), &result)
	if result["decision"] == nil {
		t.Error("Expected decision in response")
	}
}
