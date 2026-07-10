package pkg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateAndValidateToken(t *testing.T) {
	SetJWTSecret("test-secret-key")

	tests := []struct {
		name         string
		userID       string
		tenantID     string
		permissions  []string
		isSuperAdmin bool
	}{
		{
			name:         "regular user",
			userID:       "user-123",
			tenantID:     "tenant-456",
			permissions:  []string{"datasources:read", "policies:read"},
			isSuperAdmin: false,
		},
		{
			name:         "super admin",
			userID:       "admin-001",
			tenantID:     "platform",
			permissions:  []string{"*"},
			isSuperAdmin: true,
		},
		{
			name:         "user with wildcard permission",
			userID:       "user-789",
			tenantID:     "tenant-abc",
			permissions:  []string{"datasources:*"},
			isSuperAdmin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.userID, tt.tenantID, tt.permissions, tt.isSuperAdmin)
			if err != nil {
				t.Fatalf("GenerateToken failed: %v", err)
			}

			if token == "" {
				t.Fatal("Generated token is empty")
			}

			claims, err := ValidateToken(token)
			if err != nil {
				t.Fatalf("ValidateToken failed: %v", err)
			}

			if claims.UserID != tt.userID {
				t.Errorf("UserID mismatch: got %s, want %s", claims.UserID, tt.userID)
			}
			if claims.TenantID != tt.tenantID {
				t.Errorf("TenantID mismatch: got %s, want %s", claims.TenantID, tt.tenantID)
			}
			if claims.IsSuperAdmin != tt.isSuperAdmin {
				t.Errorf("IsSuperAdmin mismatch: got %v, want %v", claims.IsSuperAdmin, tt.isSuperAdmin)
			}
		})
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	SetJWTSecret("test-secret-key")

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "not-a-jwt"},
		{"wrong signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.wrong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateToken(tt.token)
			if err == nil {
				t.Error("Expected error for invalid token, got nil")
			}
		})
	}
}

func TestHashAndCheckPassword(t *testing.T) {
	password := "secure-password-123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == password {
		t.Error("Hash should not equal plain password")
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for correct password")
	}

	if CheckPassword("wrong-password", hash) {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Test empty context
	if TenantFromCtx(ctx) != "" {
		t.Error("TenantFromCtx should return empty string for empty context")
	}
	if UserFromCtx(ctx) != "" {
		t.Error("UserFromCtx should return empty string for empty context")
	}
	if IsSuperAdminFromCtx(ctx) {
		t.Error("IsSuperAdminFromCtx should return false for empty context")
	}

	// Test with values
	ctx = context.WithValue(ctx, CtxTenantID, "tenant-123")
	ctx = context.WithValue(ctx, CtxUserID, "user-456")
	ctx = context.WithValue(ctx, CtxIsSuperAdmin, true)
	ctx = context.WithValue(ctx, CtxPermissions, []string{"read", "write"})

	if TenantFromCtx(ctx) != "tenant-123" {
		t.Errorf("TenantFromCtx got %s, want tenant-123", TenantFromCtx(ctx))
	}
	if UserFromCtx(ctx) != "user-456" {
		t.Errorf("UserFromCtx got %s, want user-456", UserFromCtx(ctx))
	}
	if !IsSuperAdminFromCtx(ctx) {
		t.Error("IsSuperAdminFromCtx should return true")
	}
}

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		superAdmin  bool
		checkPerm   string
		expected    bool
	}{
		{"super admin has all", []string{}, true, "anything:here", true},
		{"exact match", []string{"datasources:read"}, false, "datasources:read", true},
		{"no match", []string{"datasources:read"}, false, "policies:write", false},
		{"wildcard all", []string{"*"}, false, "anything:here", true},
		{"wildcard resource", []string{"datasources:*"}, false, "datasources:delete", true},
		{"wildcard no match", []string{"datasources:*"}, false, "policies:read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = context.WithValue(ctx, CtxPermissions, tt.permissions)
			ctx = context.WithValue(ctx, CtxIsSuperAdmin, tt.superAdmin)

			result := HasPermission(ctx, tt.checkPerm)
			if result != tt.expected {
				t.Errorf("HasPermission(%s) = %v, want %v", tt.checkPerm, result, tt.expected)
			}
		})
	}
}

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "ok"}

	JSON(w, data)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Error("Content-Type should be application/json")
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Errorf("Body = %s, want to contain status:ok", w.Body.String())
	}
}

func TestJSON_WithStatus(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"id": "123"}

	JSON(w, data, http.StatusCreated)

	if w.Code != http.StatusCreated {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{"not found", ErrNotFound, http.StatusNotFound},
		{"unauthorized", ErrUnauthorized, http.StatusUnauthorized},
		{"forbidden", ErrForbidden, http.StatusForbidden},
		{"bad request", ErrBadRequest, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			Error(w, tt.err)

			if w.Code != tt.expectedCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.expectedCode)
			}
		})
	}
}

func TestParseListOpts(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedLimit  int
		expectedOffset int
	}{
		{"defaults", "", 50, 0},
		{"custom limit", "limit=25", 25, 0},
		{"custom offset", "offset=100", 50, 100},
		{"both", "limit=10&offset=20", 10, 20},
		{"limit too high", "limit=500", 50, 0}, // Should cap at 100
		{"negative offset", "offset=-5", 50, 0},
		{"invalid values", "limit=abc&offset=xyz", 50, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.query, nil)
			limit, offset := ParseListOpts(req)

			if limit != tt.expectedLimit {
				t.Errorf("limit = %d, want %d", limit, tt.expectedLimit)
			}
			if offset != tt.expectedOffset {
				t.Errorf("offset = %d, want %d", offset, tt.expectedOffset)
			}
		})
	}
}
