package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

func (s *Server) registerEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	var req struct {
		Hostname     string `json:"hostname" validate:"required"`
		IP           string `json:"ip"`
		OS           string `json:"os"`
		AgentVersion string `json:"agent_version"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}
	now := time.Now()
	agent := &store.EndpointAgent{
		TenantID:     tenantID,
		Hostname:     req.Hostname,
		IP:           req.IP,
		OS:           req.OS,
		AgentVersion: req.AgentVersion,
		Status:       "active",
		LastSeenAt:   &now,
	}
	if err := s.endpointAgents.Create(ctx, agent); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, map[string]any{
		"agent_id": agent.ID,
		"status":   "registered",
		"message":  "Agent registered successfully",
	}, http.StatusCreated)
}

func (s *Server) listEndpoints(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)
	agents, _ := s.endpointAgents.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	if agents == nil {
		agents = []store.EndpointAgent{}
	}
	pkg.JSON(w, map[string]any{"endpoints": agents, "total": len(agents)})
}

func (s *Server) triggerEndpointScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")
	agent, err := s.endpointAgents.FindByID(ctx, tenantID, id)
	if err != nil || agent == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	agent.Status = "scanning"
	if err := s.endpointAgents.Update(ctx, agent); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, map[string]any{"message": "scan triggered", "endpoint_id": id, "status": "scanning"}, http.StatusAccepted)
}

func (s *Server) receiveEndpointScanResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")
	agent, err := s.endpointAgents.FindByID(ctx, tenantID, id)
	if err != nil || agent == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	var body struct {
		FilesScanned   int             `json:"files_scanned"`
		PIIFound       json.RawMessage `json:"pii_found"`
		ScanDurationMs int             `json:"scan_duration_ms"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		pkg.Error(w, pkg.ErrBadRequest, http.StatusBadRequest)
		return
	}
	resultsJSON, _ := json.Marshal(map[string]any{
		"files_scanned":    body.FilesScanned,
		"pii_found":        body.PIIFound,
		"scan_duration_ms": body.ScanDurationMs,
		"received_at":      time.Now(),
	})
	now := time.Now()
	agent.Status = "active"
	agent.LastScanAt = &now
	agent.LastSeenAt = &now
	agent.ScanResults = store.JSON(resultsJSON)
	if err := s.endpointAgents.Update(ctx, agent); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, map[string]any{"message": "scan results stored", "files_scanned": body.FilesScanned})
}

func (s *Server) getEndpointResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")
	agent, err := s.endpointAgents.FindByID(ctx, tenantID, id)
	if err != nil || agent == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	pkg.JSON(w, map[string]any{
		"endpoint_id":  id,
		"hostname":     agent.Hostname,
		"last_scan_at": agent.LastScanAt,
		"scan_results": agent.ScanResults,
	})
}

func (s *Server) reportAgentScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		AgentID    string `json:"agent_id" validate:"required"`
		ScanResult struct {
			StartTime    time.Time `json:"start_time"`
			EndTime      time.Time `json:"end_time"`
			FilesScanned int       `json:"files_scanned"`
			BytesScanned int64     `json:"bytes_scanned"`
			Findings     []struct {
				FilePath   string  `json:"file_path"`
				LineNumber int     `json:"line_number"`
				PIIType    string  `json:"pii_type"`
				Value      string  `json:"value"`
				Masked     string  `json:"masked"`
				Confidence float64 `json:"confidence"`
				Severity   string  `json:"severity"`
			} `json:"findings"`
			Errors []string `json:"errors,omitempty"`
		} `json:"scan_result"`
		Paths    []string `json:"paths"`
		Hostname string   `json:"hostname"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	agent, err := s.endpointAgents.FindByID(ctx, tenantID, req.AgentID)
	if err != nil || agent == nil {
		pkg.Error(w, fmt.Errorf("agent not found: %s", req.AgentID), http.StatusNotFound)
		return
	}

	now := time.Now()
	resultsJSON, _ := json.Marshal(map[string]any{
		"files_scanned":  req.ScanResult.FilesScanned,
		"bytes_scanned":  req.ScanResult.BytesScanned,
		"findings_count": len(req.ScanResult.Findings),
		"findings":       req.ScanResult.Findings,
		"paths":          req.Paths,
		"duration_ms":    req.ScanResult.EndTime.Sub(req.ScanResult.StartTime).Milliseconds(),
		"errors":         req.ScanResult.Errors,
		"received_at":    now,
	})

	agent.Status = "active"
	agent.LastScanAt = &now
	agent.LastSeenAt = &now
	agent.ScanResults = store.JSON(resultsJSON)
	if err := s.endpointAgents.Update(ctx, agent); err != nil {
		pkg.Error(w, err)
		return
	}

	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		Action:     "agent.scan.completed",
		Resource:   "endpoint_agent",
		ResourceID: agent.ID,
		Details: store.JSON(fmt.Sprintf(`{"files_scanned":%d,"findings":%d,"paths":%v}`,
			req.ScanResult.FilesScanned, len(req.ScanResult.Findings), req.Paths)),
	})

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://app.securelens.ai"
	}

	pkg.JSON(w, map[string]any{
		"status":    "success",
		"report_id": agent.ID,
		"message":   fmt.Sprintf("Scan results stored: %d files, %d findings", req.ScanResult.FilesScanned, len(req.ScanResult.Findings)),
		"view_url":  fmt.Sprintf("%s/endpoints?agent=%s", appURL, agent.ID),
	})
}

func (s *Server) getAgentStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	agent, err := s.endpointAgents.FindByID(ctx, tenantID, id)
	if err != nil || agent == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	var totalScans int
	var totalFindings int
	s.db.GetContext(ctx, &totalScans,
		`SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND resource_id = $2 AND action = 'agent.scan.completed'`,
		tenantID, id)

	if agent.ScanResults != nil {
		var results map[string]any
		if err := json.Unmarshal(agent.ScanResults, &results); err == nil {
			if fc, ok := results["findings_count"].(float64); ok {
				totalFindings = int(fc)
			}
		}
	}

	pkg.JSON(w, map[string]any{
		"agent_id":       agent.ID,
		"status":         agent.Status,
		"hostname":       agent.Hostname,
		"os":             agent.OS,
		"agent_version":  agent.AgentVersion,
		"last_scan":      agent.LastScanAt,
		"last_seen":      agent.LastSeenAt,
		"total_scans":    totalScans,
		"total_findings": totalFindings,
	})
}

func (s *Server) agentHeartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	agent, err := s.endpointAgents.FindByID(ctx, tenantID, id)
	if err != nil || agent == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	now := time.Now()
	agent.LastSeenAt = &now
	agent.Status = "active"
	s.endpointAgents.Update(ctx, agent)

	pkg.JSON(w, map[string]string{"status": "ok"})
}
