package api

import (
	"encoding/json"
	"net/http"
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
	pkg.JSON(w, agent, http.StatusCreated)
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
