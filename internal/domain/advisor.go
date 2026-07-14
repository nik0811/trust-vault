package domain

import (
	"encoding/json"
	"strconv"

	"github.com/trustvault/trustvault/internal/store"
)

// Recommendation represents a compliance recommendation
type Recommendation struct {
	ID            string `json:"id"`
	Severity      string `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
	Category      string `json:"category"` // pii, gdpr, ccpa, pdpb, uae_pdpl, retention, access, ai
	Title         string `json:"title"`
	Description   string `json:"description"`
	Action        string `json:"action"`
	Regulation    string `json:"regulation,omitempty"`
	AffectedCount int    `json:"affected_count,omitempty"`
}

// ComplianceGap represents a gap in compliance coverage
type ComplianceGap struct {
	Regulation string   `json:"regulation"`
	Score      float64  `json:"score"`
	Gaps       []string `json:"gaps"`
	Articles   []string `json:"articles,omitempty"`
}

// AdvisorContext holds data needed for generating recommendations
type AdvisorContext struct {
	TenantID            string
	Classifications     []store.Classification
	Policies            []store.Policy
	Labels              []store.Label
	RetentionViolations []store.RetentionViolation
	DataSources         []store.DataSource
	RoPA                []store.RoPA
	TotalDatasets       int
	LabeledDatasets     int
}

// PIITypes are entity types considered PII
var PIITypes = map[string]bool{
	"PII": true, "SSN": true, "EMAIL": true, "PHONE": true,
	"ADDRESS": true, "NAME": true, "DATE_OF_BIRTH": true,
	"PASSPORT": true, "DRIVER_LICENSE": true,
}

// HighSensitivityTypes require extra protection
var HighSensitivityTypes = map[string]bool{
	"SSN": true, "CREDIT_CARD": true, "BANK_ACCOUNT": true,
	"PHI": true, "HEALTH_RECORD": true, "GENETIC_DATA": true,
}

// GenerateRecommendations creates rule-based compliance recommendations
func GenerateRecommendations(advCtx *AdvisorContext) []Recommendation {
	if advCtx == nil {
		return []Recommendation{}
	}
	
	var recs []Recommendation

	recs = append(recs, checkUnprotectedPII(advCtx)...)
	recs = append(recs, checkGDPRCompliance(advCtx)...)
	recs = append(recs, checkRetentionViolations(advCtx)...)
	recs = append(recs, checkStaleDataSources(advCtx)...)
	recs = append(recs, checkMissingLabels(advCtx)...)
	recs = append(recs, checkHighSensitivityData(advCtx)...)
	recs = append(recs, checkAIGovernance(advCtx)...)
	recs = append(recs, checkPDPBCompliance(advCtx)...)
	recs = append(recs, checkUAEPDPLCompliance(advCtx)...)

	return recs
}

func checkUnprotectedPII(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	piiCounts := make(map[string]int)
	for _, c := range advCtx.Classifications {
		if PIITypes[c.EntityType] {
			piiCounts[c.EntityType]++
		}
	}

	coveredTypes := make(map[string]bool)
	for _, p := range advCtx.Policies {
		if !p.Active || (p.Type != "redaction" && p.Type != "access") {
			continue
		}
		var conditions PolicyConditions
		if err := json.Unmarshal(p.Conditions, &conditions); err != nil {
			continue
		}
		for _, dc := range conditions.DataClassification {
			coveredTypes[dc] = true
		}
	}

	for piiType, count := range piiCounts {
		if !coveredTypes[piiType] && !coveredTypes["*"] {
			recs = append(recs, Recommendation{
				ID:            "rec-pii-" + piiType,
				Severity:      "HIGH",
				Category:      "pii",
				Title:         "Unprotected " + piiType + " detected",
				Description:   strconv.Itoa(count) + " instances of " + piiType + " found without governance policy",
				Action:        "Create a redaction or access policy to protect " + piiType + " data",
				Regulation:    "GDPR Art. 5, CCPA 1798.100",
				AffectedCount: count,
			})
		}
	}

	return recs
}

func checkGDPRCompliance(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	hasPII := false
	for _, c := range advCtx.Classifications {
		if PIITypes[c.EntityType] {
			hasPII = true
			break
		}
	}

	if !hasPII {
		return recs
	}

	hasRetentionPolicy := false
	hasAccessPolicy := false

	for _, p := range advCtx.Policies {
		if !p.Active {
			continue
		}
		switch p.Type {
		case "retention":
			hasRetentionPolicy = true
		case "access":
			hasAccessPolicy = true
		}
	}

	if len(advCtx.RoPA) == 0 {
		recs = append(recs, Recommendation{
			ID:          "rec-gdpr-ropa",
			Severity:    "CRITICAL",
			Category:    "gdpr",
			Title:       "Missing Records of Processing Activities",
			Description: "GDPR Article 30 requires maintaining records of processing activities",
			Action:      "Create RoPA entries documenting all data processing activities",
			Regulation:  "GDPR Art. 30",
		})
	}

	if !hasRetentionPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-gdpr-retention",
			Severity:    "HIGH",
			Category:    "gdpr",
			Title:       "No data retention policy defined",
			Description: "GDPR requires data minimization and storage limitation",
			Action:      "Define retention policies for personal data categories",
			Regulation:  "GDPR Art. 5(1)(e)",
		})
	}

	if !hasAccessPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-gdpr-access",
			Severity:    "HIGH",
			Category:    "gdpr",
			Title:       "No access control policies defined",
			Description: "Personal data should have appropriate access restrictions",
			Action:      "Create access policies to limit who can view personal data",
			Regulation:  "GDPR Art. 25, Art. 32",
		})
	}

	return recs
}

func checkRetentionViolations(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	if len(advCtx.RetentionViolations) == 0 {
		return recs
	}

	overdueCount := 0
	for _, v := range advCtx.RetentionViolations {
		if v.DaysOverdue > 0 {
			overdueCount++
		}
	}

	if overdueCount > 0 {
		recs = append(recs, Recommendation{
			ID:            "rec-retention-overdue",
			Severity:      "HIGH",
			Category:      "retention",
			Title:         "Data retention violations detected",
			Description:   strconv.Itoa(overdueCount) + " datasets exceed their retention period",
			Action:        "Review and remediate retention violations - archive or delete overdue data",
			Regulation:    "GDPR Art. 17, CCPA 1798.105",
			AffectedCount: overdueCount,
		})
	}

	return recs
}

func checkStaleDataSources(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	staleCount := 0
	for _, ds := range advCtx.DataSources {
		if ds.LastScan == nil || ds.LastScan.IsZero() {
			staleCount++
		}
	}

	if staleCount > 0 {
		recs = append(recs, Recommendation{
			ID:            "rec-stale-sources",
			Severity:      "MEDIUM",
			Category:      "governance",
			Title:         "Data sources not scanned",
			Description:   strconv.Itoa(staleCount) + " data sources have never been scanned for sensitive data",
			Action:        "Schedule classification scans for all data sources",
			AffectedCount: staleCount,
		})
	}

	return recs
}

func checkMissingLabels(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	if advCtx.TotalDatasets > 0 && advCtx.LabeledDatasets < advCtx.TotalDatasets {
		unlabeled := advCtx.TotalDatasets - advCtx.LabeledDatasets
		percentage := float64(unlabeled) / float64(advCtx.TotalDatasets) * 100

		severity := "LOW"
		if percentage > 50 {
			severity = "HIGH"
		} else if percentage > 25 {
			severity = "MEDIUM"
		}

		recs = append(recs, Recommendation{
			ID:            "rec-missing-labels",
			Severity:      severity,
			Category:      "governance",
			Title:         "Datasets missing sensitivity labels",
			Description:   strconv.Itoa(unlabeled) + " datasets lack sensitivity labels",
			Action:        "Assign sensitivity labels to all datasets containing classified data",
			AffectedCount: unlabeled,
		})
	}

	return recs
}

func checkHighSensitivityData(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	totalHighSens := 0
	for _, c := range advCtx.Classifications {
		if HighSensitivityTypes[c.EntityType] {
			totalHighSens++
		}
	}

	hasHighSensPolicy := false
	for _, p := range advCtx.Policies {
		if !p.Active {
			continue
		}
		var conditions PolicyConditions
		if err := json.Unmarshal(p.Conditions, &conditions); err != nil {
			continue
		}
		for _, dc := range conditions.DataClassification {
			if HighSensitivityTypes[dc] || dc == "*" {
				hasHighSensPolicy = true
				break
			}
		}
	}

	if totalHighSens > 0 && !hasHighSensPolicy {
		recs = append(recs, Recommendation{
			ID:            "rec-high-sens-policy",
			Severity:      "CRITICAL",
			Category:      "security",
			Title:         "High-sensitivity data without protection policy",
			Description:   strconv.Itoa(totalHighSens) + " instances of high-sensitivity data found without explicit protection policies",
			Action:        "Create strict access and redaction policies for high-sensitivity data types",
			Regulation:    "HIPAA, PCI-DSS, GDPR Art. 9",
			AffectedCount: totalHighSens,
		})
	}

	return recs
}

func checkAIGovernance(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	hasAIPolicy := false
	for _, p := range advCtx.Policies {
		if p.Active && p.Type == "ai" {
			hasAIPolicy = true
			break
		}
	}

	hasSensitiveData := len(advCtx.Classifications) > 0

	if hasSensitiveData && !hasAIPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-ai-governance",
			Severity:    "MEDIUM",
			Category:    "ai",
			Title:       "No AI governance policies defined",
			Description: "Sensitive data exists but no policies govern its use in AI/LLM systems",
			Action:      "Create AI governance policies to control what data can be sent to LLMs",
			Regulation:  "EU AI Act",
		})
	}

	return recs
}

// checkPDPBCompliance checks India's Digital Personal Data Protection Act 2023 requirements
func checkPDPBCompliance(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	hasPII := false
	for _, c := range advCtx.Classifications {
		if PIITypes[c.EntityType] {
			hasPII = true
			break
		}
	}

	if !hasPII {
		return recs
	}

	hasConsentPolicy := false
	hasDataLocalizationPolicy := false
	hasChildrenDataPolicy := false
	hasRetentionPolicy := false

	for _, p := range advCtx.Policies {
		if !p.Active {
			continue
		}
		switch p.Type {
		case "consent":
			hasConsentPolicy = true
		case "localization", "data_localization":
			hasDataLocalizationPolicy = true
		case "children", "minor_protection":
			hasChildrenDataPolicy = true
		case "retention":
			hasRetentionPolicy = true
		}
	}

	if !hasConsentPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-pdpb-consent",
			Severity:    "CRITICAL",
			Category:    "pdpb",
			Title:       "Missing consent management for DPDP Act",
			Description: "India's DPDP Act requires explicit consent before processing personal data",
			Action:      "Implement consent management system with clear purpose specification",
			Regulation:  "DPDP Act 2023 Section 6",
		})
	}

	if !hasDataLocalizationPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-pdpb-localization",
			Severity:    "HIGH",
			Category:    "pdpb",
			Title:       "No data localization policy defined",
			Description: "Critical personal data must be stored and processed within India under DPDP Act",
			Action:      "Define data localization policies for critical personal data categories",
			Regulation:  "DPDP Act 2023 Section 16",
		})
	}

	if !hasChildrenDataPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-pdpb-children",
			Severity:    "HIGH",
			Category:    "pdpb",
			Title:       "Missing children's data protection policy",
			Description: "DPDP Act requires verifiable parental consent for processing data of persons under 18",
			Action:      "Implement age verification and parental consent mechanisms",
			Regulation:  "DPDP Act 2023 Section 9",
		})
	}

	if !hasRetentionPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-pdpb-retention",
			Severity:    "HIGH",
			Category:    "pdpb",
			Title:       "No data retention limits defined",
			Description: "DPDP Act requires data to be erased when purpose is fulfilled or consent withdrawn",
			Action:      "Define retention policies with automatic deletion when purpose is complete",
			Regulation:  "DPDP Act 2023 Section 8(7)",
		})
	}

	if len(advCtx.RoPA) == 0 {
		recs = append(recs, Recommendation{
			ID:          "rec-pdpb-dpo",
			Severity:    "CRITICAL",
			Category:    "pdpb",
			Title:       "Significant Data Fiduciary obligations not documented",
			Description: "Significant Data Fiduciaries must appoint a DPO and conduct Data Protection Impact Assessments",
			Action:      "Document processing activities and appoint Data Protection Officer if applicable",
			Regulation:  "DPDP Act 2023 Section 10",
		})
	}

	return recs
}

// checkUAEPDPLCompliance checks UAE Personal Data Protection Law requirements
func checkUAEPDPLCompliance(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	hasPII := false
	for _, c := range advCtx.Classifications {
		if PIITypes[c.EntityType] {
			hasPII = true
			break
		}
	}

	if !hasPII {
		return recs
	}

	hasLawfulBasisPolicy := false
	hasCrossBorderPolicy := false
	hasSpecialCategoryPolicy := false
	hasRetentionPolicy := false
	hasAccessPolicy := false

	for _, p := range advCtx.Policies {
		if !p.Active {
			continue
		}
		switch p.Type {
		case "lawful_basis", "consent":
			hasLawfulBasisPolicy = true
		case "cross_border", "transfer":
			hasCrossBorderPolicy = true
		case "special_category", "sensitive":
			hasSpecialCategoryPolicy = true
		case "retention":
			hasRetentionPolicy = true
		case "access":
			hasAccessPolicy = true
		}
	}

	if !hasLawfulBasisPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-uae-lawful-basis",
			Severity:    "CRITICAL",
			Category:    "uae_pdpl",
			Title:       "Missing lawful basis documentation",
			Description: "UAE PDPL requires documented lawful basis for all personal data processing",
			Action:      "Document lawful basis (consent, contract, legal obligation, vital interests, public interest, or legitimate interests)",
			Regulation:  "UAE PDPL Art. 4",
		})
	}

	if !hasCrossBorderPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-uae-cross-border",
			Severity:    "HIGH",
			Category:    "uae_pdpl",
			Title:       "No cross-border transfer policy defined",
			Description: "UAE PDPL restricts transfer of personal data outside UAE without adequate protection",
			Action:      "Define cross-border transfer policies ensuring adequate protection level",
			Regulation:  "UAE PDPL Art. 22",
		})
	}

	hasHighSensitivity := false
	for _, c := range advCtx.Classifications {
		if HighSensitivityTypes[c.EntityType] {
			hasHighSensitivity = true
			break
		}
	}

	if hasHighSensitivity && !hasSpecialCategoryPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-uae-special-category",
			Severity:    "CRITICAL",
			Category:    "uae_pdpl",
			Title:       "Special category data without explicit protection",
			Description: "UAE PDPL requires explicit consent and additional safeguards for sensitive personal data",
			Action:      "Implement explicit consent and enhanced protection for health, biometric, genetic, and other sensitive data",
			Regulation:  "UAE PDPL Art. 7",
		})
	}

	if !hasRetentionPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-uae-retention",
			Severity:    "HIGH",
			Category:    "uae_pdpl",
			Title:       "No data retention policy for UAE PDPL",
			Description: "UAE PDPL requires data to be kept only as long as necessary for the specified purpose",
			Action:      "Define retention periods aligned with processing purposes",
			Regulation:  "UAE PDPL Art. 5",
		})
	}

	if !hasAccessPolicy {
		recs = append(recs, Recommendation{
			ID:          "rec-uae-data-subject-rights",
			Severity:    "HIGH",
			Category:    "uae_pdpl",
			Title:       "Data subject rights not fully implemented",
			Description: "UAE PDPL grants rights to access, rectification, erasure, and data portability",
			Action:      "Implement mechanisms for data subject access requests and rights fulfillment",
			Regulation:  "UAE PDPL Art. 13-18",
		})
	}

	if len(advCtx.RoPA) == 0 {
		recs = append(recs, Recommendation{
			ID:          "rec-uae-records",
			Severity:    "HIGH",
			Category:    "uae_pdpl",
			Title:       "Missing records of processing activities",
			Description: "UAE PDPL requires controllers to maintain records of processing activities",
			Action:      "Create and maintain records of all personal data processing activities",
			Regulation:  "UAE PDPL Art. 8",
		})
	}

	return recs
}
