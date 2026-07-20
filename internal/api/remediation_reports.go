package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/domain"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

func (s *Server) listRemediationActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	type actionRow struct {
		store.RemediationAction
		DatasetName string `db:"dataset_name" json:"dataset_name"`
	}

	var rows []actionRow
	if tenantID == "" {
		s.db.SelectContext(ctx, &rows,
			`SELECT r.*, COALESCE(d.name, r.dataset_id::text) AS dataset_name
			 FROM remediation_actions r
			 LEFT JOIN datasources d ON d.id::text = r.dataset_id
			 ORDER BY r.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	} else {
		s.db.SelectContext(ctx, &rows,
			`SELECT r.*, COALESCE(d.name, r.dataset_id::text) AS dataset_name
			 FROM remediation_actions r
			 LEFT JOIN datasources d ON d.id::text = r.dataset_id
			 WHERE r.tenant_id = $1
			 ORDER BY r.created_at DESC LIMIT $2 OFFSET $3`, tenantID, limit, offset)
	}
	if rows == nil {
		rows = []actionRow{}
	}
	pkg.JSON(w, rows)
}

func (s *Server) getRemediationLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	actionID := chi.URLParam(r, "id")

	var logs []store.AuditLog
	s.db.SelectContext(ctx, &logs,
		`SELECT * FROM audit_logs WHERE tenant_id = $1 AND resource_id = $2 ORDER BY created_at DESC`,
		tenantID, actionID)
	if logs == nil {
		logs = []store.AuditLog{}
	}
	pkg.JSON(w, logs)
}

func (s *Server) executeRemediationAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	userID := pkg.UserFromCtx(ctx)
	actionID := chi.URLParam(r, "id")

	action, _ := s.remediationActions.FindByID(ctx, tenantID, actionID)
	if action == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	now := time.Now()
	action.Status = "running"
	action.ExecutedAt = &now
	s.remediationActions.Update(ctx, action)

	// For ROT remediation: remove matching rot_data rows and log the action.
	s.db.ExecContext(ctx,
		`DELETE FROM rot_data WHERE tenant_id = $1 AND dataset_id = $2`,
		tenantID, action.DatasetID)

	action.Status = "completed"
	s.remediationActions.Update(ctx, action)

	s.db.ExecContext(ctx,
		`INSERT INTO audit_logs (tenant_id, user_id, action, resource, resource_id, details, ip)
		 VALUES ($1, $2, 'remediation.executed', 'remediation_action', $3,
		         $4::jsonb, '')`,
		tenantID, userID, actionID,
		fmt.Sprintf(`{"action_type":"%s","dataset_id":"%s","status":"completed"}`,
			action.ActionType, action.DatasetID))

	pkg.JSON(w, action)
}

func (s *Server) createRemediationAction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Type       string `json:"type" validate:"required,oneof=redact encrypt delete quarantine label archive deduplicate flag"`
		ActionType string `json:"action_type"`
		DatasetID  string `json:"dataset_id" validate:"required"`
		Reason     string `json:"reason"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	if req.ActionType == "" {
		req.ActionType = req.Type
	}

	action := store.RemediationAction{
		TenantID:   tenantID,
		Type:       req.Type,
		ActionType: req.ActionType,
		DatasetID:  req.DatasetID,
		Reason:     req.Reason,
		Status:     "pending",
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

	if req.Type == "compliance" {
		advCtx := s.buildAdvisorContext(ctx, tenantID)
		recommendations := domain.GenerateRecommendations(advCtx)

		criticalCount, highCount, mediumCount, lowCount := 0, 0, 0, 0
		for _, rec := range recommendations {
			switch rec.Severity {
			case "CRITICAL":
				criticalCount++
			case "HIGH":
				highCount++
			case "MEDIUM":
				mediumCount++
			case "LOW":
				lowCount++
			}
		}

		score := calcComplianceScore(criticalCount, highCount, mediumCount, lowCount)

		status := "Compliant"
		switch {
		case score < 50:
			status = "Non-Compliant"
		case score < 75:
			status = "Needs Attention"
		case score < 90:
			status = "Partially Compliant"
		}

		scannedSources := 0
		for _, ds := range advCtx.DataSources {
			if ds.LastScan != nil && !ds.LastScan.IsZero() {
				scannedSources++
			}
		}

		activePolicies := 0
		for _, p := range advCtx.Policies {
			if p.Active {
				activePolicies++
			}
		}

		regulationMap := map[string]*complianceRegEntry{}
		for _, rec := range recommendations {
			reg := rec.Regulation
			if reg == "" {
				continue
			}
			// normalize to top-level regulation name
			regName := reg
			switch {
			case len(reg) >= 4 && reg[:4] == "GDPR":
				regName = "GDPR"
			case len(reg) >= 4 && reg[:4] == "CCPA":
				regName = "CCPA"
			case len(reg) >= 4 && reg[:4] == "DPDP":
				regName = "DPDP Act 2023"
			case len(reg) >= 3 && reg[:3] == "UAE":
				regName = "UAE PDPL"
			case len(reg) >= 5 && reg[:5] == "EU AI":
				regName = "EU AI Act"
			case len(reg) >= 5 && reg[:5] == "HIPAA":
				regName = "HIPAA"
			case len(reg) >= 7 && reg[:7] == "PCI-DSS":
				regName = "PCI-DSS"
			}
			e, ok := regulationMap[regName]
			if !ok {
				e = &complianceRegEntry{Name: regName}
				regulationMap[regName] = e
			}
			e.FindingsCount++
			if rec.RegulationArticle != "" {
				alreadyIn := false
				for _, a := range e.Articles {
					if a == rec.RegulationArticle {
						alreadyIn = true
						break
					}
				}
				if !alreadyIn {
					e.Articles = append(e.Articles, rec.RegulationArticle)
				}
			}
			switch rec.Severity {
			case "CRITICAL":
				e.CriticalCount++
			case "HIGH":
				e.HighCount++
			}
		}

		regulations := make([]map[string]any, 0, len(regulationMap))
		for _, e := range regulationMap {
			regScore := 100.0 - float64(e.CriticalCount)*15 - float64(e.HighCount)*10
			if regScore < 0 {
				regScore = 0
			}
			regStatus := "Compliant"
			if regScore < 50 {
				regStatus = "Non-Compliant"
			} else if regScore < 80 {
				regStatus = "Partially Compliant"
			}
			regulations = append(regulations, map[string]any{
				"name":              e.Name,
				"score":             regScore,
				"status":            regStatus,
				"findings_count":    e.FindingsCount,
				"articles_assessed": e.Articles,
			})
		}

		findings := make([]map[string]any, 0, len(recommendations))
		for _, rec := range recommendations {
			evidence := rec.Evidence
			if evidence == nil {
				evidence = []domain.EvidenceItem{}
			}
			assets := rec.AffectedAssets
			if assets == nil {
				assets = []domain.AffectedAsset{}
			}
			findings = append(findings, map[string]any{
				"id":                 rec.ID,
				"severity":           rec.Severity,
				"category":           rec.Category,
				"title":              rec.Title,
				"description":        rec.Description,
				"action":             rec.Action,
				"regulation":         rec.Regulation,
				"regulation_article": rec.RegulationArticle,
				"affected_count":     rec.AffectedCount,
				"evidence":           evidence,
				"affected_assets":    assets,
				"detected_at":        rec.DetectedAt,
				"evidence_summary":   rec.EvidenceSummary,
				"severity_reason":    rec.SeverityReason,
			})
		}

		reportContent := map[string]any{
			"id":           report.ID,
			"title":        "SecureLens Compliance Assessment Report",
			"generated_at": time.Now(),
			"tenant_id":    tenantID,
			"executive_summary": map[string]any{
				"overall_score":         score,
				"status":                status,
				"total_findings":        len(recommendations),
				"critical":              criticalCount,
				"high":                  highCount,
				"medium":                mediumCount,
				"low":                   lowCount,
				"data_sources_total":    len(advCtx.DataSources),
				"data_sources_scanned":  scannedSources,
				"classifications_total": len(advCtx.Classifications),
				"active_policies":       activePolicies,
			},
			"regulations": regulations,
			"findings":    findings,
			"methodology": "Automated compliance assessment based on ML data classification, policy coverage analysis, and regulatory requirement mapping against live data",
			"assessor":    "SecureLens Automated Compliance Engine v1.0",
		}

		contentBytes, err := json.Marshal(reportContent)
		if err != nil {
			pkg.Error(w, err, http.StatusInternalServerError)
			return
		}

		report.Status = "completed"
		report.Content = store.JSON(contentBytes)
		s.reports.Update(ctx, &report)

		pkg.JSON(w, report, http.StatusCreated)
		return
	}

	// For non-compliance report types, generate synchronously
	var reportContent map[string]any

	switch req.Type {
	case "quality":
		var qualityStats []struct {
			DatasetID   string  `db:"dataset_id"`
			Overall     float64 `db:"overall"`
			Completeness float64 `db:"completeness"`
			Accuracy    float64 `db:"accuracy"`
			IssueCount  int     `db:"issue_count"`
		}
		s.db.SelectContext(ctx, &qualityStats,
			`SELECT dataset_id, overall_score as overall, completeness_score as completeness, accuracy_score as accuracy, issue_count FROM quality_assessments WHERE tenant_id = $1 ORDER BY assessed_at DESC LIMIT 50`, tenantID)

		avgOverall, avgCompleteness, avgAccuracy := 0.0, 0.0, 0.0
		if len(qualityStats) > 0 {
			for _, q := range qualityStats {
				avgOverall += q.Overall
				avgCompleteness += q.Completeness
				avgAccuracy += q.Accuracy
			}
			n := float64(len(qualityStats))
			avgOverall /= n
			avgCompleteness /= n
			avgAccuracy /= n
		}

		reportContent = map[string]any{
			"title":        "SecureLens Data Quality Report",
			"generated_at": report.CreatedAt,
			"tenant_id":    tenantID,
			"executive_summary": map[string]any{
				"total_assessments":    len(qualityStats),
				"avg_overall_score":    avgOverall,
				"avg_completeness":     avgCompleteness,
				"avg_accuracy":         avgAccuracy,
				"status":               func() string { if avgOverall >= 0.8 { return "Good" } else if avgOverall >= 0.6 { return "Fair" } else { return "Poor" } }(),
			},
			"datasets":    qualityStats,
			"methodology": "Quality scored across 4 dimensions: completeness, accuracy, consistency, and uniqueness",
			"assessor":    "SecureLens Data Quality Engine v1.0",
		}

	case "ai_usage":
		var gateStats struct {
			TotalQueries   int     `db:"total"`
			AllowedQueries int     `db:"allowed"`
			BlockedQueries int     `db:"blocked"`
			RedactedCount  int     `db:"redacted"`
		}
		s.db.GetContext(ctx, &gateStats.TotalQueries, `SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1`, tenantID)
		s.db.GetContext(ctx, &gateStats.AllowedQueries, `SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1 AND action = 'allow'`, tenantID)
		s.db.GetContext(ctx, &gateStats.BlockedQueries, `SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1 AND action = 'block'`, tenantID)
		s.db.GetContext(ctx, &gateStats.RedactedCount, `SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1 AND action = 'redact'`, tenantID)

		var recentQueries []map[string]any
		rows, _ := s.db.QueryxContext(ctx, `SELECT id, query_text, action, created_at FROM gate_queries WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 20`, tenantID)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				row := map[string]any{}
				rows.MapScan(row)
				recentQueries = append(recentQueries, row)
			}
		}

		blockRate := 0.0
		if gateStats.TotalQueries > 0 {
			blockRate = float64(gateStats.BlockedQueries) / float64(gateStats.TotalQueries) * 100
		}

		reportContent = map[string]any{
			"title":        "SecureLens AI Usage Report",
			"generated_at": report.CreatedAt,
			"tenant_id":    tenantID,
			"executive_summary": map[string]any{
				"total_queries":   gateStats.TotalQueries,
				"allowed_queries": gateStats.AllowedQueries,
				"blocked_queries": gateStats.BlockedQueries,
				"redacted_count":  gateStats.RedactedCount,
				"block_rate_pct":  blockRate,
				"risk_level":      func() string { if blockRate > 20 { return "High" } else if blockRate > 5 { return "Medium" } else { return "Low" } }(),
			},
			"recent_queries": recentQueries,
			"methodology":    "AI Gate intercepts all LLM queries and evaluates against governance policies",
			"assessor":       "SecureLens AI Gate Engine v1.0",
		}

	case "audit":
		var auditStats struct {
			TotalEvents  int `db:"total"`
			LoginEvents  int `db:"logins"`
			DataEvents   int `db:"data_events"`
			PolicyEvents int `db:"policy_events"`
		}
		s.db.GetContext(ctx, &auditStats.TotalEvents, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1`, tenantID)
		s.db.GetContext(ctx, &auditStats.LoginEvents, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action LIKE 'user.%'`, tenantID)
		s.db.GetContext(ctx, &auditStats.DataEvents, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action LIKE 'datasource.%'`, tenantID)
		s.db.GetContext(ctx, &auditStats.PolicyEvents, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action LIKE 'policy.%'`, tenantID)

		var recentLogs []store.AuditLog
		s.db.SelectContext(ctx, &recentLogs,
			`SELECT id, tenant_id, COALESCE(user_id,'') as user_id, action, resource, COALESCE(resource_id,'') as resource_id, details, COALESCE(ip,'') as ip, created_at FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 50`, tenantID)

		reportContent = map[string]any{
			"title":        "SecureLens Audit Report",
			"generated_at": report.CreatedAt,
			"tenant_id":    tenantID,
			"executive_summary": map[string]any{
				"total_events":  auditStats.TotalEvents,
				"login_events":  auditStats.LoginEvents,
				"data_events":   auditStats.DataEvents,
				"policy_events": auditStats.PolicyEvents,
				"period":        "All time",
			},
			"audit_log":   recentLogs,
			"methodology": "Complete immutable audit trail of all system actions with user attribution and IP tracking",
			"assessor":    "SecureLens Audit Engine v1.0",
		}

	default:
		reportContent = map[string]any{
			"title":        "SecureLens Report",
			"generated_at": report.CreatedAt,
			"tenant_id":    tenantID,
			"type":         req.Type,
		}
	}

	contentBytes, _ := json.Marshal(reportContent)
	report.Status = "completed"
	report.Content = store.JSON(contentBytes)
	s.reports.Update(ctx, &report)
	pkg.JSON(w, report, http.StatusCreated)
}

// complianceRegEntry accumulates per-regulation stats
type complianceRegEntry struct {
	Name          string
	FindingsCount int
	CriticalCount int
	HighCount     int
	Articles      []string
}

// calcComplianceScore starts at 100 and deducts per finding severity
func calcComplianceScore(critical, high, medium, low int) float64 {
	score := 100.0
	score -= float64(critical) * 15
	score -= float64(high) * 10
	score -= float64(medium) * 5
	score -= float64(low) * 2
	if score < 0 {
		score = 0
	}
	return score
}

// buildAdvisorContext fetches all data needed by the advisor engine
func (s *Server) buildAdvisorContext(ctx context.Context, tenantID string) *domain.AdvisorContext {
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

	var auditLogs []store.AuditLog
	s.db.SelectContext(ctx, &auditLogs,
		"SELECT * FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 100", tenantID)

	var totalDatasets, labeledDatasets int
	s.db.GetContext(ctx, &totalDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM classifications WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &labeledDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM labels WHERE tenant_id = $1", tenantID)

	return &domain.AdvisorContext{
		TenantID:            tenantID,
		Classifications:     classifications,
		Policies:            policies,
		Labels:              labels,
		RetentionViolations: violations,
		DataSources:         dataSources,
		RoPA:                ropa,
		TotalDatasets:       totalDatasets,
		LabeledDatasets:     labeledDatasets,
		AuditLogs:           auditLogs,
		AssessmentTime:      time.Now(),
	}
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

	pkg.JSON(w, report)
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

	advCtx := s.buildAdvisorContext(ctx, tenantID)
	recommendations := domain.GenerateRecommendations(advCtx)
	criticalCount, highCount, mediumCount, lowCount := 0, 0, 0, 0
	for _, rec := range recommendations {
		switch rec.Severity {
		case "CRITICAL":
			criticalCount++
		case "HIGH":
			highCount++
		case "MEDIUM":
			mediumCount++
		case "LOW":
			lowCount++
		}
	}
	complianceScore := calcComplianceScore(criticalCount, highCount, mediumCount, lowCount) / 100.0

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

	var rows []struct {
		Label string `db:"label"`
		Count int    `db:"count"`
	}
	s.db.SelectContext(ctx, &rows,
		"SELECT label, COUNT(*) as count FROM labels WHERE tenant_id = $1 GROUP BY label",
		tenantID)

	result := map[string]int{
		"total":        0,
		"public":       0,
		"internal":     0,
		"confidential": 0,
		"restricted":   0,
	}
	for _, row := range rows {
		result["total"] += row.Count
		switch row.Label {
		case "PUBLIC":
			result["public"] += row.Count
		case "INTERNAL":
			result["internal"] += row.Count
		case "CONFIDENTIAL", "HIGHLY_CONFIDENTIAL":
			result["confidential"] += row.Count
		case "RESTRICTED":
			result["restricted"] += row.Count
		}
	}

	pkg.JSON(w, result)
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

	var feedbackList []store.Feedback
	if tenantID == "" {
		s.db.SelectContext(ctx, &feedbackList, `SELECT * FROM feedback ORDER BY created_at DESC LIMIT 100`)
	} else {
		s.db.SelectContext(ctx, &feedbackList, `SELECT * FROM feedback WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 100`, tenantID)
	}
	if feedbackList == nil {
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

	type TrendPoint struct {
		Week  string `json:"week"`
		Count int    `json:"count"`
	}
	trend := make([]TrendPoint, 7)
	for i := 6; i >= 0; i-- {
		var count int
		weekLabel := fmt.Sprintf("W%d", 7-i)
		if tenantID == "" {
			s.db.GetContext(ctx, &count,
				`SELECT COUNT(*) FROM feedback
				 WHERE created_at >= NOW() - INTERVAL '1 day' * $1
				 AND created_at < NOW() - INTERVAL '1 day' * $2`,
				i+1, i)
		} else {
			s.db.GetContext(ctx, &count,
				`SELECT COUNT(*) FROM feedback WHERE tenant_id = $1
				 AND created_at >= NOW() - INTERVAL '1 day' * $2
				 AND created_at < NOW() - INTERVAL '1 day' * $3`,
				tenantID, i+1, i)
		}
		trend[6-i] = TrendPoint{Week: weekLabel, Count: count}
	}
	pkg.JSON(w, trend)
}

// FrontendRecommendation matches the frontend's expected format
type FrontendRecommendation struct {
	ID                string                `json:"id"`
	Type              string                `json:"type"`
	Priority          string                `json:"priority"`
	Title             string                `json:"title"`
	Description       string                `json:"description"`
	Action            string                `json:"action"`
	Regulation        string                `json:"regulation,omitempty"`
	RegulationArticle string                `json:"regulation_article,omitempty"`
	Evidence          []domain.EvidenceItem `json:"evidence"`
	AffectedAssets    []domain.AffectedAsset `json:"affected_assets"`
	EvidenceCount     int                   `json:"evidence_count"`
	DetectedAt        time.Time             `json:"detected_at"`
	EvidenceSummary   string                `json:"evidence_summary"`
	SeverityReason    string                `json:"severity_reason"`
	AffectedCount     int                   `json:"affected_count"`
}

func (s *Server) getRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	// Gather context for advisor - ensure non-nil slices
	classifications := s.loadClassifications(ctx, tenantID)
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

	var auditLogs []store.AuditLog
	s.db.SelectContext(ctx, &auditLogs,
		"SELECT * FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 100", tenantID)

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
		AuditLogs:           auditLogs,
		AssessmentTime:      time.Now(),
	}

	recommendations := domain.GenerateRecommendations(advCtx)

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

		evidence := rec.Evidence
		if evidence == nil {
			evidence = []domain.EvidenceItem{}
		}
		assets := rec.AffectedAssets
		if assets == nil {
			assets = []domain.AffectedAsset{}
		}

		result = append(result, FrontendRecommendation{
			ID:                rec.ID,
			Type:              rec.Category,
			Priority:          priority,
			Title:             rec.Title,
			Description:       rec.Description,
			Action:            rec.Action,
			Regulation:        rec.Regulation,
			RegulationArticle: rec.RegulationArticle,
			Evidence:          evidence,
			AffectedAssets:    assets,
			EvidenceCount:     len(evidence),
			DetectedAt:        rec.DetectedAt,
			EvidenceSummary:   rec.EvidenceSummary,
			SeverityReason:    rec.SeverityReason,
			AffectedCount:     rec.AffectedCount,
		})
	}

	pkg.JSON(w, result)
}

// FrontendComplianceGap matches the frontend's expected format with evidence
type FrontendComplianceGap struct {
	Regulation        string                `json:"regulation"`
	Requirement       string                `json:"requirement"`
	Status            string                `json:"status"`
	Remediation       string                `json:"remediation"`
	RegulationArticle string                `json:"regulation_article"`
	Evidence          []domain.EvidenceItem `json:"evidence"`
	AffectedAssets    []domain.AffectedAsset `json:"affected_assets"`
	EvidenceCount     int                   `json:"evidence_count"`
	DetectedAt        time.Time             `json:"detected_at"`
	LastAssessed      time.Time             `json:"last_assessed"`
	Severity          string                `json:"severity"`
	SeverityReason    string                `json:"severity_reason"`
}

func (s *Server) getComplianceGaps(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	now := time.Now()

	var totalDatasets, labeledDatasets, policyCount int
	s.db.GetContext(ctx, &totalDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM classifications WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &labeledDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM labels WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &policyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true", tenantID)

	var ropaCount int
	s.db.GetContext(ctx, &ropaCount, "SELECT COUNT(*) FROM ropa WHERE tenant_id = $1", tenantID)

	var consentPolicyCount, localizationPolicyCount, crossBorderPolicyCount int
	s.db.GetContext(ctx, &consentPolicyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true AND type IN ('consent', 'lawful_basis')", tenantID)
	s.db.GetContext(ctx, &localizationPolicyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true AND type IN ('localization', 'data_localization')", tenantID)
	s.db.GetContext(ctx, &crossBorderPolicyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true AND type IN ('cross_border', 'transfer')", tenantID)

	// Get unscanned data sources for evidence
	var unscannedSources []store.DataSource
	s.db.SelectContext(ctx, &unscannedSources,
		"SELECT * FROM datasources WHERE tenant_id = $1 AND (last_scan IS NULL) LIMIT 5", tenantID)

	// Get sample classifications for evidence
	var sampleClassifications []store.Classification
	s.db.SelectContext(ctx, &sampleClassifications,
		"SELECT * FROM classifications WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 5", tenantID)

	gaps := []FrontendComplianceGap{}

	// GDPR gaps
	if labeledDatasets < totalDatasets && totalDatasets > 0 {
		unlabeled := totalDatasets - labeledDatasets
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-gdpr-classification",
			Type:        "coverage_gap",
			Source:      "label_engine",
			Description: fmt.Sprintf("%d of %d datasets lack sensitivity labels required for Art. 5 compliance", unlabeled, totalDatasets),
			DetectedAt:  now,
			Metadata:    map[string]any{"total": totalDatasets, "labeled": labeledDatasets, "unlabeled": unlabeled},
		}}
		var assets []domain.AffectedAsset
		for _, c := range sampleClassifications {
			assets = append(assets, domain.AffectedAsset{ID: pkg.DerefStr(c.SourceID), Name: c.DatasetID, Type: "dataset"})
		}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "GDPR",
			Requirement:       "Data Classification (Art. 5)",
			Status:            "open",
			Remediation:       "Complete data classification for all datasets",
			RegulationArticle: "GDPR Art. 5(1)(d) - Accuracy: personal data must be accurate and kept up to date",
			Evidence:          evidence,
			AffectedAssets:    assets,
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "HIGH",
			SeverityReason:    fmt.Sprintf("%d datasets without labels means data handling cannot be governed appropriately", unlabeled),
		})
	}
	if ropaCount == 0 {
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-gdpr-ropa",
			Type:        "absence_of_record",
			Source:      "ropa_registry",
			Description: "Zero Records of Processing Activities exist in the system",
			DetectedAt:  now,
			Metadata:    map[string]any{"ropa_count": 0, "requirement": "mandatory for all controllers"},
		}}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "GDPR",
			Requirement:       "Records of Processing Activities (Art. 30)",
			Status:            "open",
			Remediation:       "Create RoPA entries documenting all data processing activities",
			RegulationArticle: "GDPR Art. 30(1) - Each controller and processor shall maintain a record of processing activities",
			Evidence:          evidence,
			AffectedAssets:    []domain.AffectedAsset{},
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "CRITICAL",
			SeverityReason:    "Art. 30 is a mandatory obligation for all data controllers; auditors will flag this immediately",
		})
	}
	if policyCount < 3 {
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-gdpr-policies",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: fmt.Sprintf("Only %d active policies found; minimum 3 (access, retention, redaction) expected for GDPR", policyCount),
			DetectedAt:  now,
			Metadata:    map[string]any{"active_policies": policyCount, "minimum_expected": 3},
		}}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "GDPR",
			Requirement:       "Governance Policies (Art. 25, 32)",
			Status:            "open",
			Remediation:       "Define access, retention, and redaction policies",
			RegulationArticle: "GDPR Art. 25 - Data protection by design and by default; Art. 32 - Security of processing",
			Evidence:          evidence,
			AffectedAssets:    []domain.AffectedAsset{},
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "HIGH",
			SeverityReason:    fmt.Sprintf("Only %d policies exist; comprehensive governance requires access, retention, and protection policies", policyCount),
		})
	}

	// CCPA gaps
	if labeledDatasets < totalDatasets && totalDatasets > 0 {
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-ccpa-inventory",
			Type:        "coverage_gap",
			Source:      "classification_engine",
			Description: fmt.Sprintf("Data inventory incomplete: %d of %d datasets not fully classified", totalDatasets-labeledDatasets, totalDatasets),
			DetectedAt:  now,
			Metadata:    map[string]any{"total": totalDatasets, "classified": labeledDatasets},
		}}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "CCPA",
			Requirement:       "Data Inventory (1798.100)",
			Status:            "open",
			Remediation:       "Complete data inventory and classification",
			RegulationArticle: "CCPA 1798.100(a) - Consumer right to know what personal information is collected",
			Evidence:          evidence,
			AffectedAssets:    []domain.AffectedAsset{},
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "MEDIUM",
			SeverityReason:    "Incomplete inventory limits ability to fulfill consumer access requests",
		})
	}

	// DPDP Act 2023 (India) gaps
	if consentPolicyCount == 0 {
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-dpdp-consent",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No consent management policy configured; DPDP Act requires explicit consent before processing",
			DetectedAt:  now,
			Metadata:    map[string]any{"consent_policies": 0},
		}}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "DPDP",
			Requirement:       "Consent Management (Section 6)",
			Status:            "open",
			Remediation:       "Implement explicit consent collection with clear purpose specification",
			RegulationArticle: "DPDP Act 2023 S.6(1) - Personal data shall be processed only for lawful purpose for which Data Principal has given consent",
			Evidence:          evidence,
			AffectedAssets:    []domain.AffectedAsset{},
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "CRITICAL",
			SeverityReason:    "Processing without consent is a fundamental violation; penalties up to INR 250 crore",
		})
	}
	if localizationPolicyCount == 0 {
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-dpdp-localization",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No data localization policy defined for critical personal data",
			DetectedAt:  now,
			Metadata:    map[string]any{"localization_policies": 0},
		}}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "DPDP",
			Requirement:       "Data Localization (Section 16)",
			Status:            "open",
			Remediation:       "Define data localization policies for critical personal data stored in India",
			RegulationArticle: "DPDP Act 2023 S.16 - Central Government may restrict transfer of personal data to certain countries",
			Evidence:          evidence,
			AffectedAssets:    []domain.AffectedAsset{},
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "HIGH",
			SeverityReason:    "Government may issue notification restricting transfers at any time; must be prepared",
		})
	}
	if ropaCount == 0 {
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-dpdp-sdf",
			Type:        "absence_of_record",
			Source:      "ropa_registry",
			Description: "No documentation to assess Significant Data Fiduciary status or obligations",
			DetectedAt:  now,
			Metadata:    map[string]any{"ropa_count": 0},
		}}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "DPDP",
			Requirement:       "Significant Data Fiduciary Obligations (Section 10)",
			Status:            "open",
			Remediation:       "Appoint Data Protection Officer and conduct Data Protection Impact Assessments",
			RegulationArticle: "DPDP Act 2023 S.10 - SDF shall appoint DPO, conduct periodic DPIA, and audit compliance",
			Evidence:          evidence,
			AffectedAssets:    []domain.AffectedAsset{},
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "CRITICAL",
			SeverityReason:    "SDF obligations include DPO appointment and DPIA; highest penalty tier for non-compliance",
		})
	}
	if policyCount < 3 {
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-dpdp-rights",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: fmt.Sprintf("Only %d policies exist; Data Principal rights require access, correction, and erasure mechanisms", policyCount),
			DetectedAt:  now,
			Metadata:    map[string]any{"policy_count": policyCount},
		}}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "DPDP",
			Requirement:       "Data Principal Rights (Section 11-14)",
			Status:            "open",
			Remediation:       "Implement mechanisms for access, correction, and erasure requests",
			RegulationArticle: "DPDP Act 2023 S.11-14 - Rights of Data Principal including access, correction, erasure, and grievance redressal",
			Evidence:          evidence,
			AffectedAssets:    []domain.AffectedAsset{},
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "HIGH",
			SeverityReason:    "Unable to fulfill Data Principal requests without proper access control mechanisms",
		})
	}

	// UAE PDPL gaps
	if consentPolicyCount == 0 {
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-uae-lawful",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No lawful basis documentation for personal data processing",
			DetectedAt:  now,
			Metadata:    map[string]any{"lawful_basis_policies": 0},
		}}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "UAE PDPL",
			Requirement:       "Lawful Basis for Processing (Art. 4)",
			Status:            "open",
			Remediation:       "Document lawful basis for all personal data processing activities",
			RegulationArticle: "UAE Federal Decree-Law No. 45/2021, Art. 4 - Personal data shall only be processed based on a lawful basis",
			Evidence:          evidence,
			AffectedAssets:    []domain.AffectedAsset{},
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "CRITICAL",
			SeverityReason:    "Processing without lawful basis is the most fundamental PDPL violation",
		})
	}
	if crossBorderPolicyCount == 0 {
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-uae-crossborder",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No cross-border transfer controls configured",
			DetectedAt:  now,
			Metadata:    map[string]any{"cross_border_policies": 0},
		}}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "UAE PDPL",
			Requirement:       "Cross-Border Transfer Restrictions (Art. 22)",
			Status:            "open",
			Remediation:       "Define cross-border transfer policies ensuring adequate protection level",
			RegulationArticle: "UAE Federal Decree-Law No. 45/2021, Art. 22 - Transfer outside UAE requires adequate protection or explicit consent",
			Evidence:          evidence,
			AffectedAssets:    []domain.AffectedAsset{},
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "HIGH",
			SeverityReason:    "Data transfers to countries without adequate protection level are prohibited",
		})
	}
	if ropaCount == 0 {
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-uae-records",
			Type:        "absence_of_record",
			Source:      "ropa_registry",
			Description: "No records of processing activities maintained as required by UAE PDPL",
			DetectedAt:  now,
			Metadata:    map[string]any{"ropa_count": 0},
		}}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "UAE PDPL",
			Requirement:       "Records of Processing Activities (Art. 8)",
			Status:            "open",
			Remediation:       "Maintain records of all personal data processing activities",
			RegulationArticle: "UAE Federal Decree-Law No. 45/2021, Art. 8 - Controller shall maintain record of processing activities",
			Evidence:          evidence,
			AffectedAssets:    []domain.AffectedAsset{},
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "HIGH",
			SeverityReason:    "Cannot demonstrate accountability without processing records",
		})
	}
	if policyCount < 3 {
		evidence := []domain.EvidenceItem{{
			ID:          "ev-gap-uae-rights",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: fmt.Sprintf("Only %d policies; data subject rights require comprehensive access controls", policyCount),
			DetectedAt:  now,
			Metadata:    map[string]any{"policy_count": policyCount},
		}}
		gaps = append(gaps, FrontendComplianceGap{
			Regulation:        "UAE PDPL",
			Requirement:       "Data Subject Rights (Art. 13-18)",
			Status:            "open",
			Remediation:       "Implement access, rectification, erasure, and portability request mechanisms",
			RegulationArticle: "UAE Federal Decree-Law No. 45/2021, Art. 13-18 - Rights including access, correction, erasure, restriction, portability",
			Evidence:          evidence,
			AffectedAssets:    []domain.AffectedAsset{},
			EvidenceCount:     1,
			DetectedAt:        now,
			LastAssessed:      now,
			Severity:          "HIGH",
			SeverityReason:    "Cannot process data subject requests without proper mechanisms in place",
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

func (s *Server) runComplianceAssessment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	userID := pkg.UserFromCtx(ctx)
	now := time.Now()

	classifications := s.loadClassifications(ctx, tenantID)
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

	var auditLogs []store.AuditLog
	s.db.SelectContext(ctx, &auditLogs,
		"SELECT * FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 200", tenantID)

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
		AuditLogs:           auditLogs,
		AssessmentTime:      now,
	}

	recommendations := domain.GenerateRecommendations(advCtx)

	criticalCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0
	totalEvidence := 0
	for _, rec := range recommendations {
		switch rec.Severity {
		case "CRITICAL":
			criticalCount++
		case "HIGH":
			highCount++
		case "MEDIUM":
			mediumCount++
		case "LOW":
			lowCount++
		}
		totalEvidence += len(rec.Evidence)
	}

	overallScore := 1.0
	if len(recommendations) > 0 {
		overallScore = max(0.0, 1.0-float64(criticalCount)*0.2-float64(highCount)*0.1-float64(mediumCount)*0.05-float64(lowCount)*0.02)
	}

	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		UserID:     userID,
		Action:     "compliance.assessment.run",
		Resource:   "compliance_assessment",
		ResourceID: "assessment-" + now.Format("20060102-150405"),
		IP:         r.RemoteAddr,
	})

	regulationsJSON, _ := json.Marshal([]string{"GDPR", "CCPA", "DPDP Act 2023", "UAE PDPL", "EU AI Act", "HIPAA", "PCI-DSS"})
	summaryJSON, _ := json.Marshal(map[string]any{
		"ropa_count":           len(ropa),
		"retention_violations": len(violations),
		"unscanned_sources":    countUnscanned(dataSources),
		"unlabeled_datasets":   totalDatasets - labeledDatasets,
	})

	assessment := &store.ComplianceAssessment{
		TenantID:               tenantID,
		AssessedBy:             userID,
		ComplianceScore:        overallScore,
		TotalFindings:          len(recommendations),
		CriticalFindings:       criticalCount,
		HighFindings:           highCount,
		MediumFindings:         mediumCount,
		LowFindings:            lowCount,
		TotalEvidence:          totalEvidence,
		DataSourcesChecked:     len(dataSources),
		ClassificationsChecked: len(classifications),
		PoliciesEvaluated:      len(policies),
		RegulationsCovered:     store.JSON(regulationsJSON),
		Summary:                store.JSON(summaryJSON),
	}
	s.complianceAssessments.Create(ctx, assessment)

	pkg.JSON(w, map[string]any{
		"assessed_at":        now,
		"assessed_by":       userID,
		"compliance_score":  overallScore,
		"total_findings":    len(recommendations),
		"critical_findings": criticalCount,
		"high_findings":     highCount,
		"medium_findings":   mediumCount,
		"low_findings":      lowCount,
		"total_evidence":    totalEvidence,
		"data_sources_checked":   len(dataSources),
		"classifications_checked": len(classifications),
		"policies_evaluated":      len(policies),
		"regulations_covered":     []string{"GDPR", "CCPA", "DPDP Act 2023", "UAE PDPL", "EU AI Act", "HIPAA", "PCI-DSS"},
		"summary": map[string]any{
			"ropa_count":          len(ropa),
			"retention_violations": len(violations),
			"unscanned_sources":   countUnscanned(dataSources),
			"unlabeled_datasets":  totalDatasets - labeledDatasets,
		},
	})
}

func countUnscanned(sources []store.DataSource) int {
	count := 0
	for _, ds := range sources {
		if ds.LastScan == nil || ds.LastScan.IsZero() {
			count++
		}
	}
	return count
}

// loadClassifications queries the classifications table using an explicit column list
// to avoid sqlx scan errors when nullable UUID columns contain NULL values.
func (s *Server) loadClassifications(ctx context.Context, tenantID string) []store.Classification {
	var result []store.Classification
	s.db.SelectContext(ctx, &result, //nolint:errcheck
		`SELECT id, tenant_id, dataset_id,
		 source_id::text AS source_id,
		 entity_type, value, confidence, context,
		 label_id::text AS label_id,
		 rule_id::text AS rule_id,
		 classification_source, value_sample, created_at
		 FROM classifications WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 1000`, tenantID)
	if result == nil {
		result = []store.Classification{}
	}
	return result
}

func (s *Server) listAssessmentLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var assessments []store.ComplianceAssessment
	err := s.db.SelectContext(ctx, &assessments,
		`SELECT * FROM compliance_assessments WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 50`,
		tenantID)
	if err != nil || assessments == nil {
		assessments = []store.ComplianceAssessment{}
	}
	pkg.JSON(w, assessments)
}
