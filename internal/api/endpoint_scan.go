package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/securelens/securelens/internal/pkg"
	"github.com/securelens/securelens/internal/store"
)

// ── CRUD ────────────────────────────────────────────────────────────────────

func (s *Server) listEndpointScans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)
	items, _ := s.endpointScans.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	if items == nil {
		items = []store.EndpointScan{}
	}
	pkg.JSON(w, map[string]any{"endpoints": items, "total": len(items)})
}

func (s *Server) createEndpointScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	var req store.EndpointScan
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}
	req.TenantID = tenantID
	req.Status = "pending"
	req.RiskLevel = "unknown"
	if req.Method == "" {
		req.Method = "GET"
	}
	if len(req.Headers) == 0 {
		req.Headers = store.JSON([]byte("{}"))
	}
	if len(req.AuthConfig) == 0 {
		req.AuthConfig = store.JSON([]byte("{}"))
	}
	if len(req.Findings) == 0 {
		req.Findings = store.JSON([]byte("[]"))
	}
	if err := s.endpointScans.Create(ctx, &req); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, req, http.StatusCreated)
}

func (s *Server) getEndpointScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")
	item, err := s.endpointScans.FindByID(ctx, tenantID, id)
	if err != nil || item == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	pkg.JSON(w, item)
}

func (s *Server) updateEndpointScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")
	existing, err := s.endpointScans.FindByID(ctx, tenantID, id)
	if err != nil || existing == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkg.Error(w, pkg.ErrBadRequest, http.StatusBadRequest)
		return
	}
	if v, ok := req["name"].(string); ok {
		existing.Name = v
	}
	if v, ok := req["url"].(string); ok {
		existing.URL = v
	}
	if v, ok := req["method"].(string); ok {
		existing.Method = v
	}
	if v, ok := req["auth_type"].(string); ok {
		existing.AuthType = v
	}
	if v, ok := req["headers"]; ok {
		if b, e := json.Marshal(v); e == nil {
			existing.Headers = store.JSON(b)
		}
	}
	if v, ok := req["auth_config"]; ok {
		if b, e := json.Marshal(v); e == nil {
			existing.AuthConfig = store.JSON(b)
		}
	}
	if err := s.endpointScans.Update(ctx, existing); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, existing)
}

func (s *Server) deleteEndpointScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")
	existing, err := s.endpointScans.FindByID(ctx, tenantID, id)
	if err != nil || existing == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	if err := s.endpointScans.Delete(ctx, tenantID, id); err != nil {
		pkg.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── SCAN ────────────────────────────────────────────────────────────────────

type endpointFinding struct {
	Field       string  `json:"field"`
	ValueMasked string  `json:"value_masked"`
	EntityType  string  `json:"entity_type"`
	Confidence  float64 `json:"confidence"`
}

func (s *Server) runEndpointScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	ep, err := s.endpointScans.FindByID(ctx, tenantID, id)
	if err != nil || ep == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}

	// Build request to the target endpoint
	method := ep.Method
	if method == "" {
		method = "GET"
	}
	req, err := http.NewRequestWithContext(ctx, method, ep.URL, nil)
	if err != nil {
		pkg.Error(w, fmt.Errorf("invalid URL: %w", err), http.StatusBadRequest)
		return
	}

	// Apply headers
	var hdrs map[string]string
	if len(ep.Headers) > 0 {
		json.Unmarshal(ep.Headers, &hdrs)
	}
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}

	// Apply auth
	if ep.AuthType != "none" && ep.AuthType != "" {
		var authCfg map[string]string
		json.Unmarshal(ep.AuthConfig, &authCfg)
		switch ep.AuthType {
		case "bearer":
			req.Header.Set("Authorization", "Bearer "+authCfg["token"])
		case "basic":
			req.SetBasicAuth(authCfg["username"], authCfg["password"])
		case "api_key":
			headerName := authCfg["header"]
			if headerName == "" {
				headerName = "X-API-Key"
			}
			req.Header.Set(headerName, authCfg["key"])
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		pkg.Error(w, fmt.Errorf("failed to fetch endpoint: %w", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // max 1MB

	// Extract string values from response
	stringValues := extractStringValues(body, resp.Header.Get("Content-Type"))

	// Classify with GLiNER
	findings := classifyEndpointValues(stringValues)

	// Determine risk level
	riskLevel := computeRiskLevel(findings)
	now := time.Now()
	ep.Status = "scanned"
	ep.LastScan = &now
	ep.RiskLevel = riskLevel

	findingsJSON, _ := json.Marshal(findings)
	ep.Findings = store.JSON(findingsJSON)

	if err := s.endpointScans.Update(ctx, ep); err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, ep)
}

func (s *Server) getEndpointScanFindings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")
	ep, err := s.endpointScans.FindByID(ctx, tenantID, id)
	if err != nil || ep == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	var findings []endpointFinding
	if len(ep.Findings) > 0 {
		json.Unmarshal(ep.Findings, &findings)
	}
	if findings == nil {
		findings = []endpointFinding{}
	}
	pkg.JSON(w, map[string]any{
		"endpoint_id": id,
		"name":        ep.Name,
		"url":         ep.URL,
		"risk_level":  ep.RiskLevel,
		"last_scan":   ep.LastScan,
		"findings":    findings,
		"total":       len(findings),
	})
}

// ── helpers ──────────────────────────────────────────────────────────────────

// extractStringValues pulls all leaf-level string values from JSON or plain text.
func extractStringValues(body []byte, contentType string) map[string]string {
	values := map[string]string{}
	ct := strings.ToLower(contentType)

	if strings.Contains(ct, "json") || (len(body) > 0 && body[0] == '{') {
		var obj map[string]any
		if err := json.Unmarshal(body, &obj); err == nil {
			flattenJSON("", obj, values)
			return values
		}
		var arr []any
		if err := json.Unmarshal(body, &arr); err == nil {
			for i, v := range arr {
				if obj, ok := v.(map[string]any); ok {
					flattenJSON(fmt.Sprintf("[%d]", i), obj, values)
				}
			}
			return values
		}
	}
	// Plain text — treat entire body as single value
	if len(body) > 0 {
		values["body"] = string(body)
	}
	return values
}

func flattenJSON(prefix string, obj map[string]any, out map[string]string) {
	for k, v := range obj {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch tv := v.(type) {
		case string:
			if tv != "" {
				out[key] = tv
			}
		case map[string]any:
			flattenJSON(key, tv, out)
		case []any:
			for i, elem := range tv {
				if s, ok := elem.(string); ok && s != "" {
					out[fmt.Sprintf("%s[%d]", key, i)] = s
				} else if m, ok := elem.(map[string]any); ok {
					flattenJSON(fmt.Sprintf("%s[%d]", key, i), m, out)
				}
			}
		}
	}
}

type glinerRequest struct {
	Texts  []string `json:"texts"`
	Labels []string `json:"labels"`
}

type glinerEntity struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
	Start int     `json:"start"`
	End   int     `json:"end"`
	Text  string  `json:"text"`
}

type glinerResponse struct {
	Entities [][]glinerEntity `json:"entities"`
}

func classifyEndpointValues(values map[string]string) []endpointFinding {
	findings := []endpointFinding{}
	if len(values) == 0 {
		return findings
	}

	// Prepare texts and field-index mapping
	texts := make([]string, 0, len(values))
	fields := make([]string, 0, len(values))
	for f, v := range values {
		fields = append(fields, f)
		texts = append(texts, v)
	}

	labels := []string{
		"PERSON", "EMAIL", "PHONE", "SSN", "CREDIT_CARD",
		"ADDRESS", "DATE_OF_BIRTH", "PASSPORT", "DRIVER_LICENSE",
		"IBAN", "BANK_ACCOUNT", "IP_ADDRESS", "MEDICAL_RECORD",
	}

	payload, _ := json.Marshal(glinerRequest{Texts: texts, Labels: labels})
	classifyCtx, classifyCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer classifyCancel()
	classifyReq, err := http.NewRequestWithContext(classifyCtx, "POST", "http://securelens-classifier:8085/classify", bytes.NewReader(payload))
	if err != nil {
		return localClassifyValues(values)
	}
	classifyReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(classifyReq)
	if err != nil || resp == nil {
		// Classifier unavailable — fall back to local pattern matching
		return localClassifyValues(values)
	}
	defer resp.Body.Close()

	var glResp glinerResponse
	if err := json.NewDecoder(resp.Body).Decode(&glResp); err != nil {
		return localClassifyValues(values)
	}

	for i, entityList := range glResp.Entities {
		if i >= len(fields) {
			break
		}
		field := fields[i]
		rawValue := texts[i]
		for _, ent := range entityList {
			if ent.Score < 0.5 {
				continue
			}
			findings = append(findings, endpointFinding{
				Field:       field,
				ValueMasked: maskEndpointValue(rawValue, ent.Label),
				EntityType:  ent.Label,
				Confidence:  ent.Score,
			})
		}
	}
	return findings
}

// localClassifyValues is a fallback regex-based classifier when GLiNER is unavailable.
func localClassifyValues(values map[string]string) []endpointFinding {
	findings := []endpointFinding{}
	for field, val := range values {
		for entityType, confidence := range detectPIILocal(val) {
			findings = append(findings, endpointFinding{
				Field:       field,
				ValueMasked: maskEndpointValue(val, entityType),
				EntityType:  entityType,
				Confidence:  confidence,
			})
		}
	}
	return findings
}

func detectPIILocal(val string) map[string]float64 {
	results := map[string]float64{}
	s := strings.TrimSpace(val)
	if len(s) == 0 {
		return results
	}
	// Email
	if strings.Contains(s, "@") && strings.Contains(s, ".") {
		results["EMAIL"] = 0.85
	}
	// Phone (rough)
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
	if len(digits) == 10 || len(digits) == 11 {
		results["PHONE"] = 0.7
	}
	// SSN xxx-xx-xxxx
	if len(s) == 11 && s[3] == '-' && s[6] == '-' {
		results["SSN"] = 0.9
	}
	// Credit card (16 digits)
	if len(digits) == 16 {
		results["CREDIT_CARD"] = 0.8
	}
	return results
}

func maskEndpointValue(val, entityType string) string {
	if len(val) == 0 {
		return "***"
	}
	switch entityType {
	case "EMAIL":
		if idx := strings.Index(val, "@"); idx > 0 {
			name := val[:idx]
			domain := val[idx:]
			if len(name) > 2 {
				return name[:1] + strings.Repeat("*", len(name)-1) + domain
			}
		}
	case "SSN":
		if len(val) >= 4 {
			return "***-**-" + val[len(val)-4:]
		}
	case "CREDIT_CARD":
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, val)
		if len(digits) >= 4 {
			return "****-****-****-" + digits[len(digits)-4:]
		}
	case "PHONE":
		if len(val) >= 4 {
			return strings.Repeat("*", len(val)-4) + val[len(val)-4:]
		}
	}
	// Generic mask: show first 2, mask rest
	if len(val) > 4 {
		return val[:2] + strings.Repeat("*", len(val)-2)
	}
	return "***"
}

func computeRiskLevel(findings []endpointFinding) string {
	if len(findings) == 0 {
		return "low"
	}
	critical := map[string]bool{
		"SSN": true, "CREDIT_CARD": true, "PASSPORT": true,
		"DRIVER_LICENSE": true, "IBAN": true, "BANK_ACCOUNT": true,
		"MEDICAL_RECORD": true,
	}
	high := map[string]bool{
		"EMAIL": true, "PHONE": true, "DATE_OF_BIRTH": true,
	}
	medium := map[string]bool{
		"PERSON": true, "ADDRESS": true, "IP_ADDRESS": true,
	}
	for _, f := range findings {
		if critical[f.EntityType] {
			return "critical"
		}
	}
	for _, f := range findings {
		if high[f.EntityType] {
			return "high"
		}
	}
	for _, f := range findings {
		if medium[f.EntityType] {
			return "medium"
		}
	}
	return "low"
}
