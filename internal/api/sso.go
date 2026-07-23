package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

// SSO Provider CRUD handlers

func (s *Server) listSSOProviders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var providers []store.SSOProvider
	err := s.db.SelectContext(ctx, &providers,
		`SELECT id, tenant_id, name, type, enabled, issuer_url, client_id, scopes,
		        idp_metadata_url, idp_entity_id, idp_sso_url, sp_entity_id,
		        attribute_mapping, default_role, auto_create_users, created_at, updated_at
		 FROM sso_providers WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	if providers == nil {
		providers = []store.SSOProvider{}
	}
	pkg.JSON(w, providers)
}

func (s *Server) createSSOProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Name             string   `json:"name" validate:"required"`
		Type             string   `json:"type" validate:"required,oneof=oidc saml"`
		Enabled          bool     `json:"enabled"`
		IssuerURL        string   `json:"issuer_url"`
		ClientID         string   `json:"client_id"`
		ClientSecret     string   `json:"client_secret"`
		Scopes           []string `json:"scopes"`
		IDPMetadataURL   string   `json:"idp_metadata_url"`
		IDPEntityID      string   `json:"idp_entity_id"`
		IDPSSOURL        string   `json:"idp_sso_url"`
		IDPCertificate   string   `json:"idp_certificate"`
		AttributeMapping map[string]string `json:"attribute_mapping"`
		DefaultRole      string   `json:"default_role"`
		AutoCreateUsers  bool     `json:"auto_create_users"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Encrypt client secret if provided
	var encryptedSecret *string
	if req.ClientSecret != "" {
		encrypted := encryptSecret(req.ClientSecret)
		encryptedSecret = &encrypted
	}

	// Set defaults
	if len(req.Scopes) == 0 && req.Type == "oidc" {
		req.Scopes = []string{"openid", "email", "profile"}
	}
	if req.DefaultRole == "" {
		req.DefaultRole = "user"
	}

	// Generate SP Entity ID for SAML
	var spEntityID *string
	if req.Type == "saml" {
		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = "https://app.securelens.ai"
		}
		sp := fmt.Sprintf("%s/api/v1/auth/sso/saml/metadata/%s", baseURL, tenantID)
		spEntityID = &sp
	}

	var provider store.SSOProvider
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO sso_providers 
		 (tenant_id, name, type, enabled, issuer_url, client_id, client_secret_encrypted, 
		  scopes, idp_metadata_url, idp_entity_id, idp_sso_url, idp_certificate, sp_entity_id,
		  attribute_mapping, default_role, auto_create_users)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		 RETURNING id, tenant_id, name, type, enabled, issuer_url, client_id, scopes,
		           idp_metadata_url, idp_entity_id, idp_sso_url, sp_entity_id,
		           attribute_mapping, default_role, auto_create_users, created_at, updated_at`,
		tenantID, req.Name, req.Type, req.Enabled,
		nullStr(req.IssuerURL), nullStr(req.ClientID), encryptedSecret,
		pkg.PGArray(req.Scopes),
		nullStr(req.IDPMetadataURL), nullStr(req.IDPEntityID), nullStr(req.IDPSSOURL),
		nullStr(req.IDPCertificate), spEntityID,
		store.JSON(mustMarshal(req.AttributeMapping)), req.DefaultRole, req.AutoCreateUsers,
	).Scan(&provider.ID, &provider.TenantID, &provider.Name, &provider.Type, &provider.Enabled,
		&provider.IssuerURL, &provider.ClientID, &provider.Scopes,
		&provider.IDPMetadataURL, &provider.IDPEntityID, &provider.IDPSSOURL, &provider.SPEntityID,
		&provider.AttributeMapping, &provider.DefaultRole, &provider.AutoCreateUsers,
		&provider.CreatedAt, &provider.UpdatedAt)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	pkg.JSON(w, provider, http.StatusCreated)
}

func (s *Server) getSSOProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	var provider store.SSOProvider
	err := s.db.GetContext(ctx, &provider,
		`SELECT id, tenant_id, name, type, enabled, issuer_url, client_id, scopes,
		        idp_metadata_url, idp_entity_id, idp_sso_url, sp_entity_id,
		        attribute_mapping, default_role, auto_create_users, created_at, updated_at
		 FROM sso_providers WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	pkg.JSON(w, provider)
}

func (s *Server) updateSSOProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	var req struct {
		Name             *string  `json:"name"`
		Enabled          *bool    `json:"enabled"`
		IssuerURL        *string  `json:"issuer_url"`
		ClientID         *string  `json:"client_id"`
		ClientSecret     *string  `json:"client_secret"`
		Scopes           []string `json:"scopes"`
		IDPMetadataURL   *string  `json:"idp_metadata_url"`
		IDPEntityID      *string  `json:"idp_entity_id"`
		IDPSSOURL        *string  `json:"idp_sso_url"`
		IDPCertificate   *string  `json:"idp_certificate"`
		AttributeMapping map[string]string `json:"attribute_mapping"`
		DefaultRole      *string  `json:"default_role"`
		AutoCreateUsers  *bool    `json:"auto_create_users"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	updates := []string{"updated_at = NOW()"}
	args := []any{id, tenantID}
	argIdx := 3

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Enabled != nil {
		updates = append(updates, fmt.Sprintf("enabled = $%d", argIdx))
		args = append(args, *req.Enabled)
		argIdx++
	}
	if req.IssuerURL != nil {
		updates = append(updates, fmt.Sprintf("issuer_url = $%d", argIdx))
		args = append(args, *req.IssuerURL)
		argIdx++
	}
	if req.ClientID != nil {
		updates = append(updates, fmt.Sprintf("client_id = $%d", argIdx))
		args = append(args, *req.ClientID)
		argIdx++
	}
	if req.ClientSecret != nil && *req.ClientSecret != "" {
		encrypted := encryptSecret(*req.ClientSecret)
		updates = append(updates, fmt.Sprintf("client_secret_encrypted = $%d", argIdx))
		args = append(args, encrypted)
		argIdx++
	}
	if len(req.Scopes) > 0 {
		updates = append(updates, fmt.Sprintf("scopes = $%d", argIdx))
		args = append(args, pkg.PGArray(req.Scopes))
		argIdx++
	}
	if req.IDPMetadataURL != nil {
		updates = append(updates, fmt.Sprintf("idp_metadata_url = $%d", argIdx))
		args = append(args, *req.IDPMetadataURL)
		argIdx++
	}
	if req.IDPEntityID != nil {
		updates = append(updates, fmt.Sprintf("idp_entity_id = $%d", argIdx))
		args = append(args, *req.IDPEntityID)
		argIdx++
	}
	if req.IDPSSOURL != nil {
		updates = append(updates, fmt.Sprintf("idp_sso_url = $%d", argIdx))
		args = append(args, *req.IDPSSOURL)
		argIdx++
	}
	if req.IDPCertificate != nil {
		updates = append(updates, fmt.Sprintf("idp_certificate = $%d", argIdx))
		args = append(args, *req.IDPCertificate)
		argIdx++
	}
	if req.AttributeMapping != nil {
		updates = append(updates, fmt.Sprintf("attribute_mapping = $%d", argIdx))
		args = append(args, store.JSON(mustMarshal(req.AttributeMapping)))
		argIdx++
	}
	if req.DefaultRole != nil {
		updates = append(updates, fmt.Sprintf("default_role = $%d", argIdx))
		args = append(args, *req.DefaultRole)
		argIdx++
	}
	if req.AutoCreateUsers != nil {
		updates = append(updates, fmt.Sprintf("auto_create_users = $%d", argIdx))
		args = append(args, *req.AutoCreateUsers)
		argIdx++
	}

	query := fmt.Sprintf(`UPDATE sso_providers SET %s WHERE id = $1 AND tenant_id = $2
		RETURNING id, tenant_id, name, type, enabled, issuer_url, client_id, scopes,
		          idp_metadata_url, idp_entity_id, idp_sso_url, sp_entity_id,
		          attribute_mapping, default_role, auto_create_users, created_at, updated_at`,
		strings.Join(updates, ", "))

	var provider store.SSOProvider
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&provider.ID, &provider.TenantID, &provider.Name, &provider.Type, &provider.Enabled,
		&provider.IssuerURL, &provider.ClientID, &provider.Scopes,
		&provider.IDPMetadataURL, &provider.IDPEntityID, &provider.IDPSSOURL, &provider.SPEntityID,
		&provider.AttributeMapping, &provider.DefaultRole, &provider.AutoCreateUsers,
		&provider.CreatedAt, &provider.UpdatedAt)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	pkg.JSON(w, provider)
}

func (s *Server) deleteSSOProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	_, err := s.db.ExecContext(ctx,
		"DELETE FROM sso_providers WHERE id = $1 AND tenant_id = $2", id, tenantID)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, map[string]string{"status": "deleted"})
}

// Helper functions
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func mustMarshal(v any) []byte {
	if v == nil {
		return []byte("{}")
	}
	b, _ := json.Marshal(v)
	return b
}

func encryptSecret(secret string) string {
	// Simple base64 encoding for now - in production use proper encryption
	return base64.StdEncoding.EncodeToString([]byte(secret))
}

func decryptSecret(encrypted string) string {
	b, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return ""
	}
	return string(b)
}

func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
