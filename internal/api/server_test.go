package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/pkg"
)

func TestHealthCheck(t *testing.T) {
	s := &Server{router: chi.NewRouter()}
	s.router.Get("/health", s.healthCheck)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("Status = %s, want ok", resp["status"])
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	s := &Server{router: chi.NewRouter()}
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			pkg.JSON(w, map[string]string{"ok": "true"})
		})
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	s := &Server{router: chi.NewRouter()}
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			pkg.JSON(w, map[string]string{"ok": "true"})
		})
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	pkg.SetJWTSecret("test-secret")
	token, _ := pkg.GenerateToken("user-1", "tenant-1", []string{"test:read"}, false)

	s := &Server{router: chi.NewRouter()}
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			tenantID := pkg.TenantFromCtx(r.Context())
			pkg.JSON(w, map[string]string{"tenant_id": tenantID})
		})
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["tenant_id"] != "tenant-1" {
		t.Errorf("tenant_id = %s, want tenant-1", resp["tenant_id"])
	}
}

func TestRBACMiddleware_NoPermission(t *testing.T) {
	pkg.SetJWTSecret("test-secret")
	token, _ := pkg.GenerateToken("user-1", "tenant-1", []string{"other:read"}, false)

	s := &Server{router: chi.NewRouter()}
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.rbacMiddleware("datasources:read"))
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			pkg.JSON(w, map[string]string{"ok": "true"})
		})
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %d, want %d (forbidden)", w.Code, http.StatusForbidden)
	}
}

func TestRBACMiddleware_WithPermission(t *testing.T) {
	pkg.SetJWTSecret("test-secret")
	token, _ := pkg.GenerateToken("user-1", "tenant-1", []string{"datasources:read"}, false)

	s := &Server{router: chi.NewRouter()}
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.rbacMiddleware("datasources:read"))
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			pkg.JSON(w, map[string]string{"ok": "true"})
		})
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestSuperAdminBypassesRBAC(t *testing.T) {
	pkg.SetJWTSecret("test-secret")
	token, _ := pkg.GenerateToken("admin-1", "platform", []string{}, true)

	s := &Server{router: chi.NewRouter()}
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Use(s.rbacMiddleware("any:permission"))
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			pkg.JSON(w, map[string]string{"ok": "true"})
		})
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d (super admin should bypass RBAC)", w.Code, http.StatusOK)
	}
}

func TestTenantIsolation(t *testing.T) {
	pkg.SetJWTSecret("test-secret")
	
	// Token for tenant-1
	token1, _ := pkg.GenerateToken("user-1", "tenant-1", []string{"*"}, false)
	// Token for tenant-2
	token2, _ := pkg.GenerateToken("user-2", "tenant-2", []string{"*"}, false)

	s := &Server{router: chi.NewRouter()}
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Get("/tenant", func(w http.ResponseWriter, r *http.Request) {
			tenantID := pkg.TenantFromCtx(r.Context())
			pkg.JSON(w, map[string]string{"tenant_id": tenantID})
		})
	})

	// Request with tenant-1 token
	req1 := httptest.NewRequest("GET", "/api/v1/tenant", nil)
	req1.Header.Set("Authorization", "Bearer "+token1)
	w1 := httptest.NewRecorder()
	s.router.ServeHTTP(w1, req1)

	var resp1 map[string]string
	json.Unmarshal(w1.Body.Bytes(), &resp1)
	if resp1["tenant_id"] != "tenant-1" {
		t.Errorf("Tenant 1 request got tenant_id = %s, want tenant-1", resp1["tenant_id"])
	}

	// Request with tenant-2 token
	req2 := httptest.NewRequest("GET", "/api/v1/tenant", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)
	w2 := httptest.NewRecorder()
	s.router.ServeHTTP(w2, req2)

	var resp2 map[string]string
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2["tenant_id"] != "tenant-2" {
		t.Errorf("Tenant 2 request got tenant_id = %s, want tenant-2", resp2["tenant_id"])
	}
}
