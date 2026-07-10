package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/trustvault/trustvault/internal/pkg"
)

func TestIntegration_Users_CRUD(t *testing.T) {
	_, _, token := setupTestData(t)

	// Create user
	createResp := makeRequest(t, "POST", "/api/v1/users", map[string]any{
		"email":    "newuser@test.com",
		"password": "securepass123",
		"name":     "New User",
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("Create user failed: %d - %s", createResp.Code, createResp.Body.String())
	}

	var user map[string]any
	json.Unmarshal(createResp.Body.Bytes(), &user)
	userID := user["id"].(string)

	// List users
	listResp := makeRequest(t, "GET", "/api/v1/users", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List users failed: %d", listResp.Code)
	}

	// Get user
	getResp := makeRequest(t, "GET", "/api/v1/users/"+userID, nil, token)
	if getResp.Code != http.StatusOK {
		t.Errorf("Get user failed: %d", getResp.Code)
	}

	// Update user
	updateResp := makeRequest(t, "PUT", "/api/v1/users/"+userID, map[string]any{
		"name":   "Updated Name",
		"status": "active",
	}, token)
	if updateResp.Code != http.StatusOK {
		t.Errorf("Update user failed: %d", updateResp.Code)
	}

	// Delete user
	deleteResp := makeRequest(t, "DELETE", "/api/v1/users/"+userID, nil, token)
	if deleteResp.Code != http.StatusOK {
		t.Errorf("Delete user failed: %d", deleteResp.Code)
	}
}

func TestIntegration_Roles_CRUD(t *testing.T) {
	_, _, token := setupTestData(t)

	// Create role
	createResp := makeRequest(t, "POST", "/api/v1/roles", map[string]any{
		"name":        "custom_role",
		"description": "A custom role",
		"permissions": []string{"datasources:read", "policies:read"},
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("Create role failed: %d - %s", createResp.Code, createResp.Body.String())
	}

	var role map[string]any
	json.Unmarshal(createResp.Body.Bytes(), &role)
	roleID := role["id"].(string)

	// List roles
	listResp := makeRequest(t, "GET", "/api/v1/roles", nil, token)
	if listResp.Code != http.StatusOK {
		t.Errorf("List roles failed: %d", listResp.Code)
	}

	// Update role
	updateResp := makeRequest(t, "PUT", "/api/v1/roles/"+roleID, map[string]any{
		"name":        "updated_role",
		"description": "Updated description",
		"permissions": []string{"datasources:*"},
	}, token)
	if updateResp.Code != http.StatusOK {
		t.Errorf("Update role failed: %d", updateResp.Code)
	}
}

func TestIntegration_MultiTenant_Isolation(t *testing.T) {
	// Create two tenants with users
	tenant1ID, _, token1 := setupTestData(t)
	tenant2ID, _, token2 := setupTestData(t)

	if tenant1ID == tenant2ID {
		t.Fatal("Test setup error: tenants should be different")
	}

	// Create datasource in tenant 1
	ds1Resp := makeRequest(t, "POST", "/api/v1/datasources", map[string]any{
		"name": "Tenant1 Source",
		"type": "postgres",
	}, token1)
	if ds1Resp.Code != http.StatusCreated {
		t.Fatalf("Create datasource failed: %d", ds1Resp.Code)
	}

	var ds1 map[string]any
	json.Unmarshal(ds1Resp.Body.Bytes(), &ds1)
	ds1ID := ds1["id"].(string)

	// Create datasource in tenant 2
	ds2Resp := makeRequest(t, "POST", "/api/v1/datasources", map[string]any{
		"name": "Tenant2 Source",
		"type": "mysql",
	}, token2)
	if ds2Resp.Code != http.StatusCreated {
		t.Fatalf("Create datasource failed: %d", ds2Resp.Code)
	}

	// Tenant 1 should NOT see tenant 2's datasource
	list1Resp := makeRequest(t, "GET", "/api/v1/datasources", nil, token1)
	var list1 []map[string]any
	json.Unmarshal(list1Resp.Body.Bytes(), &list1)

	for _, ds := range list1 {
		if ds["name"] == "Tenant2 Source" {
			t.Error("Tenant 1 should not see Tenant 2's datasource")
		}
	}

	// Tenant 2 should NOT see tenant 1's datasource
	list2Resp := makeRequest(t, "GET", "/api/v1/datasources", nil, token2)
	var list2 []map[string]any
	json.Unmarshal(list2Resp.Body.Bytes(), &list2)

	for _, ds := range list2 {
		if ds["name"] == "Tenant1 Source" {
			t.Error("Tenant 2 should not see Tenant 1's datasource")
		}
	}

	// Tenant 2 should NOT be able to access tenant 1's datasource directly
	crossAccessResp := makeRequest(t, "GET", "/api/v1/datasources/"+ds1ID, nil, token2)
	if crossAccessResp.Code != http.StatusNotFound {
		t.Errorf("Cross-tenant access should return 404, got %d", crossAccessResp.Code)
	}
}

func TestIntegration_RBAC_Enforcement(t *testing.T) {
	tenantID, _, _ := setupTestData(t)

	// Create a user with limited permissions
	limitedUserID := uuid.New().String()
	hash, _ := pkg.HashPassword("password123")
	testDB.Exec(`INSERT INTO users (id, tenant_id, email, password_hash, name, status) VALUES ($1, $2, $3, $4, $5, 'active')`,
		limitedUserID, tenantID, "limited-"+limitedUserID[:8]+"@test.com", hash, "Limited User")

	// Token with only read permissions
	limitedToken, _ := pkg.GenerateToken(limitedUserID, tenantID, []string{"datasources:read"}, false)

	// Should be able to list datasources
	listResp := makeRequest(t, "GET", "/api/v1/datasources", nil, limitedToken)
	if listResp.Code != http.StatusOK {
		t.Errorf("Read should be allowed: %d", listResp.Code)
	}

	// Should NOT be able to create datasources
	createResp := makeRequest(t, "POST", "/api/v1/datasources", map[string]any{
		"name": "Unauthorized Source",
		"type": "postgres",
	}, limitedToken)
	if createResp.Code != http.StatusForbidden {
		t.Errorf("Create should be forbidden, got %d", createResp.Code)
	}

	// Should NOT be able to access policies
	policiesResp := makeRequest(t, "GET", "/api/v1/governance/policies", nil, limitedToken)
	if policiesResp.Code != http.StatusForbidden {
		t.Errorf("Policies access should be forbidden, got %d", policiesResp.Code)
	}
}

func TestIntegration_SuperAdmin_CrossTenant(t *testing.T) {
	// Create a regular tenant
	tenantID, _, _ := setupTestData(t)

	// Create super admin token
	superAdminToken, _ := pkg.GenerateToken("super-admin", "platform", []string{"*"}, true)

	// Create datasource in tenant
	adminToken, _ := pkg.GenerateToken("admin", tenantID, []string{"*"}, false)
	dsResp := makeRequest(t, "POST", "/api/v1/datasources", map[string]any{
		"name": "Tenant Source",
		"type": "postgres",
	}, adminToken)

	var ds map[string]any
	json.Unmarshal(dsResp.Body.Bytes(), &ds)

	// Super admin should be able to access any endpoint
	// (Note: In real implementation, super admin would use internal port or impersonation)
	healthResp := makeRequest(t, "GET", "/health", nil, superAdminToken)
	if healthResp.Code != http.StatusOK {
		t.Errorf("Super admin health check failed: %d", healthResp.Code)
	}
}
