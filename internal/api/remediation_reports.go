package api

import (
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
	if req.ClassificationID != "" {
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
	if req.ClassificationID != "" {
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
	if req.ClassificationID != "" {
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
	var accuracy float64
	if total > 0 {
		accuracy = float64(confirmations) / float64(total)
	}

	pkg.JSON(w, map[string]any{
		"total_corrections":    corrections,
		"total_confirmations":  confirmations,
		"accuracy_improvement": accuracy,
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

func (s *Server) listCorrections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	corrections, err := s.feedback.List(ctx, tenantID, store.ListOpts{Limit: 100})
	if err != nil || corrections == nil {
		corrections = []store.Feedback{}
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
	
	// Ensure we return an empty array, not null
	if recommendations == nil {
		recommendations = []domain.Recommendation{}
	}

	pkg.JSON(w, recommendations)
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

	gdprScore := 0.7
	gdprGaps := []string{}
	if labeledDatasets < totalDatasets {
		gdprGaps = append(gdprGaps, "Incomplete data classification")
		gdprScore -= 0.1
	}
	if ropaCount == 0 {
		gdprGaps = append(gdprGaps, "Missing Records of Processing Activities")
		gdprScore -= 0.15
	}
	if policyCount < 3 {
		gdprGaps = append(gdprGaps, "Insufficient governance policies")
		gdprScore -= 0.1
	}

	ccpaScore := 0.8
	ccpaGaps := []string{}
	if labeledDatasets < totalDatasets {
		ccpaGaps = append(ccpaGaps, "Incomplete data inventory")
		ccpaScore -= 0.1
	}

	pkg.JSON(w, map[string]any{
		"gdpr": map[string]any{"score": max(0, gdprScore), "gaps": gdprGaps},
		"ccpa": map[string]any{"score": max(0, ccpaScore), "gaps": ccpaGaps},
	})
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
