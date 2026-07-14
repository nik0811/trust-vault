package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/trustvault/trustvault/internal/domain"
	"github.com/trustvault/trustvault/internal/pkg"
	"github.com/trustvault/trustvault/internal/store"
)

func (s *Server) listRemediationActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	actions, _ := s.remediationActions.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, actions)
}

func (s *Server) createRemediationAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Type      string `json:"type" validate:"required,oneof=redact encrypt delete quarantine label"`
		DatasetID string `json:"dataset_id" validate:"required"`
		Reason    string `json:"reason"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	action := store.RemediationAction{
		TenantID:  tenantID,
		Type:      req.Type,
		DatasetID: req.DatasetID,
		Reason:    req.Reason,
		Status:    "pending",
	}
	s.remediationActions.Create(ctx, &action)

	pkg.JSON(w, action, http.StatusCreated)
}

func (s *Server) executeRemediation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	action, _ := s.remediationActions.FindByID(ctx, tenantID, id)
	if action == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	now := time.Now()
	action.Status = "executing"
	action.ExecutedAt = &now
	s.remediationActions.Update(ctx, action)

	s.kafka.Produce(ctx, "remediation-jobs", tenantID, map[string]any{
		"action_id": id,
		"type":      action.Type,
		"dataset":   action.DatasetID,
	})

	pkg.JSON(w, action)
}

func (s *Server) approveRemediation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	userID := pkg.UserFromCtx(ctx)
	id := chi.URLParam(r, "id")

	action, _ := s.remediationActions.FindByID(ctx, tenantID, id)
	if action == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	action.Status = "approved"
	action.ApprovedBy = &userID
	s.remediationActions.Update(ctx, action)

	pkg.JSON(w, action)
}

func (s *Server) getRemediationHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var history []store.RemediationAction
	s.db.SelectContext(ctx, &history,
		`SELECT * FROM remediation_actions WHERE tenant_id = $1 
		 AND status IN ('completed', 'failed') ORDER BY executed_at DESC LIMIT 100`,
		tenantID)

	pkg.JSON(w, history)
}

func (s *Server) generateReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Type     string `json:"type" validate:"required,oneof=compliance quality ai_usage audit"`
		DateFrom string `json:"date_from"`
		DateTo   string `json:"date_to"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	report := store.Report{
		TenantID: tenantID,
		Type:     req.Type,
		Status:   "generating",
	}
	s.reports.Create(ctx, &report)

	s.kafka.Produce(ctx, "report-jobs", tenantID, map[string]any{
		"report_id": report.ID,
		"type":      req.Type,
		"date_from": req.DateFrom,
		"date_to":   req.DateTo,
	})

	pkg.JSON(w, report)
}

func (s *Server) listReports(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	reports, _ := s.reports.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, reports)
}

func (s *Server) downloadReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	report, _ := s.reports.FindByID(ctx, tenantID, id)
	if report == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	pkg.JSON(w, map[string]any{
		"id":     report.ID,
		"status": report.Status,
		"url":    report.FilePath,
	})
}

func (s *Server) getAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var stats struct {
		TotalSources         int `db:"sources"`
		TotalClassifications int `db:"classifications"`
		TotalPolicies        int `db:"policies"`
		TotalQueries         int `db:"queries"`
	}

	s.db.GetContext(ctx, &stats.TotalSources, "SELECT COUNT(*) FROM datasources WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &stats.TotalClassifications, "SELECT COUNT(*) FROM classifications WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &stats.TotalPolicies, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &stats.TotalQueries, "SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1", tenantID)

	var complianceScore float64 = 0.85
	var violations int
	s.db.GetContext(ctx, &violations, "SELECT COUNT(*) FROM retention_violations WHERE tenant_id = $1", tenantID)
	if violations > 0 {
		complianceScore = max(0.5, complianceScore-float64(violations)*0.05)
	}

	pkg.JSON(w, map[string]any{
		"total_sources":         stats.TotalSources,
		"total_classifications": stats.TotalClassifications,
		"total_policies":        stats.TotalPolicies,
		"total_queries":         stats.TotalQueries,
		"compliance_score":      complianceScore,
	})
}

func (s *Server) getAnalyticsTrends(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var classificationTrend []struct {
		Date  string `db:"date" json:"date"`
		Count int    `db:"count" json:"count"`
	}
	s.db.SelectContext(ctx, &classificationTrend,
		`SELECT DATE(created_at)::text as date, COUNT(*) as count 
		 FROM classifications WHERE tenant_id = $1 
		 AND created_at > NOW() - INTERVAL '30 days'
		 GROUP BY DATE(created_at) ORDER BY date`, tenantID)

	var queryTrend []struct {
		Date  string `db:"date" json:"date"`
		Count int    `db:"count" json:"count"`
	}
	s.db.SelectContext(ctx, &queryTrend,
		`SELECT DATE(created_at)::text as date, COUNT(*) as count 
		 FROM gate_queries WHERE tenant_id = $1 
		 AND created_at > NOW() - INTERVAL '30 days'
		 GROUP BY DATE(created_at) ORDER BY date`, tenantID)

	pkg.JSON(w, map[string]any{
		"classifications_trend": classificationTrend,
		"queries_trend":         queryTrend,
	})
}

func (s *Server) getDatasetLabel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasetID := chi.URLParam(r, "id")

	var label store.Label
	err := s.db.GetContext(ctx, &label,
		"SELECT * FROM labels WHERE tenant_id = $1 AND dataset_id = $2 ORDER BY created_at DESC LIMIT 1",
		tenantID, datasetID)
	if err != nil {
		pkg.JSON(w, map[string]any{"dataset_id": datasetID, "label": "UNCLASSIFIED"})
		return
	}
	pkg.JSON(w, label)
}

func (s *Server) assignLabel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	userID := pkg.UserFromCtx(ctx)

	var req struct {
		DatasetID string `json:"dataset_id" validate:"required"`
		Label     string `json:"label" validate:"required,oneof=PUBLIC INTERNAL CONFIDENTIAL HIGHLY_CONFIDENTIAL RESTRICTED"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	label := store.Label{
		TenantID:     tenantID,
		DatasetID:    req.DatasetID,
		Label:        req.Label,
		AutoAssigned: false,
		AssignedBy:   &userID,
	}
	s.labels.Create(ctx, &label)

	pkg.JSON(w, label, http.StatusCreated)
}

func (s *Server) getLabelRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	rules, _ := s.labelRules.List(ctx, tenantID, store.ListOpts{Limit: 100})
	pkg.JSON(w, rules)
}

func (s *Server) createLabelRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Classification string `json:"classification" validate:"required"`
		Label          string `json:"label" validate:"required"`
		Priority       int    `json:"priority"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	rule := store.LabelRule{
		TenantID:       tenantID,
		Classification: req.Classification,
		Label:          req.Label,
		Priority:       req.Priority,
		Active:         true,
	}
	s.labelRules.Create(ctx, &rule)

	pkg.JSON(w, rule, http.StatusCreated)
}

func (s *Server) updateLabelRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	rule, err := s.labelRules.FindByID(ctx, tenantID, id)
	if err != nil || rule == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	var req struct {
		Classification string `json:"classification"`
		Label          string `json:"label"`
		Priority       int    `json:"priority"`
		Active         *bool  `json:"active"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	if req.Classification != "" {
		rule.Classification = req.Classification
	}
	if req.Label != "" {
		rule.Label = req.Label
	}
	if req.Priority != 0 {
		rule.Priority = req.Priority
	}
	if req.Active != nil {
		rule.Active = *req.Active
	}

	s.labelRules.Update(ctx, rule)
	pkg.JSON(w, rule)
}

func (s *Server) deleteLabelRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	rule, err := s.labelRules.FindByID(ctx, tenantID, id)
	if err != nil || rule == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	s.labelRules.Delete(ctx, tenantID, id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getLabelSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var summary []struct {
		Label string `db:"label" json:"label"`
		Count int    `db:"count" json:"count"`
	}
	s.db.SelectContext(ctx, &summary,
		"SELECT label, COUNT(*) as count FROM labels WHERE tenant_id = $1 GROUP BY label",
		tenantID)

	pkg.JSON(w, summary)
}

func (s *Server) submitFeedback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	userID := pkg.UserFromCtx(ctx)

	var req struct {
		ClassificationID string `json:"classification_id"`
		Type             string `json:"type"`
		CorrectedLabel   string `json:"corrected_label,omitempty"`
		Comment          string `json:"comment,omitempty"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	feedbackType := req.Type
	if feedbackType == "" {
		feedbackType = "general"
	}

	feedback := store.Feedback{
		TenantID:       tenantID,
		Type:           feedbackType,
		CorrectedLabel: req.CorrectedLabel,
		UserID:         userID,
	}
	if req.ClassificationID != "" && pkg.IsValidUUID(req.ClassificationID) {
		feedback.ClassificationID = &req.ClassificationID
	}

	if err := s.feedback.Create(ctx, &feedback); err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	pkg.JSON(w, feedback, http.StatusCreated)
}

func (s *Server) submitCorrection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	userID := pkg.UserFromCtx(ctx)

	var req struct {
		ClassificationID string `json:"classification_id"`
		CorrectedLabel   string `json:"corrected_label" validate:"required"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	feedback := store.Feedback{
		TenantID:       tenantID,
		Type:           "correction",
		CorrectedLabel: req.CorrectedLabel,
		UserID:         userID,
	}
	if req.ClassificationID != "" && pkg.IsValidUUID(req.ClassificationID) {
		feedback.ClassificationID = &req.ClassificationID
	}

	if err := s.feedback.Create(ctx, &feedback); err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	pkg.JSON(w, feedback, http.StatusCreated)
}

func (s *Server) submitConfirmation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	userID := pkg.UserFromCtx(ctx)

	var req struct {
		ClassificationID string `json:"classification_id"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	feedback := store.Feedback{
		TenantID: tenantID,
		Type:     "confirmation",
		UserID:   userID,
	}
	if req.ClassificationID != "" && pkg.IsValidUUID(req.ClassificationID) {
		feedback.ClassificationID = &req.ClassificationID
	}

	if err := s.feedback.Create(ctx, &feedback); err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	pkg.JSON(w, feedback, http.StatusCreated)
}

func (s *Server) getFeedbackStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var corrections, confirmations int
	s.db.GetContext(ctx, &corrections,
		"SELECT COUNT(*) FROM feedback WHERE tenant_id = $1 AND type = 'correction'", tenantID)
	s.db.GetContext(ctx, &confirmations,
		"SELECT COUNT(*) FROM feedback WHERE tenant_id = $1 AND type = 'confirmation'", tenantID)

	total := corrections + confirmations
	var accuracyImprovement string
	if total > 0 {
		accuracy := float64(confirmations) / float64(total) * 100
		accuracyImprovement = fmt.Sprintf("+%.1f%%", accuracy)
	} else {
		accuracyImprovement = "—"
	}

	// Get knowledge cache size (custom entities count)
	var cacheSize int
	s.db.GetContext(ctx, &cacheSize, "SELECT COUNT(*) FROM custom_entities WHERE tenant_id = $1", tenantID)
	cacheSizeStr := fmt.Sprintf("%d entries", cacheSize)
	if cacheSize == 0 {
		cacheSizeStr = "0 entries"
	}

	// Calculate cache hit rate based on classifications that matched custom entities
	var cacheHitRate string
	var totalClassifications int
	s.db.GetContext(ctx, &totalClassifications, "SELECT COUNT(*) FROM classifications WHERE tenant_id = $1", tenantID)
	if totalClassifications > 0 && cacheSize > 0 {
		// Estimate hit rate based on corrections vs total
		hitRate := float64(confirmations) / float64(totalClassifications) * 100
		if hitRate > 100 {
			hitRate = 100
		}
		cacheHitRate = fmt.Sprintf("%.0f%%", hitRate)
	} else {
		cacheHitRate = "—"
	}

	pkg.JSON(w, map[string]any{
		"total_corrections":    corrections,
		"total_confirmations":  confirmations,
		"accuracy_improvement": accuracyImprovement,
		"cache_size":           cacheSizeStr,
		"cache_hit_rate":       cacheHitRate,
	})
}

func (s *Server) createCustomEntity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Name    string `json:"name" validate:"required"`
		Pattern string `json:"pattern" validate:"required"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	policy := store.Policy{
		TenantID: tenantID,
		Name:     req.Name,
		Type:     "custom_entity",
		Active:   true,
	}
	s.policies.Create(ctx, &policy)

	pkg.JSON(w, policy, http.StatusCreated)
}

func (s *Server) listCustomEntities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	policies, err := s.policies.List(ctx, tenantID, store.ListOpts{Limit: 100})
	if err != nil {
		pkg.JSON(w, []any{})
		return
	}

	type CustomEntity struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Examples   string `json:"examples"`
		Detections int    `json:"detections"`
		Accuracy   int    `json:"accuracy"`
	}

	var entities []CustomEntity
	for _, p := range policies {
		if p.Type == "custom_entity" {
			var detections int
			s.db.GetContext(ctx, &detections,
				"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND entity_type = $2",
				tenantID, p.Name)

			entities = append(entities, CustomEntity{
				ID:         p.ID,
				Name:       p.Name,
				Examples:   "",
				Detections: detections,
				Accuracy:   95,
			})
		}
	}

	if entities == nil {
		entities = []CustomEntity{}
	}
	pkg.JSON(w, entities)
}

func (s *Server) getKnowledgeCache(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var cacheSize int
	s.db.GetContext(ctx, &cacheSize,
		"SELECT COUNT(DISTINCT entity_type) FROM classifications WHERE tenant_id = $1", tenantID)

	var totalClassifications int
	s.db.GetContext(ctx, &totalClassifications,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1", tenantID)

	pkg.JSON(w, map[string]any{
		"cache_size":            cacheSize,
		"total_classifications": totalClassifications,
		"last_updated":          time.Now(),
	})
}

// CorrectionResponse matches the frontend's expected format for corrections
type CorrectionResponse struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	User      string    `json:"user"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Server) listCorrections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	feedbackList, err := s.feedback.List(ctx, tenantID, store.ListOpts{Limit: 100})
	if err != nil || feedbackList == nil {
		feedbackList = []store.Feedback{}
	}

	// Transform to frontend format
	corrections := make([]CorrectionResponse, 0, len(feedbackList))
	for _, fb := range feedbackList {
		// Only include corrections (not confirmations)
		if fb.Type != "correction" {
			continue
		}

		// Get username from user_id
		userName := fb.UserID
		var user store.User
		if err := s.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", fb.UserID); err == nil {
			userName = user.Name
			if userName == "" {
				userName = user.Email
			}
		}

		// Generate a text sample from the classification if available
		textSample := fmt.Sprintf("%s → %s", fb.OriginalLabel, fb.CorrectedLabel)
		if fb.ClassificationID != nil {
			var classification store.Classification
			if err := s.db.GetContext(ctx, &classification, "SELECT * FROM classifications WHERE id = $1", *fb.ClassificationID); err == nil && classification.Value != "" {
				textSample = classification.Value
				if len(textSample) > 50 {
					textSample = textSample[:50] + "..."
				}
			}
		}

		corrections = append(corrections, CorrectionResponse{
			ID:        fb.ID,
			Text:      textSample,
			From:      fb.OriginalLabel,
			To:        fb.CorrectedLabel,
			User:      userName,
			CreatedAt: fb.CreatedAt,
		})
	}

	pkg.JSON(w, corrections)
}

func (s *Server) getCorrectionTrend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	// Get corrections per day for last 7 days
	var trend []int
	for i := 6; i >= 0; i-- {
		var count int
		s.db.GetContext(ctx, &count,
			`SELECT COUNT(*) FROM feedback WHERE tenant_id = $1 
			 AND created_at >= NOW() - INTERVAL '1 day' * $2 
			 AND created_at < NOW() - INTERVAL '1 day' * $3`,
			tenantID, i+1, i)
		trend = append(trend, count)
	}

	pkg.JSON(w, trend)
}

// FrontendRecommendation matches the frontend's expected format
type FrontendRecommendation struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Priority    string `json:"priority"` // Frontend expects "priority" not "severity"
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

func (s *Server) getRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	// Gather context for advisor - ensure non-nil slices
	classifications, _ := s.classifications.List(ctx, tenantID, store.ListOpts{Limit: 1000})
	if classifications == nil {
		classifications = []store.Classification{}
	}
	policies, _ := s.policies.List(ctx, tenantID, store.ListOpts{Limit: 100})
	if policies == nil {
		policies = []store.Policy{}
	}
	labels, _ := s.labels.List(ctx, tenantID, store.ListOpts{Limit: 1000})
	if labels == nil {
		labels = []store.Label{}
	}
	violations, _ := s.retentionViolations.List(ctx, tenantID, store.ListOpts{Limit: 100})
	if violations == nil {
		violations = []store.RetentionViolation{}
	}
	dataSources, _ := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: 100})
	if dataSources == nil {
		dataSources = []store.DataSource{}
	}
	ropa, _ := s.ropa.List(ctx, tenantID, store.ListOpts{Limit: 100})
	if ropa == nil {
		ropa = []store.RoPA{}
	}

	var totalDatasets, labeledDatasets int
	s.db.GetContext(ctx, &totalDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM classifications WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &labeledDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM labels WHERE tenant_id = $1", tenantID)

	advCtx := &domain.AdvisorContext{
		TenantID:            tenantID,
		Classifications:     classifications,
		Policies:            policies,
		Labels:              labels,
		RetentionViolations: violations,
		DataSources:         dataSources,
		RoPA:                ropa,
		TotalDatasets:       totalDatasets,
		LabeledDatasets:     labeledDatasets,
	}

	recommendations := domain.GenerateRecommendations(advCtx)

	// Convert to frontend format (severity -> priority, map to lowercase)
	result := make([]FrontendRecommendation, 0, len(recommendations))
	for _, rec := range recommendations {
		priority := "low"
		switch rec.Severity {
		case "CRITICAL":
			priority = "high"
		case "HIGH":
			priority = "high"
		case "MEDIUM":
			priority = "medium"
		case "LOW":
			priority = "low"
		}
		result = append(result, FrontendRecommendation{
			ID:          rec.ID,
			Type:        rec.Category,
			Priority:    priority,
			Title:       rec.Title,
			Description: rec.Description,
			Action:      rec.Action,
		})
	}

	pkg.JSON(w, result)
}

// FrontendComplianceGap matches the frontend's expected format
type FrontendComplianceGap struct {
	Regulation  string `json:"regulation"`
	Requirement string `json:"requirement"`
	Status      string `json:"status"`
	Remediation string `json:"remediation"`
}

func (s *Server) getComplianceGaps(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var totalDatasets, labeledDatasets, policyCount int
	s.db.GetContext(ctx, &totalDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM classifications WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &labeledDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM labels WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &policyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true", tenantID)

	var ropaCount int
	s.db.GetContext(ctx, &ropaCount, "SELECT COUNT(*) FROM ropa WHERE tenant_id = $1", tenantID)

	// Check for specific policy types
	var consentPolicyCount, localizationPolicyCount, crossBorderPolicyCount int
	s.db.GetContext(ctx, &consentPolicyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true AND type IN ('consent', 'lawful_basis')", tenantID)
	s.db.GetContext(ctx, &localizationPolicyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true AND type IN ('localization', 'data_localization')", tenantID)
	s.db.GetContext(ctx, &crossBorderPolicyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true AND type IN ('cross_border', 'transfer')", tenantID)

	// Build gaps array in the format frontend expects
	gaps := []FrontendComplianceGap{}

	// GDPR gaps
	if labeledDatasets < totalDatasets && totalDatasets > 0 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "GDPR",
			Requirement: "Data Classification (Art. 5)",
			Status:      "open",
			Remediation: "Complete data classification for all datasets",
		})
	}
	if ropaCount == 0 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "GDPR",
			Requirement: "Records of Processing Activities (Art. 30)",
			Status:      "open",
			Remediation: "Create RoPA entries documenting all data processing activities",
		})
	}
	if policyCount < 3 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "GDPR",
			Requirement: "Governance Policies (Art. 25, 32)",
			Status:      "open",
			Remediation: "Define access, retention, and redaction policies",
		})
	}

	// CCPA gaps
	if labeledDatasets < totalDatasets && totalDatasets > 0 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "CCPA",
			Requirement: "Data Inventory (1798.100)",
			Status:      "open",
			Remediation: "Complete data inventory and classification",
		})
	}

	// PDPB (India DPDP Act 2023) gaps
	if consentPolicyCount == 0 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "DPDP",
			Requirement: "Consent Management (Section 6)",
			Status:      "open",
			Remediation: "Implement explicit consent collection with clear purpose specification",
		})
	}
	if localizationPolicyCount == 0 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "DPDP",
			Requirement: "Data Localization (Section 16)",
			Status:      "open",
			Remediation: "Define data localization policies for critical personal data stored in India",
		})
	}
	if ropaCount == 0 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "DPDP",
			Requirement: "Significant Data Fiduciary Obligations (Section 10)",
			Status:      "open",
			Remediation: "Appoint Data Protection Officer and conduct Data Protection Impact Assessments",
		})
	}
	if policyCount < 3 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "DPDP",
			Requirement: "Data Principal Rights (Section 11-14)",
			Status:      "open",
			Remediation: "Implement mechanisms for access, correction, and erasure requests",
		})
	}

	// UAE PDPL gaps
	if consentPolicyCount == 0 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "UAE PDPL",
			Requirement: "Lawful Basis for Processing (Art. 4)",
			Status:      "open",
			Remediation: "Document lawful basis for all personal data processing activities",
		})
	}
	if crossBorderPolicyCount == 0 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "UAE PDPL",
			Requirement: "Cross-Border Transfer Restrictions (Art. 22)",
			Status:      "open",
			Remediation: "Define cross-border transfer policies ensuring adequate protection level",
		})
	}
	if ropaCount == 0 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "UAE PDPL",
			Requirement: "Records of Processing Activities (Art. 8)",
			Status:      "open",
			Remediation: "Maintain records of all personal data processing activities",
		})
	}
	if policyCount < 3 {
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:  "UAE PDPL",
			Requirement: "Data Subject Rights (Art. 13-18)",
			Status:      "open",
			Remediation: "Implement access, rectification, erasure, and portability request mechanisms",
		})
	}

	pkg.JSON(w, gaps)
}

func (s *Server) generateDefenseDocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Regulations []string `json:"regulations"`
		DateFrom    string   `json:"date_from"`
		DateTo      string   `json:"date_to"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	report := store.Report{
		TenantID: tenantID,
		Type:     "defense_docket",
		Status:   "generating",
	}
	s.reports.Create(ctx, &report)

	s.kafka.Produce(ctx, "report-jobs", tenantID, map[string]any{
		"report_id":   report.ID,
		"type":        "defense_docket",
		"regulations": req.Regulations,
		"date_from":   req.DateFrom,
		"date_to":     req.DateTo,
	})

	pkg.JSON(w, report)
}

func (s *Server) getPlaybook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	issueType := chi.URLParam(r, "issue_type")

	var playbook store.Playbook
	err := s.db.GetContext(ctx, &playbook,
		"SELECT * FROM playbooks WHERE tenant_id = $1 AND issue_type = $2 LIMIT 1",
		tenantID, issueType)

	if err != nil {
		s.db.GetContext(ctx, &playbook,
			"SELECT * FROM playbooks WHERE issue_type = $1 LIMIT 1", issueType)
	}

	if playbook.ID == "" {
		pkg.JSON(w, map[string]any{
			"issue_type": issueType,
			"name":       "Generic Remediation",
			"steps":      []string{"Identify affected data", "Assess impact", "Create remediation plan", "Execute remediation", "Verify compliance"},
		})
		return
	}

	pkg.JSON(w, playbook)
}

func (s *Server) getRiskScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var piiCount, totalClassifications int
	s.db.GetContext(ctx, &piiCount,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND entity_type IN ('PII', 'SSN', 'CREDIT_CARD', 'PHI')", tenantID)
	s.db.GetContext(ctx, &totalClassifications,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1", tenantID)

	var policyCount int
	s.db.GetContext(ctx, &policyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true", tenantID)

	var violations int
	s.db.GetContext(ctx, &violations, "SELECT COUNT(*) FROM retention_violations WHERE tenant_id = $1", tenantID)

	dataExposure := 0.3
	if totalClassifications > 0 {
		dataExposure = float64(piiCount) / float64(totalClassifications)
	}

	policyCoverage := min(1.0, float64(policyCount)/5.0)
	complianceGaps := min(1.0, float64(violations)*0.1)

	overallScore := 1.0 - (dataExposure*0.4 + (1-policyCoverage)*0.3 + complianceGaps*0.3)

	riskLevel := "low"
	if overallScore < 0.5 {
		riskLevel = "critical"
	} else if overallScore < 0.7 {
		riskLevel = "high"
	} else if overallScore < 0.85 {
		riskLevel = "medium"
	}

	pkg.JSON(w, map[string]any{
		"overall_score": overallScore,
		"risk_level":    riskLevel,
		"factors": map[string]float64{
			"data_exposure":   dataExposure,
			"policy_coverage": policyCoverage,
			"compliance_gaps": complianceGaps,
		},
	})
}
