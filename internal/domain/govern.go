package domain

import (
	"encoding/json"

	"github.com/trustvault/trustvault/internal/store"
)

type PolicyConditions struct {
	DataClassification []string `json:"data_classification,omitempty"`
	SourceType         []string `json:"source_type,omitempty"`
	DestinationType    []string `json:"destination_type,omitempty"`
	UserRole           []string `json:"user_role,omitempty"`
	Label              []string `json:"label,omitempty"`
}

type PolicyActions struct {
	Action            string `json:"action"` // allow, deny, redact
	RedactionStrategy string `json:"redaction_strategy,omitempty"`
	NotifyOnViolation bool   `json:"notify_on_violation,omitempty"`
}

type EvaluationContext struct {
	UserID          string
	UserRole        string
	TenantID        string
	SourceType      string
	DestinationType string
	Classifications []string
	Label           string
}

type EvaluationResult struct {
	Decision        string
	AppliedPolicies []string
	Violations      []PolicyViolation
	Redactions      []RedactionAction
}

type PolicyViolation struct {
	PolicyID   string
	PolicyName string
	Reason     string
	Regulation string
}

type RedactionAction struct {
	EntityType string
	Strategy   string
}

// EvaluatePolicies checks data against all active policies
func EvaluatePolicies(policies []store.Policy, ctx EvaluationContext) *EvaluationResult {
	result := &EvaluationResult{
		Decision:        "allow",
		AppliedPolicies: []string{},
		Violations:      []PolicyViolation{},
		Redactions:      []RedactionAction{},
	}

	for _, policy := range policies {
		if !policy.Active {
			continue
		}

		var conditions PolicyConditions
		json.Unmarshal(policy.Conditions, &conditions)

		var actions PolicyActions
		json.Unmarshal(policy.Actions, &actions)

		if matchesConditions(conditions, ctx) {
			result.AppliedPolicies = append(result.AppliedPolicies, policy.ID)

			switch actions.Action {
			case "deny":
				result.Decision = "deny"
				result.Violations = append(result.Violations, PolicyViolation{
					PolicyID:   policy.ID,
					PolicyName: policy.Name,
					Reason:     "Policy conditions matched",
				})
			case "redact":
				if result.Decision != "deny" {
					result.Decision = "redact"
				}
				for _, class := range ctx.Classifications {
					result.Redactions = append(result.Redactions, RedactionAction{
						EntityType: class,
						Strategy:   actions.RedactionStrategy,
					})
				}
			}
		}
	}

	return result
}

func matchesConditions(conditions PolicyConditions, ctx EvaluationContext) bool {
	// Check data classification
	if len(conditions.DataClassification) > 0 {
		matched := false
		for _, c := range conditions.DataClassification {
			for _, ctxClass := range ctx.Classifications {
				if c == ctxClass {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}

	// Check destination type
	if len(conditions.DestinationType) > 0 {
		matched := false
		for _, d := range conditions.DestinationType {
			if d == ctx.DestinationType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check user role
	if len(conditions.UserRole) > 0 {
		matched := false
		for _, r := range conditions.UserRole {
			if r == ctx.UserRole {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check label
	if len(conditions.Label) > 0 {
		matched := false
		for _, l := range conditions.Label {
			if l == ctx.Label {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// CheckAIEligibility determines if data can be used for AI
func CheckAIEligibility(policies []store.Policy, classifications []string, purpose string) (bool, []string) {
	reasons := []string{}

	for _, policy := range policies {
		if !policy.Active || policy.Type != "ai" {
			continue
		}

		var conditions PolicyConditions
		json.Unmarshal(policy.Conditions, &conditions)

		var actions PolicyActions
		json.Unmarshal(policy.Actions, &actions)

		// Check if any classification is blocked for AI
		for _, class := range classifications {
			for _, blocked := range conditions.DataClassification {
				if class == blocked && actions.Action == "deny" {
					reasons = append(reasons, "Classification "+class+" blocked by policy "+policy.Name)
				}
			}
		}
	}

	return len(reasons) == 0, reasons
}
