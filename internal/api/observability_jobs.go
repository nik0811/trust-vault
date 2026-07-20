package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/events"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

func (s *Server) getSystemHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	services := map[string]string{
		"gateway": "healthy",
	}

	if err := s.db.PingContext(ctx); err != nil {
		services["postgres"] = "unhealthy"
	} else {
		services["postgres"] = "healthy"
	}

	if s.kafka != nil {
		services["kafka"] = "healthy"
	} else {
		services["kafka"] = "unknown"
	}

	if s.qdrant != nil {
		services["qdrant"] = "healthy"
	} else {
		services["qdrant"] = "unknown"
	}

	overallStatus := "healthy"
	for _, status := range services {
		if status == "unhealthy" {
			overallStatus = "degraded"
			break
		}
	}

	pkg.JSON(w, map[string]any{
		"status":    overallStatus,
		"services":  services,
		"timestamp": time.Now(),
	})
}

func (s *Server) getSourceHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	ds, _ := s.datasources.FindByID(ctx, tenantID, id)
	if ds == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	health := "healthy"
	volumeTrend := "stable"

	if ds.Status == "error" || ds.Status == "disconnected" {
		health = "unhealthy"
	}

	var daysSinceLastScan float64
	if ds.LastScan != nil {
		daysSinceLastScan = time.Since(*ds.LastScan).Hours() / 24
	} else {
		daysSinceLastScan = 999 // Never scanned
	}
	if daysSinceLastScan > 7 {
		health = "stale"
	}

	var recentClassifications int
	s.db.GetContext(ctx, &recentClassifications,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND source_id = $2 AND created_at > NOW() - INTERVAL '7 days'",
		tenantID, id)

	var olderClassifications int
	s.db.GetContext(ctx, &olderClassifications,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND source_id = $2 AND created_at BETWEEN NOW() - INTERVAL '14 days' AND NOW() - INTERVAL '7 days'",
		tenantID, id)

	if recentClassifications > olderClassifications*2 {
		volumeTrend = "increasing"
	} else if recentClassifications < olderClassifications/2 {
		volumeTrend = "decreasing"
	}

	pkg.JSON(w, map[string]any{
		"source_id":    id,
		"status":       ds.Status,
		"last_scan":    ds.LastScan,
		"health":       health,
		"volume_trend": volumeTrend,
	})
}

func (s *Server) getMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var totalQueries, totalClassifications int
	s.db.GetContext(ctx, &totalQueries, "SELECT COUNT(*) FROM gate_queries")
	s.db.GetContext(ctx, &totalClassifications, "SELECT COUNT(*) FROM classifications")

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(pkg.Sprintf(`# HELP securelens_queries_total Total gate queries
# TYPE securelens_queries_total counter
securelens_queries_total %d
# HELP securelens_classifications_total Total classifications
# TYPE securelens_classifications_total counter
securelens_classifications_total %d
`, totalQueries, totalClassifications)))
}

func (s *Server) getAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	notifications, _ := s.notifications.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, notifications)
}

func (s *Server) createAlertRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Name      string `json:"name" validate:"required"`
		Condition string `json:"condition" validate:"required"`
		Severity  string `json:"severity"`
		Channel   string `json:"channel"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	policy := store.Policy{
		TenantID: tenantID,
		Name:     req.Name,
		Type:     "alert",
		Active:   true,
	}
	s.policies.Create(ctx, &policy)

	pkg.JSON(w, policy, http.StatusCreated)
}

func (s *Server) listAIGovPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var policies []store.Policy
	s.db.SelectContext(ctx, &policies,
		"SELECT * FROM policies WHERE tenant_id = $1 AND type = 'ai' ORDER BY created_at DESC",
		tenantID)

	pkg.JSON(w, policies)
}

func (s *Server) createAIGovPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var policy store.Policy
	if err := pkg.Bind(r, &policy); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	policy.TenantID = tenantID
	policy.Type = "ai"
	policy.Active = true
	s.policies.Create(ctx, &policy)

	pkg.JSON(w, policy, http.StatusCreated)
}

func (s *Server) evaluateAIEligibility(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		DatasetID string `json:"dataset_id" validate:"required"`
		Purpose   string `json:"purpose"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	var policies []store.Policy
	s.db.SelectContext(ctx, &policies,
		"SELECT * FROM policies WHERE tenant_id = $1 AND type = 'ai' AND active = true",
		tenantID)

	var label store.Label
	s.db.GetContext(ctx, &label,
		"SELECT * FROM labels WHERE tenant_id = $1 AND dataset_id = $2 ORDER BY created_at DESC LIMIT 1",
		tenantID, req.DatasetID)

	eligible := true
	reasons := []string{}

	if label.Label == "RESTRICTED" || label.Label == "HIGHLY_CONFIDENTIAL" {
		eligible = false
		reasons = append(reasons, "Dataset has restricted sensitivity label")
	}

	var piiCount int
	s.db.GetContext(ctx, &piiCount,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND dataset_id = $2 AND entity_type IN ('SSN', 'CREDIT_CARD', 'PHI')",
		tenantID, req.DatasetID)

	if piiCount > 0 && req.Purpose == "training" {
		eligible = false
		reasons = append(reasons, pkg.Sprintf("Dataset contains %d high-risk PII entities", piiCount))
	}

	pkg.JSON(w, map[string]any{
		"dataset_id": req.DatasetID,
		"eligible":   eligible,
		"reasons":    reasons,
		"policies":   len(policies),
	})
}

func (s *Server) getAIEligibility(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasetID := chi.URLParam(r, "dataset_id")

	var label store.Label
	s.db.GetContext(ctx, &label,
		"SELECT * FROM labels WHERE tenant_id = $1 AND dataset_id = $2 ORDER BY created_at DESC LIMIT 1",
		tenantID, datasetID)

	eligible := true
	status := "approved"

	if label.Label == "RESTRICTED" || label.Label == "HIGHLY_CONFIDENTIAL" {
		eligible = false
		status = "restricted"
	}

	var piiCount int
	s.db.GetContext(ctx, &piiCount,
		"SELECT COUNT(*) FROM classifications WHERE tenant_id = $1 AND dataset_id = $2 AND entity_type IN ('SSN', 'CREDIT_CARD', 'PHI')",
		tenantID, datasetID)

	if piiCount > 0 {
		status = "requires_review"
	}

	pkg.JSON(w, map[string]any{
		"dataset_id": datasetID,
		"eligible":   eligible,
		"status":     status,
		"pii_count":  piiCount,
	})
}

func (s *Server) getModelLineage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	modelID := chi.URLParam(r, "model_id")

	var trainingData []store.ModelLineage
	s.db.SelectContext(ctx, &trainingData,
		"SELECT * FROM model_lineage WHERE tenant_id = $1 AND model_id = $2 AND usage_type = 'training'",
		tenantID, modelID)

	var inferenceUsage []store.ModelLineage
	s.db.SelectContext(ctx, &inferenceUsage,
		"SELECT * FROM model_lineage WHERE tenant_id = $1 AND model_id = $2 AND usage_type = 'inference'",
		tenantID, modelID)

	trainingDatasets := []string{}
	for _, td := range trainingData {
		trainingDatasets = append(trainingDatasets, td.DatasetID)
	}

	inferenceDatasets := []string{}
	for _, iu := range inferenceUsage {
		inferenceDatasets = append(inferenceDatasets, iu.DatasetID)
	}

	pkg.JSON(w, map[string]any{
		"model_id":        modelID,
		"training_data":   trainingDatasets,
		"inference_usage": inferenceDatasets,
	})
}

func (s *Server) generateModelCard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		ModelID string `json:"model_id" validate:"required"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	var trainingData []store.ModelLineage
	s.db.SelectContext(ctx, &trainingData,
		"SELECT * FROM model_lineage WHERE tenant_id = $1 AND model_id = $2 AND usage_type = 'training'",
		tenantID, req.ModelID)

	sources := []string{}
	classifications := []string{}
	for _, td := range trainingData {
		sources = append(sources, td.DatasetID)

		var entityTypes []string
		s.db.SelectContext(ctx, &entityTypes,
			"SELECT DISTINCT entity_type FROM classifications WHERE tenant_id = $1 AND dataset_id = $2",
			tenantID, td.DatasetID)
		classifications = append(classifications, entityTypes...)
	}

	governance := "compliant"
	for _, td := range trainingData {
		var label store.Label
		s.db.GetContext(ctx, &label,
			"SELECT * FROM labels WHERE tenant_id = $1 AND dataset_id = $2 ORDER BY created_at DESC LIMIT 1",
			tenantID, td.DatasetID)
		if label.Label == "RESTRICTED" {
			governance = "non_compliant"
			break
		}
	}

	pkg.JSON(w, map[string]any{
		"model_id":  req.ModelID,
		"generated": time.Now(),
		"training_data": map[string]any{
			"sources":         sources,
			"classifications": classifications,
			"governance":      governance,
		},
	})
}

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	notifications, _ := s.notifications.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, notifications)
}

func (s *Server) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	s.db.ExecContext(ctx, "UPDATE notifications SET read = true WHERE tenant_id = $1 AND id = $2",
		tenantID, id)

	pkg.JSON(w, map[string]string{"status": "read"})
}

func (s *Server) markAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	s.db.ExecContext(ctx, "UPDATE notifications SET read = true WHERE tenant_id = $1", tenantID)
	pkg.JSON(w, map[string]string{"status": "all_read"})
}

func (s *Server) deleteNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	s.db.ExecContext(ctx, "DELETE FROM notifications WHERE tenant_id = $1 AND id = $2", tenantID, id)
	pkg.JSON(w, map[string]string{"status": "deleted"})
}

func (s *Server) createWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var webhook store.Webhook
	if err := pkg.Bind(r, &webhook); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	webhook.TenantID = tenantID
	webhook.Active = true
	s.webhooks.Create(ctx, &webhook)

	pkg.JSON(w, webhook, http.StatusCreated)
}

func (s *Server) listWebhooks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	webhooks, _ := s.webhooks.List(ctx, tenantID, store.ListOpts{Limit: 50})
	pkg.JSON(w, webhooks)
}

func (s *Server) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	s.webhooks.Delete(ctx, tenantID, id)
	pkg.JSON(w, map[string]string{"status": "deleted"})
}

func (s *Server) streamEvents(w http.ResponseWriter, r *http.Request) {
	// Explicitly set CORS headers for SSE — some proxies strip them from streaming responses
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	// Generate unique client ID
	clientID := pkg.GenerateID()

	// Register SSE client
	client := events.RegisterSSEClient(clientID, tenantID)
	defer events.UnregisterSSEClient(clientID)

	// Send initial connection event
	w.Write([]byte("event: connected\n"))
	w.Write([]byte(pkg.Sprintf("data: {\"client_id\":\"%s\",\"tenant_id\":\"%s\"}\n\n", clientID, tenantID)))
	flusher.Flush()

	// Heartbeat ticker
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-client.Done:
			return
		case msg := <-client.Events:
			data, err := json.Marshal(msg.Data)
			if err != nil {
				continue
			}
			w.Write([]byte(pkg.Sprintf("event: %s\n", msg.Event)))
			w.Write([]byte(pkg.Sprintf("data: %s\n\n", string(data))))
			flusher.Flush()
		case <-heartbeat.C:
			w.Write([]byte("event: heartbeat\n"))
			w.Write([]byte(pkg.Sprintf("data: {\"time\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))))
			flusher.Flush()
		}
	}
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	jobs, _ := s.jobs.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	pkg.JSON(w, jobs)
}

func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var job store.Job
	if err := pkg.Bind(r, &job); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	job.TenantID = tenantID
	job.Status = "scheduled"

	// Calculate next run time if schedule is provided
	if job.Schedule != "" {
		nextRun := calculateNextRunTime(job.Schedule)
		job.NextRun = &nextRun
	}

	s.jobs.Create(ctx, &job)

	events.Emit("job.created", map[string]any{
		"job_id":    job.ID,
		"tenant_id": tenantID,
		"name":      job.Name,
		"type":      job.Type,
		"schedule":  job.Schedule,
	})

	pkg.JSON(w, job, http.StatusCreated)
}

// calculateNextRunTime calculates the next run time based on schedule
func calculateNextRunTime(schedule string) time.Time {
	now := time.Now()
	switch schedule {
	case "@hourly":
		return now.Add(time.Hour)
	case "@daily":
		return now.Add(24 * time.Hour)
	case "@weekly":
		return now.Add(7 * 24 * time.Hour)
	default:
		if d, err := time.ParseDuration(schedule); err == nil {
			return now.Add(d)
		}
		return now.Add(24 * time.Hour)
	}
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	job, _ := s.jobs.FindByID(ctx, tenantID, id)
	if job == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	pkg.JSON(w, job)
}

func (s *Server) deleteJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	s.jobs.Delete(ctx, tenantID, id)
	pkg.JSON(w, map[string]string{"status": "deleted"})
}

func (s *Server) updateJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	job, _ := s.jobs.FindByID(ctx, tenantID, id)
	if job == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	var req struct {
		Name     *string    `json:"name"`
		Type     *string    `json:"type"`
		Schedule *string    `json:"schedule"`
		Config   store.JSON `json:"config"`
		Status   *string    `json:"status"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		job.Name = *req.Name
	}
	if req.Type != nil {
		job.Type = *req.Type
	}
	if req.Schedule != nil {
		job.Schedule = *req.Schedule
		nextRun := calculateNextRunTime(*req.Schedule)
		job.NextRun = &nextRun
	}
	if req.Config != nil {
		job.Config = req.Config
	}
	if req.Status != nil {
		job.Status = *req.Status
	}

	s.jobs.Update(ctx, job)

	events.Emit("job.updated", map[string]any{
		"job_id":    job.ID,
		"tenant_id": tenantID,
		"name":      job.Name,
	})

	pkg.JSON(w, job)
}

func (s *Server) runJobNow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	job, _ := s.jobs.FindByID(ctx, tenantID, id)
	if job == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	// Check if job is already running
	if job.Status == "running" {
		pkg.JSON(w, map[string]string{
			"status":  "already_running",
			"message": "This job is already running",
		}, http.StatusConflict)
		return
	}

	// Create execution record
	executionID := pkg.GenerateID()
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO job_executions (id, tenant_id, job_id, status, started_at, created_at, updated_at)
		 VALUES ($1, $2, $3, 'running', $4, $4, $4)`,
		executionID, tenantID, id, now)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	// Update status to running
	job.Status = "running"
	s.jobs.Update(ctx, job)

	// Queue job for immediate execution
	err = s.kafka.Produce(ctx, "job-executions", tenantID, map[string]any{
		"job_id":       id,
		"execution_id": executionID,
		"tenant_id":    tenantID,
		"type":         job.Type,
		"config":       job.Config,
	})
	if err != nil {
		job.Status = "scheduled"
		s.jobs.Update(ctx, job)
		// Mark execution as failed
		s.db.ExecContext(ctx,
			`UPDATE job_executions SET status = 'failed', error = $1, completed_at = NOW(), updated_at = NOW() WHERE id = $2`,
			err.Error(), executionID)
		pkg.Error(w, err)
		return
	}

	events.Emit("job.queued", map[string]any{
		"job_id":       id,
		"execution_id": executionID,
		"tenant_id":    tenantID,
		"name":         job.Name,
		"type":         job.Type,
	})

	pkg.JSON(w, map[string]any{
		"status":       "running",
		"job_id":       id,
		"execution_id": executionID,
		"message":      "Job started",
	})
}

func (s *Server) getJobHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")
	limit, offset := pkg.ParseListOpts(r)

	job, _ := s.jobs.FindByID(ctx, tenantID, id)
	if job == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	var executions []store.JobExecution
	err := s.db.SelectContext(ctx, &executions,
		`SELECT * FROM job_executions 
		 WHERE tenant_id = $1 AND job_id = $2 
		 ORDER BY created_at DESC 
		 LIMIT $3 OFFSET $4`,
		tenantID, id, limit, offset)
	if err != nil {
		pkg.Error(w, err)
		return
	}

	pkg.JSON(w, executions)
}

func (s *Server) getObservabilitySummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var totalSources, healthySources int
	s.db.GetContext(ctx, &totalSources, `SELECT COUNT(*) FROM datasources WHERE tenant_id = $1`, tenantID)
	s.db.GetContext(ctx, &healthySources, `SELECT COUNT(*) FROM datasources WHERE tenant_id = $1 AND status = 'active'`, tenantID)

	var schemaDrifts int
	s.db.GetContext(ctx, &schemaDrifts, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action LIKE '%schema%'`, tenantID)

	var freshnessIssues int
	s.db.GetContext(ctx, &freshnessIssues,
		`SELECT COUNT(*) FROM datasources WHERE tenant_id = $1 AND (last_scan IS NULL OR last_scan < NOW() - INTERVAL '7 days')`, tenantID)

	var recentAlerts int
	s.db.GetContext(ctx, &recentAlerts,
		`SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND created_at > NOW() - INTERVAL '24 hours'`, tenantID)

	pkg.JSON(w, map[string]any{
		"total_sources":    totalSources,
		"healthy_sources":  healthySources,
		"schema_drifts":    schemaDrifts,
		"freshness_issues": freshnessIssues,
		"recent_alerts":    recentAlerts,
		"status":           "healthy",
		"last_checked":     time.Now(),
	})
}


