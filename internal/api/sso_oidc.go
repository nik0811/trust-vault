package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/securelens/securelens/internal/events"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
	"golang.org/x/oauth2"
)

// ssoOIDCInitiate starts the OIDC login flow
func (s *Server) ssoOIDCInitiate(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "provider_id")
	tenantSlug := r.URL.Query().Get("tenant")

	if tenantSlug == "" {
		pkg.Error(w, fmt.Errorf("tenant parameter required"), http.StatusBadRequest)
		return
	}

	// Look up tenant by slug
	var tenantID string
	err := s.db.GetContext(r.Context(), &tenantID,
		"SELECT id FROM tenants WHERE slug = $1", tenantSlug)
	if err != nil {
		pkg.Error(w, fmt.Errorf("tenant not found"), http.StatusNotFound)
		return
	}

	// Get provider config
	var provider store.SSOProvider
	err = s.db.GetContext(r.Context(), &provider,
		`SELECT id, tenant_id, name, type, enabled, issuer_url, client_id, 
		        client_secret_encrypted, scopes
		 FROM sso_providers WHERE id = $1 AND tenant_id = $2 AND enabled = true AND type = 'oidc'`,
		providerID, tenantID)
	if err != nil {
		pkg.Error(w, fmt.Errorf("SSO provider not found or disabled"), http.StatusNotFound)
		return
	}

	if provider.IssuerURL == nil || provider.ClientID == nil {
		pkg.Error(w, fmt.Errorf("OIDC provider not properly configured"), http.StatusBadRequest)
		return
	}

	// Create OIDC provider
	ctx := r.Context()
	oidcProvider, err := oidc.NewProvider(ctx, *provider.IssuerURL)
	if err != nil {
		log.Error().Err(err).Str("issuer", *provider.IssuerURL).Msg("Failed to create OIDC provider")
		pkg.Error(w, fmt.Errorf("failed to connect to identity provider"), http.StatusBadGateway)
		return
	}

	// Get client secret
	var clientSecret string
	if provider.ClientSecretEncrypted != nil {
		clientSecret = decryptSecret(*provider.ClientSecretEncrypted)
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://app.securelens.ai"
	}
	redirectURI := fmt.Sprintf("%s/api/v1/auth/sso/oidc/callback", baseURL)

	oauth2Config := oauth2.Config{
		ClientID:     *provider.ClientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       provider.Scopes,
	}

	// Generate state and nonce
	state := generateState()
	nonce := generateNonce()

	// Store session
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sso_sessions (tenant_id, provider_id, state, nonce, redirect_uri, expires_at)
		 VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '10 minutes')`,
		tenantID, provider.ID, state, nonce, redirectURI)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	// Redirect to IdP
	authURL := oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce))
	http.Redirect(w, r, authURL, http.StatusFound)
}

// ssoOIDCCallback handles the OIDC callback
func (s *Server) ssoOIDCCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get state and code from query
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		errorDesc := r.URL.Query().Get("error_description")
		log.Error().Str("error", errorParam).Str("description", errorDesc).Msg("OIDC error")
		redirectToLoginWithError(w, r, "SSO login failed: "+errorDesc)
		return
	}

	if state == "" || code == "" {
		redirectToLoginWithError(w, r, "Invalid SSO callback")
		return
	}

	// Look up session by state
	var session struct {
		TenantID   string `db:"tenant_id"`
		ProviderID string `db:"provider_id"`
		Nonce      string `db:"nonce"`
	}
	err := s.db.GetContext(ctx, &session,
		`SELECT tenant_id, provider_id, nonce FROM sso_sessions 
		 WHERE state = $1 AND expires_at > NOW()`, state)
	if err != nil {
		redirectToLoginWithError(w, r, "SSO session expired or invalid")
		return
	}

	// Delete the session (one-time use)
	s.db.ExecContext(ctx, "DELETE FROM sso_sessions WHERE state = $1", state)

	// Get provider config
	var provider store.SSOProvider
	err = s.db.GetContext(ctx, &provider,
		`SELECT id, tenant_id, name, type, issuer_url, client_id, client_secret_encrypted, 
		        scopes, attribute_mapping, default_role, auto_create_users
		 FROM sso_providers WHERE id = $1`, session.ProviderID)
	if err != nil {
		redirectToLoginWithError(w, r, "SSO provider not found")
		return
	}

	// Create OIDC provider
	oidcProvider, err := oidc.NewProvider(ctx, *provider.IssuerURL)
	if err != nil {
		redirectToLoginWithError(w, r, "Failed to connect to identity provider")
		return
	}

	var clientSecret string
	if provider.ClientSecretEncrypted != nil {
		clientSecret = decryptSecret(*provider.ClientSecretEncrypted)
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://app.securelens.ai"
	}
	redirectURI := fmt.Sprintf("%s/api/v1/auth/sso/oidc/callback", baseURL)

	oauth2Config := oauth2.Config{
		ClientID:     *provider.ClientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       provider.Scopes,
	}

	// Exchange code for token
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		log.Error().Err(err).Msg("Failed to exchange OIDC code")
		redirectToLoginWithError(w, r, "Failed to complete SSO login")
		return
	}

	// Extract ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		redirectToLoginWithError(w, r, "No ID token in response")
		return
	}

	// Verify ID token
	verifier := oidcProvider.Verifier(&oidc.Config{ClientID: *provider.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Error().Err(err).Msg("Failed to verify ID token")
		redirectToLoginWithError(w, r, "Invalid ID token")
		return
	}

	// Verify nonce
	var claims struct {
		Nonce         string `json:"nonce"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Sub           string `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		redirectToLoginWithError(w, r, "Failed to parse ID token claims")
		return
	}

	if claims.Nonce != session.Nonce {
		redirectToLoginWithError(w, r, "Invalid nonce")
		return
	}

	if claims.Email == "" {
		redirectToLoginWithError(w, r, "Email not provided by identity provider")
		return
	}

	// Find or create user
	user, err := s.findOrCreateSSOUser(ctx, session.TenantID, provider, claims.Sub, claims.Email, claims.Name)
	if err != nil {
		log.Error().Err(err).Msg("Failed to find/create SSO user")
		redirectToLoginWithError(w, r, "Failed to create user account")
		return
	}

	// Generate JWT
	token, err := pkg.GenerateToken(user.ID, user.TenantID, []string{}, user.IsSuperAdmin)
	if err != nil {
		redirectToLoginWithError(w, r, "Failed to generate session")
		return
	}

	// Update last login
	s.db.ExecContext(ctx, "UPDATE users SET last_login_at = $1 WHERE id = $2", time.Now(), user.ID)

	// Audit log
	clientIP := pkg.GetClientIP(r)
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   user.TenantID,
		UserID:     user.ID,
		Action:     "user.sso_login",
		Resource:   "user",
		ResourceID: user.ID,
		Details:    store.JSON(fmt.Sprintf(`{"provider":"%s","email":"%s"}`, provider.Name, user.Email)),
		IP:         clientIP,
	})

	events.Emit("user.sso_login", map[string]string{
		"user_id":     user.ID,
		"provider_id": provider.ID,
		"email":       user.Email,
	})

	// Redirect to frontend with token
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://app.securelens.ai"
	}
	redirectURL := fmt.Sprintf("%s/auth/sso-callback?token=%s", frontendURL, token)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// findOrCreateSSOUser finds an existing user or creates a new one for SSO login
func (s *Server) findOrCreateSSOUser(ctx context.Context, tenantID string, provider store.SSOProvider, subjectID, email, name string) (*store.User, error) {
	// First try to find by SSO subject ID
	var user store.User
	err := s.db.GetContext(ctx, &user,
		`SELECT * FROM users WHERE tenant_id = $1 AND sso_provider_id = $2 AND sso_subject_id = $3`,
		tenantID, provider.ID, subjectID)
	if err == nil {
		return &user, nil
	}

	// Try to find by email
	err = s.db.GetContext(ctx, &user,
		`SELECT * FROM users WHERE tenant_id = $1 AND email = $2`, tenantID, email)
	if err == nil {
		// Link existing user to SSO
		s.db.ExecContext(ctx,
			`UPDATE users SET sso_provider_id = $1, sso_subject_id = $2 WHERE id = $3`,
			provider.ID, subjectID, user.ID)
		return &user, nil
	}

	// Create new user if auto-create is enabled
	if !provider.AutoCreateUsers {
		return nil, fmt.Errorf("user not found and auto-creation is disabled")
	}

	if name == "" {
		name = email
	}

	err = s.db.QueryRowContext(ctx,
		`INSERT INTO users (tenant_id, email, name, status, sso_provider_id, sso_subject_id, password_hash)
		 VALUES ($1, $2, $3, 'active', $4, $5, '')
		 RETURNING id, tenant_id, email, name, status, is_super_admin, created_at, updated_at`,
		tenantID, email, name, provider.ID, subjectID,
	).Scan(&user.ID, &user.TenantID, &user.Email, &user.Name, &user.Status, &user.IsSuperAdmin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Assign default role if specified
	if provider.DefaultRole != "" && provider.DefaultRole != "user" {
		var roleID string
		err = s.db.GetContext(ctx, &roleID,
			"SELECT id FROM roles WHERE tenant_id = $1 AND name = $2", tenantID, provider.DefaultRole)
		if err == nil && roleID != "" {
			s.db.ExecContext(ctx,
				"INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
				user.ID, roleID)
		}
	}

	return &user, nil
}

func redirectToLoginWithError(w http.ResponseWriter, r *http.Request, message string) {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://app.securelens.ai"
	}
	redirectURL := fmt.Sprintf("%s/login?error=%s", frontendURL, message)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// listTenantSSOProviders returns enabled SSO providers for a tenant (public endpoint)
func (s *Server) listTenantSSOProviders(w http.ResponseWriter, r *http.Request) {
	tenantSlug := r.URL.Query().Get("tenant")
	if tenantSlug == "" {
		pkg.Error(w, fmt.Errorf("tenant parameter required"), http.StatusBadRequest)
		return
	}

	var tenantID string
	err := s.db.GetContext(r.Context(), &tenantID,
		"SELECT id FROM tenants WHERE slug = $1", tenantSlug)
	if err != nil {
		pkg.Error(w, fmt.Errorf("tenant not found"), http.StatusNotFound)
		return
	}

	type PublicProvider struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}

	var providers []PublicProvider
	err = s.db.SelectContext(r.Context(), &providers,
		`SELECT id, name, type FROM sso_providers 
		 WHERE tenant_id = $1 AND enabled = true ORDER BY name`, tenantID)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	if providers == nil {
		providers = []PublicProvider{}
	}
	pkg.JSON(w, providers)
}
