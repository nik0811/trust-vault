package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/securelens/securelens/internal/store"
)

// sensitivityLabelForEntity maps entity types to their required sensitivity labels (highest wins).
// Returns label string matching store.Label.Label values.
var entityLabelMap = map[string]string{
	// Restricted
	"SSN":            "RESTRICTED",
	"CREDIT_CARD":    "RESTRICTED",
	"CREDIT_CARD_FORMATTED": "RESTRICTED",
	"BANK_ACCOUNT":   "RESTRICTED",
	"ROUTING_NUMBER": "RESTRICTED",
	"IBAN":           "RESTRICTED",
	"PHI":            "RESTRICTED",
	"HEALTH_RECORD":  "RESTRICTED",
	"MEDICAL_RECORD": "RESTRICTED",
	"HEALTH_INSURANCE_ID": "RESTRICTED",
	// Confidential
	"EMAIL":          "CONFIDENTIAL",
	"PHONE":          "CONFIDENTIAL",
	"ADDRESS":        "CONFIDENTIAL",
	"PASSPORT":       "CONFIDENTIAL",
	"DRIVER_LICENSE": "CONFIDENTIAL",
	"DATE_OF_BIRTH":  "CONFIDENTIAL",
	"AWS_ACCESS_KEY": "CONFIDENTIAL",
	"AWS_SECRET_KEY": "CONFIDENTIAL",
	"API_KEY":        "CONFIDENTIAL",
	"JWT_TOKEN":      "CONFIDENTIAL",
	"IP_ADDRESS":     "CONFIDENTIAL",
	"IPV6_ADDRESS":   "CONFIDENTIAL",
	"MAC_ADDRESS":    "CONFIDENTIAL",
	"VIN":            "CONFIDENTIAL",
	// Internal
	"NAME":         "INTERNAL",
	"COMPANY":      "INTERNAL",
	"PERSON_NAME":  "INTERNAL",
	"US_ZIP":       "INTERNAL",
	"UK_POSTCODE":  "INTERNAL",
}

var labelPriority = map[string]int{
	"PUBLIC":             0,
	"INTERNAL":           1,
	"CONFIDENTIAL":       2,
	"HIGHLY_CONFIDENTIAL": 3,
	"RESTRICTED":         4,
}

// autoApplyGovernance runs after classifications are saved. It is non-blocking:
// call it in a goroutine so it never delays the API response.
func (s *Server) autoApplyGovernance(ctx context.Context, tenantID string, datasetID string, classifications []store.Classification) {
	if len(classifications) == 0 {
		return
	}

	// --- 1. Determine highest required sensitivity label ---
	highestLabel := "INTERNAL"
	detectedTypes := make([]string, 0, len(classifications))
	seen := map[string]bool{}

	for _, c := range classifications {
		if !seen[c.EntityType] {
			detectedTypes = append(detectedTypes, c.EntityType)
			seen[c.EntityType] = true
		}
		if label, ok := entityLabelMap[c.EntityType]; ok {
			if labelPriority[label] > labelPriority[highestLabel] {
				highestLabel = label
			}
		}
	}

	// --- 2. Upsert sensitivity label for this dataset ---
	if datasetID != "" {
		autoTrue := true
		existing := store.Label{}
		err := s.db.GetContext(ctx, &existing,
			"SELECT * FROM labels WHERE tenant_id = $1 AND dataset_id = $2 LIMIT 1",
			tenantID, datasetID)

		if err != nil {
			// No existing label — create one
			lbl := store.Label{
				TenantID:     tenantID,
				DatasetID:    datasetID,
				Label:        highestLabel,
				AutoAssigned: autoTrue,
			}
			if createErr := s.labels.Create(ctx, &lbl); createErr != nil {
				log.Error().Err(createErr).Str("dataset_id", datasetID).Msg("auto_governance: failed to create label")
			}
		} else if labelPriority[highestLabel] > labelPriority[existing.Label] {
			// Upgrade label only (never downgrade automatically)
			_, _ = s.db.ExecContext(ctx,
				"UPDATE labels SET label = $1, auto_assigned = true, updated_at = NOW() WHERE tenant_id = $2 AND dataset_id = $3",
				highestLabel, tenantID, datasetID)
		}
	}

	// --- 3. Audit log: classification.auto_governance ---
	details, _ := json.Marshal(map[string]any{
		"detected_types":    detectedTypes,
		"assigned_label":    highestLabel,
		"dataset_id":        datasetID,
		"classifications_count": len(classifications),
	})
	s.auditLogs.Create(ctx, &store.AuditLog{
		TenantID:   tenantID,
		UserID:     "system",
		Action:     "classification.auto_governance",
		Resource:   "dataset",
		ResourceID: datasetID,
		Details:    store.JSON(details),
	})

	// --- 4. Check policy coverage gaps ---
	var activePolicies []store.Policy
	_ = s.db.SelectContext(ctx, &activePolicies,
		"SELECT * FROM policies WHERE tenant_id = $1 AND active = true AND (type = 'redaction' OR type = 'access')",
		tenantID)

	// Build a set of entity types covered by at least one policy
	coveredTypes := map[string]bool{}
	for _, p := range activePolicies {
		var conditions map[string]any
		if p.Conditions != nil {
			_ = json.Unmarshal(p.Conditions, &conditions)
		}
		// Policies store covered entity types in conditions.entity_types or conditions.pii_types
		for _, key := range []string{"entity_types", "pii_types", "entity_type"} {
			if raw, ok := conditions[key]; ok {
				switch v := raw.(type) {
				case string:
					coveredTypes[v] = true
				case []any:
					for _, item := range v {
						if s, ok := item.(string); ok {
							coveredTypes[s] = true
						}
					}
				}
			}
		}
	}

	// Find PII/sensitive types with no policy coverage
	var gaps []string
	for _, entityType := range detectedTypes {
		label := entityLabelMap[entityType]
		if (label == "RESTRICTED" || label == "CONFIDENTIAL") && !coveredTypes[entityType] {
			gaps = append(gaps, entityType)
		}
	}

	if len(gaps) > 0 {
		gapDetails, _ := json.Marshal(map[string]any{
			"unprotected_types": gaps,
			"dataset_id":        datasetID,
			"message":           fmt.Sprintf("%d sensitive PII type(s) detected with no active redaction/access policy", len(gaps)),
		})
		s.auditLogs.Create(ctx, &store.AuditLog{
			TenantID:   tenantID,
			UserID:     "system",
			Action:     "compliance.gap_detected",
			Resource:   "dataset",
			ResourceID: datasetID,
			Details:    store.JSON(gapDetails),
		})
		log.Warn().
			Str("tenant_id", tenantID).
			Str("dataset_id", datasetID).
			Strs("gaps", gaps).
			Msg("auto_governance: compliance gaps detected")
	}

	log.Info().
		Str("tenant_id", tenantID).
		Str("dataset_id", datasetID).
		Str("label", highestLabel).
		Strs("detected", detectedTypes).
		Int("gaps", len(gaps)).
		Msg("auto_governance: completed")
}
