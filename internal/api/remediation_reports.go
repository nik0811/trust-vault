package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/securelens/securelens/internal/domain"
	"github.com/securelens/securelens/internal/events"
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
		// Respond immediately with 202; heavy advisor work runs in background.
		pkg.JSON(w, report, http.StatusAccepted)

		go func(reportID, tid string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			advCtx := s.buildAdvisorContext(bgCtx, tid)
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
			"id":           reportID,
			"title":        "SecureLens Compliance Assessment Report",
			"generated_at": time.Now(),
			"tenant_id":    tid,
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
			log.Error().Err(err).Str("report_id", reportID).Msg("failed to marshal compliance report content")
			s.db.ExecContext(bgCtx, `UPDATE reports SET status='failed', updated_at=NOW() WHERE id=$1`, reportID)
			events.Emit("report.completed", map[string]any{"report_id": reportID, "tenant_id": tid, "status": "failed"})
			return
		}

		s.db.ExecContext(bgCtx,
			`UPDATE reports SET status='completed', content=$1, updated_at=NOW() WHERE id=$2`,
			store.JSON(contentBytes), reportID)

		events.Emit("report.completed", map[string]any{"report_id": reportID, "tenant_id": tid, "status": "completed"})
	}(report.ID, tenantID)
		return
	}

	// For non-compliance report types, generate synchronously
	var reportContent map[string]any

	switch req.Type {
	case "quality":
		type qualityStat struct {
			DatasetID    string  `db:"dataset_id" json:"dataset_id"`
			Overall      float64 `db:"overall" json:"overall"`
			Completeness float64 `db:"completeness" json:"completeness"`
			Accuracy     float64 `db:"accuracy" json:"accuracy"`
			IssueCount   int     `db:"issue_count" json:"issue_count"`
		}
		qualityStats := []qualityStat{}
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
			TotalQueries   int `db:"total"`
			AllowedQueries int `db:"allowed"`
			BlockedQueries int `db:"blocked"`
			RedactedCount  int `db:"redacted"`
		}
		s.db.GetContext(ctx, &gateStats.TotalQueries, `SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1`, tenantID)
		s.db.GetContext(ctx, &gateStats.AllowedQueries, `SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1 AND action = 'allow'`, tenantID)
		s.db.GetContext(ctx, &gateStats.BlockedQueries, `SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1 AND action = 'block'`, tenantID)
		s.db.GetContext(ctx, &gateStats.RedactedCount, `SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1 AND action = 'redact'`, tenantID)

		recentQueries := []map[string]any{}
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

		recentLogs := []store.AuditLog{}
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

	var totalSources, totalClassifications, totalPolicies, totalQueries int
	s.db.GetContext(ctx, &totalSources, "SELECT COUNT(*) FROM datasources WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &totalClassifications, "SELECT COUNT(*) FROM classifications WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &totalPolicies, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &totalQueries, "SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1", tenantID)

	// Get total records scanned (count of classifications as proxy for records)
	var totalRecords int
	s.db.GetContext(ctx, &totalRecords, 
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1", tenantID)

	// Get PII detected count
	var piiDetected int
	s.db.GetContext(ctx, &piiDetected,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND entity_type IN ('PII', 'SSN', 'EMAIL', 'PHONE', 'CREDIT_CARD', 'ADDRESS', 'NAME', 'DOB', 'PHI')", tenantID)

	// Get columns classified (distinct column_name from classifications)
	var columnsClassified int
	s.db.GetContext(ctx, &columnsClassified,
		"SELECT COUNT(DISTINCT column_name) FROM classifications WHERE tenant_id = $1", tenantID)

	// Get documents processed
	var documentsProcessed int
	s.db.GetContext(ctx, &documentsProcessed,
		"SELECT COUNT(*) FROM document_classifications WHERE tenant_id = $1", tenantID)

	// Get average confidence
	var avgConfidence float64
	s.db.GetContext(ctx, &avgConfidence,
		"SELECT COALESCE(AVG(confidence), 0) FROM classifications WHERE tenant_id = $1", tenantID)

	// Get top entity types
	type EntityCount struct {
		Type  string `db:"entity_type" json:"type"`
		Count int    `db:"count" json:"count"`
	}
	var topEntities []EntityCount
	s.db.SelectContext(ctx, &topEntities,
		`SELECT entity_type, COUNT(*) as count 
		 FROM classifications WHERE tenant_id = $1 
		 GROUP BY entity_type ORDER BY count DESC LIMIT 10`, tenantID)
	if topEntities == nil {
		topEntities = []EntityCount{}
	}

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
		"total_sources":         totalSources,
		"total_classifications": totalClassifications,
		"total_policies":        totalPolicies,
		"total_queries":         totalQueries,
		"total_records":         totalRecords,
		"pii_detected":          piiDetected,
		"columns_classified":    columnsClassified,
		"documents_processed":   documentsProcessed,
		"avg_confidence":        avgConfidence,
		"top_entities":          topEntities,
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

	var totalCorrections, correctionsThisMonth, correctionsPrevMonth int
	s.db.GetContext(ctx, &totalCorrections,
		"SELECT COUNT(*) FROM feedback WHERE tenant_id = $1 AND type = 'correction'", tenantID)
	s.db.GetContext(ctx, &correctionsThisMonth,
		`SELECT COUNT(*) FROM feedback WHERE tenant_id = $1 AND type = 'correction'
		 AND created_at >= date_trunc('month', NOW())`, tenantID)
	s.db.GetContext(ctx, &correctionsPrevMonth,
		`SELECT COUNT(*) FROM feedback WHERE tenant_id = $1 AND type = 'correction'
		 AND created_at >= date_trunc('month', NOW()) - INTERVAL '1 month'
		 AND created_at < date_trunc('month', NOW())`, tenantID)

	// Accuracy improvement: percentage of corrections resolved (month-over-month decline = improvement)
	var accuracyImprovement string
	if totalCorrections > 0 && correctionsPrevMonth > 0 {
		// Declining corrections means the model needs fewer corrections → improving
		delta := float64(correctionsPrevMonth-correctionsThisMonth) / float64(correctionsPrevMonth) * 100
		if delta >= 0 {
			accuracyImprovement = fmt.Sprintf("+%.1f%%", delta)
		} else {
			accuracyImprovement = fmt.Sprintf("%.1f%%", delta)
		}
	} else if totalCorrections > 0 {
		accuracyImprovement = "+0.0%"
	} else {
		accuracyImprovement = "—"
	}

	// Knowledge cache size from dedicated table
	var cacheSize int
	s.db.GetContext(ctx, &cacheSize, "SELECT COUNT(*) FROM custom_entities WHERE tenant_id = $1", tenantID)
	var cacheSizeStr string
	if cacheSize == 1 {
		cacheSizeStr = "1 entry"
	} else {
		cacheSizeStr = fmt.Sprintf("%d entries", cacheSize)
	}

	// Cache hit rate: sum of hit_counts across knowledge_cache entries / total corrections
	var totalHits int
	s.db.GetContext(ctx, &totalHits,
		"SELECT COALESCE(SUM(hit_count), 0) FROM knowledge_cache WHERE tenant_id = $1", tenantID)
	var cacheHitRate string
	if totalCorrections > 0 && totalHits > 0 {
		rate := float64(totalHits) / float64(totalCorrections) * 100
		if rate > 100 {
			rate = 100
		}
		cacheHitRate = fmt.Sprintf("%.0f%%", rate)
	} else {
		cacheHitRate = "—"
	}

	pkg.JSON(w, map[string]any{
		"total_corrections":    totalCorrections,
		"accuracy_improvement": accuracyImprovement,
		"cache_size":           cacheSizeStr,
		"cache_hit_rate":       cacheHitRate,
	})
}

func (s *Server) createCustomEntity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Name        string `json:"name" validate:"required"`
		Pattern     string `json:"pattern" validate:"required"`
		Description string `json:"description"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	entity := store.CustomEntity{
		TenantID:    tenantID,
		Name:        req.Name,
		Pattern:     req.Pattern,
		Description: req.Description,
	}
	if _, err := s.db.NamedExecContext(ctx,
		`INSERT INTO custom_entities (tenant_id, name, pattern, description)
		 VALUES (:tenant_id, :name, :pattern, :description)
		 ON CONFLICT (tenant_id, name) DO UPDATE SET pattern = :pattern, description = :description, updated_at = NOW()
		 RETURNING id, created_at, updated_at`,
		&entity); err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	// Fetch the inserted row so we return the generated ID
	if err := s.db.GetContext(ctx, &entity,
		`SELECT * FROM custom_entities WHERE tenant_id = $1 AND name = $2`,
		tenantID, req.Name); err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	pkg.JSON(w, map[string]any{
		"id":         entity.ID,
		"name":       entity.Name,
		"examples":   entity.Pattern,
		"detections": entity.Detections,
		"accuracy":   95,
	}, http.StatusCreated)
}

func (s *Server) listCustomEntities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var entities []store.CustomEntity
	if err := s.db.SelectContext(ctx, &entities,
		`SELECT * FROM custom_entities WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID); err != nil {
		pkg.JSON(w, []any{})
		return
	}

	type CustomEntityResponse struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Examples   string `json:"examples"`
		Detections int    `json:"detections"`
		Accuracy   int    `json:"accuracy"`
	}

	result := make([]CustomEntityResponse, 0, len(entities))
	for _, e := range entities {
		// Count how many classifications matched this entity type
		var detections int
		s.db.GetContext(ctx, &detections,
			"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND entity_type = $2",
			tenantID, e.Name)

		result = append(result, CustomEntityResponse{
			ID:         e.ID,
			Name:       e.Name,
			Examples:   e.Pattern,
			Detections: detections,
			Accuracy:   95,
		})
	}

	pkg.JSON(w, result)
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
	// Return 7 weeks of data, oldest first
	trend := make([]TrendPoint, 7)
	for i := 6; i >= 0; i-- {
		var count int
		// Week i: from (i+1)*7 days ago to i*7 days ago
		weeksAgo := 6 - i
		weekLabel := fmt.Sprintf("W%d", weeksAgo+1)
		if tenantID == "" {
			s.db.GetContext(ctx, &count,
				`SELECT COUNT(*) FROM feedback
				 WHERE type = 'correction'
				 AND created_at >= NOW() - INTERVAL '1 week' * $1
				 AND created_at < NOW() - INTERVAL '1 week' * $2`,
				i+1, i)
		} else {
			s.db.GetContext(ctx, &count,
				`SELECT COUNT(*) FROM feedback
				 WHERE tenant_id = $1 AND type = 'correction'
				 AND created_at >= NOW() - INTERVAL '1 week' * $2
				 AND created_at < NOW() - INTERVAL '1 week' * $3`,
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

	var (
		totalDatasets           int
		labeledDatasets         int
		policyCount             int
		ropaCount               int
		consentPolicyCount      int
		localizationPolicyCount int
		crossBorderPolicyCount  int
	)

	type countTask struct {
		dest  *int
		query string
		args  []any
	}
	tasks := []countTask{
		{&totalDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM classifications WHERE tenant_id = $1", []any{tenantID}},
		{&labeledDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM labels WHERE tenant_id = $1", []any{tenantID}},
		{&policyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true", []any{tenantID}},
		{&ropaCount, "SELECT COUNT(*) FROM ropa WHERE tenant_id = $1", []any{tenantID}},
		{&consentPolicyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true AND type IN ('consent', 'lawful_basis')", []any{tenantID}},
		{&localizationPolicyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true AND type IN ('localization', 'data_localization')", []any{tenantID}},
		{&crossBorderPolicyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true AND type IN ('cross_border', 'transfer')", []any{tenantID}},
	}

	var wg sync.WaitGroup
	for i := range tasks {
		wg.Add(1)
		go func(t *countTask) {
			defer wg.Done()
			s.db.GetContext(ctx, t.dest, t.query, t.args...)
		}(&tasks[i])
	}
	wg.Wait()

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

	// Parse date range
	dateFrom, _ := time.Parse(time.RFC3339, req.DateFrom)
	dateTo, _ := time.Parse(time.RFC3339, req.DateTo)
	if dateFrom.IsZero() {
		dateFrom = time.Now().AddDate(0, -3, 0) // Default to 90 days ago
	}
	if dateTo.IsZero() {
		dateTo = time.Now()
	}

	// Build the defense docket with real data
	docket := map[string]any{
		"generated_at": time.Now(),
		"date_range": map[string]any{
			"from": dateFrom,
			"to":   dateTo,
		},
		"regulations": req.Regulations,
		"sections":    []map[string]any{},
	}

	sections := []map[string]any{}

	// 1. Data Classification Summary
	var classificationStats struct {
		TotalClassifications int `db:"total"`
		PIICount             int `db:"pii_count"`
		UniqueDatasets       int `db:"unique_datasets"`
	}
	s.db.GetContext(ctx, &classificationStats, `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE entity_type IN ('PII', 'SSN', 'CREDIT_CARD', 'EMAIL', 'PHONE', 'ADDRESS', 'NAME', 'DOB', 'PHI')) as pii_count,
			COUNT(DISTINCT dataset_id) as unique_datasets
		FROM classifications 
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
	`, tenantID, dateFrom, dateTo)

	sections = append(sections, map[string]any{
		"title": "Data Classification Summary",
		"type":  "classification",
		"data": map[string]any{
			"total_classifications": classificationStats.TotalClassifications,
			"pii_detections":        classificationStats.PIICount,
			"datasets_scanned":      classificationStats.UniqueDatasets,
			"coverage_percentage":   calculateCoverage(classificationStats.UniqueDatasets, classificationStats.TotalClassifications),
		},
	})

	// 2. Policy Enforcement Evidence
	var policyStats struct {
		ActivePolicies   int `db:"active"`
		TotalEvaluations int `db:"evaluations"`
	}
	s.db.GetContext(ctx, &policyStats, `
		SELECT 
			COUNT(*) FILTER (WHERE active = true) as active,
			(SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action LIKE 'policy%' AND created_at BETWEEN $2 AND $3) as evaluations
		FROM policies WHERE tenant_id = $1
	`, tenantID, dateFrom, dateTo)

	var policies []store.Policy
	s.db.SelectContext(ctx, &policies, `
		SELECT id, name, type, active, created_at FROM policies 
		WHERE tenant_id = $1 AND active = true ORDER BY created_at DESC LIMIT 10
	`, tenantID)

	policyList := make([]map[string]any, 0, len(policies))
	for _, p := range policies {
		policyList = append(policyList, map[string]any{
			"name":       p.Name,
			"type":       p.Type,
			"status":     "active",
			"created_at": p.CreatedAt,
		})
	}

	sections = append(sections, map[string]any{
		"title": "Policy Enforcement",
		"type":  "policies",
		"data": map[string]any{
			"active_policies":    policyStats.ActivePolicies,
			"policy_evaluations": policyStats.TotalEvaluations,
			"policies":           policyList,
		},
	})

	// 3. Access Control Audit Trail
	var accessLogs []store.AuditLog
	s.db.SelectContext(ctx, &accessLogs, `
		SELECT id, user_id, action, resource_type, resource_id, created_at 
		FROM audit_logs 
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
		ORDER BY created_at DESC LIMIT 50
	`, tenantID, dateFrom, dateTo)

	auditEntries := make([]map[string]any, 0, len(accessLogs))
	for _, log := range accessLogs {
		auditEntries = append(auditEntries, map[string]any{
			"timestamp":     log.CreatedAt,
			"user_id":       log.UserID,
			"action":        log.Action,
			"resource_type": log.Resource,
			"resource_id":   log.ResourceID,
		})
	}

	sections = append(sections, map[string]any{
		"title": "Access Control Audit Trail",
		"type":  "audit",
		"data": map[string]any{
			"total_events":  len(accessLogs),
			"audit_entries": auditEntries,
		},
	})

	// 4. DSAR Processing Records
	var dsarStats struct {
		TotalDSARs    int `db:"total"`
		CompletedDSARs int `db:"completed"`
		PendingDSARs   int `db:"pending"`
	}
	s.db.GetContext(ctx, &dsarStats, `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status IN ('pending', 'in_progress')) as pending
		FROM dsars 
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
	`, tenantID, dateFrom, dateTo)

	sections = append(sections, map[string]any{
		"title": "Data Subject Request Processing",
		"type":  "dsar",
		"data": map[string]any{
			"total_requests":     dsarStats.TotalDSARs,
			"completed_requests": dsarStats.CompletedDSARs,
			"pending_requests":   dsarStats.PendingDSARs,
			"compliance_rate":    calculateRate(dsarStats.CompletedDSARs, dsarStats.TotalDSARs),
		},
	})

	// 5. Data Quality Scores
	var qualityStats struct {
		AvgCompleteness float64 `db:"avg_completeness"`
		AvgAccuracy     float64 `db:"avg_accuracy"`
		AvgConsistency  float64 `db:"avg_consistency"`
	}
	s.db.GetContext(ctx, &qualityStats, `
		SELECT 
			COALESCE(AVG(completeness), 0) as avg_completeness,
			COALESCE(AVG(accuracy), 0) as avg_accuracy,
			COALESCE(AVG(consistency), 0) as avg_consistency
		FROM quality_scores 
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
	`, tenantID, dateFrom, dateTo)

	sections = append(sections, map[string]any{
		"title": "Data Quality Assessment",
		"type":  "quality",
		"data": map[string]any{
			"average_completeness": qualityStats.AvgCompleteness,
			"average_accuracy":     qualityStats.AvgAccuracy,
			"average_consistency":  qualityStats.AvgConsistency,
			"overall_score":        (qualityStats.AvgCompleteness + qualityStats.AvgAccuracy + qualityStats.AvgConsistency) / 3,
		},
	})

	// 6. Retention Policy Compliance
	var retentionStats struct {
		TotalViolations int `db:"violations"`
		ResolvedCount   int `db:"resolved"`
	}
	s.db.GetContext(ctx, &retentionStats, `
		SELECT 
			COUNT(*) as violations,
			COUNT(*) FILTER (WHERE status = 'resolved') as resolved
		FROM retention_violations 
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
	`, tenantID, dateFrom, dateTo)

	sections = append(sections, map[string]any{
		"title": "Retention Policy Compliance",
		"type":  "retention",
		"data": map[string]any{
			"total_violations":    retentionStats.TotalViolations,
			"resolved_violations": retentionStats.ResolvedCount,
			"compliance_rate":     calculateRate(retentionStats.ResolvedCount, retentionStats.TotalViolations),
		},
	})

	// 7. RoPA Summary
	var ropaCount int
	s.db.GetContext(ctx, &ropaCount, `SELECT COUNT(*) FROM ropa WHERE tenant_id = $1`, tenantID)

	var ropaEntries []store.RoPA
	s.db.SelectContext(ctx, &ropaEntries, `
		SELECT id, name, purpose, legal_basis, created_at FROM ropa 
		WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 10
	`, tenantID)

	ropaList := make([]map[string]any, 0, len(ropaEntries))
	for _, r := range ropaEntries {
		ropaList = append(ropaList, map[string]any{
			"name":        r.Name,
			"purpose":     r.Purpose,
			"legal_basis": r.LegalBasis,
			"created_at":  r.CreatedAt,
		})
	}

	sections = append(sections, map[string]any{
		"title": "Records of Processing Activities",
		"type":  "ropa",
		"data": map[string]any{
			"total_records": ropaCount,
			"entries":       ropaList,
		},
	})

	docket["sections"] = sections

	// Also create a report record for history
	report := store.Report{
		TenantID: tenantID,
		Type:     "defense_docket",
		Status:   "completed",
	}
	s.reports.Create(ctx, &report)

	pkg.JSON(w, docket)
}

func calculateCoverage(scanned, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(scanned) / float64(total) * 100
}

func calculateRate(completed, total int) float64 {
	if total == 0 {
		return 100 // No violations = 100% compliance
	}
	return float64(completed) / float64(total) * 100
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
		// Return predefined playbooks based on issue type
		pkg.JSON(w, getDefaultPlaybook(issueType))
		return
	}

	pkg.JSON(w, playbook)
}

type PlaybookStep struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func getDefaultPlaybook(issueType string) map[string]any {
	playbooks := map[string]map[string]any{
		"data_breach": {
			"issue_type": "data_breach",
			"name":       "Data Breach Response Playbook",
			"steps": []PlaybookStep{
				{Title: "Contain the Breach", Description: "Immediately isolate affected systems to prevent further data exposure. Disable compromised accounts and revoke access tokens."},
				{Title: "Assess the Scope", Description: "Identify what data was accessed, how many records were affected, and which data subjects are impacted."},
				{Title: "Notify Stakeholders", Description: "Alert internal security team, legal counsel, and executive leadership within 24 hours of discovery."},
				{Title: "Regulatory Notification", Description: "Notify relevant supervisory authorities within 72 hours (GDPR) or as required by applicable regulations."},
				{Title: "Affected Party Notification", Description: "Prepare and send breach notifications to affected data subjects with clear information about the incident and protective measures."},
				{Title: "Document and Review", Description: "Create detailed incident report, conduct root cause analysis, and implement preventive measures."},
			},
		},
		"dsar_overdue": {
			"issue_type": "dsar_overdue",
			"name":       "Overdue DSAR Response Playbook",
			"steps": []PlaybookStep{
				{Title: "Prioritize Request", Description: "Immediately escalate the overdue request to the data protection team and assign a dedicated handler."},
				{Title: "Communicate with Requestor", Description: "Send acknowledgment to the data subject explaining the delay and providing a revised timeline."},
				{Title: "Expedite Data Collection", Description: "Fast-track data gathering from all relevant systems and departments."},
				{Title: "Verify Identity", Description: "Confirm the requestor's identity if not already verified to prevent unauthorized disclosure."},
				{Title: "Compile Response Package", Description: "Prepare the data package with all personal data held, processing purposes, and third-party disclosures."},
				{Title: "Review and Deliver", Description: "Have legal/compliance review the response before delivery. Document completion in the DSAR log."},
			},
		},
		"retention_violation": {
			"issue_type": "retention_violation",
			"name":       "Data Retention Violation Playbook",
			"steps": []PlaybookStep{
				{Title: "Identify Affected Data", Description: "Locate all data that has exceeded its retention period using data discovery tools."},
				{Title: "Assess Legal Holds", Description: "Check if any data is subject to litigation holds or regulatory preservation requirements."},
				{Title: "Create Deletion Plan", Description: "Document which data will be deleted, from which systems, and the timeline for completion."},
				{Title: "Execute Secure Deletion", Description: "Perform secure deletion using approved methods. Ensure backups are also purged."},
				{Title: "Update Retention Policies", Description: "Review and update retention schedules to prevent future violations."},
				{Title: "Generate Compliance Report", Description: "Document the remediation actions taken and update the data inventory."},
			},
		},
		"consent_withdrawal": {
			"issue_type": "consent_withdrawal",
			"name":       "Consent Withdrawal Processing Playbook",
			"steps": []PlaybookStep{
				{Title: "Verify Withdrawal Request", Description: "Confirm the identity of the data subject and validate the withdrawal request."},
				{Title: "Identify Processing Activities", Description: "Map all processing activities that rely on the withdrawn consent as their legal basis."},
				{Title: "Cease Processing", Description: "Immediately stop all processing activities that were based solely on the withdrawn consent."},
				{Title: "Assess Data Retention", Description: "Determine if data can be retained under alternative legal bases or must be deleted."},
				{Title: "Update Systems", Description: "Update consent management systems and marketing preferences across all platforms."},
				{Title: "Confirm to Data Subject", Description: "Send confirmation to the data subject that their withdrawal has been processed."},
			},
		},
		"quality_degradation": {
			"issue_type": "quality_degradation",
			"name":       "Data Quality Remediation Playbook",
			"steps": []PlaybookStep{
				{Title: "Identify Quality Issues", Description: "Review data quality reports to identify specific issues: duplicates, missing values, format errors, or stale data."},
				{Title: "Assess Business Impact", Description: "Determine which business processes and decisions are affected by the quality issues."},
				{Title: "Root Cause Analysis", Description: "Investigate the source of quality degradation: data entry errors, integration issues, or system bugs."},
				{Title: "Implement Corrections", Description: "Apply data cleansing rules, merge duplicates, and fill missing values where possible."},
				{Title: "Add Quality Controls", Description: "Implement validation rules at data entry points to prevent future quality issues."},
				{Title: "Monitor and Report", Description: "Set up ongoing quality monitoring and establish quality score thresholds."},
			},
		},
		"unauthorized_access": {
			"issue_type": "unauthorized_access",
			"name":       "Unauthorized Access Response Playbook",
			"steps": []PlaybookStep{
				{Title: "Revoke Access Immediately", Description: "Disable the unauthorized user's access and revoke all active sessions and tokens."},
				{Title: "Preserve Evidence", Description: "Capture audit logs, access records, and system snapshots before any changes."},
				{Title: "Assess Data Exposure", Description: "Determine what data was accessed, viewed, or potentially exfiltrated."},
				{Title: "Investigate Root Cause", Description: "Identify how unauthorized access was obtained: credential theft, privilege escalation, or misconfiguration."},
				{Title: "Remediate Vulnerabilities", Description: "Fix the security gap that allowed unauthorized access and strengthen access controls."},
				{Title: "Report and Document", Description: "File incident report, notify affected parties if required, and update security policies."},
			},
		},
	}

	if playbook, exists := playbooks[issueType]; exists {
		return playbook
	}

	// Generic fallback
	return map[string]any{
		"issue_type": issueType,
		"name":       "Generic Compliance Remediation Playbook",
		"steps": []PlaybookStep{
			{Title: "Identify Affected Data", Description: "Locate and catalog all data assets affected by the compliance issue."},
			{Title: "Assess Impact", Description: "Evaluate the regulatory, business, and reputational impact of the issue."},
			{Title: "Create Remediation Plan", Description: "Develop a detailed action plan with timelines and responsible parties."},
			{Title: "Execute Remediation", Description: "Implement the corrective actions according to the plan."},
			{Title: "Verify Compliance", Description: "Validate that the issue has been resolved and compliance is restored."},
			{Title: "Document and Report", Description: "Create audit trail documentation and update compliance records."},
		},
	}
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

	var ropaCount int
	s.db.GetContext(ctx, &ropaCount, "SELECT COUNT(*) FROM ropa WHERE tenant_id = $1", tenantID)

	var consentCount int
	s.db.GetContext(ctx, &consentCount, "SELECT COUNT(*) FROM consent_records WHERE tenant_id = $1", tenantID)

	var dpiaCount int
	s.db.GetContext(ctx, &dpiaCount, "SELECT COUNT(*) FROM dpias WHERE tenant_id = $1", tenantID)

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

	// Calculate per-regulation scores based on actual compliance indicators
	gdprScore := calculateRegulationScore(policyCount, ropaCount, consentCount, totalClassifications, "gdpr")
	ccpaScore := calculateRegulationScore(policyCount, ropaCount, consentCount, totalClassifications, "ccpa")
	hipaaScore := calculateRegulationScore(policyCount, ropaCount, consentCount, totalClassifications, "hipaa")
	pciScore := calculateRegulationScore(policyCount, ropaCount, consentCount, totalClassifications, "pci")
	dpdpScore := calculateRegulationScore(policyCount, ropaCount, consentCount, totalClassifications, "dpdp")
	uaePdplScore := calculateRegulationScore(policyCount, ropaCount, consentCount, totalClassifications, "uae_pdpl")
	euAiActScore := calculateRegulationScore(policyCount, ropaCount, consentCount, totalClassifications, "eu_ai_act")

	pkg.JSON(w, map[string]any{
		"overall_score":    overallScore,
		"risk_level":       riskLevel,
		"gdpr_score":       gdprScore,
		"ccpa_score":       ccpaScore,
		"hipaa_score":      hipaaScore,
		"pci_score":        pciScore,
		"dpdp_score":       dpdpScore,
		"uae_pdpl_score":   uaePdplScore,
		"eu_ai_act_score":  euAiActScore,
		"factors": map[string]float64{
			"data_exposure":   dataExposure,
			"policy_coverage": policyCoverage,
			"compliance_gaps": complianceGaps,
		},
	})
}

func calculateRegulationScore(policyCount, ropaCount, consentCount, classificationCount int, regulation string) float64 {
	baseScore := 0.5

	// Policy coverage contributes 30%
	policyContrib := min(1.0, float64(policyCount)/3.0) * 0.3

	// RoPA coverage contributes 20% (especially important for GDPR, DPDP)
	ropaContrib := 0.0
	if ropaCount > 0 {
		ropaContrib = 0.2
	}

	// Consent management contributes 20% (important for GDPR, CCPA, DPDP)
	consentContrib := 0.0
	if consentCount > 0 {
		consentContrib = 0.2
	}

	// Classification coverage contributes 30%
	classContrib := 0.0
	if classificationCount > 0 {
		classContrib = 0.3
	}

	score := baseScore + policyContrib + ropaContrib + consentContrib + classContrib

	// Adjust based on regulation-specific requirements
	switch regulation {
	case "gdpr":
		if ropaCount == 0 {
			score -= 0.15 // RoPA is mandatory for GDPR
		}
	case "hipaa":
		// HIPAA requires specific PHI handling
		score = min(score, 0.85) // Cap until PHI-specific policies exist
	case "pci":
		// PCI-DSS requires specific card data handling
		score = min(score, 0.80) // Cap until PCI-specific controls exist
	case "eu_ai_act":
		// EU AI Act is newer, give benefit of doubt if AI governance exists
		score = min(score+0.1, 1.0)
	}

	return min(1.0, max(0.0, score))
}

func (s *Server) runComplianceAssessment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	userID := pkg.UserFromCtx(ctx)

	assessmentID := pkg.GenerateID()

	// Insert a pending record so callers can poll for completion.
	s.db.ExecContext(ctx,
		`INSERT INTO compliance_assessments (id, tenant_id, status, assessed_by, created_at, updated_at)
		 VALUES ($1, $2, 'running', $3, NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		assessmentID, tenantID, userID)

	// Respond immediately with 202 so the HTTP handler is not blocked.
	pkg.JSON(w, map[string]any{"assessment_id": assessmentID, "status": "running"}, http.StatusAccepted)

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		now := time.Now()

		classifications := s.loadClassifications(bgCtx, tenantID)
		policies, _ := s.policies.List(bgCtx, tenantID, store.ListOpts{Limit: 100})
		if policies == nil {
			policies = []store.Policy{}
		}
		labels, _ := s.labels.List(bgCtx, tenantID, store.ListOpts{Limit: 1000})
		if labels == nil {
			labels = []store.Label{}
		}
		violations, _ := s.retentionViolations.List(bgCtx, tenantID, store.ListOpts{Limit: 100})
		if violations == nil {
			violations = []store.RetentionViolation{}
		}
		dataSources, _ := s.datasources.List(bgCtx, tenantID, store.ListOpts{Limit: 100})
		if dataSources == nil {
			dataSources = []store.DataSource{}
		}
		ropa, _ := s.ropa.List(bgCtx, tenantID, store.ListOpts{Limit: 100})
		if ropa == nil {
			ropa = []store.RoPA{}
		}

		var auditLogs []store.AuditLog
		s.db.SelectContext(bgCtx, &auditLogs,
			"SELECT * FROM audit_logs WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 200", tenantID)

		var totalDatasets, labeledDatasets int
		s.db.GetContext(bgCtx, &totalDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM classifications WHERE tenant_id = $1", tenantID)
		s.db.GetContext(bgCtx, &labeledDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM labels WHERE tenant_id = $1", tenantID)

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

		criticalCount, highCount, mediumCount, lowCount, totalEvidence := 0, 0, 0, 0, 0
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

		regulationsJSON, _ := json.Marshal([]string{"GDPR", "CCPA", "DPDP Act 2023", "UAE PDPL", "EU AI Act", "HIPAA", "PCI-DSS"})
		summaryJSON, _ := json.Marshal(map[string]any{
			"ropa_count":           len(ropa),
			"retention_violations": len(violations),
			"unscanned_sources":    countUnscanned(dataSources),
			"unlabeled_datasets":   totalDatasets - labeledDatasets,
		})

		// Upsert the final result into the previously inserted row.
		s.db.ExecContext(bgCtx,
			`UPDATE compliance_assessments
			 SET status = 'completed',
			     assessed_by = $1,
			     compliance_score = $2,
			     total_findings = $3,
			     critical_findings = $4,
			     high_findings = $5,
			     medium_findings = $6,
			     low_findings = $7,
			     total_evidence = $8,
			     data_sources_checked = $9,
			     classifications_checked = $10,
			     policies_evaluated = $11,
			     regulations_covered = $12,
			     summary = $13,
			     updated_at = NOW()
			 WHERE id = $14`,
			userID, overallScore, len(recommendations),
			criticalCount, highCount, mediumCount, lowCount,
			totalEvidence, len(dataSources), len(classifications),
			len(policies), regulationsJSON, summaryJSON, assessmentID)

		s.auditLogs.Create(bgCtx, &store.AuditLog{
			TenantID:   tenantID,
			UserID:     userID,
			Action:     "compliance.assessment.run",
			Resource:   "compliance_assessment",
			ResourceID: assessmentID,
		})

		events.Emit("compliance.assessment.completed", map[string]any{
			"assessment_id":    assessmentID,
			"tenant_id":        tenantID,
			"compliance_score": overallScore,
			"total_findings":   len(recommendations),
		})
	}()
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
	err := s.db.SelectContext(ctx, &result,
		`SELECT id, tenant_id,
		 COALESCE(dataset_id, '') AS dataset_id,
		 source_id::text AS source_id,
		 COALESCE(entity_type, '') AS entity_type,
		 COALESCE(value, '') AS value,
		 COALESCE(confidence, 0) AS confidence,
		 COALESCE(context, '{}') AS context,
		 label_id::text AS label_id,
		 rule_id::text AS rule_id,
		 classification_source, value_sample, created_at
		 FROM classifications WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 1000`, tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("loadClassifications failed")
	}
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

func (s *Server) getAdvisorOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var piiCount, totalClassifications int
	s.db.GetContext(ctx, &piiCount,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND entity_type IN ('PII', 'SSN', 'CREDIT_CARD', 'PHI')", tenantID)
	s.db.GetContext(ctx, &totalClassifications,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1", tenantID)

	var policyCount int
	s.db.GetContext(ctx, &policyCount, "SELECT COUNT(*) FROM policies WHERE tenant_id = $1 AND active = true", tenantID)

	var dsarOverdue int
	s.db.GetContext(ctx, &dsarOverdue,
		"SELECT COUNT(*) FROM dsars WHERE tenant_id = $1 AND status = 'pending' AND due_date < NOW()", tenantID)

	var retentionViolations int
	s.db.GetContext(ctx, &retentionViolations,
		"SELECT COUNT(*) FROM retention_violations WHERE tenant_id = $1 AND resolved = false", tenantID)

	var gapsCount int
	s.db.GetContext(ctx, &gapsCount,
		"SELECT COUNT(*) FROM compliance_assessments WHERE tenant_id = $1 AND score < 70", tenantID)

	riskScore := 100
	if totalClassifications > 0 {
		piiRatio := float64(piiCount) / float64(totalClassifications)
		riskScore -= int(piiRatio * 30)
	}
	if policyCount < 3 {
		riskScore -= 20
	}
	riskScore -= dsarOverdue * 5
	riskScore -= retentionViolations * 3
	if riskScore < 0 {
		riskScore = 0
	}

	riskLevel := "low"
	if riskScore < 50 {
		riskLevel = "critical"
	} else if riskScore < 70 {
		riskLevel = "high"
	} else if riskScore < 85 {
		riskLevel = "medium"
	}

	var recentAssessments []store.ComplianceAssessment
	s.db.SelectContext(ctx, &recentAssessments,
		`SELECT * FROM compliance_assessments WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 5`, tenantID)

	pkg.JSON(w, map[string]any{
		"risk_score":           riskScore,
		"risk_level":           riskLevel,
		"compliance_gaps":      gapsCount,
		"dsar_overdue":         dsarOverdue,
		"retention_violations": retentionViolations,
		"active_policies":      policyCount,
		"pii_exposure":         piiCount,
		"recent_assessments":   recentAssessments,
	})
}

func (s *Server) listPlaybooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var playbooks []store.Playbook
	err := s.db.SelectContext(ctx, &playbooks,
		`SELECT * FROM playbooks WHERE tenant_id = $1 OR tenant_id IS NULL ORDER BY name`, tenantID)
	if err != nil || len(playbooks) == 0 {
		defaultPlaybooks := []map[string]any{
			{
				"id":          "data_breach",
				"issue_type":  "data_breach",
				"name":        "Data Breach Response Playbook",
				"description": "Step-by-step guide for responding to data breaches",
				"steps_count": 6,
			},
			{
				"id":          "dsar_overdue",
				"issue_type":  "dsar_overdue",
				"name":        "Overdue DSAR Response Playbook",
				"description": "Process for handling overdue data subject access requests",
				"steps_count": 6,
			},
			{
				"id":          "retention_violation",
				"issue_type":  "retention_violation",
				"name":        "Data Retention Violation Playbook",
				"description": "Remediation steps for data retention policy violations",
				"steps_count": 6,
			},
			{
				"id":          "consent_withdrawal",
				"issue_type":  "consent_withdrawal",
				"name":        "Consent Withdrawal Processing Playbook",
				"description": "Handling consent withdrawal requests from data subjects",
				"steps_count": 6,
			},
			{
				"id":          "quality_degradation",
				"issue_type":  "quality_degradation",
				"name":        "Data Quality Remediation Playbook",
				"description": "Steps to address data quality issues and degradation",
				"steps_count": 6,
			},
			{
				"id":          "unauthorized_access",
				"issue_type":  "unauthorized_access",
				"name":        "Unauthorized Access Response Playbook",
				"description": "Response procedures for unauthorized data access incidents",
				"steps_count": 6,
			},
		}
		pkg.JSON(w, defaultPlaybooks)
		return
	}

	pkg.JSON(w, playbooks)
}

func (s *Server) getDefenseDocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	dateFrom := time.Now().AddDate(0, -3, 0)
	dateTo := time.Now()

	if fromStr := r.URL.Query().Get("date_from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			dateFrom = t
		}
	}
	if toStr := r.URL.Query().Get("date_to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			dateTo = t
		}
	}

	docket := map[string]any{
		"generated_at": time.Now(),
		"date_range": map[string]any{
			"from": dateFrom,
			"to":   dateTo,
		},
		"sections": []map[string]any{},
	}

	sections := []map[string]any{}

	var classificationStats struct {
		TotalClassifications int `db:"total"`
		PIICount             int `db:"pii_count"`
		UniqueDatasets       int `db:"unique_datasets"`
	}
	s.db.GetContext(ctx, &classificationStats, `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE entity_type IN ('PII', 'SSN', 'CREDIT_CARD', 'EMAIL', 'PHONE', 'ADDRESS', 'NAME', 'DOB', 'PHI')) as pii_count,
			COUNT(DISTINCT dataset_id) as unique_datasets
		FROM classifications 
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
	`, tenantID, dateFrom, dateTo)

	sections = append(sections, map[string]any{
		"title": "Data Classification Summary",
		"type":  "classification",
		"data": map[string]any{
			"total_classifications": classificationStats.TotalClassifications,
			"pii_detections":        classificationStats.PIICount,
			"datasets_scanned":      classificationStats.UniqueDatasets,
		},
	})

	var policyStats struct {
		ActivePolicies   int `db:"active"`
		TotalEvaluations int `db:"evaluations"`
	}
	s.db.GetContext(ctx, &policyStats, `
		SELECT 
			COUNT(*) FILTER (WHERE active = true) as active,
			(SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action LIKE 'policy%' AND created_at BETWEEN $2 AND $3) as evaluations
		FROM policies WHERE tenant_id = $1
	`, tenantID, dateFrom, dateTo)

	sections = append(sections, map[string]any{
		"title": "Policy Enforcement",
		"type":  "policies",
		"data": map[string]any{
			"active_policies":    policyStats.ActivePolicies,
			"policy_evaluations": policyStats.TotalEvaluations,
		},
	})

	var dsarStats struct {
		TotalDSARs     int `db:"total"`
		CompletedDSARs int `db:"completed"`
		PendingDSARs   int `db:"pending"`
	}
	s.db.GetContext(ctx, &dsarStats, `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'pending') as pending
		FROM dsars WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
	`, tenantID, dateFrom, dateTo)

	sections = append(sections, map[string]any{
		"title": "DSAR Processing",
		"type":  "dsar",
		"data": map[string]any{
			"total_requests":     dsarStats.TotalDSARs,
			"completed_requests": dsarStats.CompletedDSARs,
			"pending_requests":   dsarStats.PendingDSARs,
		},
	})

	docket["sections"] = sections
	pkg.JSON(w, docket)
}
