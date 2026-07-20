package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/events"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
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
	Decision        string            `json:"decision"` // allow, deny, redact
	Redactions      []Redaction       `json:"redactions,omitempty"`
	Violations      []PolicyViolation `json:"violations,omitempty"`
	AppliedPolicies []string          `json:"applied_policies"`
	RedactedData    string            `json:"redacted_data,omitempty"`
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

	// Step 1: Classify the input data to find PII
	classificationResults := s.runBasicClassification(req.Data, nil)

	// Step 2: Get active policies sorted by priority
	policies, _ := s.policies.List(ctx, tenantID, store.ListOpts{Limit: 100})

	response := EvaluateResponse{
		Decision:        "allow",
		AppliedPolicies: []string{},
		Redactions:      []Redaction{},
		Violations:      []PolicyViolation{},
	}

	// Build a map of detected entity types
	detectedTypes := make(map[string]bool)
	for _, result := range classificationResults {
		if entityType, ok := result["entity_type"].(string); ok {
			detectedTypes[entityType] = true
		}
	}

	// Step 3: Evaluate each active policy
	redactedData := req.Data
	for _, policy := range policies {
		if !policy.Active {
			continue
		}

		// Check if policy applies based on type
		switch policy.Type {
		case "access":
			// Access policies can deny based on detected PII types
			if len(detectedTypes) > 0 {
				// Check if policy conditions match detected types
				for entityType := range detectedTypes {
					if isHighRiskPII(entityType) {
						response.Violations = append(response.Violations, PolicyViolation{
							PolicyID:   policy.ID,
							PolicyName: policy.Name,
							Reason:     fmt.Sprintf("High-risk PII detected: %s", entityType),
						})
						response.Decision = "deny"
					}
				}
			}
			response.AppliedPolicies = append(response.AppliedPolicies, policy.Name)

		case "redaction":
			// Redaction policies mask sensitive data
			for _, result := range classificationResults {
				value, _ := result["value"].(string)
				entityType, _ := result["entity_type"].(string)
				start, _ := result["start"].(int)
				end, _ := result["end"].(int)
				
				masked := maskValue(value, entityType)
				redaction := Redaction{
					Start:  start,
					End:    end,
					Type:   entityType,
					Masked: masked,
				}
				response.Redactions = append(response.Redactions, redaction)
				// Apply redaction to the data
				if start >= 0 && end <= len(redactedData) && start < end {
					redactedData = redactedData[:start] + masked + redactedData[end:]
				}
			}
			if len(response.Redactions) > 0 {
				response.Decision = "redact"
			}
			response.AppliedPolicies = append(response.AppliedPolicies, policy.Name)

		case "ai":
			// AI policies control what data can be sent to LLMs
			for entityType := range detectedTypes {
				if isHighRiskPII(entityType) {
					response.Violations = append(response.Violations, PolicyViolation{
						PolicyID:   policy.ID,
						PolicyName: policy.Name,
						Reason:     fmt.Sprintf("AI policy blocks %s from LLM access", entityType),
					})
				}
			}
			response.AppliedPolicies = append(response.AppliedPolicies, policy.Name)

		case "retention":
			// Retention policies don't affect real-time evaluation
			response.AppliedPolicies = append(response.AppliedPolicies, policy.Name)
		}
	}

	// If no policies matched but PII was found, apply default redaction
	if len(response.AppliedPolicies) == 0 && len(classificationResults) > 0 {
		response.Decision = "redact"
		for _, result := range classificationResults {
			value, _ := result["value"].(string)
			entityType, _ := result["entity_type"].(string)
			start, _ := result["start"].(int)
			end, _ := result["end"].(int)
			
			masked := maskValue(value, entityType)
			response.Redactions = append(response.Redactions, Redaction{
				Start:  start,
				End:    end,
				Type:   entityType,
				Masked: masked,
			})
			if start >= 0 && end <= len(redactedData) && start < end {
				redactedData = redactedData[:start] + masked + redactedData[end:]
			}
		}
		response.AppliedPolicies = append(response.AppliedPolicies, "Default PII Protection")
	}

	// Add redacted data to response if any redactions were made
	if len(response.Redactions) > 0 {
		response.RedactedData = redactedData
	}

	pkg.JSON(w, response)
}

// isHighRiskPII returns true for PII types that are considered high risk
func isHighRiskPII(entityType string) bool {
	highRisk := map[string]bool{
		"SSN":               true,
		"CREDIT_CARD":       true,
		"CREDIT_CARD_FORMATTED": true,
		"PASSPORT":          true,
		"DRIVER_LICENSE":    true,
		"BANK_ACCOUNT":      true,
		"IBAN":              true,
		"MEDICAL_RECORD":    true,
		"HEALTH_INSURANCE_ID": true,
	}
	return highRisk[entityType]
}

// maskValue masks a PII value based on its type
func maskValue(value, entityType string) string {
	if len(value) <= 4 {
		return "****"
	}
	switch entityType {
	case "EMAIL":
		parts := strings.Split(value, "@")
		if len(parts) == 2 {
			masked := parts[0][:min(3, len(parts[0]))] + "***@" + parts[1]
			return masked
		}
	case "SSN":
		return "***-**-" + value[len(value)-4:]
	case "CREDIT_CARD", "CREDIT_CARD_FORMATTED":
		clean := strings.ReplaceAll(strings.ReplaceAll(value, "-", ""), " ", "")
		if len(clean) >= 4 {
			return "****-****-****-" + clean[len(clean)-4:]
		}
	case "PHONE":
		if len(value) >= 4 {
			return "***-***-" + value[len(value)-4:]
		}
	}
	// Default masking: show first 2 and last 2 chars
	if len(value) > 4 {
		return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
	}
	return "****"
}

// getGovernanceStats returns governance statistics for the dashboard
func (s *Server) getGovernanceStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	// Count policies
	var totalPolicies, activePolicies int
	s.db.GetContext(ctx, &totalPolicies, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &activePolicies, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true", tenantID)

	// Count policy evaluations from audit logs (gate queries that applied policies)
	var evaluationCount int
	s.db.GetContext(ctx, &evaluationCount,
		`SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1 AND created_at >= NOW() - INTERVAL '24 hours'`,
		tenantID)

	// Count evaluations from audit logs
	var auditEvalCount int
	s.db.GetContext(ctx, &auditEvalCount,
		`SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action LIKE 'policy%' AND created_at >= NOW() - INTERVAL '24 hours'`,
		tenantID)

	totalEvaluations := evaluationCount + auditEvalCount

	pkg.JSON(w, map[string]any{
		"total_policies":       totalPolicies,
		"active_policies":      activePolicies,
		"evaluations_24h":      totalEvaluations,
		"evaluation_status":    "active",
	})
}
