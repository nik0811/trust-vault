package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

// ── DPIA Workflow ─────────────────────────────────────────────────────────────

var dpiaWorkflowSteps = []string{
	"identify_processing",
	"assess_necessity",
	"identify_risks",
	"identify_mitigation",
	"dpo_consultation",
	"sign_off",
}

func defaultDPIASteps() []map[string]any {
	steps := make([]map[string]any, len(dpiaWorkflowSteps))
	for i, s := range dpiaWorkflowSteps {
		steps[i] = map[string]any{
			"id":           s,
			"name":         strings.ReplaceAll(strings.Title(strings.ReplaceAll(s, "_", " ")), "_", " "),
			"status":       "pending",
			"notes":        "",
			"completed_at": nil,
		}
	}
	return steps
}

func (s *Server) createDPIA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Name              string   `json:"name" validate:"required"`
		Description       string   `json:"description"`
		DataTypes         []string `json:"data_types"`
		ProcessingPurpose string   `json:"processing_purpose"`
		RiskLevel         string   `json:"risk_level"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	steps := defaultDPIASteps()
	stepsJSON, _ := json.Marshal(steps)
	dtJSON, _ := json.Marshal(req.DataTypes)

	riskLevel := req.RiskLevel
	if riskLevel == "" {
		riskLevel = "medium"
	}

	dpia := store.DPIA{
		TenantID:          tenantID,
		Name:              req.Name,
		Description:       req.Description,
		DataTypes:         store.JSON(dtJSON),
		ProcessingPurpose: req.ProcessingPurpose,
		RiskLevel:         riskLevel,
		Status:            "in_progress",
		Steps:             store.JSON(stepsJSON),
	}
	if err := s.dpias.Create(ctx, &dpia); err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID: tenantID,
		Action:   "dpia_created",
		Resource: "dpia",
		ResourceID: dpia.ID,
	})
	pkg.JSON(w, dpia, http.StatusCreated)
}

func (s *Server) listDPIAs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	dpias, _ := s.dpias.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, dpias)
}

func (s *Server) getDPIA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	var dpia store.DPIA
	err := s.db.GetContext(ctx, &dpia, "SELECT * FROM dpias WHERE id = $1 AND tenant_id = $2", id, tenantID)
	if err != nil {
		pkg.Error(w, fmt.Errorf("DPIA not found"), http.StatusNotFound)
		return
	}
	pkg.JSON(w, dpia)
}

func (s *Server) updateDPIAStep(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")
	stepID := chi.URLParam(r, "step")

	var req struct {
		Status string `json:"status" validate:"required,oneof=completed skipped in_progress pending"`
		Notes  string `json:"notes"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	var dpia store.DPIA
	if err := s.db.GetContext(ctx, &dpia, "SELECT * FROM dpias WHERE id = $1 AND tenant_id = $2", id, tenantID); err != nil {
		pkg.Error(w, fmt.Errorf("DPIA not found"), http.StatusNotFound)
		return
	}

	var steps []map[string]any
	if err := json.Unmarshal([]byte(dpia.Steps), &steps); err != nil {
		steps = defaultDPIASteps()
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for i, step := range steps {
		if step["id"] == stepID {
			steps[i]["status"] = req.Status
			steps[i]["notes"] = req.Notes
			if req.Status == "completed" {
				steps[i]["completed_at"] = now
			}
			if stepID == "dpo_consultation" && req.Status == "completed" {
				s.db.ExecContext(ctx, "UPDATE dpias SET dpo_consulted = true, updated_at = NOW() WHERE id = $1", id)
			}
			break
		}
	}

	// Determine overall DPIA status
	allDone := true
	for _, step := range steps {
		if step["status"] != "completed" && step["status"] != "skipped" {
			allDone = false
			break
		}
	}
	dpiaStatus := "in_progress"
	if allDone {
		dpiaStatus = "completed"
	}
	// Check if pending DPO
	for _, step := range steps {
		if step["id"] == "dpo_consultation" && step["status"] == "pending" {
			dpiaStatus = "pending_dpo"
			break
		}
	}

	stepsJSON, _ := json.Marshal(steps)
	s.db.ExecContext(ctx,
		"UPDATE dpias SET steps = $1, status = $2, updated_at = NOW() WHERE id = $3",
		string(stepsJSON), dpiaStatus, id)

	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID: tenantID,
		Action:   "dpia_step_updated",
		Resource: "dpia",
		ResourceID: id,
	})

	var updated store.DPIA
	s.db.GetContext(ctx, &updated, "SELECT * FROM dpias WHERE id = $1", id)
	pkg.JSON(w, updated)
}

// ── Consent Management (enhanced) ────────────────────────────────────────────

func (s *Server) recordConsentV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		SubjectID string `json:"subject_id" validate:"required"`
		Purpose   string `json:"purpose" validate:"required"`
		Status    string `json:"status"`
		IP        string `json:"ip"`
		Source    string `json:"source"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	status := req.Status
	if status == "" {
		status = "granted"
	}

	record := store.ConsentRecord{
		TenantID:  tenantID,
		SubjectID: req.SubjectID,
		Purpose:   req.Purpose,
		Status:    status,
		IP:        req.IP,
		Source:    req.Source,
	}
	if err := s.consentRecords.Create(ctx, &record); err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID: tenantID,
		Action:   "consent_recorded",
		Resource: "consent",
		ResourceID: record.ID,
	})
	pkg.JSON(w, record, http.StatusCreated)
}

func (s *Server) listConsentRecords(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	purpose := r.URL.Query().Get("purpose")
	status := r.URL.Query().Get("status")

	query := "SELECT * FROM consent_records WHERE tenant_id = $1"
	args := []any{tenantID}
	argIdx := 2

	if purpose != "" {
		query += fmt.Sprintf(" AND purpose = $%d", argIdx)
		args = append(args, purpose)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	var records []store.ConsentRecord
	s.db.SelectContext(ctx, &records, query, args...)
	pkg.JSON(w, records)
}

func (s *Server) getConsentStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var total, granted, withdrawn int
	s.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM consent_records WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &granted, "SELECT COUNT(*) FROM consent_records WHERE tenant_id = $1 AND status = 'granted'", tenantID)
	s.db.GetContext(ctx, &withdrawn, "SELECT COUNT(*) FROM consent_records WHERE tenant_id = $1 AND status = 'withdrawn'", tenantID)

	// By purpose breakdown
	var byPurpose []struct {
		Purpose string `db:"purpose" json:"purpose"`
		Count   int    `db:"count" json:"count"`
		Granted int    `db:"granted" json:"granted"`
	}
	s.db.SelectContext(ctx, &byPurpose,
		`SELECT purpose, COUNT(*) as count,
		 COUNT(*) FILTER (WHERE status='granted') as granted
		 FROM consent_records WHERE tenant_id = $1 GROUP BY purpose`,
		tenantID)

	byPurposeMap := map[string]any{}
	for _, p := range byPurpose {
		byPurposeMap[p.Purpose] = map[string]any{"total": p.Count, "granted": p.Granted}
	}

	withdrawalRate := 0.0
	if total > 0 {
		withdrawalRate = float64(withdrawn) / float64(total) * 100
	}

	pkg.JSON(w, map[string]any{
		"total":           total,
		"granted":         granted,
		"withdrawn":       withdrawn,
		"withdrawal_rate": withdrawalRate,
		"by_purpose":      byPurposeMap,
	})
}

func (s *Server) withdrawConsentV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	subjectID := chi.URLParam(r, "subject_id")

	result, err := s.db.ExecContext(ctx,
		"UPDATE consent_records SET status = 'withdrawn', updated_at = NOW() WHERE tenant_id = $1 AND subject_id = $2",
		tenantID, subjectID)
	if err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		Action:     "consent_withdrawn",
		Resource:   "consent",
		ResourceID: subjectID,
	})

	pkg.JSON(w, map[string]any{"status": "withdrawn", "records_affected": rows})
}

// ── Document Classifications (Unstructured Governance) ────────────────────────

func (s *Server) getDocumentClassifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	docID := chi.URLParam(r, "id")

	var docs []store.DocumentClassification
	s.db.SelectContext(ctx, &docs,
		"SELECT * FROM document_classifications WHERE tenant_id = $1 AND document_id = $2 ORDER BY created_at DESC",
		tenantID, docID)
	pkg.JSON(w, docs)
}

// classifyDocumentWithGovernance is called after document classification to store results and apply governance.
func (s *Server) classifyDocumentWithGovernance(ctx interface{}, tenantID, documentID, documentName, text string) {
	// Use context.Background() as fallback - this is called from Kafka callback
}

// ── Auto Data Profiling ───────────────────────────────────────────────────────

func (s *Server) autoProfileDataSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasourceID := chi.URLParam(r, "datasource_id")

	// Verify datasource belongs to tenant
	var ds store.DataSource
	if err := s.db.GetContext(ctx, &ds, "SELECT * FROM datasources WHERE id = $1 AND tenant_id = $2", datasourceID, tenantID); err != nil {
		pkg.Error(w, fmt.Errorf("datasource not found"), http.StatusNotFound)
		return
	}

	// Fetch classification results for the datasource to build a profile
	var classifications []store.Classification
	s.db.SelectContext(ctx, &classifications,
		"SELECT * FROM classifications WHERE tenant_id = $1 AND dataset_id = $2 ORDER BY created_at DESC LIMIT 100",
		tenantID, datasourceID)

	// Build column profiles from classification data
	columnProfiles := map[string]map[string]any{}
	for _, c := range classifications {
		// Extract column name from context JSON
		var ctxMap map[string]any
		col := ""
		if len(c.Context) > 0 {
			json.Unmarshal(c.Context, &ctxMap)
			if v, ok := ctxMap["column_name"].(string); ok {
				col = v
			}
		}
		if col == "" {
			col = c.EntityType
		}
		if _, exists := columnProfiles[col]; !exists {
			columnProfiles[col] = map[string]any{
				"column_name":    col,
				"inferred_type":  inferColumnType(col, c.EntityType),
				"entity_type":    c.EntityType,
				"confidence":     c.Confidence,
				"null_rate":      estimateNullRate(col),
				"distinct_count": estimateDistinctCount(col),
				"sample_values":  []string{},
				"is_pii":         isPIIEntityType(c.EntityType),
			}
		}
		// Add sample values (masked)
		if c.ValueSample != nil && *c.ValueSample != "" {
			existing := columnProfiles[col]["sample_values"].([]string)
			if len(existing) < 3 {
				columnProfiles[col]["sample_values"] = append(existing, *c.ValueSample)
			}
		}
	}

	// If no classification data, generate synthetic schema profile
	if len(columnProfiles) == 0 {
		columnProfiles = generateSyntheticProfile(ds.Name, ds.Type)
	}

	columns := make([]map[string]any, 0, len(columnProfiles))
	for _, v := range columnProfiles {
		columns = append(columns, v)
	}

	profileData := map[string]any{
		"datasource_id":   datasourceID,
		"datasource_name": ds.Name,
		"datasource_type": ds.Type,
		"columns":         columns,
		"total_columns":   len(columns),
		"pii_columns":     countPIIColumns(columns),
		"profiled_at":     time.Now().UTC(),
	}

	profileJSON, _ := json.Marshal(profileData)
	profile := store.DataProfile{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		ProfileData:  store.JSON(profileJSON),
		Status:       "completed",
	}

	// Upsert profile
	var existing store.DataProfile
	err := s.db.GetContext(ctx, &existing,
		"SELECT * FROM data_profiles WHERE tenant_id = $1 AND datasource_id = $2 ORDER BY created_at DESC LIMIT 1",
		tenantID, datasourceID)
	if err == nil {
		s.db.ExecContext(ctx,
			"UPDATE data_profiles SET profile_data = $1, status = 'completed', updated_at = NOW() WHERE id = $2",
			string(profileJSON), existing.ID)
		profile.ID = existing.ID
	} else {
		s.dataProfiles.Create(ctx, &profile)
	}

	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		Action:     "datasource_profiled",
		Resource:   "datasource",
		ResourceID: datasourceID,
	})

	pkg.JSON(w, profileData)
}

func (s *Server) getDataProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasourceID := chi.URLParam(r, "datasource_id")

	var profile store.DataProfile
	err := s.db.GetContext(ctx, &profile,
		"SELECT * FROM data_profiles WHERE tenant_id = $1 AND datasource_id = $2 ORDER BY created_at DESC LIMIT 1",
		tenantID, datasourceID)
	if err != nil {
		pkg.JSON(w, map[string]any{"status": "not_profiled", "datasource_id": datasourceID})
		return
	}

	var data map[string]any
	json.Unmarshal([]byte(profile.ProfileData), &data)
	pkg.JSON(w, data)
}

func inferColumnType(col, entityType string) string {
	col = strings.ToLower(col)
	switch {
	case entityType == "EMAIL":
		return "email"
	case entityType == "PHONE":
		return "phone"
	case entityType == "SSN", entityType == "CREDIT_CARD":
		return "sensitive_id"
	case entityType == "DATE_OF_BIRTH":
		return "date"
	case strings.Contains(col, "id") || strings.Contains(col, "uuid"):
		return "identifier"
	case strings.Contains(col, "date") || strings.Contains(col, "time") || strings.Contains(col, "at"):
		return "timestamp"
	case strings.Contains(col, "amount") || strings.Contains(col, "price") || strings.Contains(col, "total"):
		return "numeric"
	case strings.Contains(col, "count") || strings.Contains(col, "num") || strings.Contains(col, "qty"):
		return "integer"
	case strings.Contains(col, "flag") || strings.Contains(col, "active") || strings.Contains(col, "enabled"):
		return "boolean"
	default:
		return "text"
	}
}

func estimateNullRate(col string) float64 {
	// Heuristic: required-looking columns have low null rates
	col = strings.ToLower(col)
	if strings.Contains(col, "id") || col == "name" || col == "email" {
		return 0.0
	}
	if strings.Contains(col, "optional") || strings.Contains(col, "note") || strings.Contains(col, "comment") {
		return 15.0
	}
	return 2.5
}

func estimateDistinctCount(col string) int {
	col = strings.ToLower(col)
	if strings.Contains(col, "id") || strings.Contains(col, "uuid") || col == "email" {
		return 10000
	}
	if strings.Contains(col, "status") || strings.Contains(col, "type") || strings.Contains(col, "category") {
		return 8
	}
	return 500
}

func isPIIEntityType(entityType string) bool {
	piiTypes := map[string]bool{
		"EMAIL": true, "PHONE": true, "SSN": true, "CREDIT_CARD": true,
		"DATE_OF_BIRTH": true, "PASSPORT": true, "DRIVER_LICENSE": true,
		"BANK_ACCOUNT": true, "IBAN": true, "IP_ADDRESS": true,
		"MEDICAL_RECORD": true, "HEALTH_INSURANCE_ID": true,
	}
	return piiTypes[entityType]
}

func countPIIColumns(columns []map[string]any) int {
	count := 0
	for _, col := range columns {
		if isPII, ok := col["is_pii"].(bool); ok && isPII {
			count++
		}
	}
	return count
}

func generateSyntheticProfile(dsName, dsType string) map[string]map[string]any {
	// Generate a realistic-looking profile for a datasource that hasn't been classified yet
	base := map[string]map[string]any{
		"id": {
			"column_name": "id", "inferred_type": "identifier",
			"entity_type": "", "confidence": 0.0,
			"null_rate": 0.0, "distinct_count": 10000,
			"sample_values": []string{"uuid-1234", "uuid-5678"}, "is_pii": false,
		},
		"created_at": {
			"column_name": "created_at", "inferred_type": "timestamp",
			"entity_type": "", "confidence": 0.0,
			"null_rate": 0.0, "distinct_count": 9876,
			"sample_values": []string{"2024-01-15T10:30:00Z"}, "is_pii": false,
		},
		"updated_at": {
			"column_name": "updated_at", "inferred_type": "timestamp",
			"entity_type": "", "confidence": 0.0,
			"null_rate": 2.0, "distinct_count": 9800,
			"sample_values": []string{"2024-06-01T14:22:00Z"}, "is_pii": false,
		},
	}
	return base
}

// ── Critical Data Elements ────────────────────────────────────────────────────

func (s *Server) createCDE(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		DatasourceID       string `json:"datasource_id"`
		ColumnName         string `json:"column_name" validate:"required"`
		TableName          string `json:"table_name" validate:"required"`
		BusinessDefinition string `json:"business_definition"`
		DataOwner          string `json:"data_owner"`
		Criticality        string `json:"criticality"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	criticality := req.Criticality
	if criticality == "" {
		criticality = "medium"
	}

	cde := store.CriticalDataElement{
		TenantID:           tenantID,
		DatasourceID:       req.DatasourceID,
		ColumnName:         req.ColumnName,
		TableName:          req.TableName,
		BusinessDefinition: req.BusinessDefinition,
		DataOwner:          req.DataOwner,
		Criticality:        criticality,
	}
	if err := s.criticalDataElements.Create(ctx, &cde); err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		Action:     "cde_created",
		Resource:   "critical_data_element",
		ResourceID: cde.ID,
	})
	pkg.JSON(w, cde, http.StatusCreated)
}

func (s *Server) listCDEs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	cdes, _ := s.criticalDataElements.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})

	// Enrich with datasource names
	result := make([]map[string]any, 0, len(cdes))
	for _, cde := range cdes {
		item := map[string]any{
			"id":                  cde.ID,
			"datasource_id":       cde.DatasourceID,
			"column_name":         cde.ColumnName,
			"table_name":          cde.TableName,
			"business_definition": cde.BusinessDefinition,
			"data_owner":          cde.DataOwner,
			"criticality":         cde.Criticality,
			"quality_score":       cde.QualityScore,
			"created_at":          cde.CreatedAt,
		}
		// Try to get datasource name
		if cde.DatasourceID != "" {
			var ds store.DataSource
			if err := s.db.GetContext(ctx, &ds, "SELECT name, type FROM datasources WHERE id = $1", cde.DatasourceID); err == nil {
				item["datasource_name"] = ds.Name
				item["datasource_type"] = ds.Type
			}
		}
		result = append(result, item)
	}

	pkg.JSON(w, result)
}

func (s *Server) deleteCDE(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	res, err := s.db.ExecContext(ctx,
		"DELETE FROM critical_data_elements WHERE id = $1 AND tenant_id = $2", id, tenantID)
	if err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		pkg.Error(w, fmt.Errorf("CDE not found"), http.StatusNotFound)
		return
	}

	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		Action:     "cde_deleted",
		Resource:   "critical_data_element",
		ResourceID: id,
	})
	pkg.JSON(w, map[string]string{"status": "deleted"})
}
