package domain

import (
	"encoding/json"
	"testing"

	"github.com/securelens/securelens/internal/store"
)

func TestEvaluatePolicies(t *testing.T) {
	tests := []struct {
		name             string
		policies         []store.Policy
		context          EvaluationContext
		expectedDecision string
		expectViolations bool
	}{
		{
			name:     "no policies - allow",
			policies: []store.Policy{},
			context: EvaluationContext{
				UserID:          "user-1",
				Classifications: []string{"EMAIL"},
			},
			expectedDecision: "allow",
			expectViolations: false,
		},
		{
			name: "inactive policy - ignored",
			policies: []store.Policy{
				{
					ID:     "policy-1",
					Name:   "Block PII",
					Active: false,
					Conditions: mustJSON(PolicyConditions{
						DataClassification: []string{"PII"},
					}),
					Actions: mustJSON(PolicyActions{
						Action: "deny",
					}),
				},
			},
			context: EvaluationContext{
				Classifications: []string{"PII"},
			},
			expectedDecision: "allow",
			expectViolations: false,
		},
		{
			name: "deny policy - blocks",
			policies: []store.Policy{
				{
					ID:     "policy-1",
					Name:   "Block PII to external LLM",
					Active: true,
					Conditions: mustJSON(PolicyConditions{
						DataClassification: []string{"PII"},
						DestinationType:    []string{"external_llm"},
					}),
					Actions: mustJSON(PolicyActions{
						Action: "deny",
					}),
				},
			},
			context: EvaluationContext{
				Classifications: []string{"PII"},
				DestinationType: "external_llm",
			},
			expectedDecision: "deny",
			expectViolations: true,
		},
		{
			name: "redact policy",
			policies: []store.Policy{
				{
					ID:     "policy-1",
					Name:   "Redact PII",
					Active: true,
					Conditions: mustJSON(PolicyConditions{
						DataClassification: []string{"EMAIL"},
					}),
					Actions: mustJSON(PolicyActions{
						Action:            "redact",
						RedactionStrategy: "mask",
					}),
				},
			},
			context: EvaluationContext{
				Classifications: []string{"EMAIL"},
			},
			expectedDecision: "redact",
			expectViolations: false,
		},
		{
			name: "conditions not matched - allow",
			policies: []store.Policy{
				{
					ID:     "policy-1",
					Name:   "Block PHI",
					Active: true,
					Conditions: mustJSON(PolicyConditions{
						DataClassification: []string{"PHI"},
					}),
					Actions: mustJSON(PolicyActions{
						Action: "deny",
					}),
				},
			},
			context: EvaluationContext{
				Classifications: []string{"EMAIL"}, // Not PHI
			},
			expectedDecision: "allow",
			expectViolations: false,
		},
		{
			name: "role-based policy",
			policies: []store.Policy{
				{
					ID:     "policy-1",
					Name:   "Analysts cannot access PII",
					Active: true,
					Conditions: mustJSON(PolicyConditions{
						DataClassification: []string{"PII"},
						UserRole:           []string{"analyst"},
					}),
					Actions: mustJSON(PolicyActions{
						Action: "deny",
					}),
				},
			},
			context: EvaluationContext{
				UserRole:        "analyst",
				Classifications: []string{"PII"},
			},
			expectedDecision: "deny",
			expectViolations: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluatePolicies(tt.policies, tt.context)

			if result.Decision != tt.expectedDecision {
				t.Errorf("Decision = %s, want %s", result.Decision, tt.expectedDecision)
			}

			hasViolations := len(result.Violations) > 0
			if hasViolations != tt.expectViolations {
				t.Errorf("Has violations = %v, want %v", hasViolations, tt.expectViolations)
			}
		})
	}
}

func TestCheckAIEligibility(t *testing.T) {
	tests := []struct {
		name            string
		policies        []store.Policy
		classifications []string
		purpose         string
		expectedOK      bool
	}{
		{
			name:            "no policies - eligible",
			policies:        []store.Policy{},
			classifications: []string{"EMAIL"},
			purpose:         "inference",
			expectedOK:      true,
		},
		{
			name: "blocked classification",
			policies: []store.Policy{
				{
					ID:     "ai-policy-1",
					Name:   "No PHI for AI",
					Type:   "ai",
					Active: true,
					Conditions: mustJSON(PolicyConditions{
						DataClassification: []string{"PHI"},
					}),
					Actions: mustJSON(PolicyActions{
						Action: "deny",
					}),
				},
			},
			classifications: []string{"PHI"},
			purpose:         "training",
			expectedOK:      false,
		},
		{
			name: "allowed classification",
			policies: []store.Policy{
				{
					ID:     "ai-policy-1",
					Name:   "No PHI for AI",
					Type:   "ai",
					Active: true,
					Conditions: mustJSON(PolicyConditions{
						DataClassification: []string{"PHI"},
					}),
					Actions: mustJSON(PolicyActions{
						Action: "deny",
					}),
				},
			},
			classifications: []string{"EMAIL"}, // Not PHI
			purpose:         "inference",
			expectedOK:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eligible, reasons := CheckAIEligibility(tt.policies, tt.classifications, tt.purpose)

			if eligible != tt.expectedOK {
				t.Errorf("Eligible = %v, want %v. Reasons: %v", eligible, tt.expectedOK, reasons)
			}
		})
	}
}

func mustJSON(v any) store.JSON {
	b, _ := json.Marshal(v)
	return store.JSON(b)
}
