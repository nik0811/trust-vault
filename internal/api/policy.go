package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/trustvault/trustvault/internal/events"
	"github.com/trustvault/trustvault/internal/pkg"
	"github.com/trustvault/trustvault/internal/store"
)

func (s *Server) listPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	policies, err := s.policies.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	if err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, policies)
}

func (s *Server) createPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var policy store.Policy
	if err := pkg.Bind(r, &policy); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	policy.TenantID = tenantID
	policy.Active = true

	if err := s.policies.Create(ctx, &policy); err != nil {
		pkg.Error(w, err)
		return
	}

	events.Emit("policy.created", policy)
	
	// Audit log
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		UserID:     pkg.UserFromCtx(ctx),
		Action:     "policy.created",
		Resource:   "policy",
		ResourceID: policy.ID,
		Details:    store.JSON(fmt.Sprintf(`{"name":"%s","type":"%s"}`, policy.Name, policy.Type)),
		IP:         pkg.ClientIPFromCtx(ctx),
	})
	
	pkg.JSON(w, policy, http.StatusCreated)
}

func (s *Server) getPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	policy, err := s.policies.FindByID(ctx, tenantID, id)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	if policy == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	pkg.JSON(w, policy)
}

func (s *Server) updatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	policy, err := s.policies.FindByID(ctx, tenantID, id)
	if err != nil || policy == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	var req store.Policy
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	policy.Name = req.Name
	policy.Description = req.Description
	policy.Conditions = req.Conditions
	policy.Actions = req.Actions
	policy.Regulations = req.Regulations
	policy.Active = req.Active
	policy.Priority = req.Priority

	if err := s.policies.Update(ctx, policy); err != nil {
		pkg.Error(w, err)
		return
	}

	events.Emit("policy.updated", policy)
	
	// Audit log
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		UserID:     pkg.UserFromCtx(ctx),
		Action:     "policy.updated",
		Resource:   "policy",
		ResourceID: policy.ID,
		Details:    store.JSON(fmt.Sprintf(`{"name":"%s"}`, policy.Name)),
		IP:         pkg.ClientIPFromCtx(ctx),
	})
	
	pkg.JSON(w, policy)
}

func (s *Server) deletePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	if err := s.policies.Delete(ctx, tenantID, id); err != nil {
		pkg.Error(w, err)
		return
	}

	events.Emit("policy.deleted", map[string]string{"id": id})
	
	// Audit log
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		UserID:     pkg.UserFromCtx(ctx),
		Action:     "policy.deleted",
		Resource:   "policy",
		ResourceID: id,
		IP:         pkg.ClientIPFromCtx(ctx),
	})
	
	pkg.JSON(w, map[string]string{"status": "deleted"})
}

type EvaluateRequest struct {
	Data       string            `json:"data"`
	Context    map[string]any    `json:"context"`
	PolicyIDs  []string          `json:"policy_ids,omitempty"`
}

type EvaluateResponse struct {
	Decision    string            `json:"decision"` // allow, deny, redact
	Redactions  []Redaction       `json:"redactions,omitempty"`
	Violations  []PolicyViolation `json:"violations,omitempty"`
	AppliedPolicies []string      `json:"applied_policies"`
}

type Redaction struct {
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Type   string `json:"type"`
	Masked string `json:"masked"`
}

type PolicyViolation struct {
	PolicyID   string `json:"policy_id"`
	PolicyName string `json:"policy_name"`
	Reason     string `json:"reason"`
}

func (s *Server) evaluatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req EvaluateRequest
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Get active policies
	policies, _ := s.policies.List(ctx, tenantID, store.ListOpts{Limit: 100})

	response := EvaluateResponse{
		Decision:        "allow",
		AppliedPolicies: []string{},
	}

	for _, policy := range policies {
		if !policy.Active {
			continue
		}
		// Evaluate policy conditions against data
		// This is simplified - real implementation would parse conditions JSON
		response.AppliedPolicies = append(response.AppliedPolicies, policy.ID)
	}

	pkg.JSON(w, response)
}
