package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/trustvault/trustvault/internal/events"
	"github.com/trustvault/trustvault/internal/pkg"
	"github.com/trustvault/trustvault/internal/store"
)

func (s *Server) classifyText(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Text        string   `json:"text" validate:"required"`
		EntityTypes []string `json:"entity_types,omitempty"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Queue classification via Kafka for async GLiNER processing
	s.kafka.Produce(ctx, "classification-jobs", tenantID, map[string]any{
		"text":         req.Text,
		"entity_types": req.EntityTypes,
		"tenant_id":    tenantID,
		"mode":         "text",
	})

	// For sync response, run basic pattern matching
	results := s.runBasicClassification(req.Text, req.EntityTypes)

	// Store classification results
	for _, res := range results {
		c := store.Classification{
			TenantID:   tenantID,
			EntityType: res["entity"].(string),
			Value:      res["value"].(string),
			Confidence: res["confidence"].(float64),
		}
		s.classifications.Create(ctx, &c)
	}

	pkg.JSON(w, map[string]any{"entities": results})
}

// PIIPattern defines a PII detection pattern with validation
type PIIPattern struct {
	Pattern    string
	Regex      *regexp.Regexp
	Confidence float64
	Validator  func(string) bool
}

// piiPatterns contains all PII detection patterns with confidence scores
var piiPatterns map[string]PIIPattern

func init() {
	piiPatterns = make(map[string]PIIPattern)
	
	patterns := map[string]struct {
		Pattern    string
		Confidence float64
		Validator  func(string) bool
	}{
		"EMAIL": {
			Pattern:    `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
			Confidence: 0.95,
			Validator:  validateEmail,
		},
		"PHONE": {
			Pattern:    `(?:\+?1[-.\s]?)?\(?[2-9]\d{2}\)?[-.\s]?\d{3}[-.\s]?\d{4}`,
			Confidence: 0.90,
			Validator:  validatePhone,
		},
		"SSN": {
			Pattern:    `\b(?!000|666|9\d{2})\d{3}[-\s]?(?!00)\d{2}[-\s]?(?!0000)\d{4}\b`,
			Confidence: 0.95,
			Validator:  validateSSN,
		},
		"CREDIT_CARD": {
			Pattern:    `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12}|(?:2131|1800|35\d{3})\d{11})\b`,
			Confidence: 0.95,
			Validator:  validateCreditCard,
		},
		"CREDIT_CARD_FORMATTED": {
			Pattern:    `\b(?:4[0-9]{3}|5[1-5][0-9]{2}|3[47][0-9]{2}|6(?:011|5[0-9]{2}))[-\s]?[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}\b`,
			Confidence: 0.95,
			Validator:  validateCreditCard,
		},
		"IP_ADDRESS": {
			Pattern:    `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`,
			Confidence: 0.90,
			Validator:  validateIPAddress,
		},
		"DATE_OF_BIRTH": {
			Pattern:    `\b(?:0?[1-9]|1[0-2])[-/](?:0?[1-9]|[12][0-9]|3[01])[-/](?:19|20)\d{2}\b`,
			Confidence: 0.75,
			Validator:  nil,
		},
		"PASSPORT": {
			Pattern:    `\b[A-Z]{1,2}[0-9]{6,9}\b`,
			Confidence: 0.70,
			Validator:  nil,
		},
		"DRIVER_LICENSE": {
			Pattern:    `\b[A-Z]{1,2}[0-9]{5,8}\b`,
			Confidence: 0.65,
			Validator:  nil,
		},
		"IBAN": {
			Pattern:    `\b[A-Z]{2}[0-9]{2}[A-Z0-9]{4}[0-9]{7}(?:[A-Z0-9]?){0,16}\b`,
			Confidence: 0.90,
			Validator:  validateIBAN,
		},
		"BANK_ACCOUNT": {
			Pattern:    `\b[0-9]{8,17}\b`,
			Confidence: 0.50,
			Validator:  nil,
		},
		"ROUTING_NUMBER": {
			Pattern:    `\b[0-9]{9}\b`,
			Confidence: 0.60,
			Validator:  validateRoutingNumber,
		},
		"MAC_ADDRESS": {
			Pattern:    `\b(?:[0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}\b`,
			Confidence: 0.95,
			Validator:  nil,
		},
		"IPV6_ADDRESS": {
			Pattern:    `\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`,
			Confidence: 0.90,
			Validator:  nil,
		},
		"AWS_ACCESS_KEY": {
			Pattern:    `\bAKIA[0-9A-Z]{16}\b`,
			Confidence: 0.98,
			Validator:  nil,
		},
		"AWS_SECRET_KEY": {
			Pattern:    `\b[A-Za-z0-9/+=]{40}\b`,
			Confidence: 0.70,
			Validator:  nil,
		},
		"API_KEY": {
			Pattern:    `(?:api[_-]?key|apikey|api_secret)[=:]\s*['"]?([a-zA-Z0-9_-]{20,})['"]?`,
			Confidence: 0.85,
			Validator:  nil,
		},
		"JWT_TOKEN": {
			Pattern:    `\beyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*\b`,
			Confidence: 0.95,
			Validator:  nil,
		},
		"MEDICAL_RECORD": {
			Pattern:    `(?:MRN|Medical Record)[:\s#]*[A-Z0-9]{6,12}`,
			Confidence: 0.85,
			Validator:  nil,
		},
		"HEALTH_INSURANCE_ID": {
			Pattern:    `\b[A-Z]{3}[0-9]{9}\b`,
			Confidence: 0.70,
			Validator:  nil,
		},
		"VIN": {
			Pattern:    `\b[A-HJ-NPR-Z0-9]{17}\b`,
			Confidence: 0.85,
			Validator:  validateVIN,
		},
		"US_ZIP": {
			Pattern:    `\b[0-9]{5}(?:-[0-9]{4})?\b`,
			Confidence: 0.70,
			Validator:  nil,
		},
		"UK_POSTCODE": {
			Pattern:    `\b[A-Z]{1,2}[0-9][0-9A-Z]?\s?[0-9][A-Z]{2}\b`,
			Confidence: 0.85,
			Validator:  nil,
		},
	}
	
	for name, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			continue
		}
		piiPatterns[name] = PIIPattern{
			Pattern:    p.Pattern,
			Regex:      re,
			Confidence: p.Confidence,
			Validator:  p.Validator,
		}
	}
}

func validateEmail(s string) bool {
	return strings.Contains(s, "@") && strings.Contains(s, ".")
}

func validatePhone(s string) bool {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
	return len(digits) >= 10 && len(digits) <= 15
}

func validateSSN(s string) bool {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
	if len(digits) != 9 {
		return false
	}
	area := digits[0:3]
	if area == "000" || area == "666" || area[0] == '9' {
		return false
	}
	return digits[3:5] != "00" && digits[5:9] != "0000"
}

func validateCreditCard(s string) bool {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	return luhnCheck(digits)
}

func luhnCheck(digits string) bool {
	sum := 0
	alt := false
	for i := len(digits) - 1; i >= 0; i-- {
		n := int(digits[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}

func validateIPAddress(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
			n = n*10 + int(c-'0')
		}
		if n > 255 {
			return false
		}
	}
	return true
}

func validateIBAN(s string) bool {
	return len(s) >= 15 && len(s) <= 34
}

func validateRoutingNumber(s string) bool {
	if len(s) != 9 {
		return false
	}
	weights := []int{3, 7, 1, 3, 7, 1, 3, 7, 1}
	sum := 0
	for i, c := range s {
		if c < '0' || c > '9' {
			return false
		}
		sum += int(c-'0') * weights[i]
	}
	return sum%10 == 0
}

func validateVIN(s string) bool {
	if len(s) != 17 {
		return false
	}
	for _, c := range s {
		if c == 'I' || c == 'O' || c == 'Q' {
			return false
		}
	}
	return true
}

func (s *Server) runBasicClassification(text string, entityTypes []string) []map[string]any {
	results := make([]map[string]any, 0)

	for entityType, piiPattern := range piiPatterns {
		if len(entityTypes) > 0 && !containsIgnoreCase(entityTypes, entityType) {
			continue
		}
		if piiPattern.Regex == nil {
			continue
		}
		matches := piiPattern.Regex.FindAllStringIndex(text, -1)
		for _, m := range matches {
			value := text[m[0]:m[1]]
			confidence := piiPattern.Confidence
			if piiPattern.Validator != nil && !piiPattern.Validator(value) {
				confidence *= 0.5
			}
			if confidence < 0.5 {
				continue
			}
			results = append(results, map[string]any{
				"entity":     entityType,
				"value":      value,
				"confidence": confidence,
				"start":      m[0],
				"end":        m[1],
			})
		}
	}
	return results
}

func containsIgnoreCase(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

func (s *Server) classifyDataset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		DatasetID string `json:"dataset_id" validate:"required"`
		Async     bool   `json:"async,omitempty"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	s.kafka.Produce(ctx, "classification-jobs", tenantID, map[string]any{
		"dataset_id": req.DatasetID,
		"tenant_id":  tenantID,
	})

	pkg.JSON(w, map[string]string{"status": "queued", "job_id": req.DatasetID})
}

func (s *Server) getClassificationResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasetID := chi.URLParam(r, "dataset_id")

	var results []store.Classification
	s.db.SelectContext(ctx, &results,
		"SELECT * FROM classifications WHERE tenant_id = $1 AND dataset_id = $2 ORDER BY confidence DESC",
		tenantID, datasetID)

	pkg.JSON(w, results)
}

func (s *Server) createClassificationRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var req struct {
		Name          string  `json:"name" validate:"required"`
		Type          string  `json:"type" validate:"required,oneof=override pattern whitelist threshold"`
		ColumnPattern string  `json:"column_pattern"`
		ValuePattern  string  `json:"value_pattern"`
		EntityType    string  `json:"entity_type"`
		Confidence    float64 `json:"confidence"`
		Priority      int     `json:"priority"`
		Active        bool    `json:"active"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Default confidence to 0.95 if not provided
	if req.Confidence == 0 {
		req.Confidence = 0.95
	}

	rule := store.ClassificationRule{
		TenantID:      tenantID,
		Name:          req.Name,
		Type:          req.Type,
		ColumnPattern: req.ColumnPattern,
		ValuePattern:  req.ValuePattern,
		EntityType:    req.EntityType,
		Confidence:    req.Confidence,
		Priority:      req.Priority,
		Active:        req.Active,
	}
	s.classificationRules.Create(ctx, &rule)

	pkg.JSON(w, rule, http.StatusCreated)
}

// builtInModels are the default classification models available
var builtInModels = []map[string]any{
	{
		"id":          "trustvault-pii-edge",
		"name":        "TrustVault PII Edge",
		"description": "Fast, lightweight PII detection using pattern matching and heuristics. Optimized for edge deployment.",
		"size":        "197MB",
		"accuracy":    0.96,
		"speed":       "4M chars/sec",
		"default":     true,
		"active":      true,
		"type":        "Pattern",
		"version":     "1.0.0",
		"entity_types": []string{
			"EMAIL", "PHONE", "SSN", "CREDIT_CARD", "IP_ADDRESS", "DATE_OF_BIRTH",
			"PASSPORT", "DRIVER_LICENSE", "IBAN", "MAC_ADDRESS", "AWS_ACCESS_KEY",
			"JWT_TOKEN", "VIN", "US_ZIP", "UK_POSTCODE",
		},
	},
	{
		"id":          "trustvault-pii-pro",
		"name":        "TrustVault PII Pro",
		"description": "High-accuracy PII detection using advanced ML model. Best for comprehensive classification.",
		"size":        "330MB",
		"accuracy":    0.98,
		"speed":       "2M chars/sec",
		"default":     false,
		"active":      true,
		"type":        "ML",
		"version":     "1.0.0",
		"entity_types": []string{
			"EMAIL", "PHONE", "SSN", "CREDIT_CARD", "IP_ADDRESS", "DATE_OF_BIRTH",
			"PASSPORT", "DRIVER_LICENSE", "IBAN", "BANK_ACCOUNT", "ROUTING_NUMBER",
			"MAC_ADDRESS", "IPV6_ADDRESS", "AWS_ACCESS_KEY", "AWS_SECRET_KEY",
			"API_KEY", "JWT_TOKEN", "MEDICAL_RECORD", "HEALTH_INSURANCE_ID",
			"VIN", "US_ZIP", "UK_POSTCODE", "PERSON_NAME", "ADDRESS", "ORGANIZATION",
		},
	},
	{
		"id":          "trustvault-phi-detector",
		"name":        "TrustVault PHI Detector",
		"description": "Specialized model for Protected Health Information (PHI) detection. HIPAA compliant.",
		"size":        "280MB",
		"accuracy":    0.97,
		"speed":       "3M chars/sec",
		"default":     false,
		"active":      true,
		"type":        "ML",
		"version":     "1.0.0",
		"entity_types": []string{
			"MEDICAL_RECORD", "HEALTH_INSURANCE_ID", "DIAGNOSIS_CODE", "PROCEDURE_CODE",
			"MEDICATION", "PATIENT_ID", "PROVIDER_ID", "DATE_OF_SERVICE",
		},
	},
}

func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var models []store.ClassificationModel
	err := s.db.SelectContext(ctx, &models,
		"SELECT * FROM classification_models WHERE active = true ORDER BY is_default DESC, name")
	if err != nil || len(models) == 0 {
		pkg.JSON(w, builtInModels)
		return
	}

	result := make([]map[string]any, 0, len(models)+len(builtInModels))
	for _, m := range builtInModels {
		result = append(result, m)
	}
	for _, m := range models {
		result = append(result, map[string]any{
			"id":       m.ID,
			"name":     m.Name,
			"size":     m.Size,
			"accuracy": m.Accuracy,
			"speed":    m.Speed,
			"default":  m.IsDefault,
			"active":   m.Active,
		})
	}

	pkg.JSON(w, result)
}

func (s *Server) listClassificationRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)

	var rules []store.ClassificationRule
	s.db.SelectContext(ctx, &rules,
		"SELECT * FROM classification_rules WHERE tenant_id = $1 ORDER BY priority DESC, created_at DESC",
		tenantID)

	pkg.JSON(w, rules)
}

func (s *Server) getClassificationRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	ruleID := chi.URLParam(r, "id")

	var rule store.ClassificationRule
	err := s.db.GetContext(ctx, &rule,
		"SELECT * FROM classification_rules WHERE tenant_id = $1 AND id = $2",
		tenantID, ruleID)
	if err != nil {
		pkg.Error(w, err, http.StatusNotFound)
		return
	}

	pkg.JSON(w, rule)
}

func (s *Server) updateClassificationRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	ruleID := chi.URLParam(r, "id")

	var req struct {
		Name          string  `json:"name"`
		Type          string  `json:"type"`
		ColumnPattern string  `json:"column_pattern"`
		ValuePattern  string  `json:"value_pattern"`
		EntityType    string  `json:"entity_type"`
		Confidence    float64 `json:"confidence"`
		Priority      int     `json:"priority"`
		Active        *bool   `json:"active"`
	}
	if err := pkg.Bind(r, &req); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	updates := []string{"updated_at = NOW()"}
	args := []any{}
	argIdx := 1

	if req.Name != "" {
		updates = append(updates, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, req.Name)
		argIdx++
	}
	if req.Type != "" {
		updates = append(updates, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, req.Type)
		argIdx++
	}
	if req.ColumnPattern != "" {
		updates = append(updates, fmt.Sprintf("column_pattern = $%d", argIdx))
		args = append(args, req.ColumnPattern)
		argIdx++
	}
	if req.ValuePattern != "" {
		updates = append(updates, fmt.Sprintf("value_pattern = $%d", argIdx))
		args = append(args, req.ValuePattern)
		argIdx++
	}
	if req.EntityType != "" {
		updates = append(updates, fmt.Sprintf("entity_type = $%d", argIdx))
		args = append(args, req.EntityType)
		argIdx++
	}
	if req.Confidence > 0 {
		updates = append(updates, fmt.Sprintf("confidence = $%d", argIdx))
		args = append(args, req.Confidence)
		argIdx++
	}
	if req.Priority != 0 {
		updates = append(updates, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, req.Priority)
		argIdx++
	}
	if req.Active != nil {
		updates = append(updates, fmt.Sprintf("active = $%d", argIdx))
		args = append(args, *req.Active)
		argIdx++
	}

	args = append(args, tenantID, ruleID)
	query := fmt.Sprintf("UPDATE classification_rules SET %s WHERE tenant_id = $%d AND id = $%d",
		strings.Join(updates, ", "), argIdx, argIdx+1)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	// Fetch updated rule
	var rule store.ClassificationRule
	s.db.GetContext(ctx, &rule,
		"SELECT * FROM classification_rules WHERE tenant_id = $1 AND id = $2",
		tenantID, ruleID)

	pkg.JSON(w, rule)
}

func (s *Server) deleteClassificationRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	ruleID := chi.URLParam(r, "id")

	_, err := s.db.ExecContext(ctx,
		"DELETE FROM classification_rules WHERE tenant_id = $1 AND id = $2",
		tenantID, ruleID)
	if err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	pkg.JSON(w, map[string]string{"status": "deleted"})
}

// ColumnClassification represents classification results for a single column
type ColumnClassification struct {
	ID                string  `json:"id"`
	ColumnName        string  `json:"column_name"`
	DataType          string  `json:"data_type"`
	SensitivityLevel  string  `json:"sensitivity_level"`
	Confidence        float64 `json:"confidence"`
	ClassificationTag string  `json:"classification_tag"`
	Status            string  `json:"status"`
}

// DatasetClassificationResponse represents the full classification summary for a dataset
type DatasetClassificationResponse struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	SourceID          string                 `json:"source_id"`
	TotalColumns      int                    `json:"total_columns"`
	ClassifiedColumns int                    `json:"classified_columns"`
	PendingColumns    int                    `json:"pending_columns"`
	AvgConfidence     float64                `json:"avg_confidence"`
	Columns           []ColumnClassification `json:"columns"`
}

// getDatasetClassification returns classification summary for a dataset
func (s *Server) getDatasetClassification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasetID := chi.URLParam(r, "id")

	// Get classifications for this dataset
	var classifications []store.Classification
	err := s.db.SelectContext(ctx, &classifications,
		"SELECT * FROM classifications WHERE tenant_id = $1 AND dataset_id = $2 ORDER BY confidence DESC",
		tenantID, datasetID)
	if err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	// Get datasource info for the dataset name
	var datasource store.DataSource
	err = s.db.GetContext(ctx, &datasource,
		"SELECT * FROM datasources WHERE tenant_id = $1 AND id = $2",
		tenantID, datasetID)
	
	datasetName := datasetID
	sourceID := ""
	if err == nil {
		datasetName = datasource.Name
		sourceID = datasource.ID
	}

	// Build column classifications from the classification results
	columns := make([]ColumnClassification, 0, len(classifications))
	var totalConfidence float64
	classifiedCount := 0
	pendingCount := 0

	for _, c := range classifications {
		status := "classified"
		if c.Confidence < 0.7 {
			status = "review"
			pendingCount++
		} else {
			classifiedCount++
		}

		sensitivityLevel := determineSensitivityLevel(c.EntityType, c.Confidence)

		columns = append(columns, ColumnClassification{
			ID:                c.ID,
			ColumnName:        c.Value,
			DataType:          "string",
			SensitivityLevel:  sensitivityLevel,
			Confidence:        c.Confidence,
			ClassificationTag: c.EntityType,
			Status:            status,
		})
		totalConfidence += c.Confidence
	}

	avgConfidence := 0.0
	if len(classifications) > 0 {
		avgConfidence = totalConfidence / float64(len(classifications))
	}

	response := DatasetClassificationResponse{
		ID:                datasetID,
		Name:              datasetName,
		SourceID:          sourceID,
		TotalColumns:      len(columns),
		ClassifiedColumns: classifiedCount,
		PendingColumns:    pendingCount,
		AvgConfidence:     avgConfidence,
		Columns:           columns,
	}

	pkg.JSON(w, response)
}

// getDatasetColumns returns column-level classification results for a dataset
func (s *Server) getDatasetColumns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasetID := chi.URLParam(r, "id")

	var classifications []store.Classification
	err := s.db.SelectContext(ctx, &classifications,
		"SELECT * FROM classifications WHERE tenant_id = $1 AND dataset_id = $2 ORDER BY confidence DESC",
		tenantID, datasetID)
	if err != nil {
		pkg.Error(w, err, http.StatusInternalServerError)
		return
	}

	columns := make([]ColumnClassification, 0, len(classifications))
	for _, c := range classifications {
		status := "classified"
		if c.Confidence < 0.7 {
			status = "review"
		}

		sensitivityLevel := determineSensitivityLevel(c.EntityType, c.Confidence)

		columns = append(columns, ColumnClassification{
			ID:                c.ID,
			ColumnName:        c.Value,
			DataType:          "string",
			SensitivityLevel:  sensitivityLevel,
			Confidence:        c.Confidence,
			ClassificationTag: c.EntityType,
			Status:            status,
		})
	}

	pkg.JSON(w, columns)
}

// reclassifyDataset triggers a re-classification job for a dataset
func (s *Server) reclassifyDataset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := pkg.TenantFromCtx(ctx)
	datasetID := chi.URLParam(r, "id")

	// Queue re-classification via Kafka
	s.kafka.Produce(ctx, "classification-jobs", tenantID, map[string]any{
		"dataset_id":    datasetID,
		"tenant_id":     tenantID,
		"reclassify":    true,
	})

	pkg.JSON(w, map[string]string{
		"status":     "queued",
		"job_id":     datasetID,
		"message":    "Re-classification job has been queued",
	})
}

// determineSensitivityLevel maps entity types to sensitivity levels
func determineSensitivityLevel(entityType string, confidence float64) string {
	highSensitivity := map[string]bool{
		"SSN": true, "CREDIT_CARD": true, "CREDIT_CARD_FORMATTED": true,
		"BANK_ACCOUNT": true, "ROUTING_NUMBER": true, "IBAN": true,
		"AWS_ACCESS_KEY": true, "AWS_SECRET_KEY": true, "API_KEY": true,
		"JWT_TOKEN": true, "MEDICAL_RECORD": true, "HEALTH_INSURANCE_ID": true,
	}

	mediumSensitivity := map[string]bool{
		"EMAIL": true, "PHONE": true, "DATE_OF_BIRTH": true,
		"PASSPORT": true, "DRIVER_LICENSE": true, "VIN": true,
	}

	if highSensitivity[entityType] {
		if confidence >= 0.9 {
			return "critical"
		}
		return "high"
	}

	if mediumSensitivity[entityType] {
		return "medium"
	}

	return "low"
}

// classificationCallback handles completion callbacks from the worker
func (s *Server) classificationCallback(w http.ResponseWriter, r *http.Request) {
	var callback struct {
		TenantID          string `json:"tenant_id"`
		DatasetID         string `json:"dataset_id"`
		Status            string `json:"status"`
		ColumnsClassified int    `json:"columns_classified"`
		Message           string `json:"message"`
		Error             string `json:"error,omitempty"`
	}
	if err := pkg.Bind(r, &callback); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	log.Info().
		Str("dataset_id", callback.DatasetID).
		Str("status", callback.Status).
		Int("columns_classified", callback.ColumnsClassified).
		Msg("Classification callback received")

	// Emit SSE event for completion
	events.Emit("classification.completed", map[string]any{
		"tenant_id":          callback.TenantID,
		"dataset_id":         callback.DatasetID,
		"status":             callback.Status,
		"columns_classified": callback.ColumnsClassified,
		"message":            callback.Message,
		"error":              callback.Error,
	})

	pkg.JSON(w, map[string]string{"status": "ok"})
}

// classificationProgress handles progress callbacks from the worker
func (s *Server) classificationProgress(w http.ResponseWriter, r *http.Request) {
	var progress struct {
		TenantID  string `json:"tenant_id"`
		DatasetID string `json:"dataset_id"`
		Message   string `json:"message"`
		Progress  struct {
			Current int `json:"current"`
			Total   int `json:"total"`
		} `json:"progress"`
	}
	if err := pkg.Bind(r, &progress); err != nil {
		pkg.Error(w, err, http.StatusBadRequest)
		return
	}

	log.Debug().
		Str("dataset_id", progress.DatasetID).
		Str("message", progress.Message).
		Int("current", progress.Progress.Current).
		Int("total", progress.Progress.Total).
		Msg("Classification progress received")

	// Emit SSE event for progress
	events.Emit("classification.progress", map[string]any{
		"tenant_id":  progress.TenantID,
		"dataset_id": progress.DatasetID,
		"message":    progress.Message,
		"progress":   progress.Progress,
		"status":     "running",
	})

	pkg.JSON(w, map[string]string{"status": "ok"})
}
