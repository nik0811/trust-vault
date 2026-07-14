package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/trustvault/trustvault/internal/domain"
	"github.com/trustvault/trustvault/internal/events"
	"github.com/trustvault/trustvault/internal/external"
	"github.com/trustvault/trustvault/internal/pkg"
	"github.com/trustvault/trustvault/internal/store"
)

type GateQueryRequest struct {
	Query       string         `json:"query" validate:"required"`
	Context     map[string]any `json:"context,omitempty"`
	MaxChunks   int            `json:"max_chunks,omitempty"`
	LLMEndpoint string         `json:"llm_endpoint,omitempty"`
	Model       string         `json:"model,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
}

type GateQueryResponse struct {
	ID         string         `json:"id"`
	Response   string         `json:"response"`
	Context    []ChunkResult  `json:"context"`
	Decision   string         `json:"decision"`
	Redactions []Redaction    `json:"redactions,omitempty"`
	LatencyMs  int            `json:"latency_ms"`
}

type ChunkResult struct {
	ID       string         `json:"id"`
	Content  string         `json:"content"`
	Source   string         `json:"source"`
	Score    float32        `json:"score"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// redactionMasks maps entity types to their redaction placeholders
var redactionMasks = map[string]string{
	"SSN":           "[REDACTED_SSN]",
	"CREDIT_CARD":   "[REDACTED_CC]",
	"EMAIL":         "[REDACTED_EMAIL]",
	"PHONE":         "[REDACTED_PHONE]",
	"ADDRESS":       "[REDACTED_ADDRESS]",
	"PASSPORT":      "[REDACTED_PASSPORT]",
	"DRIVER_LICENSE": "[REDACTED_DL]",
	"BANK_ACCOUNT":  "[REDACTED_BANK]",
	"PHI":           "[REDACTED_PHI]",
	"HEALTH_RECORD": "[REDACTED_HEALTH]",
	"GENETIC_DATA":  "[REDACTED_GENETIC]",
	"PII":           "[REDACTED_PII]",
	"IP_ADDRESS":    "[REDACTED_IP]",
	"DATE_OF_BIRTH": "[REDACTED_DOB]",
	"NAME":          "[REDACTED_NAME]",
}

// sensitiveTypes are entity types that should always be flagged in output validation
var sensitiveTypes = map[string]bool{
	"SSN": true, "CREDIT_CARD": true, "BANK_ACCOUNT": true,
	"PHI": true, "HEALTH_RECORD": true, "GENETIC_DATA": true,
	"PASSPORT": true, "DRIVER_LICENSE": true,
}

// getRedactionMask returns the appropriate mask for an entity type
func getRedactionMask(entityType string) string {
	if mask, ok := redactionMasks[entityType]; ok {
		return mask
	}
	return "[REDACTED_" + strings.ToUpper(entityType) + "]"
}

// shouldRedact checks if a classification type should be redacted based on policies
func shouldRedact(entityType string, policies []store.Policy) bool {
	for _, policy := range policies {
		if !policy.Active || (policy.Type != "redaction" && policy.Type != "ai") {
			continue
		}

		var conditions domain.PolicyConditions
		if err := json.Unmarshal(policy.Conditions, &conditions); err != nil {
			continue
		}

		var actions domain.PolicyActions
		if err := json.Unmarshal(policy.Actions, &actions); err != nil {
			continue
		}

		// Check if this entity type matches policy conditions
		for _, dc := range conditions.DataClassification {
			if dc == entityType || dc == "*" {
				if actions.Action == "redact" || actions.Action == "deny" {
					return true
				}
			}
		}
	}

	// Default: redact high-sensitivity types even without explicit policy
	return sensitiveTypes[entityType]
}

// redactSensitiveData applies redaction to text based on classifications and policies
func redactSensitiveData(text string, entities []domain.Entity, policies []store.Policy) (string, []Redaction) {
	if len(entities) == 0 {
		return text, nil
	}

	// Sort by position (reverse) to maintain offsets during replacement
	sorted := make([]domain.Entity, len(entities))
	copy(sorted, entities)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Start > sorted[j].Start
	})

	var redactions []Redaction
	result := text

	for _, e := range sorted {
		if !shouldRedact(e.Type, policies) {
			continue
		}

		mask := getRedactionMask(e.Type)

		// Handle byte-based positions vs rune-based
		if e.Start >= 0 && e.End <= len(result) && e.Start < e.End {
			result = result[:e.Start] + mask + result[e.End:]
			redactions = append(redactions, Redaction{
				Start:  e.Start,
				End:    e.End,
				Type:   e.Type,
				Masked: mask,
			})
		} else {
			// Fallback: replace by value
			result = strings.Replace(result, e.Value, mask, 1)
			redactions = append(redactions, Redaction{
				Start:  -1,
				End:    -1,
				Type:   e.Type,
				Masked: mask,
			})
		}
	}

	return result, redactions
}

// validateOutput checks LLM response for potential data leakage
func (s *Server) validateOutput(ctx context.Context, response string, tenantID string) (bool, []string) {
	// Run classification on the response
	result, err := domain.ClassifyText(ctx, response, nil)
	if err != nil {
		return true, nil // Allow on classification error
	}

	var leaks []string
	for _, entity := range result.Entities {
		if sensitiveTypes[entity.Type] {
			leaks = append(leaks, entity.Type)
		}
	}

	return len(leaks) == 0, leaks
}

func (s *Server) gateQuery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	userID := pkg.UserFromCtx(ctx)
	start := time.Now()

	var req GateQueryRequest
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	if req.MaxChunks == 0 {
		req.MaxChunks = 5
	}

	// 1. Retrieve context from vector DB
	chunks, err := s.qdrant.SearchText(ctx, tenantID, req.Query, req.MaxChunks)
	if err != nil {
		chunks = nil
	}

	var contextChunks []ChunkResult
	for _, c := range chunks {
		contextChunks = append(contextChunks, ChunkResult{
			ID:       c.ID,
			Content:  c.Payload["content"].(string),
			Source:   c.Payload["source"].(string),
			Score:    c.Score,
			Metadata: c.Payload,
		})
	}

	// 2. Get active policies for redaction
	policies, _ := s.policies.List(ctx, tenantID, store.ListOpts{Limit: 100})

	// 3. Classify and redact the query
	queryClassification, _ := domain.ClassifyText(ctx, req.Query, nil)
	redactedQuery, queryRedactions := redactSensitiveData(req.Query, queryClassification.Entities, policies)

	// 4. Classify and redact context chunks
	var allRedactions []Redaction
	allRedactions = append(allRedactions, queryRedactions...)

	var redactedContextText string
	for i, c := range contextChunks {
		chunkClassification, _ := domain.ClassifyText(ctx, c.Content, nil)
		redactedContent, chunkRedactions := redactSensitiveData(c.Content, chunkClassification.Entities, policies)
		contextChunks[i].Content = redactedContent
		redactedContextText += redactedContent + "\n\n"
		allRedactions = append(allRedactions, chunkRedactions...)
	}

	// 5. Evaluate governance policies for decision
	var classifications []string
	for _, e := range queryClassification.Entities {
		classifications = append(classifications, e.Type)
	}

	evalCtx := domain.EvaluationContext{
		UserID:          userID,
		TenantID:        tenantID,
		DestinationType: "llm",
		Classifications: classifications,
	}
	evalResult := domain.EvaluatePolicies(policies, evalCtx)

	// If policy denies, return early
	if evalResult.Decision == "deny" {
		latency := int(time.Since(start).Milliseconds())
		gateQuery := store.GateQuery{
			TenantID:  tenantID,
			UserID:    userID,
			Query:     req.Query,
			Response:  "Request blocked by governance policy",
			Decision:  "deny",
			LatencyMs: latency,
		}
		s.gateQueries.Create(ctx, &gateQuery)
		events.Emit("gate.query.blocked", gateQuery)

		pkg.JSON(w, GateQueryResponse{
			ID:         gateQuery.ID,
			Response:   "Request blocked by governance policy",
			Context:    contextChunks,
			Decision:   "deny",
			Redactions: allRedactions,
			LatencyMs:  latency,
		})
		return
	}

	// 6. Build prompt with redacted context
	messages := []external.ChatMessage{
		{Role: "system", Content: "You are a helpful assistant. Use the following context to answer questions:\n\n" + redactedContextText},
		{Role: "user", Content: redactedQuery},
	}

	// 7. Call LLM
	llmEndpoint := req.LLMEndpoint
	if llmEndpoint == "" {
		llmEndpoint = os.Getenv("LLM_ENDPOINT")
		if llmEndpoint == "" {
			llmEndpoint = "http://localhost:11434/v1"
		}
	}
	
	model := req.Model
	if model == "" {
		model = os.Getenv("LLM_MODEL")
		if model == "" {
			model = "llama3.2"
		}
	}

	llm := external.NewLLM(llmEndpoint, "", model)
	llmResp, err := llm.Chat(ctx, messages)

	var responseText string
	if err != nil {
		responseText = "Error calling LLM: " + err.Error()
	} else if len(llmResp.Choices) > 0 {
		responseText = llmResp.Choices[0].Message.Content
	}

	// 8. Validate output for data leakage
	decision := evalResult.Decision
	if decision == "" {
		decision = "allow"
	}

	isClean, leakedTypes := s.validateOutput(ctx, responseText, tenantID)
	if !isClean {
		decision = "flagged"
		responseText = "[WARNING: Potential data leakage detected (" + strings.Join(leakedTypes, ", ") + ")] " + responseText

		events.Emit("gate.leakage.detected", map[string]any{
			"tenant_id":    tenantID,
			"user_id":      userID,
			"leaked_types": leakedTypes,
		})
	}

	latency := int(time.Since(start).Milliseconds())

	// 9. Record audit trail
	redactionsJSON, _ := json.Marshal(allRedactions)
	gateQuery := store.GateQuery{
		TenantID:    tenantID,
		UserID:      userID,
		Query:       req.Query,
		Response:    responseText,
		Decision:    decision,
		Redactions:  store.JSON(redactionsJSON),
		LatencyMs:   latency,
		LLMEndpoint: llmEndpoint,
	}
	s.gateQueries.Create(ctx, &gateQuery)

	events.Emit("gate.query", gateQuery)

	// 10. Emit OpenLineage event for AI Gate data flow
	var inputDatasets []map[string]any
	for _, chunk := range contextChunks {
		inputDatasets = append(inputDatasets, map[string]any{
			"namespace": "trustvault",
			"name":      chunk.Source,
		})
	}
	if len(inputDatasets) == 0 {
		inputDatasets = []map[string]any{
			{"namespace": "trustvault", "name": "query_context"},
		}
	}

	lineageEvent := map[string]any{
		"eventType": "COMPLETE",
		"eventTime": time.Now().UTC().Format(time.RFC3339),
		"run": map[string]any{
			"runId": gateQuery.ID,
		},
		"job": map[string]any{
			"namespace": "trustvault",
			"name":      "ai_gate_query",
		},
		"inputs": inputDatasets,
		"outputs": []map[string]any{
			{
				"namespace": "llm",
				"name":      req.Model,
			},
		},
		"producer": "trustvault-gateway",
	}
	if err := s.datahub.EmitLineage(ctx, lineageEvent); err != nil {
		_ = err
	}

	// Create data flow entry for lineage tracking
	for _, chunk := range contextChunks {
		if chunk.Source != "" {
			dataFlow := store.DataFlow{
				TenantID:        tenantID,
				SourceDatasetID: chunk.Source,
				TargetDatasetID: "ai-gate-" + gateQuery.ID,
				FlowType:        "ai_consumption",
			}
			s.dataFlows.Create(ctx, &dataFlow)
		}
	}

	pkg.JSON(w, GateQueryResponse{
		ID:         gateQuery.ID,
		Response:   responseText,
		Context:    contextChunks,
		Decision:   decision,
		Redactions: allRedactions,
		LatencyMs:  latency,
	})
}

func (s *Server) gateRetrieve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Query     string `json:"query" validate:"required"`
		MaxChunks int    `json:"max_chunks,omitempty"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	if req.MaxChunks == 0 {
		req.MaxChunks = 5
	}

	chunks, _ := s.qdrant.Search(ctx, tenantID, []float32{0.1, 0.2, 0.3}, req.MaxChunks)

	var results []ChunkResult
	for _, c := range chunks {
		content := ""
		if v, ok := c.Payload["content"].(string); ok {
			content = v
		}
		source := ""
		if v, ok := c.Payload["source"].(string); ok {
			source = v
		}
		results = append(results, ChunkResult{
			ID:       c.ID,
			Content:  content,
			Source:   source,
			Score:    c.Score,
			Metadata: c.Payload,
		})
	}

	pkg.JSON(w, map[string]any{"chunks": results})
}

func (s *Server) gateValidate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Response string `json:"response" validate:"required"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Run classification on response to detect data leakage
	isClean, leakedTypes := s.validateOutput(ctx, req.Response, tenantID)

	policies, _ := s.policies.List(ctx, tenantID, store.ListOpts{Limit: 100})

	decision := "allow"
	var violations []PolicyViolation

	if !isClean {
		decision = "flagged"
		for _, leakType := range leakedTypes {
			violations = append(violations, PolicyViolation{
				PolicyID:   "output_validation",
				PolicyName: "Data Leakage Detection",
				Reason:     "Sensitive data type detected in output: " + leakType,
			})
		}
	}

	var appliedPolicies []string
	for _, policy := range policies {
		if policy.Active {
			appliedPolicies = append(appliedPolicies, policy.ID)
		}
	}

	pkg.JSON(w, EvaluateResponse{
		Decision:        decision,
		AppliedPolicies: appliedPolicies,
		Violations:      violations,
	})
}

func (s *Server) gateStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var stats struct {
		TotalQueries   int     `json:"total_queries"`
		QueriesBlocked int     `json:"queries_blocked"`
		AvgLatencyMs   float64 `json:"avg_latency_ms"`
		QueriesPerMin  float64 `json:"queries_per_min"`
	}

	s.db.GetContext(ctx, &stats.TotalQueries,
		"SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1", tenantID)
	s.db.GetContext(ctx, &stats.QueriesBlocked,
		"SELECT COUNT(*) FROM gate_queries WHERE tenant_id = $1 AND decision = 'deny'", tenantID)
	s.db.GetContext(ctx, &stats.AvgLatencyMs,
		"SELECT COALESCE(AVG(latency_ms), 0) FROM gate_queries WHERE tenant_id = $1", tenantID)

	pkg.JSON(w, stats)
}

func (s *Server) listGateQueries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	limit, offset := pkg.ParseListOpts(r)

	queries, err := s.gateQueries.List(ctx, tenantID, store.ListOpts{Limit: limit, Offset: offset})
	if err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, queries)
}

func (s *Server) getGateQuery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	id := chi.URLParam(r, "id")

	query, err := s.gateQueries.FindByID(ctx, tenantID, id)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	if query == nil {
		pkg.Error(w, pkg.ErrNotFound, http.StatusNotFound)
		return
	}
	pkg.JSON(w, query)
}
