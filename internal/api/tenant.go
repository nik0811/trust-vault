package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

// Internal/Super Admin handlers

func (s *Server) createTenant(w http.ResponseWriter, r *http.Request) {
	var tenant store.Tenant
	if err := pkg.Bind(r, &tenant); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	tenant.Status = "active"
	if err := s.tenants.Create(r.Context(), &tenant); err != nil {
		pkg.Error(w, err)
		return
	}

	// Create default roles for tenant
	defaultRoles := []store.Role{
		{TenantID: tenant.ID, Name: "tenant_admin", IsSystem: true, Permissions: []byte(`["*"]`)},
		{TenantID: tenant.ID, Name: "governance_admin", IsSystem: true, Permissions: []byte(`["policies:*","classifications:*"]`)},
		{TenantID: tenant.ID, Name: "data_steward", IsSystem: true, Permissions: []byte(`["datasources:read","classifications:read","quality:read"]`)},
		{TenantID: tenant.ID, Name: "analyst", IsSystem: true, Permissions: []byte(`["audit:read","reports:read"]`)},
		{TenantID: tenant.ID, Name: "ai_consumer", IsSystem: true, Permissions: []byte(`["gate:query"]`)},
	}
	for _, role := range defaultRoles {
		s.roles.Create(r.Context(), &role)
	}

	pkg.JSON(w, tenant, http.StatusCreated)
}

func (s *Server) listTenants(w http.ResponseWriter, r *http.Request) {
	var tenants []store.Tenant
	s.db.SelectContext(r.Context(), &tenants, "SELECT * FROM tenants ORDER BY created_at DESC")
	pkg.JSON(w, tenants)
}

func (s *Server) getTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var tenant store.Tenant
	err := s.db.GetContext(r.Context(), &tenant, "SELECT * FROM tenants WHERE id = $1", id)
	if err != nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	pkg.JSON(w, tenant)
}

func (s *Server) suspendTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, err := s.db.ExecContext(r.Context(), "UPDATE tenants SET status = 'suspended' WHERE id = $1", id)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	pkg.JSON(w, map[string]string{"status": "suspended"})
}

func (s *Server) deleteTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, err := s.db.ExecContext(r.Context(), "UPDATE tenants SET status = 'deleted' WHERE id = $1", id)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	pkg.JSON(w, map[string]string{"status": "deleted"})
}

func (s *Server) impersonateTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Generate impersonation token
	token, err := pkg.GenerateToken(req.UserID, id, []string{"*"}, false)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	pkg.JSON(w, map[string]string{
		"access_token": token,
		"tenant_id":    id,
		"user_id":      req.UserID,
	})
}
