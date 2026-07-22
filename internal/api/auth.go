package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/events"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	var user store.User
	err := s.db.GetContext(r.Context(), &user,
		"SELECT * FROM users WHERE email = $1 AND status = 'active'", req.Email)
	if err != nil {
		pkg.Error(w, pkg.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	if !pkg.CheckPassword(req.Password, user.PasswordHash) {
		pkg.Error(w, pkg.ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	var permissions []string
	rows, _ := s.db.QueryContext(r.Context(),
		`SELECT r.permissions FROM roles r 
		 JOIN user_roles ur ON ur.role_id = r.id 
		 WHERE ur.user_id = $1`, user.ID)
	defer rows.Close()
	for rows.Next() {
		var perms store.JSON
		rows.Scan(&perms)
		// Parse and merge permissions
	}

	token, err := pkg.GenerateToken(user.ID, user.TenantID, permissions, user.IsSuperAdmin)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	s.db.ExecContext(r.Context(), "UPDATE users SET last_login_at = $1 WHERE id = $2", time.Now(), user.ID)
	
	// Audit log for login
	clientIP := pkg.GetClientIP(r)
	s.auditLogs.Create(r.Context(), &store.AuditLog{
		TenantID:   user.TenantID,
		UserID:     user.ID,
		Action:     "user.login",
		Resource:   "user",
		ResourceID: user.ID,
		Details:    store.JSON(fmt.Sprintf(`{"email":"%s"}`, user.Email)),
		IP:         clientIP,
	})

	pkg.JSON(w, LoginResponse{
		AccessToken:  token,
		RefreshToken: token, // Simplified - in production use separate refresh token
		ExpiresIn:    86400,
	})
}

func (s *Server) refreshToken(w http.ResponseWriter, r *http.Request) {
	// Token refresh logic
	pkg.JSON(w, map[string]string{"status": "ok"})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	// Invalidate token logic
	pkg.JSON(w, map[string]string{"status": "ok"})
}

func (s *Server) createAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	userID := pkg.UserFromCtx(ctx)

	var req struct {
		Name        string   `json:"name" validate:"required"`
		Permissions []string `json:"permissions"`
		ExpiresIn   int      `json:"expires_in"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Default to 30 days if not specified
	expiresInDays := req.ExpiresIn
	if expiresInDays <= 0 {
		expiresInDays = 30
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, expiresInDays)

	// Generate a secure API key: sl_<32 random hex chars>
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		pkg.Error(w, fmt.Errorf("failed to generate key: %w", err))
		return
	}
	plainKey := "sl_" + hex.EncodeToString(keyBytes)
	prefix := plainKey[:10] // Store prefix for display (sl_XXXXXX)

	// Hash the key for storage
	keyHash, err := pkg.HashPassword(plainKey)
	if err != nil {
		pkg.Error(w, fmt.Errorf("failed to hash key: %w", err))
		return
	}

	var id string
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO api_keys (id, tenant_id, user_id, key_hash, key_prefix, name, permissions, expires_at, created_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8) 
		 RETURNING id`,
		tenantID, userID, keyHash, prefix, req.Name, "{}", expiresAt, now,
	).Scan(&id)

	if err != nil {
		pkg.Error(w, err)
		return
	}

	// Return the plain key - this is the only time it will be visible
	pkg.JSON(w, map[string]any{
		"id":         id,
		"key":        plainKey,
		"name":       req.Name,
		"prefix":     prefix,
		"expires_at": expiresAt,
		"created_at": now,
	}, http.StatusCreated)
}

func (s *Server) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	_, err := s.db.ExecContext(ctx, "DELETE FROM api_keys WHERE tenant_id = $1 AND id = $2", tenantID, id)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	pkg.JSON(w, map[string]string{"status": "deleted"})
}

func (s *Server) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	type APIKeyResponse struct {
		ID        string     `db:"id" json:"id"`
		Name      string     `db:"name" json:"name"`
		Prefix    *string    `db:"key_prefix" json:"prefix"`
		UserID    string     `db:"user_id" json:"user_id"`
		ExpiresAt time.Time  `db:"expires_at" json:"expires_at"`
		LastUsed  *time.Time `db:"last_used_at" json:"last_used"`
		CreatedAt time.Time  `db:"created_at" json:"created_at"`
	}

	var keys []APIKeyResponse
	err := s.db.SelectContext(ctx, &keys,
		`SELECT id, name, COALESCE(key_prefix, '') AS key_prefix, user_id, expires_at, last_used_at, created_at 
		 FROM api_keys WHERE tenant_id = $1 
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	if keys == nil {
		keys = []APIKeyResponse{}
	}

	pkg.JSON(w, keys)
}

// User handlers
func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	users, err := s.users.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	if err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, users)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
		Name     string `json:"name"`
		RoleID   string `json:"role_id"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	hash, _ := pkg.HashPassword(req.Password)
	user := store.User{
		TenantID:     tenantID,
		Email:        req.Email,
		PasswordHash: hash,
		Name:         req.Name,
		Status:       "active",
	}

	if err := s.users.Create(ctx, &user); err != nil {
		pkg.Error(w, err)
		return
	}

	events.Emit("user.created", user)
	pkg.JSON(w, user, http.StatusCreated)
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	user, err := s.users.FindByID(ctx, tenantID, id)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	if user == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	pkg.JSON(w, user)
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	user, err := s.users.FindByID(ctx, tenantID, id)
	if err != nil || user == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	var req struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Status != "" {
		user.Status = req.Status
	}

	if err := s.users.Update(ctx, user); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, user)
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	if err := s.users.Delete(ctx, tenantID, id); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, map[string]string{"status": "deleted"})
}

// Role handlers
func (s *Server) listRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	roles, err := s.roles.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	if err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, roles)
}

func (s *Server) createRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var role store.Role
	if err := pkg.Bind(r, &role); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	role.TenantID = tenantID
	if err := s.roles.Create(ctx, &role); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, role, http.StatusCreated)
}

func (s *Server) updateRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	role, err := s.roles.FindByID(ctx, tenantID, id)
	if err != nil || role == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	if role.IsSystem {
		pkg.Error(w, pkg.ErrForbidden, http.StatusForbidden)
		return
	}

	var req store.Role
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	role.Name = req.Name
	role.Description = req.Description
	role.Permissions = req.Permissions

	if err := s.roles.Update(ctx, role); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, role)
}

// Invitation handlers

func generateSecureToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type CreateInvitationRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required"`
}

type InvitationResponse struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	InvitedBy  *string    `json:"invited_by,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	InviteURL  string     `json:"invite_url,omitempty"`
}

func (s *Server) createInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	userID := pkg.UserFromCtx(ctx)
	isSuperAdmin := pkg.IsSuperAdminFromCtx(ctx)

	var req CreateInvitationRequest
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// RBAC: superadmin can invite tenant_admin, tenant_admin can invite other roles
	if req.Role == "tenant_admin" && !isSuperAdmin {
		pkg.Error(w, errors.New("only superadmin can invite tenant admins"), http.StatusForbidden)
		return
	}

	// Check if user already exists
	var existingCount int
	s.db.GetContext(ctx, &existingCount, "SELECT COUNT(*) FROM users WHERE email = $1", req.Email)
	if existingCount > 0 {
		pkg.Error(w, errors.New("user with this email already exists"), http.StatusConflict)
		return
	}

	// Check if pending invitation exists
	var pendingCount int
	s.db.GetContext(ctx, &pendingCount,
		"SELECT COUNT(*) FROM invitations WHERE email = $1 AND accepted_at IS NULL AND expires_at > NOW()",
		req.Email)
	if pendingCount > 0 {
		pkg.Error(w, errors.New("pending invitation already exists for this email"), http.StatusConflict)
		return
	}

	token := generateSecureToken()
	expiresAt := time.Now().AddDate(0, 0, 7) // 7 days

	var invitation store.Invitation
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO invitations (tenant_id, email, role, token, invited_by, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW()) 
		 RETURNING id, tenant_id, email, role, invited_by, expires_at, created_at`,
		tenantID, req.Email, req.Role, token, userID, expiresAt,
	).Scan(&invitation.ID, &invitation.TenantID, &invitation.Email, &invitation.Role,
		&invitation.InvitedBy, &invitation.ExpiresAt, &invitation.CreatedAt)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	events.Emit("invitation.created", invitation)

	pkg.JSON(w, InvitationResponse{
		ID:        invitation.ID,
		TenantID:  invitation.TenantID,
		Email:     invitation.Email,
		Role:      invitation.Role,
		InvitedBy: invitation.InvitedBy,
		ExpiresAt: invitation.ExpiresAt,
		CreatedAt: invitation.CreatedAt,
		InviteURL: "/register?token=" + token,
	}, http.StatusCreated)
}

func (s *Server) listInvitations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	var invitations []store.Invitation
	err := s.db.SelectContext(ctx, &invitations,
		`SELECT id, tenant_id, email, role, invited_by, expires_at, accepted_at, created_at 
		 FROM invitations WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	pkg.JSON(w, invitations)
}

func (s *Server) cancelInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	result, err := s.db.ExecContext(ctx,
		"DELETE FROM invitations WHERE id = $1 AND tenant_id = $2 AND accepted_at IS NULL",
		id, tenantID)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	pkg.JSON(w, map[string]string{"status": "cancelled"})
}

func (s *Server) verifyInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := chi.URLParam(r, "token")

	var invitation store.Invitation
	err := s.db.GetContext(ctx, &invitation,
		`SELECT i.id, i.tenant_id, i.email, i.role, i.expires_at, i.accepted_at, i.created_at,
		        t.name as tenant_name
		 FROM invitations i
		 JOIN tenants t ON t.id = i.tenant_id
		 WHERE i.token = $1`, token)
	if err != nil {
		pkg.Error(w, errors.New("invalid invitation token"), http.StatusNotFound)
		return
	}

	if invitation.AcceptedAt != nil {
		pkg.Error(w, errors.New("invitation already accepted"), http.StatusBadRequest)
		return
	}

	if time.Now().After(invitation.ExpiresAt) {
		pkg.Error(w, errors.New("invitation has expired"), http.StatusBadRequest)
		return
	}

	// Get tenant name
	var tenantName string
	s.db.GetContext(ctx, &tenantName, "SELECT name FROM tenants WHERE id = $1", invitation.TenantID)

	pkg.JSON(w, map[string]any{
		"id":          invitation.ID,
		"email":       invitation.Email,
		"role":        invitation.Role,
		"tenant_id":   invitation.TenantID,
		"tenant_name": tenantName,
		"expires_at":  invitation.ExpiresAt,
	})
}

type RegisterRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name"`
}

func (s *Server) registerWithInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RegisterRequest
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Verify invitation
	var invitation store.Invitation
	err := s.db.GetContext(ctx, &invitation,
		`SELECT id, tenant_id, email, role, expires_at, accepted_at 
		 FROM invitations WHERE token = $1`, req.Token)
	if err != nil {
		pkg.Error(w, errors.New("invalid invitation token"), http.StatusNotFound)
		return
	}

	if invitation.AcceptedAt != nil {
		pkg.Error(w, errors.New("invitation already accepted"), http.StatusBadRequest)
		return
	}

	if time.Now().After(invitation.ExpiresAt) {
		pkg.Error(w, errors.New("invitation has expired"), http.StatusBadRequest)
		return
	}

	// Create user
	hash, err := pkg.HashPassword(req.Password)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	name := req.Name
	if name == "" {
		name = invitation.Email
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	defer tx.Rollback()

	var userID string
	err = tx.QueryRowContext(ctx,
		`INSERT INTO users (tenant_id, email, password_hash, name, status)
		 VALUES ($1, $2, $3, $4, 'active') RETURNING id`,
		invitation.TenantID, invitation.Email, hash, name,
	).Scan(&userID)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	// Mark invitation as accepted
	_, err = tx.ExecContext(ctx,
		"UPDATE invitations SET accepted_at = NOW() WHERE id = $1",
		invitation.ID)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	// Assign role if specified
	if invitation.Role != "" && invitation.Role != "user" {
		var roleID string
		err = tx.GetContext(ctx, &roleID,
			"SELECT id FROM roles WHERE tenant_id = $1 AND name = $2",
			invitation.TenantID, invitation.Role)
		if err == nil && roleID != "" {
			tx.ExecContext(ctx,
				"INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)",
				userID, roleID)
		}
	}

	if err := tx.Commit(); err != nil {
		pkg.Error(w, err)
		return
	}

	events.Emit("user.registered", map[string]string{
		"user_id":       userID,
		"email":         invitation.Email,
		"invitation_id": invitation.ID,
	})

	pkg.JSON(w, map[string]string{
		"status":  "registered",
		"user_id": userID,
		"email":   invitation.Email,
	}, http.StatusCreated)
}

func (s *Server) resendInvitation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	// Generate new token and extend expiry
	newToken := generateSecureToken()
	newExpiry := time.Now().AddDate(0, 0, 7)

	result, err := s.db.ExecContext(ctx,
		`UPDATE invitations SET token = $1, expires_at = $2 
		 WHERE id = $3 AND tenant_id = $4 AND accepted_at IS NULL`,
		newToken, newExpiry, id, tenantID)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	pkg.JSON(w, map[string]string{
		"status":     "resent",
		"invite_url": "/register?token=" + newToken,
	})
}
