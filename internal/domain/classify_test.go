package domain

import (
	"context"
	"testing"

	"github.com/trustvault/trustvault/internal/store"
)

func TestClassifyText(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		expectedTypes  []string
		minEntities    int
	}{
		{
			name:          "detect email",
			text:          "Contact us at support@example.com for help",
			expectedTypes: []string{"EMAIL"},
			minEntities:   1,
		},
		{
			name:          "detect phone",
			text:          "Call us at 555-123-4567 today",
			expectedTypes: []string{"PHONE"},
			minEntities:   1,
		},
		{
			name:          "detect multiple",
			text:          "Email john@test.com or call 1234567890",
			expectedTypes: []string{"EMAIL", "PHONE"},
			minEntities:   2,
		},
		{
			name:          "no entities",
			text:          "This is plain text with no sensitive data",
			expectedTypes: []string{},
			minEntities:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ClassifyText(context.Background(), tt.text, nil)
			if err != nil {
				t.Fatalf("ClassifyText failed: %v", err)
			}

			if len(result.Entities) < tt.minEntities {
				t.Errorf("Expected at least %d entities, got %d", tt.minEntities, len(result.Entities))
			}

			for _, expectedType := range tt.expectedTypes {
				found := false
				for _, entity := range result.Entities {
					if entity.Type == expectedType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find entity type %s", expectedType)
				}
			}
		})
	}
}

func TestApplyRedaction(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		entities []Entity
		strategy string
		expected string
	}{
		{
			name: "mask strategy",
			text: "Email: test@example.com",
			entities: []Entity{
				{Type: "EMAIL", Value: "test@example.com"},
			},
			strategy: "mask",
			expected: "Email: ****************",
		},
		{
			name: "hash strategy",
			text: "Email: test@example.com",
			entities: []Entity{
				{Type: "EMAIL", Value: "test@example.com"},
			},
			strategy: "hash",
			expected: "Email: [REDACTED:EMAIL]",
		},
		{
			name: "remove strategy",
			text: "Email: test@example.com",
			entities: []Entity{
				{Type: "EMAIL", Value: "test@example.com"},
			},
			strategy: "remove",
			expected: "Email: ",
		},
		{
			name: "default strategy",
			text: "Email: test@example.com",
			entities: []Entity{
				{Type: "EMAIL", Value: "test@example.com"},
			},
			strategy: "unknown",
			expected: "Email: [EMAIL]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyRedaction(tt.text, tt.entities, tt.strategy)
			if result != tt.expected {
				t.Errorf("ApplyRedaction = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestAssignLabel(t *testing.T) {
	tests := []struct {
		name            string
		classifications []store.Classification
		expectedLabel   string
	}{
		{
			name:            "no classifications",
			classifications: []store.Classification{},
			expectedLabel:   "PUBLIC",
		},
		{
			name: "PII data",
			classifications: []store.Classification{
				{EntityType: "EMAIL"},
				{EntityType: "PHONE"},
			},
			expectedLabel: "CONFIDENTIAL",
		},
		{
			name: "PHI data (restricted)",
			classifications: []store.Classification{
				{EntityType: "PHI"},
				{EntityType: "EMAIL"},
			},
			expectedLabel: "RESTRICTED",
		},
		{
			name: "PCI data (highly confidential)",
			classifications: []store.Classification{
				{EntityType: "CREDIT_CARD"},
			},
			expectedLabel: "HIGHLY_CONFIDENTIAL",
		},
		{
			name: "internal only",
			classifications: []store.Classification{
				{EntityType: "EMPLOYEE_ID"},
			},
			expectedLabel: "INTERNAL",
		},
		{
			name: "mixed - highest wins",
			classifications: []store.Classification{
				{EntityType: "EMAIL"},           // CONFIDENTIAL
				{EntityType: "CREDIT_CARD"},     // HIGHLY_CONFIDENTIAL
				{EntityType: "HEALTH_RECORD"},   // RESTRICTED
			},
			expectedLabel: "RESTRICTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AssignLabel(tt.classifications)
			if result != tt.expectedLabel {
				t.Errorf("AssignLabel = %s, want %s", result, tt.expectedLabel)
			}
		})
	}
}
