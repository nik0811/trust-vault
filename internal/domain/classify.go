package domain

import (
	"context"
	"strings"

	"github.com/trustvault/trustvault/internal/store"
)

type ClassificationResult struct {
	Entities []Entity `json:"entities"`
	Text     string   `json:"text"`
}

type Entity struct {
	Type       string  `json:"type"`
	Value      string  `json:"value"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Confidence float64 `json:"confidence"`
}

// ClassifyText runs classification on text using pattern matching and ML model
func ClassifyText(ctx context.Context, text string, entityTypes []string) (*ClassificationResult, error) {
	result := &ClassificationResult{
		Text:     text,
		Entities: []Entity{},
	}

	// Tier 1: Fast pattern matching
	result.Entities = append(result.Entities, detectPatterns(text)...)

	// Tier 2: ML model (GLiNER) - would call model service in production
	// For now, return pattern-based results

	return result, nil
}

func detectPatterns(text string) []Entity {
	var entities []Entity

	// Email pattern
	words := strings.Fields(text)
	for i, word := range words {
		if strings.Contains(word, "@") && strings.Contains(word, ".") {
			entities = append(entities, Entity{
				Type:       "EMAIL",
				Value:      word,
				Start:      i,
				End:        i + len(word),
				Confidence: 0.95,
			})
		}
	}

	// Phone pattern (simplified)
	for i, word := range words {
		cleaned := strings.ReplaceAll(strings.ReplaceAll(word, "-", ""), " ", "")
		if len(cleaned) >= 10 && isNumeric(cleaned) {
			entities = append(entities, Entity{
				Type:       "PHONE",
				Value:      word,
				Start:      i,
				End:        i + len(word),
				Confidence: 0.85,
			})
		}
	}

	return entities
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ApplyRedaction masks sensitive entities based on policy
func ApplyRedaction(text string, entities []Entity, strategy string) string {
	result := text
	for _, e := range entities {
		var masked string
		switch strategy {
		case "mask":
			masked = strings.Repeat("*", len(e.Value))
		case "hash":
			masked = "[REDACTED:" + e.Type + "]"
		case "remove":
			masked = ""
		default:
			masked = "[" + e.Type + "]"
		}
		result = strings.Replace(result, e.Value, masked, 1)
	}
	return result
}

// AssignLabel determines sensitivity label based on classifications
func AssignLabel(classifications []store.Classification) string {
	hasRestricted := false
	hasHighlyConfidential := false
	hasConfidential := false
	hasInternal := false

	for _, c := range classifications {
		switch c.EntityType {
		case "PHI", "HEALTH_RECORD", "GENETIC_DATA":
			hasRestricted = true
		case "PCI", "CREDIT_CARD", "BANK_ACCOUNT", "LEGAL_PRIVILEGE":
			hasHighlyConfidential = true
		case "PII", "SSN", "PASSPORT", "DRIVER_LICENSE", "EMAIL", "PHONE", "ADDRESS":
			hasConfidential = true
		case "INTERNAL_ID", "EMPLOYEE_ID":
			hasInternal = true
		}
	}

	if hasRestricted {
		return "RESTRICTED"
	}
	if hasHighlyConfidential {
		return "HIGHLY_CONFIDENTIAL"
	}
	if hasConfidential {
		return "CONFIDENTIAL"
	}
	if hasInternal {
		return "INTERNAL"
	}
	return "PUBLIC"
}
