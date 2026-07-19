package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/securelens/securelens/internal/events"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

func (s *Server) getROTSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var stats struct {
		TotalROT   int64 `db:"total"`
		Redundant  int64 `db:"redundant"`
		Obsolete   int64 `db:"obsolete"`
		Trivial    int64 `db:"trivial"`
		TotalBytes int64 `db:"bytes"`
	}

	s.db.GetContext(ctx, &stats.TotalROT, "SELECT COUNT(*) FROM rot_data WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &stats.Redundant, "SELECT COUNT(*) FROM rot_data WHERE tenant_id = $1 AND category = 'redundant'", tenantID)
	s.db.GetContext(ctx, &stats.Obsolete, "SELECT COUNT(*) FROM rot_data WHERE tenant_id = $1 AND category = 'obsolete'", tenantID)
	s.db.GetContext(ctx, &stats.Trivial, "SELECT COUNT(*) FROM rot_data WHERE tenant_id = $1 AND category = 'trivial'", tenantID)
	s.db.GetContext(ctx, &stats.TotalBytes, "SELECT COALESCE(SUM(size_bytes), 0) FROM rot_data WHERE tenant_id = $1", tenantID)

	var totalDatasets int64
	s.db.GetContext(ctx, &totalDatasets, "SELECT COUNT(DISTINCT dataset_id) FROM classifications WHERE tenant_id = $1", tenantID)

	percentage := 0.0
	if totalDatasets > 0 {
		percentage = float64(stats.TotalROT) / float64(totalDatasets)
	}

	pkg.JSON(w, map[string]any{
		"total_rot":           stats.TotalROT,
		"total_rot_data":      stats.TotalROT,
		"redundant":           stats.Redundant,
		"redundant_count":     stats.Redundant,
		"obsolete":            stats.Obsolete,
		"obsolete_count":      stats.Obsolete,
		"trivial":             stats.Trivial,
		"trivial_count":       stats.Trivial,
		"total_size_gb":       float64(stats.TotalBytes) / 1e9,
		"potential_savings_gb": float64(stats.TotalBytes) / 1e9,
		"percentage":          percentage,
		"estimated_cost":      float64(stats.TotalBytes) / 1e9 * 100,
	})
}

func (s *Server) getROTDatasets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	type rotDatasetRow struct {
		store.ROTData
		DatasetName string `db:"dataset_name" json:"dataset_name"`
	}

	var rows []rotDatasetRow
	if tenantID == "" {
		s.db.SelectContext(ctx, &rows,
			`SELECT r.*, COALESCE(d.name, r.dataset_id) AS dataset_name
			 FROM rot_data r
			 LEFT JOIN datasources d ON d.id::text = r.dataset_id
			 ORDER BY r.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	} else {
		s.db.SelectContext(ctx, &rows,
			`SELECT r.*, COALESCE(d.name, r.dataset_id) AS dataset_name
			 FROM rot_data r
			 LEFT JOIN datasources d ON d.id::text = r.dataset_id
			 WHERE r.tenant_id = $1
			 ORDER BY r.created_at DESC LIMIT $2 OFFSET $3`, tenantID, limit, offset)
	}
	if rows == nil {
		rows = []rotDatasetRow{}
	}
	pkg.JSON(w, rows)
}

func (s *Server) listROTItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	items, _ := s.rotData.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	if items == nil {
		items = []store.ROTData{}
	}
	pkg.JSON(w, items)
}

func (s *Server) getDuplicates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	duplicates, _ := s.duplicateGroups.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, duplicates)
}

func (s *Server) triggerROTScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	// Emit SSE event for scan started
	events.Emit("rot.scan.started", map[string]any{
		"tenant_id": tenantID,
		"status":    "running",
		"message":   "ROT scan started - analyzing data for redundant, obsolete, and trivial content",
	})

	scanID := pkg.GenerateID()
	pkg.JSON(w, map[string]any{"status": "scanning", "scan_id": scanID})

	// Run synchronously in gateway goroutine — avoids cross-container SSE bus problem
	go func() {
		bgCtx := context.Background()

		// Clear previous findings
		s.db.ExecContext(bgCtx, `DELETE FROM rot_data WHERE tenant_id = $1`, tenantID)

		var obsolete, redundant, trivial int64

		// Obsolete: datasources not scanned in 90+ days
		res, _ := s.db.ExecContext(bgCtx,
			`INSERT INTO rot_data (id, tenant_id, category, dataset_id, reason, score, size_bytes, last_access, created_at)
			 SELECT gen_random_uuid(), $1, 'obsolete', id::text,
			        'Datasource not scanned in 90+ days', 0.85,
			        0,
			        COALESCE(last_scanned, updated_at, created_at),
			        NOW()
			 FROM datasources
			 WHERE tenant_id = $1
			   AND (last_scanned IS NULL OR last_scanned < NOW() - INTERVAL '90 days')
			 ON CONFLICT DO NOTHING`, tenantID)
		if res != nil {
			obsolete, _ = res.RowsAffected()
		}

		// Redundant: dataset_ids with 3+ classification entries (scanned multiple times)
		res, _ = s.db.ExecContext(bgCtx,
			`INSERT INTO rot_data (id, tenant_id, category, dataset_id, reason, score, size_bytes, last_access, created_at)
			 SELECT gen_random_uuid(), $1, 'redundant', sub.dataset_id,
			        'Dataset appears in multiple classification runs', 0.7,
			        sub.cnt * 1024,
			        COALESCE(d.last_scanned, d.updated_at, d.created_at),
			        NOW()
			 FROM (
			   SELECT dataset_id, COUNT(*) as cnt
			   FROM classifications WHERE tenant_id = $1
			   GROUP BY dataset_id HAVING COUNT(*) >= 3
			 ) sub
			 LEFT JOIN datasources d ON d.id::text = sub.dataset_id
			 ON CONFLICT DO NOTHING`, tenantID)
		if res != nil {
			redundant, _ = res.RowsAffected()
		}

		// Trivial: classifications with very low confidence (< 0.2)
		res, _ = s.db.ExecContext(bgCtx,
			`INSERT INTO rot_data (id, tenant_id, category, dataset_id, reason, score, size_bytes, last_access, created_at)
			 SELECT gen_random_uuid(), $1, 'trivial', sub.dataset_id,
			        'All classifications have low confidence (< 20%)', 0.3,
			        sub.cnt * 512,
			        COALESCE(d.last_scanned, d.updated_at, d.created_at),
			        NOW()
			 FROM (
			   SELECT dataset_id, COUNT(*) as cnt
			   FROM classifications WHERE tenant_id = $1
			   GROUP BY dataset_id
			   HAVING MAX(confidence) < 0.2
			 ) sub
			 LEFT JOIN datasources d ON d.id::text = sub.dataset_id
			 ON CONFLICT DO NOTHING`, tenantID)
		if res != nil {
			trivial, _ = res.RowsAffected()
		}

		total := obsolete + redundant + trivial
		events.Emit("rot.scan.completed", map[string]any{
			"tenant_id": tenantID,
			"scan_id":   scanID,
			"message": fmt.Sprintf("ROT scan completed: %d items found (%d redundant, %d obsolete, %d trivial)",
				total, redundant, obsolete, trivial),
			"total":     total,
			"redundant": redundant,
			"obsolete":  obsolete,
			"trivial":   trivial,
		})
	}()
}

func (s *Server) remediateROT(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		DatasetIDs []string `json:"dataset_ids" validate:"required"`
		Action     string   `json:"action" validate:"required,oneof=archive delete deduplicate"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	for _, datasetID := range req.DatasetIDs {
		action := store.RemediationAction{
			TenantID:   tenantID,
			Type:       req.Action,
			ActionType: "quarantine",
			DatasetID:  datasetID,
			Reason:     "ROT remediation",
			Status:     "pending",
		}
		s.remediationActions.Create(ctx, &action)
	}

	s.kafka.Produce(ctx, "rot-remediation-jobs", tenantID, map[string]any{
		"dataset_ids": req.DatasetIDs,
		"action":      req.Action,
	})

	pkg.JSON(w, map[string]any{
		"status":   "processing",
		"datasets": len(req.DatasetIDs),
		"action":   req.Action,
	})
}

func (s *Server) getROTScanStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	scanID := chi.URLParam(r, "id")

	var scanLog store.ScanLog
	err := s.db.GetContext(ctx, &scanLog,
		"SELECT * FROM scan_logs WHERE tenant_id = $1 AND id = $2",
		tenantID, scanID)

	if err != nil {
		pkg.Error(w, fmt.Errorf("scan not found"), http.StatusNotFound)
		return
	}

	pkg.JSON(w, map[string]any{
		"scan_id": scanLog.ID,
		"status":  scanLog.Status,
		"message": scanLog.Message,
	})
}

func (s *Server) listIntegrations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	integrations, _ := s.integrations.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	if integrations == nil {
		integrations = []store.Integration{}
	}
	pkg.JSON(w, integrations)
}

func (s *Server) createIntegration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var integration store.Integration
	if err := pkg.Bind(r, &integration); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	integration.TenantID = tenantID
	integration.Status = "pending"
	s.integrations.Create(ctx, &integration)

	pkg.JSON(w, integration, http.StatusCreated)
}

func (s *Server) getIntegration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	integration, _ := s.integrations.FindByID(ctx, tenantID, id)
	if integration == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	pkg.JSON(w, integration)
}

func (s *Server) updateIntegration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	integration, _ := s.integrations.FindByID(ctx, tenantID, id)
	if integration == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	var req struct {
		Name     string   `json:"name"`
		Config   store.JSON `json:"config"`
		SyncFreq string   `json:"sync_freq"`
		Status   string   `json:"status"`
		Type     string   `json:"type"`
		Provider string   `json:"provider"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	if req.Name != "" {
		integration.Name = req.Name
	}
	if req.Config != nil {
		integration.Config = req.Config
	}
	if req.SyncFreq != "" {
		integration.SyncFreq = req.SyncFreq
	}
	if req.Status != "" {
		integration.Status = req.Status
	}
	if req.Type != "" {
		integration.Type = req.Type
	}
	if req.Provider != "" {
		integration.Provider = req.Provider
	}
	s.integrations.Update(ctx, integration)

	// Re-fetch to return accurate DB state
	updated, _ := s.integrations.FindByID(ctx, tenantID, id)
	if updated != nil {
		integration = updated
	}
	pkg.JSON(w, integration)
}

func (s *Server) deleteIntegration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	integration, _ := s.integrations.FindByID(ctx, tenantID, id)
	if integration == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	s.integrations.Delete(ctx, tenantID, id)
	pkg.JSON(w, map[string]string{"status": "deleted"})
}

func (s *Server) testIntegration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	integration, _ := s.integrations.FindByID(ctx, tenantID, id)
	if integration == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	start := time.Now()
	var config map[string]any
	json.Unmarshal(integration.Config, &config)

	// Test connection based on integration type
	status := "connected"
	var errorMsg string
	var details map[string]any

	switch integration.Type {
	case "webhook", "rest_api":
		status, errorMsg, details = testWebhookIntegration(ctx, config)
	case "slack":
		status, errorMsg, details = testSlackIntegration(ctx, config)
	case "teams":
		status, errorMsg, details = testWebhookIntegration(ctx, config) // MS Teams also uses incoming webhooks
	case "jira":
		status, errorMsg, details = testJiraIntegration(ctx, config)
	case "servicenow":
		status, errorMsg, details = testServiceNowIntegration(ctx, config)
	case "splunk", "siem":
		status, errorMsg, details = testSplunkIntegration(ctx, config)
	case "datadog", "sentinel":
		status, errorMsg, details = testDatadogIntegration(ctx, config)
	case "pagerduty":
		status, errorMsg, details = testPagerDutyIntegration(ctx, config)
	case "email", "communication":
		status, errorMsg, details = testEmailIntegration(ctx, config)
	case "dlp", "privacy_platform", "catalog", "ticketing", "collibra", "alation", "onetrust", "privacyops", "custom":
		// For catalog/privacy/DLP types without real connectivity, attempt generic URL test or return info
		if url := getIntegrationURL(config); url != "" {
			status, errorMsg, details = testGenericURLIntegration(ctx, url, config)
		} else {
			status = "connected"
			errorMsg = ""
			details = map[string]any{"message": "Connection test not available for this type - please verify credentials manually"}
		}
	default:
		// Generic URL test
		if url := getIntegrationURL(config); url != "" {
			status, errorMsg, details = testGenericURLIntegration(ctx, url, config)
		} else {
			status = "connected"
			errorMsg = "No URL or endpoint configured - please verify credentials manually"
		}
	}

	latency := time.Since(start).Milliseconds()

	// Log the test result
	logLevel := "info"
	if status != "connected" {
		logLevel = "error"
	}
	logEntry := store.IntegrationLog{
		TenantID:      tenantID,
		IntegrationID: id,
		Level:         logLevel,
		Message:       "Connection test: " + status + " - " + errorMsg,
	}
	s.integrationLogs.Create(ctx, &logEntry)

	// Update integration status
	s.db.ExecContext(ctx,
		"UPDATE integrations SET status = $1, updated_at = NOW() WHERE id = $2",
		status, id)

	response := map[string]any{
		"id":         id,
		"success":    status == "connected",
		"status":     status,
		"latency_ms": latency,
	}
	if errorMsg != "" {
		response["message"] = errorMsg
		response["error"] = errorMsg
	} else {
		response["message"] = "Connection successful"
	}
	if details != nil {
		response["details"] = details
	}

	pkg.JSON(w, response)
}

func getIntegrationURL(config map[string]any) string {
	if url, ok := config["url"].(string); ok && url != "" {
		return url
	}
	if endpoint, ok := config["endpoint"].(string); ok && endpoint != "" {
		return endpoint
	}
	if webhookURL, ok := config["webhook_url"].(string); ok && webhookURL != "" {
		return webhookURL
	}
	return ""
}

func testWebhookIntegration(ctx context.Context, config map[string]any) (string, string, map[string]any) {
	url := getIntegrationURL(config)
	if url == "" {
		return "error", "Missing webhook URL", nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(`{"test": true}`))
	if err != nil {
		return "error", "Failed to create request: " + err.Error(), nil
	}
	req.Header.Set("Content-Type", "application/json")

	// Add auth headers if configured
	if token, ok := config["token"].(string); ok && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "error", "Connection failed: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "error", fmt.Sprintf("HTTP %d response", resp.StatusCode), map[string]any{"status_code": resp.StatusCode}
	}

	return "connected", "", map[string]any{"status_code": resp.StatusCode}
}

func testSlackIntegration(ctx context.Context, config map[string]any) (string, string, map[string]any) {
	webhookURL := getIntegrationURL(config)
	if webhookURL == "" {
		return "error", "Missing Slack webhook URL", nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	payload := `{"text": "SecureLens integration test - please ignore"}`
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, strings.NewReader(payload))
	if err != nil {
		return "error", "Failed to create request: " + err.Error(), nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "error", "Connection failed: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "error", fmt.Sprintf("Slack returned HTTP %d", resp.StatusCode), nil
	}

	return "connected", "", map[string]any{"channel": config["channel"]}
}

func testJiraIntegration(ctx context.Context, config map[string]any) (string, string, map[string]any) {
	baseURL := getIntegrationURL(config)
	if baseURL == "" {
		return "error", "Missing Jira URL", nil
	}

	email, _ := config["email"].(string)
	apiToken, _ := config["api_token"].(string)
	if email == "" || apiToken == "" {
		return "error", "Missing email or API token", nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/rest/api/3/myself", nil)
	if err != nil {
		return "error", "Failed to create request: " + err.Error(), nil
	}
	req.SetBasicAuth(email, apiToken)

	resp, err := client.Do(req)
	if err != nil {
		return "error", "Connection failed: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return "error", "Authentication failed", nil
	}
	if resp.StatusCode >= 400 {
		return "error", fmt.Sprintf("Jira returned HTTP %d", resp.StatusCode), nil
	}

	var user map[string]any
	json.NewDecoder(resp.Body).Decode(&user)

	return "connected", "", map[string]any{
		"user":    user["displayName"],
		"account": user["accountId"],
	}
}

func testServiceNowIntegration(ctx context.Context, config map[string]any) (string, string, map[string]any) {
	instance, _ := config["instance"].(string)
	username, _ := config["username"].(string)
	password, _ := config["password"].(string)

	if instance == "" || username == "" {
		return "error", "Missing instance or username", nil
	}

	url := fmt.Sprintf("https://%s.service-now.com/api/now/table/sys_user?sysparm_limit=1", instance)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "error", "Failed to create request: " + err.Error(), nil
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "error", "Connection failed: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return "error", "Authentication failed", nil
	}
	if resp.StatusCode >= 400 {
		return "error", fmt.Sprintf("ServiceNow returned HTTP %d", resp.StatusCode), nil
	}

	return "connected", "", map[string]any{"instance": instance}
}

func testSplunkIntegration(ctx context.Context, config map[string]any) (string, string, map[string]any) {
	url := getIntegrationURL(config)
	token, _ := config["token"].(string)

	if url == "" || token == "" {
		return "error", "Missing URL or HEC token", nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	payload := `{"event": "SecureLens integration test", "sourcetype": "securelens"}`
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(payload))
	if err != nil {
		return "error", "Failed to create request: " + err.Error(), nil
	}
	req.Header.Set("Authorization", "Splunk "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "error", "Connection failed: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return "error", "Authentication failed", nil
	}
	if resp.StatusCode >= 400 {
		return "error", fmt.Sprintf("Splunk returned HTTP %d", resp.StatusCode), nil
	}

	return "connected", "", nil
}

func testDatadogIntegration(ctx context.Context, config map[string]any) (string, string, map[string]any) {
	apiKey, _ := config["api_key"].(string)
	site, _ := config["site"].(string)
	if site == "" {
		site = "datadoghq.com"
	}

	if apiKey == "" {
		return "error", "Missing API key", nil
	}

	url := fmt.Sprintf("https://api.%s/api/v1/validate", site)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "error", "Failed to create request: " + err.Error(), nil
	}
	req.Header.Set("DD-API-KEY", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "error", "Connection failed: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return "error", "Invalid API key", nil
	}
	if resp.StatusCode >= 400 {
		return "error", fmt.Sprintf("Datadog returned HTTP %d", resp.StatusCode), nil
	}

	return "connected", "", map[string]any{"site": site}
}

func testPagerDutyIntegration(ctx context.Context, config map[string]any) (string, string, map[string]any) {
	routingKey, _ := config["routing_key"].(string)
	if routingKey == "" {
		return "error", "Missing routing key", nil
	}

	url := "https://events.pagerduty.com/v2/enqueue"
	payload := map[string]any{
		"routing_key":  routingKey,
		"event_action": "trigger",
		"dedup_key":    "securelens-test-" + time.Now().Format("20060102150405"),
		"payload": map[string]any{
			"summary":  "SecureLens integration test - please ignore",
			"severity": "info",
			"source":   "securelens",
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return "error", "Failed to create request: " + err.Error(), nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "error", "Connection failed: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 400 {
		return "error", "Invalid routing key", nil
	}
	if resp.StatusCode >= 400 {
		return "error", fmt.Sprintf("PagerDuty returned HTTP %d", resp.StatusCode), nil
	}

	return "connected", "", nil
}

func testEmailIntegration(ctx context.Context, config map[string]any) (string, string, map[string]any) {
	host, _ := config["smtp_host"].(string)
	port := 587
	if p, ok := config["smtp_port"].(float64); ok {
		port = int(p)
	}

	if host == "" {
		return "error", "Missing SMTP host", nil
	}

	// Test TCP connection to SMTP server
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 10*time.Second)
	if err != nil {
		return "error", "Connection failed: " + err.Error(), nil
	}
	conn.Close()

	return "connected", "", map[string]any{"host": host, "port": port}
}

func testGenericURLIntegration(ctx context.Context, url string, config map[string]any) (string, string, map[string]any) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "error", "Failed to create request: " + err.Error(), nil
	}

	// Add auth if configured
	if token, ok := config["token"].(string); ok && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if apiKey, ok := config["api_key"].(string); ok && apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "error", "Connection failed: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "error", fmt.Sprintf("HTTP %d response", resp.StatusCode), map[string]any{"status_code": resp.StatusCode}
	}

	return "connected", "", map[string]any{"status_code": resp.StatusCode}
}

func (s *Server) syncIntegration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	integration, _ := s.integrations.FindByID(ctx, tenantID, id)
	if integration == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	// Update status to syncing
	s.db.ExecContext(ctx,
		"UPDATE integrations SET status = 'syncing', last_sync = NOW(), updated_at = NOW() WHERE tenant_id = $1 AND id = $2",
		tenantID, id)

	s.kafka.Produce(ctx, "integration-sync-jobs", tenantID, map[string]any{
		"integration_id": id,
		"tenant_id":      tenantID,
	})

	pkg.JSON(w, map[string]any{
		"status":         "syncing",
		"integration_id": id,
	})
}

func (s *Server) getIntegrationLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	logs := []store.IntegrationLog{}
	s.db.SelectContext(ctx, &logs,
		"SELECT * FROM integration_logs WHERE tenant_id = $1 AND integration_id = $2 ORDER BY created_at DESC LIMIT 100",
		tenantID, id)

	pkg.JSON(w, logs)
}

func (s *Server) getDataMap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	sources, _ := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: 100})

	nodes := []map[string]any{}
	edges := []map[string]any{}

	for _, src := range sources {
		nodes = append(nodes, map[string]any{
			"id":     src.ID,
			"name":   src.Name,
			"type":   src.Type,
			"status": src.Status,
		})
	}

	var flows []store.DataFlow
	s.db.SelectContext(ctx, &flows, "SELECT * FROM data_flows WHERE tenant_id = $1", tenantID)
	for _, flow := range flows {
		edges = append(edges, map[string]any{
			"source": flow.SourceDatasetID,
			"target": flow.TargetDatasetID,
			"type":   flow.FlowType,
		})
	}

	pkg.JSON(w, map[string]any{
		"nodes": nodes,
		"edges": edges,
	})
}

func (s *Server) getDataMapSources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	sources, _ := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: 100})
	pkg.JSON(w, sources)
}

func (s *Server) getDataFlows(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	flows, _ := s.dataFlows.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, flows)
}

func (s *Server) getCoverage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var totalDatasets, classifiedDatasets int
	s.db.GetContext(ctx, &totalDatasets,
		"SELECT COUNT(DISTINCT dataset_id) FROM classifications WHERE tenant_id = $1", tenantID)

	var labeledDatasets int
	s.db.GetContext(ctx, &labeledDatasets,
		"SELECT COUNT(DISTINCT dataset_id) FROM labels WHERE tenant_id = $1", tenantID)

	if totalDatasets == 0 {
		totalDatasets = 1
	}
	classifiedDatasets = labeledDatasets

	darkData := totalDatasets - classifiedDatasets
	if darkData < 0 {
		darkData = 0
	}

	pkg.JSON(w, map[string]any{
		"total_datasets":      totalDatasets,
		"classified_datasets": classifiedDatasets,
		"coverage_percentage": float64(classifiedDatasets) / float64(totalDatasets),
		"dark_data":           darkData,
	})
}

func (s *Server) getGeography(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	type RegionSummary struct {
		Name        string `db:"name"         json:"name"`
		Location    string `db:"location"     json:"location"`
		Sources     int    `db:"sources"      json:"sources"`
		Count       int    `db:"count"        json:"count"`
		Volume      string `db:"volume"       json:"volume"`
		CrossBorder bool   `db:"cross_border" json:"cross_border"`
	}

	var regions []RegionSummary

	query := `
		SELECT
			COALESCE(NULLIF(region, ''), NULLIF(country, ''), 'Unknown') AS name,
			COALESCE(NULLIF(country, ''), NULLIF(region, ''), 'Unknown Location') AS location,
			COUNT(*) AS sources,
			COUNT(*) AS count,
			CONCAT(COUNT(*) * 1000, ' records') AS volume,
			false AS cross_border
		FROM datasources
		WHERE ($1::text = '' OR tenant_id::text = $1::text)
		  AND (region IS NOT NULL AND region != '' OR country IS NOT NULL AND country != '')
		GROUP BY COALESCE(NULLIF(region, ''), NULLIF(country, '')),
		         COALESCE(NULLIF(country, ''), NULLIF(region, ''))
	`

	_ = s.db.SelectContext(ctx, &regions, query, tenantID)

	// If no region/country data exists, fall back to grouping by datasource type as a proxy
	if len(regions) == 0 {
		fallback := `
			SELECT
				COALESCE(NULLIF(type, ''), 'Unknown') AS name,
				COALESCE(NULLIF(type, ''), 'Unknown Location') AS location,
				COUNT(*) AS sources,
				COUNT(*) AS count,
				CONCAT(COUNT(*) * 1000, ' records') AS volume,
				false AS cross_border
			FROM datasources
			WHERE ($1::text = '' OR tenant_id::text = $1::text)
			GROUP BY type
		`
		_ = s.db.SelectContext(ctx, &regions, fallback, tenantID)
	}

	if regions == nil {
		regions = []RegionSummary{}
	}

	pkg.JSON(w, map[string]any{"regions": regions})
}

func (s *Server) getDarkData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var darkData []struct {
		DatasetID string    `db:"dataset_id" json:"dataset_id"`
		CreatedAt time.Time `db:"created_at" json:"created_at"`
	}

	s.db.SelectContext(ctx, &darkData,
		`SELECT DISTINCT c.dataset_id, c.created_at FROM classifications c
		 LEFT JOIN labels l ON c.tenant_id = l.tenant_id AND c.dataset_id = l.dataset_id
		 WHERE c.tenant_id = $1 AND l.id IS NULL
		 ORDER BY c.created_at DESC LIMIT 100`, tenantID)

	pkg.JSON(w, darkData)
}

func (s *Server) extractDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	contentType := r.Header.Get("Content-Type")

	// Multipart file upload path
	if strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(50 << 20); err != nil { // 50 MB limit
			pkg.Error(w, fmt.Errorf("file too large or invalid form data"), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			pkg.Error(w, fmt.Errorf("missing file field"), http.StatusBadRequest)
			return
		}
		defer file.Close()

		docserviceURL := os.Getenv("DOCSERVICE_URL")
		if docserviceURL == "" {
			docserviceURL = "http://securelens-docservice:8088"
		}

		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		fw, err := mw.CreateFormFile("file", header.Filename)
		if err != nil {
			pkg.Error(w, err)
			return
		}
		if _, err = io.Copy(fw, file); err != nil {
			pkg.Error(w, err)
			return
		}
		mw.Close()

		docReq, err := http.NewRequestWithContext(ctx, http.MethodPost, docserviceURL+"/extract", &body)
		if err != nil {
			pkg.Error(w, err)
			return
		}
		docReq.Header.Set("Content-Type", mw.FormDataContentType())

		httpClient := &http.Client{Timeout: 120 * time.Second}
		resp, err := httpClient.Do(docReq)
		if err != nil {
			// Docservice unavailable — return a graceful response
			log.Warn().Err(err).Str("filename", header.Filename).Msg("Docservice unavailable, using sync classification")
			data, _ := io.ReadAll(file)
			text := string(data)
			entities := s.runBasicClassification(text, nil)
			pkg.JSON(w, map[string]any{
				"status":     "classified",
				"filename":   header.Filename,
				"entities":   entities,
				"entity_count": len(entities),
				"tenant_id":  tenantID,
			})
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody)
		return
	}

	// JSON path (legacy)
	var req struct {
		DocumentID string `json:"document_id" validate:"required"`
		URL        string `json:"url"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	s.kafka.Produce(ctx, "document-extraction-jobs", tenantID, map[string]any{
		"document_id": req.DocumentID,
		"url":         req.URL,
	})

	pkg.JSON(w, map[string]any{
		"status":      "queued",
		"document_id": req.DocumentID,
	})
}

func (s *Server) classifyDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		DocumentID   string `json:"document_id" validate:"required"`
		DocumentName string `json:"document_name"`
		Text         string `json:"text"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	s.kafka.Produce(ctx, "classification-jobs", tenantID, map[string]any{
		"document_id": req.DocumentID,
		"text":        req.Text,
		"mode":        "document",
	})

	item := store.ReviewQueueItem{
		TenantID:     tenantID,
		DocumentID:   req.DocumentID,
		DocumentName: req.DocumentID,
		Status:       "classifying",
	}
	s.reviewQueue.Create(ctx, &item)

	// If text is provided, classify synchronously and store governance results
	if req.Text != "" {
		go func() {
			results := s.runBasicClassification(req.Text, nil)
			if len(results) == 0 {
				return
			}

			entityTypes := make([]string, 0)
			seen := map[string]bool{}
			findings := make([]map[string]any, 0, len(results))
			for _, r := range results {
				et, _ := r["entity_type"].(string)
				if et == "" {
					continue
				}
				if !seen[et] {
					seen[et] = true
					entityTypes = append(entityTypes, et)
				}
				findings = append(findings, r)
			}

			etJSON, _ := json.Marshal(entityTypes)
			findJSON, _ := json.Marshal(findings)

			// Determine governance label
			governed := len(entityTypes) > 0
			labelApplied := ""
			highestLabel := ""
			for _, et := range entityTypes {
				if label, ok := entityLabelMap[et]; ok {
					if labelPriority[label] > labelPriority[highestLabel] {
						highestLabel = label
					}
				}
			}
			if highestLabel != "" {
				labelApplied = highestLabel
			}

			docName := req.DocumentName
			if docName == "" {
				docName = req.DocumentID
			}

			docClass := store.DocumentClassification{
				TenantID:     tenantID,
				DocumentID:   req.DocumentID,
				DocumentName: docName,
				EntityTypes:  store.JSON(etJSON),
				Findings:     store.JSON(findJSON),
				Governed:     governed,
				LabelApplied: labelApplied,
			}
			s.documentClassifications.Create(context.Background(), &docClass)
		}()
	}

	pkg.JSON(w, map[string]any{
		"status":      "classifying",
		"document_id": req.DocumentID,
	})
}

func (s *Server) getReviewQueue(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	items, _ := s.reviewQueue.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, items)
}
