package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/trustvault/trustvault/internal/events"
	"github.com/trustvault/trustvault/internal/pkg"
	"github.com/trustvault/trustvault/internal/store"
)

// sensitiveFields are config keys that should be masked in API responses
var sensitiveFields = []string{
	"password", "secret", "api_key", "apikey", "api_secret", "apisecret",
	"access_key", "accesskey", "secret_key", "secretkey", "private_key",
	"privatekey", "token", "credential", "credentials", "auth_token",
}

// maskSensitiveConfig masks sensitive fields in datasource config
func maskSensitiveConfig(config store.JSON) store.JSON {
	if len(config) == 0 {
		return config
	}

	var configMap map[string]any
	if err := json.Unmarshal(config, &configMap); err != nil {
		return config
	}

	for key := range configMap {
		keyLower := key
		for _, sensitive := range sensitiveFields {
			if keyLower == sensitive || contains(keyLower, sensitive) {
				configMap[key] = "********"
				break
			}
		}
	}

	masked, err := json.Marshal(configMap)
	if err != nil {
		return config
	}
	return store.JSON(masked)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)))
}

// maskDataSource returns a copy with sensitive config fields masked
func maskDataSource(ds *store.DataSource) store.DataSource {
	masked := *ds
	masked.Config = maskSensitiveConfig(ds.Config)
	return masked
}

func (s *Server) listDataSources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	sources, err := s.datasources.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	if err != nil {
		pkg.Error(w, err)
		return
	}
	if sources == nil {
		sources = []store.DataSource{}
	}
	
	// Mask sensitive fields before returning
	maskedSources := make([]store.DataSource, len(sources))
	for i, ds := range sources {
		maskedSources[i] = maskDataSource(&ds)
	}
	pkg.JSON(w, maskedSources)
}

func (s *Server) createDataSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var ds store.DataSource
	if err := pkg.Bind(r, &ds); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	ds.TenantID = tenantID
	ds.Status = "pending"

	if err := s.datasources.Create(ctx, &ds); err != nil {
		pkg.Error(w, err)
		return
	}

	events.Emit("datasource.created", ds)
	
	// Audit log
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		UserID:     pkg.UserFromCtx(ctx),
		Action:     "datasource.created",
		Resource:   "datasource",
		ResourceID: ds.ID,
		Details:    store.JSON(fmt.Sprintf(`{"name":"%s","type":"%s"}`, ds.Name, ds.Type)),
	})
	
	// Mask sensitive fields before returning
	masked := maskDataSource(&ds)
	pkg.JSON(w, masked, http.StatusCreated)
}

func (s *Server) getDataSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	ds, err := s.datasources.FindByID(ctx, tenantID, id)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	if ds == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	
	// Mask sensitive fields before returning
	masked := maskDataSource(ds)
	pkg.JSON(w, masked)
}

func (s *Server) updateDataSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	ds, err := s.datasources.FindByID(ctx, tenantID, id)
	if err != nil || ds == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	var req struct {
		Name   string     `json:"name"`
		Config store.JSON `json:"config"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	if req.Name != "" {
		ds.Name = req.Name
	}
	if req.Config != nil {
		ds.Config = req.Config
	}

	if err := s.datasources.Update(ctx, ds); err != nil {
		pkg.Error(w, err)
		return
	}

	events.Emit("datasource.updated", ds)
	
	// Audit log
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		UserID:     pkg.UserFromCtx(ctx),
		Action:     "datasource.updated",
		Resource:   "datasource",
		ResourceID: ds.ID,
		Details:    store.JSON(fmt.Sprintf(`{"name":"%s"}`, ds.Name)),
	})
	
	// Mask sensitive fields before returning
	masked := maskDataSource(ds)
	pkg.JSON(w, masked)
}

func (s *Server) deleteDataSource(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	if err := s.datasources.Delete(ctx, tenantID, id); err != nil {
		pkg.Error(w, err)
		return
	}

	events.Emit("datasource.deleted", map[string]string{"id": id})
	
	// Audit log
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		UserID:     pkg.UserFromCtx(ctx),
		Action:     "datasource.deleted",
		Resource:   "datasource",
		ResourceID: id,
	})
	
	pkg.JSON(w, map[string]string{"status": "deleted"})
}

func (s *Server) triggerScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	ds, err := s.datasources.FindByID(ctx, tenantID, id)
	if err != nil || ds == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	// Check if already scanning
	if ds.Status == "scanning" {
		pkg.JSON(w, map[string]string{
			"status":  "already_scanning",
			"message": "A scan is already in progress for this data source",
		}, http.StatusConflict)
		return
	}

	ds.Status = "scanning"
	if err := s.datasources.Update(ctx, ds); err != nil {
		log.Error().Err(err).Str("datasource_id", id).Str("tenant_id", tenantID).Msg("failed to update datasource status to scanning")
		pkg.Error(w, fmt.Errorf("failed to update status: %w", err), http.StatusInternalServerError)
		return
	}
	log.Info().Str("datasource_id", id).Str("tenant_id", tenantID).Msg("datasource status updated to scanning")

	// Create a new scan log entry
	scanLog := store.ScanLog{
		TenantID:     tenantID,
		DatasourceID: id,
		Status:       "running",
		StartedAt:    time.Now(),
		Message:      "Scan started",
		Logs:         store.JSON(`[]`),
	}
	if err := s.scanLogs.Create(ctx, &scanLog); err != nil {
		log.Error().Err(err).Str("datasource_id", id).Msg("failed to create scan log entry")
	}
	
	// Audit log
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		UserID:     pkg.UserFromCtx(ctx),
		Action:     "datasource.scan_started",
		Resource:   "datasource",
		ResourceID: id,
		Details:    store.JSON(fmt.Sprintf(`{"name":"%s","type":"%s"}`, ds.Name, ds.Type)),
	})

	// Refetch to get clean config (avoid buffer reuse issues)
	ds, err = s.datasources.FindByID(ctx, tenantID, id)
	if err != nil || ds == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	// Parse config to map for ingestion request
	var configMap map[string]any
	if len(ds.Config) > 0 {
		json.Unmarshal(ds.Config, &configMap)
	}

	// Call ingestion sidecar to start DataHub ingestion
	ingestionReq := map[string]any{
		"datasource_id": id,
		"tenant_id":     tenantID,
		"type":          ds.Type,
		"config":        configMap,
		"callback_url":  s.ingestionCallbackURL(),
		"scan_log_id":   scanLog.ID,
	}

	ingestionURL := s.ingestionSidecarURL() + "/ingest"
	resp, err := s.callIngestionSidecar(ctx, ingestionURL, ingestionReq)
	if err != nil {
		// Ingestion sidecar is required - fail if unavailable
		ds.Status = "error"
		s.datasources.Update(ctx, ds)
		
		// Update scan log with failure
		now := time.Now()
		scanLog.Status = "failed"
		scanLog.CompletedAt = &now
		scanLog.Message = fmt.Sprintf("Ingestion service unavailable: %v", err)
		s.scanLogs.Update(ctx, &scanLog)
		
		log.Error().Err(err).Msg("ingestion sidecar unavailable - cannot proceed with scan")
		pkg.Error(w, fmt.Errorf("ingestion service unavailable: %w", err), http.StatusServiceUnavailable)
		return
	}

	// Emit SSE event for scan started
	events.Emit("datasource.scan.started", map[string]any{
		"datasource_id": id,
		"tenant_id":     tenantID,
		"name":          ds.Name,
		"type":          ds.Type,
		"status":        "scanning",
		"scan_log_id":   scanLog.ID,
	})

	result := map[string]any{
		"status":        "scanning",
		"datasource_id": id,
		"message":       "Scan job started successfully",
		"scan_log_id":   scanLog.ID,
	}
	if resp != nil {
		result["job_id"] = resp["job_id"]
	}
	pkg.JSON(w, result)
}

func (s *Server) getScanStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	ds, err := s.datasources.FindByID(ctx, tenantID, id)
	if err != nil || ds == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	pkg.JSON(w, map[string]any{
		"status":    ds.Status,
		"last_scan": ds.LastScan,
	})
}

// ingestionSidecarURL returns the URL of the ingestion sidecar service
func (s *Server) ingestionSidecarURL() string {
	if url := os.Getenv("INGESTION_SIDECAR_URL"); url != "" {
		return url
	}
	return "http://trustvault-ingestion:8090"
}

// ingestionCallbackURL returns the callback URL for ingestion completion
func (s *Server) ingestionCallbackURL() string {
	if url := os.Getenv("INGESTION_CALLBACK_URL"); url != "" {
		return url
	}
	return "http://trustvault-gateway:8080/api/v1/datasources/callback"
}

// callIngestionSidecar makes an HTTP POST to the ingestion sidecar
func (s *Server) callIngestionSidecar(ctx context.Context, url string, payload map[string]any) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Use a separate context with longer timeout for ingestion calls
	callCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(callCtx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ingestion sidecar returned status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// scanCallback handles completion notifications from the ingestion sidecar
func (s *Server) scanCallback(w http.ResponseWriter, r *http.Request) {
	var callback struct {
		JobID              string `json:"job_id"`
		Status             string `json:"status"`
		Message            string `json:"message"`
		DatasetsDiscovered int    `json:"datasets_discovered"`
		ScanLogID          string `json:"scan_log_id"`
	}
	if err := pkg.Bind(r, &callback); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Parse job_id format: tenant_id-datasource_id
	parts := splitJobID(callback.JobID)
	if len(parts) != 2 {
		pkg.Error(w, fmt.Errorf("invalid job_id format"), http.StatusBadRequest)
		return
	}
	tenantID, datasourceID := parts[0], parts[1]

	ctx := r.Context()
	ds, err := s.datasources.FindByID(ctx, tenantID, datasourceID)
	if err != nil || ds == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	// Update datasource status based on callback
	now := time.Now()
	ds.LastScan = &now
	if callback.Status == "completed" {
		ds.Status = "connected"
	} else {
		ds.Status = "error"
	}

	if err := s.datasources.Update(ctx, ds); err != nil {
		pkg.Error(w, err)
		return
	}

	// Update scan log entry if scan_log_id provided
	if callback.ScanLogID != "" {
		scanLog, err := s.scanLogs.FindByID(ctx, tenantID, callback.ScanLogID)
		if err == nil && scanLog != nil {
			scanLog.CompletedAt = &now
			scanLog.DatasetsDiscovered = callback.DatasetsDiscovered
			scanLog.Message = callback.Message
			if callback.Status == "completed" {
				scanLog.Status = "success"
			} else {
				scanLog.Status = "failed"
			}
			s.scanLogs.Update(ctx, scanLog)
		}
	} else {
		// Fallback: find the most recent running scan log for this datasource
		query := `SELECT id, tenant_id, datasource_id, status, started_at, completed_at, message, logs, datasets_discovered, created_at 
			FROM scan_logs 
			WHERE tenant_id = $1 AND datasource_id = $2 AND status = 'running' 
			ORDER BY started_at DESC 
			LIMIT 1`
		row := s.db.QueryRowContext(ctx, query, tenantID, datasourceID)
		var scanLog store.ScanLog
		if err := row.Scan(&scanLog.ID, &scanLog.TenantID, &scanLog.DatasourceID, &scanLog.Status, &scanLog.StartedAt, &scanLog.CompletedAt, &scanLog.Message, &scanLog.Logs, &scanLog.DatasetsDiscovered, &scanLog.CreatedAt); err == nil {
			scanLog.CompletedAt = &now
			scanLog.DatasetsDiscovered = callback.DatasetsDiscovered
			scanLog.Message = callback.Message
			if callback.Status == "completed" {
				scanLog.Status = "success"
			} else {
				scanLog.Status = "failed"
			}
			s.scanLogs.Update(ctx, &scanLog)
		}
	}

	// Emit SSE event for scan completion
	events.Emit("datasource.scan.completed", map[string]any{
		"datasource_id":       datasourceID,
		"tenant_id":           tenantID,
		"status":              callback.Status,
		"message":             callback.Message,
		"datasets_discovered": callback.DatasetsDiscovered,
	})

	// Create lineage entry for successful scans
	if callback.Status == "completed" && callback.DatasetsDiscovered > 0 {
		dataFlow := store.DataFlow{
			TenantID:        tenantID,
			SourceDatasetID: datasourceID,
			TargetDatasetID: "trustvault-classification",
			FlowType:        "ingestion",
		}
		s.dataFlows.Create(ctx, &dataFlow)
	}

	log.Info().
		Str("datasource_id", datasourceID).
		Str("tenant_id", tenantID).
		Str("status", callback.Status).
		Int("datasets", callback.DatasetsDiscovered).
		Msg("scan callback received")

	pkg.JSON(w, map[string]string{"status": "ok"})
}

// splitJobID splits a job ID in format "tenant_id::datasource_id"
func splitJobID(jobID string) []string {
	parts := strings.Split(jobID, "::")
	if len(parts) != 2 {
		return nil
	}
	return parts
}

// scanProgress handles progress updates from the ingestion sidecar
func (s *Server) scanProgress(w http.ResponseWriter, r *http.Request) {
	var progress struct {
		JobID        string   `json:"job_id"`
		DatasourceID string   `json:"datasource_id"`
		TenantID     string   `json:"tenant_id"`
		Message      string   `json:"message"`
		LogLines     []string `json:"log_lines"`
		ScanLogID    string   `json:"scan_log_id"`
	}
	if err := pkg.Bind(r, &progress); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Emit SSE event for progress update
	eventData := map[string]any{
		"datasource_id": progress.DatasourceID,
		"tenant_id":     progress.TenantID,
		"message":       progress.Message,
	}
	if len(progress.LogLines) > 0 {
		eventData["log_lines"] = progress.LogLines
	}

	events.Emit("datasource.scan.progress", eventData)

	log.Debug().
		Str("datasource_id", progress.DatasourceID).
		Str("tenant_id", progress.TenantID).
		Str("message", progress.Message).
		Int("log_lines", len(progress.LogLines)).
		Msg("scan progress received")

	pkg.JSON(w, map[string]string{"status": "ok"})
}

// listScanLogs returns scan history for a datasource
func (s *Server) listScanLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasourceID := chi.URLParam(r, "id")
	limit, offset := pkg.ParseListOpts(r)

	// Verify datasource exists and belongs to tenant
	ds, err := s.datasources.FindByID(ctx, tenantID, datasourceID)
	if err != nil || ds == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	// Query scan logs for this datasource
	query := `SELECT id, tenant_id, datasource_id, status, started_at, completed_at, message, logs, datasets_discovered, created_at 
		FROM scan_logs 
		WHERE tenant_id = $1 AND datasource_id = $2 
		ORDER BY started_at DESC 
		LIMIT $3 OFFSET $4`

	rows, err := s.db.QueryContext(ctx, query, tenantID, datasourceID, limit, offset)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	defer rows.Close()

	var logs []store.ScanLog
	for rows.Next() {
		var sl store.ScanLog
		if err := rows.Scan(&sl.ID, &sl.TenantID, &sl.DatasourceID, &sl.Status, &sl.StartedAt, &sl.CompletedAt, &sl.Message, &sl.Logs, &sl.DatasetsDiscovered, &sl.CreatedAt); err != nil {
			pkg.Error(w, err)
			return
		}
		logs = append(logs, sl)
	}

	if logs == nil {
		logs = []store.ScanLog{}
	}
	pkg.JSON(w, logs)
}
