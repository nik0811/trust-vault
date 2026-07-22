package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

func (s *Server) getQualityScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasetID := chi.URLParam(r, "id")

	var score store.QualityScore
	err := s.db.GetContext(ctx, &score,
		"SELECT * FROM quality_scores WHERE tenant_id = $1 AND dataset_id = $2 ORDER BY created_at DESC LIMIT 1",
		tenantID, datasetID)
	if err != nil {
		pkg.JSON(w, map[string]any{
			"dataset_id":   datasetID,
			"overall":      0.0,
			"completeness": 0.0,
			"accuracy":     0.0,
			"consistency":  0.0,
			"timeliness":   0.0,
			"uniqueness":   0.0,
			"status":       "not_assessed",
		})
		return
	}
	pkg.JSON(w, score)
}

func (s *Server) getQualityIssues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasetID := chi.URLParam(r, "id")

	var score store.QualityScore
	s.db.GetContext(ctx, &score,
		"SELECT issues FROM quality_scores WHERE tenant_id = $1 AND dataset_id = $2 ORDER BY created_at DESC LIMIT 1",
		tenantID, datasetID)

	pkg.JSON(w, map[string]any{"issues": score.Issues})
}

func (s *Server) assessQuality(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		DatasetID string `json:"dataset_id" validate:"required"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	s.kafka.Produce(ctx, "quality-jobs", tenantID, map[string]any{
		"dataset_id": req.DatasetID,
		"tenant_id":  tenantID,
	})

	pkg.JSON(w, map[string]string{"status": "queued"})
}

func (s *Server) getQualityTrends(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var trends []struct {
		Date    time.Time `db:"date" json:"date"`
		Overall float64   `db:"overall" json:"overall"`
	}
	s.db.SelectContext(ctx, &trends,
		`SELECT DATE(created_at) as date, AVG(overall) as overall 
		 FROM quality_scores WHERE tenant_id = $1 
		 GROUP BY DATE(created_at) ORDER BY date DESC LIMIT 30`,
		tenantID)

	// If no data, return a proper structure with empty array and metadata
	if trends == nil || len(trends) == 0 {
		pkg.JSON(w, map[string]any{
			"trends":     []any{},
			"total":      0,
			"period":     "30d",
			"message":    "No quality assessments have been run yet. Run a quality assessment to see trends.",
			"has_data":   false,
		})
		return
	}

	pkg.JSON(w, map[string]any{
		"trends":   trends,
		"total":    len(trends),
		"period":   "30d",
		"has_data": true,
	})
}

func (s *Server) setQualityThresholds(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Dimension string  `json:"dimension" validate:"required"`
		Minimum   float64 `json:"minimum" validate:"required"`
		Severity  string  `json:"severity"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	policy := store.Policy{
		TenantID: tenantID,
		Name:     "Quality threshold: " + req.Dimension,
		Type:     "quality_threshold",
		Active:   true,
	}
	s.policies.Create(ctx, &policy)

	pkg.JSON(w, map[string]string{"status": "saved"})
}

func (s *Server) createDSAR(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		SubjectID string `json:"subject_id" validate:"required"`
		Type      string `json:"type" validate:"required,oneof=access delete rectify"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	dsar := store.DSAR{
		TenantID:  tenantID,
		SubjectID: req.SubjectID,
		Type:      req.Type,
		Status:    "pending",
		Deadline:  time.Now().AddDate(0, 0, 30),
	}
	s.dsars.Create(ctx, &dsar)

	pkg.JSON(w, dsar, http.StatusCreated)
}

func (s *Server) listDSARs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	dsars, _ := s.dsars.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, dsars)
}

func (s *Server) getDSAR(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	dsar, _ := s.dsars.FindByID(ctx, tenantID, id)
	if dsar == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	pkg.JSON(w, dsar)
}

func (s *Server) updateDSAR(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	dsar, err := s.dsars.FindByID(ctx, tenantID, id)
	if err != nil || dsar == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	if req.Status != "" {
		dsar.Status = req.Status
		if req.Status == "completed" {
			now := time.Now()
			dsar.CompletedAt = &now
		}
	}

	if err := s.dsars.Update(ctx, dsar); err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	pkg.JSON(w, dsar)
}

func (s *Server) deleteDSAR(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	dsar, err := s.dsars.FindByID(ctx, tenantID, id)
	if err != nil || dsar == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	if err := s.dsars.Delete(ctx, tenantID, id); err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getDSARPackage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	dsar, _ := s.dsars.FindByID(ctx, tenantID, id)
	if dsar == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	pkg.JSON(w, map[string]any{
		"dsar_id":    id,
		"subject_id": dsar.SubjectID,
		"data":       dsar.Results,
		"generated":  time.Now(),
	})
}

// DSARResult represents a single data finding for a subject
type DSARResult struct {
	SourceID     string         `json:"source_id"`
	SourceName   string         `json:"source_name"`
	SourceType   string         `json:"source_type"`
	DatasetID    string         `json:"dataset_id"`
	EntityType   string         `json:"entity_type"`
	Value        string         `json:"value"`
	Context      map[string]any `json:"context,omitempty"`
	FoundAt      time.Time      `json:"found_at"`
}

// DSARPackage is the complete response package for a DSAR
type DSARPackage struct {
	DSARID      string       `json:"dsar_id"`
	SubjectID   string       `json:"subject_id"`
	Type        string       `json:"type"`
	Status      string       `json:"status"`
	DataFound   []DSARResult `json:"data_found"`
	SourceCount int          `json:"source_count"`
	RecordCount int          `json:"record_count"`
	GeneratedAt time.Time    `json:"generated_at"`
}

func (s *Server) executeDSAR(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	// Get the DSAR
	dsar, err := s.dsars.FindByID(ctx, tenantID, id)
	if err != nil || dsar == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	// Update status to processing
	dsar.Status = "processing"
	s.dsars.Update(ctx, dsar)

	// Get all datasources for tenant
	sources, err := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: 1000})
	if err != nil {
		pkg.Error(w, err)
		return
	}

	var results []DSARResult

	// Search classifications for subject data
	// Look for classifications that match the subject identifier (email, name, ID, etc.)
	var classifications []store.Classification
	err = s.db.SelectContext(ctx, &classifications,
		`SELECT * FROM classifications 
		 WHERE tenant_id = $1 
		 AND (value ILIKE $2 OR value ILIKE $3)
		 ORDER BY created_at DESC 
		 LIMIT 1000`,
		tenantID, "%"+dsar.SubjectID+"%", dsar.SubjectID)
	if err == nil {
		for _, c := range classifications {
			// Find the source for this classification
			var sourceName, sourceType string
			cSourceID := pkg.DerefStr(c.SourceID)
			for _, src := range sources {
				if src.ID == cSourceID {
					sourceName = src.Name
					sourceType = src.Type
					break
				}
			}

			var ctxMap map[string]any
			if len(c.Context) > 0 {
				json.Unmarshal(c.Context, &ctxMap)
			}

			results = append(results, DSARResult{
				SourceID:   cSourceID,
				SourceName: sourceName,
				SourceType: sourceType,
				DatasetID:  c.DatasetID,
				EntityType: c.EntityType,
				Value:      c.Value,
				Context:    ctxMap,
				FoundAt:    c.CreatedAt,
			})
		}
	}

	// Also search audit logs for subject activity
	var auditLogs []store.AuditLog
	err = s.db.SelectContext(ctx, &auditLogs,
		`SELECT * FROM audit_logs 
		 WHERE tenant_id = $1 
		 AND (resource_id = $2 OR details::text ILIKE $3)
		 ORDER BY created_at DESC 
		 LIMIT 100`,
		tenantID, dsar.SubjectID, "%"+dsar.SubjectID+"%")
	if err == nil {
		for _, log := range auditLogs {
			var details map[string]any
			if len(log.Details) > 0 {
				json.Unmarshal(log.Details, &details)
			}
			results = append(results, DSARResult{
				SourceID:   "audit_logs",
				SourceName: "Audit Trail",
				SourceType: "system",
				DatasetID:  log.ID,
				EntityType: "ACTIVITY",
				Value:      log.Action + " on " + log.Resource,
				Context:    details,
				FoundAt:    log.CreatedAt,
			})
		}
	}

	// Build the package
	dsarPkg := DSARPackage{
		DSARID:      id,
		SubjectID:   dsar.SubjectID,
		Type:        dsar.Type,
		Status:      "completed",
		DataFound:   results,
		SourceCount: len(sources),
		RecordCount: len(results),
		GeneratedAt: time.Now(),
	}

	// Store results in DSAR
	resultsJSON, _ := json.Marshal(results)
	dsar.Results = store.JSON(resultsJSON)
	dsar.Status = "completed"
	now := time.Now()
	dsar.CompletedAt = &now
	s.dsars.Update(ctx, dsar)

	// Record audit log
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		Action:     "dsar_executed",
		Resource:   "dsar",
		ResourceID: id,
	})

	pkg.JSON(w, dsarPkg)
}

func (s *Server) generatePIA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		DatasetID string `json:"dataset_id" validate:"required"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	var piiCount int
	s.db.GetContext(ctx, &piiCount,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND dataset_id = $2 AND entity_type IN ('PII', 'SSN', 'EMAIL', 'PHONE')",
		tenantID, req.DatasetID)

	riskScore := 0.3
	if piiCount > 10 {
		riskScore = 0.8
	} else if piiCount > 5 {
		riskScore = 0.6
	} else if piiCount > 0 {
		riskScore = 0.4
	}

	riskLevel := "low"
	if riskScore >= 0.7 {
		riskLevel = "high"
	} else if riskScore >= 0.4 {
		riskLevel = "medium"
	}

	recommendations := []string{}
	if piiCount > 0 {
		recommendations = append(recommendations, "Implement data minimization")
		recommendations = append(recommendations, "Add encryption at rest")
	}
	if riskScore >= 0.5 {
		recommendations = append(recommendations, "Review retention policies")
		recommendations = append(recommendations, "Implement access controls")
	}

	pkg.JSON(w, map[string]any{
		"dataset_id":      req.DatasetID,
		"tenant_id":       tenantID,
		"risk_score":      riskScore,
		"risk_level":      riskLevel,
		"pii_count":       piiCount,
		"generated":       time.Now(),
		"recommendations": recommendations,
	})
}

func (s *Server) getPIA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasetID := chi.URLParam(r, "dataset_id")

	var piiCount int
	s.db.GetContext(ctx, &piiCount,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND dataset_id = $2 AND entity_type IN ('PII', 'SSN', 'EMAIL', 'PHONE')",
		tenantID, datasetID)

	riskScore := 0.3
	if piiCount > 10 {
		riskScore = 0.8
	} else if piiCount > 5 {
		riskScore = 0.6
	} else if piiCount > 0 {
		riskScore = 0.4
	}

	pkg.JSON(w, map[string]any{
		"dataset_id": datasetID,
		"risk_score": riskScore,
		"pii_count":  piiCount,
		"status":     "completed",
	})
}

func (s *Server) listRoPA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	ropa, _ := s.ropa.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, ropa)
}

func (s *Server) createRoPA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Name             string   `json:"name" validate:"required"`
		Purpose          string   `json:"purpose"`
		LegalBasis       string   `json:"legal_basis"`
		DataCategories   []string `json:"data_categories"`
		Recipients       []string `json:"recipients"`
		RetentionPeriod  string   `json:"retention_period"`
		SecurityMeasures string   `json:"security_measures"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	ropa := store.RoPA{
		TenantID:         tenantID,
		Name:             req.Name,
		Purpose:          req.Purpose,
		LegalBasis:       req.LegalBasis,
		RetentionPeriod:  req.RetentionPeriod,
		SecurityMeasures: req.SecurityMeasures,
	}
	s.ropa.Create(ctx, &ropa)

	pkg.JSON(w, ropa, http.StatusCreated)
}

func (s *Server) recordConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		SubjectID string   `json:"subject_id" validate:"required"`
		Purposes  []string `json:"purposes"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	log := store.AuditLog{
		TenantID: tenantID,
		Action:   "consent_recorded",
		Resource: "consent",
	}
	s.auditLogs.Create(ctx, &log)

	pkg.JSON(w, map[string]string{"status": "recorded"})
}

func (s *Server) withdrawConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	subjectID := chi.URLParam(r, "subject_id")

	log := store.AuditLog{
		TenantID:   tenantID,
		Action:     "consent_withdrawn",
		Resource:   "consent",
		ResourceID: subjectID,
	}
	s.auditLogs.Create(ctx, &log)

	pkg.JSON(w, map[string]string{"status": "withdrawn"})
}

func (s *Server) getRetentionViolations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	violations, _ := s.retentionViolations.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, violations)
}

func (s *Server) setRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Name           string `json:"name" validate:"required"`
		Classification string `json:"classification"`
		RetentionDays  int    `json:"retention_days" validate:"required"`
		Action         string `json:"action"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	policy := store.RetentionPolicy{
		TenantID:       tenantID,
		Name:           req.Name,
		Classification: req.Classification,
		RetentionDays:  req.RetentionDays,
		Action:         req.Action,
		Active:         true,
	}
	s.retentionPolicies.Create(ctx, &policy)

	pkg.JSON(w, policy, http.StatusCreated)
}

func (s *Server) getAuditTrail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	isSuperAdmin := pkg.IsSuperAdminFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	var logs []store.AuditLog
	var err error

	if isSuperAdmin && tenantID == "" {
		// Superadmin can see all audit logs across tenants
		err = s.db.SelectContext(ctx, &logs,
			`SELECT id, tenant_id, COALESCE(user_id::text, '') as user_id, action, resource, COALESCE(resource_id::text, '') as resource_id, details, COALESCE(ip, '') as ip, created_at FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
			limit, offset)
	} else {
		err = s.db.SelectContext(ctx, &logs,
			`SELECT id, tenant_id, COALESCE(user_id::text, '') as user_id, action, resource, COALESCE(resource_id::text, '') as resource_id, details, COALESCE(ip, '') as ip, created_at FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			tenantID, limit, offset)
	}

	if err != nil {
		pkg.Error(w, err)
		return
	}

	if logs == nil {
		logs = []store.AuditLog{}
	}

	pkg.JSON(w, logs)
}

func (s *Server) getAIUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasetID := chi.URLParam(r, "id")

	var usage []store.GateQuery
	s.db.SelectContext(ctx, &usage,
		"SELECT * FROM gate_queries WHERE tenant_id = $1 AND context::text LIKE $2 ORDER BY created_at DESC LIMIT 100",
		tenantID, "%"+datasetID+"%")

	pkg.JSON(w, usage)
}

func (s *Server) getComplianceReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var policyCount, dsarCount, ropaCount int
	s.db.GetContext(ctx, &policyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true", tenantID)
	s.db.GetContext(ctx, &dsarCount, "SELECT COUNT(*) FROM dsars WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &ropaCount, "SELECT COUNT(*) FROM ropa WHERE tenant_id = $1", tenantID)

	var violations []store.RetentionViolation
	s.db.SelectContext(ctx, &violations,
		"SELECT * FROM retention_violations WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 10", tenantID)

	gdprScore := 0.7
	if ropaCount > 0 {
		gdprScore += 0.1
	}
	if policyCount >= 3 {
		gdprScore += 0.1
	}
	if len(violations) == 0 {
		gdprScore += 0.1
	}

	pkg.JSON(w, map[string]any{
		"tenant_id": tenantID,
		"generated": time.Now(),
		"compliance": map[string]float64{
			"gdpr":  min(1.0, gdprScore),
			"ccpa":  min(1.0, gdprScore+0.05),
			"hipaa": min(1.0, gdprScore-0.1),
		},
		"issues":   violations,
		"policies": policyCount,
		"dsars":    dsarCount,
		"ropa":     ropaCount,
	})
}

func (s *Server) getLineageSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	type LineageNode struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}
	type LineageEdge struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Label  string `json:"label"`
	}

	sources, _ := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: 100})
	nodes := make([]LineageNode, 0, len(sources))
	sourceIDs := map[string]bool{}
	for _, ds := range sources {
		nodes = append(nodes, LineageNode{ID: ds.ID, Name: ds.Name, Type: ds.Type})
		sourceIDs[ds.ID] = true
	}

	// Build edges: datasource → classified dataset (distinct dataset_ids from classifications)
	var classRows []struct {
		DatasetID string `db:"dataset_id"`
		SourceID  string `db:"source_id"`
	}
	s.db.SelectContext(ctx, &classRows,
		`SELECT DISTINCT dataset_id, COALESCE(source_id::text,'') AS source_id
		 FROM classifications WHERE tenant_id = $1 AND dataset_id IS NOT NULL AND dataset_id <> ''
		 LIMIT 200`, tenantID)

	edges := make([]LineageEdge, 0)
	seenDatasets := map[string]bool{}
	for _, row := range classRows {
		if !seenDatasets[row.DatasetID] {
			seenDatasets[row.DatasetID] = true
			if !sourceIDs[row.DatasetID] {
				nodes = append(nodes, LineageNode{ID: row.DatasetID, Name: row.DatasetID, Type: "dataset"})
			}
		}
		if row.SourceID != "" && row.SourceID != row.DatasetID {
			edges = append(edges, LineageEdge{Source: row.SourceID, Target: row.DatasetID, Label: "has_data"})
		}
	}

	// Add model lineage edges
	var mlRows []struct {
		DatasetID string `db:"dataset_id"`
		ModelID   string `db:"model_id"`
	}
	s.db.SelectContext(ctx, &mlRows,
		"SELECT DISTINCT dataset_id, model_id FROM model_lineage WHERE tenant_id = $1 LIMIT 200", tenantID)

	modelIDs := map[string]bool{}
	for _, row := range mlRows {
		edges = append(edges, LineageEdge{Source: row.DatasetID, Target: row.ModelID, Label: "used_by"})
		if !modelIDs[row.ModelID] {
			modelIDs[row.ModelID] = true
			nodes = append(nodes, LineageNode{ID: row.ModelID, Name: row.ModelID, Type: "model"})
		}
	}

	// Add AI gate query lineage: datasource → gate → endpoint
	var gateRows []struct {
		LLMEndpoint string `db:"llm_endpoint"`
	}
	s.db.SelectContext(ctx, &gateRows,
		`SELECT DISTINCT COALESCE(llm_endpoint,'') AS llm_endpoint FROM gate_queries
		 WHERE tenant_id = $1 AND llm_endpoint IS NOT NULL AND llm_endpoint <> '' LIMIT 20`, tenantID)
	for _, row := range gateRows {
		nodes = append(nodes, LineageNode{ID: "gate-" + row.LLMEndpoint, Name: row.LLMEndpoint, Type: "llm_endpoint"})
		edges = append(edges, LineageEdge{Source: "ai-gate", Target: "gate-" + row.LLMEndpoint, Label: "routes_to"})
	}
	if len(gateRows) > 0 {
		nodes = append(nodes, LineageNode{ID: "ai-gate", Name: "AI Gate", Type: "gateway"})
	}

	pkg.JSON(w, map[string]any{"nodes": nodes, "edges": edges})
}

func (s *Server) getLineage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasetID := chi.URLParam(r, "dataset_id")

	var upstream []store.DataFlow
	s.db.SelectContext(ctx, &upstream,
		"SELECT * FROM data_flows WHERE tenant_id = $1 AND target_dataset_id = $2",
		tenantID, datasetID)

	var downstream []store.DataFlow
	s.db.SelectContext(ctx, &downstream,
		"SELECT * FROM data_flows WHERE tenant_id = $1 AND source_dataset_id = $2",
		tenantID, datasetID)

	var aiUsage []store.ModelLineage
	s.db.SelectContext(ctx, &aiUsage,
		"SELECT * FROM model_lineage WHERE tenant_id = $1 AND dataset_id = $2",
		tenantID, datasetID)

	pkg.JSON(w, map[string]any{
		"dataset_id": datasetID,
		"upstream":   upstream,
		"downstream": downstream,
		"ai_usage":   aiUsage,
	})
}

func (s *Server) getQualitySummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var totalDatasets int
	s.db.GetContext(ctx, &totalDatasets,
		"SELECT COUNT(DISTINCT dataset_id) FROM quality_scores WHERE tenant_id = $1", tenantID)

	var avgs struct {
		Overall      float64 `db:"avg_overall"`
		Completeness float64 `db:"avg_completeness"`
		Accuracy     float64 `db:"avg_accuracy"`
		Consistency  float64 `db:"avg_consistency"`
		Timeliness   float64 `db:"avg_timeliness"`
		Uniqueness   float64 `db:"avg_uniqueness"`
	}
	s.db.GetContext(ctx, &avgs,
		`SELECT COALESCE(AVG(overall),0) as avg_overall,
		        COALESCE(AVG(completeness),0) as avg_completeness,
		        COALESCE(AVG(accuracy),0) as avg_accuracy,
		        COALESCE(AVG(consistency),0) as avg_consistency,
		        COALESCE(AVG(timeliness),0) as avg_timeliness,
		        COALESCE(AVG(uniqueness),0) as avg_uniqueness
		 FROM quality_scores WHERE tenant_id = $1`, tenantID)

	// Derive per-dimension scores from real data when no quality_scores rows exist
	if totalDatasets == 0 {
		// Calculate dimensions from classifications and datasources directly
		avgs.Completeness = s.calcCompletenessScore(ctx, tenantID)
		avgs.Accuracy = s.calcAccuracyScore(ctx, tenantID)
		avgs.Consistency = s.calcConsistencyScore(ctx, tenantID)
		avgs.Timeliness = s.calcTimelinessScore(ctx, tenantID)
		avgs.Uniqueness = s.calcUniquenessScore(ctx, tenantID)
		avgs.Overall = (avgs.Completeness + avgs.Accuracy + avgs.Consistency + avgs.Timeliness + avgs.Uniqueness) / 5.0
	}

	// Issues = datasets with overall score below 70 + datasources with no quality score
	var issuesFound int
	s.db.GetContext(ctx, &issuesFound,
		`SELECT COUNT(*) FROM (
		   SELECT DISTINCT dataset_id FROM quality_scores
		   WHERE tenant_id = $1 AND overall < 0.7
		 ) sub`, tenantID)

	// Also count datasources that have never been assessed (each is a quality gap)
	var unassessedCount int
	s.db.GetContext(ctx, &unassessedCount,
		`SELECT COUNT(*) FROM datasources d
		 WHERE d.tenant_id = $1
		   AND NOT EXISTS (
		     SELECT 1 FROM quality_scores q
		     WHERE q.tenant_id = $1 AND q.dataset_id = d.id::text
		   )`, tenantID)
	issuesFound += unassessedCount

	pkg.JSON(w, map[string]any{
		"overall_score":  avgs.Overall,
		"completeness":   avgs.Completeness,
		"accuracy":       avgs.Accuracy,
		"consistency":    avgs.Consistency,
		"timeliness":     avgs.Timeliness,
		"uniqueness":     avgs.Uniqueness,
		"total_datasets": totalDatasets,
		"issues_found":   issuesFound,
	})
}

// calcCompletenessScore estimates completeness from classification null rates.
// A column with no null values scores 1.0; average null rate of >20% pulls score down.
func (s *Server) calcCompletenessScore(ctx context.Context, tenantID string) float64 {
	var totalCols, nullyCols int
	s.db.GetContext(ctx, &totalCols,
		"SELECT COUNT(DISTINCT column_name) FROM classifications WHERE tenant_id = $1", tenantID)
	if totalCols == 0 {
		return 0
	}
	// Estimate null columns as those with entity_type = 'UNKNOWN' or very low confidence
	s.db.GetContext(ctx, &nullyCols,
		"SELECT COUNT(DISTINCT column_name) FROM classifications WHERE tenant_id = $1 AND (confidence < 0.2 OR entity_type = 'UNKNOWN')", tenantID)
	score := 1.0 - (float64(nullyCols) / float64(totalCols))
	return roundScore(score)
}

// calcAccuracyScore uses average classification confidence as a proxy for accuracy.
func (s *Server) calcAccuracyScore(ctx context.Context, tenantID string) float64 {
	var avgConf float64
	s.db.GetContext(ctx, &avgConf,
		"SELECT COALESCE(AVG(confidence),0) FROM classifications WHERE tenant_id = $1", tenantID)
	return roundScore(avgConf)
}

// calcConsistencyScore measures how many datasources have been scanned vs total.
func (s *Server) calcConsistencyScore(ctx context.Context, tenantID string) float64 {
	var total, scanned int
	s.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM datasources WHERE tenant_id = $1", tenantID)
	if total == 0 {
		return 0
	}
	s.db.GetContext(ctx, &scanned,
		"SELECT COUNT(*) FROM datasources WHERE tenant_id = $1 AND status IN ('active','completed','connected')", tenantID)
	score := float64(scanned) / float64(total)
	return roundScore(score)
}

// calcTimelinessScore rewards datasources updated in the last 30 days.
func (s *Server) calcTimelinessScore(ctx context.Context, tenantID string) float64 {
	var total, recent int
	s.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM datasources WHERE tenant_id = $1", tenantID)
	if total == 0 {
		return 0
	}
	s.db.GetContext(ctx, &recent,
		"SELECT COUNT(*) FROM datasources WHERE tenant_id = $1 AND updated_at >= NOW() - INTERVAL '30 days'", tenantID)
	score := float64(recent) / float64(total)
	return roundScore(score)
}

// calcUniquenessScore estimates uniqueness from ROT redundant findings.
func (s *Server) calcUniquenessScore(ctx context.Context, tenantID string) float64 {
	var total, redundant int
	s.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM datasources WHERE tenant_id = $1", tenantID)
	if total == 0 {
		return 0
	}
	s.db.GetContext(ctx, &redundant,
		"SELECT COUNT(DISTINCT dataset_id) FROM rot_data WHERE tenant_id = $1 AND type = 'redundant'", tenantID)
	if redundant >= total {
		return 0.1
	}
	score := 1.0 - (float64(redundant) / float64(total))
	return roundScore(score)
}

func roundScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	// Round to 2 decimal places
	return float64(int(score*100+0.5)) / 100
}

func (s *Server) listPIAs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)
	_ = offset

	// Derive PIA list from classification data (datasets that have PII classifications)
	type PIAEntry struct {
		DatasetID   string  `json:"dataset_id"`
		DatasetName string  `json:"dataset_name"`
		RiskScore   float64 `json:"risk_score"`
		RiskLevel   string  `json:"risk_level"`
		PIICount    int     `json:"pii_count"`
		Status      string  `json:"status"`
	}

	var rows []struct {
		DatasetID string `db:"dataset_id"`
		Name      string `db:"name"`
		PIICount  int    `db:"pii_count"`
	}
	s.db.SelectContext(ctx, &rows,
		`SELECT c.dataset_id,
		        COALESCE(d.name, c.dataset_id) as name,
		        COUNT(*) as pii_count
		 FROM classifications c
		 LEFT JOIN datasources d ON d.id::text = c.dataset_id AND d.tenant_id = $1
		 WHERE c.tenant_id = $1
		   AND c.entity_type IN ('EMAIL','PHONE','SSN','CREDIT_CARD','PII','PERSON','LOCATION')
		 GROUP BY c.dataset_id, d.name
		 ORDER BY pii_count DESC LIMIT $2`, tenantID, limit)

	pias := make([]PIAEntry, 0, len(rows))
	for _, row := range rows {
		riskScore := 0.3
		if row.PIICount > 10 {
			riskScore = 0.8
		} else if row.PIICount > 5 {
			riskScore = 0.6
		} else if row.PIICount > 0 {
			riskScore = 0.4
		}
		riskLevel := "low"
		if riskScore >= 0.7 {
			riskLevel = "high"
		} else if riskScore >= 0.4 {
			riskLevel = "medium"
		}
		pias = append(pias, PIAEntry{
			DatasetID:   row.DatasetID,
			DatasetName: row.Name,
			RiskScore:   riskScore,
			RiskLevel:   riskLevel,
			PIICount:    row.PIICount,
			Status:      "completed",
		})
	}
	pkg.JSON(w, pias)
}

func (s *Server) listComplianceFrameworks(w http.ResponseWriter, r *http.Request) {
	frameworks := []map[string]any{
		{"id": "gdpr", "name": "GDPR", "description": "General Data Protection Regulation", "region": "EU", "active": true},
		{"id": "hipaa", "name": "HIPAA", "description": "Health Insurance Portability and Accountability Act", "region": "US", "active": true},
		{"id": "ccpa", "name": "CCPA", "description": "California Consumer Privacy Act", "region": "US-CA", "active": true},
		{"id": "soc2", "name": "SOC 2", "description": "Service Organization Control Type 2", "region": "Global", "active": true},
		{"id": "iso27001", "name": "ISO 27001", "description": "Information Security Management", "region": "Global", "active": true},
		{"id": "pdpb", "name": "PDPB", "description": "Personal Data Protection Bill (India)", "region": "India", "active": true},
		{"id": "uae_pdpl", "name": "UAE PDPL", "description": "UAE Personal Data Protection Law", "region": "UAE", "active": true},
	}
	pkg.JSON(w, frameworks)
}

func (s *Server) getQualityOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var totalDatasets int
	s.db.GetContext(ctx, &totalDatasets,
		"SELECT COUNT(DISTINCT dataset_id) FROM quality_scores WHERE tenant_id = $1", tenantID)

	var avgs struct {
		Overall      float64 `db:"avg_overall"`
		Completeness float64 `db:"avg_completeness"`
		Accuracy     float64 `db:"avg_accuracy"`
		Consistency  float64 `db:"avg_consistency"`
		Timeliness   float64 `db:"avg_timeliness"`
		Uniqueness   float64 `db:"avg_uniqueness"`
	}
	s.db.GetContext(ctx, &avgs,
		`SELECT COALESCE(AVG(overall),0) as avg_overall,
		        COALESCE(AVG(completeness),0) as avg_completeness,
		        COALESCE(AVG(accuracy),0) as avg_accuracy,
		        COALESCE(AVG(consistency),0) as avg_consistency,
		        COALESCE(AVG(timeliness),0) as avg_timeliness,
		        COALESCE(AVG(uniqueness),0) as avg_uniqueness
		 FROM quality_scores WHERE tenant_id = $1`, tenantID)

	if totalDatasets == 0 {
		avgs.Completeness = s.calcCompletenessScore(ctx, tenantID)
		avgs.Accuracy = s.calcAccuracyScore(ctx, tenantID)
		avgs.Consistency = s.calcConsistencyScore(ctx, tenantID)
		avgs.Timeliness = s.calcTimelinessScore(ctx, tenantID)
		avgs.Uniqueness = s.calcUniquenessScore(ctx, tenantID)
		avgs.Overall = (avgs.Completeness + avgs.Accuracy + avgs.Consistency + avgs.Timeliness + avgs.Uniqueness) / 5.0
	}

	var issuesFound int
	s.db.GetContext(ctx, &issuesFound,
		`SELECT COUNT(*) FROM (
		   SELECT DISTINCT dataset_id FROM quality_scores
		   WHERE tenant_id = $1 AND overall < 0.7
		 ) sub`, tenantID)

	var unassessedCount int
	s.db.GetContext(ctx, &unassessedCount,
		`SELECT COUNT(*) FROM datasources d
		 WHERE d.tenant_id = $1
		   AND NOT EXISTS (
		     SELECT 1 FROM quality_scores q
		     WHERE q.tenant_id = $1 AND q.dataset_id = d.id::text
		   )`, tenantID)
	issuesFound += unassessedCount

	var recentScores []store.QualityScore
	s.db.SelectContext(ctx, &recentScores,
		`SELECT * FROM quality_scores WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 10`, tenantID)

	pkg.JSON(w, map[string]any{
		"overall_score":  avgs.Overall,
		"completeness":   avgs.Completeness,
		"accuracy":       avgs.Accuracy,
		"consistency":    avgs.Consistency,
		"timeliness":     avgs.Timeliness,
		"uniqueness":     avgs.Uniqueness,
		"total_datasets": totalDatasets,
		"issues_found":   issuesFound,
		"recent_scores":  recentScores,
	})
}

func (s *Server) getQualityDimensions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	completeness := s.calcCompletenessScore(ctx, tenantID)
	accuracy := s.calcAccuracyScore(ctx, tenantID)
	consistency := s.calcConsistencyScore(ctx, tenantID)
	timeliness := s.calcTimelinessScore(ctx, tenantID)
	uniqueness := s.calcUniquenessScore(ctx, tenantID)

	dimensions := []map[string]any{
		{
			"id":          "completeness",
			"name":        "Completeness",
			"description": "Measures the extent to which data is not missing and has all required values",
			"score":       completeness,
			"weight":      0.2,
			"status":      dimensionStatus(completeness),
		},
		{
			"id":          "accuracy",
			"name":        "Accuracy",
			"description": "Measures how well data reflects the real-world entities it represents",
			"score":       accuracy,
			"weight":      0.25,
			"status":      dimensionStatus(accuracy),
		},
		{
			"id":          "consistency",
			"name":        "Consistency",
			"description": "Measures uniformity of data across different sources and systems",
			"score":       consistency,
			"weight":      0.2,
			"status":      dimensionStatus(consistency),
		},
		{
			"id":          "timeliness",
			"name":        "Timeliness",
			"description": "Measures how current and up-to-date the data is",
			"score":       timeliness,
			"weight":      0.15,
			"status":      dimensionStatus(timeliness),
		},
		{
			"id":          "uniqueness",
			"name":        "Uniqueness",
			"description": "Measures the absence of duplicate records in the data",
			"score":       uniqueness,
			"weight":      0.2,
			"status":      dimensionStatus(uniqueness),
		},
	}

	overall := (completeness*0.2 + accuracy*0.25 + consistency*0.2 + timeliness*0.15 + uniqueness*0.2)

	pkg.JSON(w, map[string]any{
		"dimensions":    dimensions,
		"overall_score": overall,
	})
}

func dimensionStatus(score float64) string {
	if score >= 0.8 {
		return "good"
	}
	if score >= 0.6 {
		return "warning"
	}
	return "critical"
}

func (s *Server) getPrivacyOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var dsarStats struct {
		Total     int `db:"total"`
		Pending   int `db:"pending"`
		Completed int `db:"completed"`
		Overdue   int `db:"overdue"`
	}
	s.db.GetContext(ctx, &dsarStats, `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'pending' AND due_date < NOW()) as overdue
		FROM dsars WHERE tenant_id = $1
	`, tenantID)

	var dpiaStats struct {
		Total      int `db:"total"`
		InProgress int `db:"in_progress"`
		Completed  int `db:"completed"`
	}
	s.db.GetContext(ctx, &dpiaStats, `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status IN ('draft', 'in_progress', 'review')) as in_progress,
			COUNT(*) FILTER (WHERE status = 'completed') as completed
		FROM dpias WHERE tenant_id = $1
	`, tenantID)

	var consentStats struct {
		Total     int `db:"total"`
		Active    int `db:"active"`
		Withdrawn int `db:"withdrawn"`
	}
	s.db.GetContext(ctx, &consentStats, `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'active') as active,
			COUNT(*) FILTER (WHERE status = 'withdrawn') as withdrawn
		FROM consent_records WHERE tenant_id = $1
	`, tenantID)

	var retentionViolations int
	s.db.GetContext(ctx, &retentionViolations,
		"SELECT COUNT(*) FROM retention_violations WHERE tenant_id = $1 AND resolved = false", tenantID)

	var piiDatasets int
	s.db.GetContext(ctx, &piiDatasets, `
		SELECT COUNT(DISTINCT dataset_id) FROM classifications 
		WHERE tenant_id = $1 AND entity_type IN ('PII', 'SSN', 'CREDIT_CARD', 'EMAIL', 'PHONE', 'ADDRESS', 'NAME', 'DOB', 'PHI')
	`, tenantID)

	pkg.JSON(w, map[string]any{
		"dsar": map[string]any{
			"total":     dsarStats.Total,
			"pending":   dsarStats.Pending,
			"completed": dsarStats.Completed,
			"overdue":   dsarStats.Overdue,
		},
		"dpia": map[string]any{
			"total":       dpiaStats.Total,
			"in_progress": dpiaStats.InProgress,
			"completed":   dpiaStats.Completed,
		},
		"consent": map[string]any{
			"total":     consentStats.Total,
			"active":    consentStats.Active,
			"withdrawn": consentStats.Withdrawn,
		},
		"retention_violations": retentionViolations,
		"pii_datasets":         piiDatasets,
	})
}
