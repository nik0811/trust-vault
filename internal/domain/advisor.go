package domain

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/securelens/securelens/internal/store"
)

// EvidenceItem represents a verifiable piece of evidence supporting a compliance finding
type EvidenceItem struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Source      string    `json:"source"`
	Description string    `json:"description"`
	ResourceID  string    `json:"resource_id,omitempty"`
	ResourceRef string    `json:"resource_ref,omitempty"`
	DetectedAt  time.Time `json:"detected_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// AffectedAsset represents a data source/dataset impacted by a finding
type AffectedAsset struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Recommendation represents a compliance recommendation
type Recommendation struct {
	ID                string          `json:"id"`
	Severity          string          `json:"severity"`
	Category          string          `json:"category"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	Action            string          `json:"action"`
	Regulation        string          `json:"regulation,omitempty"`
	RegulationArticle string          `json:"regulation_article,omitempty"`
	AffectedCount     int             `json:"affected_count,omitempty"`
	Evidence          []EvidenceItem  `json:"evidence,omitempty"`
	AffectedAssets    []AffectedAsset `json:"affected_assets,omitempty"`
	DetectedAt        time.Time       `json:"detected_at"`
	EvidenceSummary   string          `json:"evidence_summary,omitempty"`
	SeverityReason    string          `json:"severity_reason,omitempty"`
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
	AuditLogs           []store.AuditLog
	AssessmentTime      time.Time
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

	if advCtx.AssessmentTime.IsZero() {
		advCtx.AssessmentTime = time.Now()
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
	piiEvidence := make(map[string][]EvidenceItem)
	piiAssets := make(map[string]map[string]AffectedAsset)

	for _, c := range advCtx.Classifications {
		if PIITypes[c.EntityType] {
			piiCounts[c.EntityType]++
			if piiAssets[c.EntityType] == nil {
				piiAssets[c.EntityType] = make(map[string]AffectedAsset)
			}
			piiAssets[c.EntityType][c.SourceID] = AffectedAsset{
				ID: c.SourceID, Name: c.DatasetID, Type: "dataset",
			}
			if len(piiEvidence[c.EntityType]) < 5 {
				piiEvidence[c.EntityType] = append(piiEvidence[c.EntityType], EvidenceItem{
					ID:          c.ID,
					Type:        "classification_result",
					Source:      "classification_engine",
					Description: c.EntityType + " detected in dataset " + c.DatasetID + " with confidence " + strconv.FormatFloat(c.Confidence, 'f', 2, 64),
					ResourceID:  c.DatasetID,
					ResourceRef: "classifications/" + c.ID,
					DetectedAt:  c.CreatedAt,
					Metadata:    map[string]any{"confidence": c.Confidence, "entity_type": c.EntityType, "source_id": c.SourceID},
				})
			}
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
			assets := make([]AffectedAsset, 0, len(piiAssets[piiType]))
			for _, a := range piiAssets[piiType] {
				assets = append(assets, a)
			}
			recs = append(recs, Recommendation{
				ID:                "rec-pii-" + piiType,
				Severity:          "HIGH",
				Category:          "pii",
				Title:             "Unprotected " + piiType + " detected",
				Description:       strconv.Itoa(count) + " instances of " + piiType + " found without governance policy",
				Action:            "Create a redaction or access policy to protect " + piiType + " data",
				Regulation:        "GDPR Art. 5, CCPA 1798.100",
				RegulationArticle: "GDPR Art. 5(1)(f) - Integrity and Confidentiality",
				AffectedCount:     count,
				Evidence:          piiEvidence[piiType],
				AffectedAssets:    assets,
				DetectedAt:        advCtx.AssessmentTime,
				EvidenceSummary:   strconv.Itoa(count) + " " + piiType + " instances detected across " + strconv.Itoa(len(assets)) + " data sources with no redaction/access policy covering this type",
				SeverityReason:    "HIGH: Personal data exposed without protection violates fundamental data protection principles",
			})
		}
	}

	return recs
}

func checkGDPRCompliance(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	hasPII := false
	var piiClassifications []store.Classification
	for _, c := range advCtx.Classifications {
		if PIITypes[c.EntityType] {
			hasPII = true
			if len(piiClassifications) < 5 {
				piiClassifications = append(piiClassifications, c)
			}
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
		var evidence []EvidenceItem
		evidence = append(evidence, EvidenceItem{
			ID:          "ev-gdpr-ropa-absence",
			Type:        "absence_of_record",
			Source:      "ropa_registry",
			Description: "No Records of Processing Activities found in the system. GDPR Art. 30 mandates maintaining these records.",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"table": "ropa", "count": 0},
		})
		for _, c := range piiClassifications[:min(3, len(piiClassifications))] {
			evidence = append(evidence, EvidenceItem{
				ID:          "ev-gdpr-ropa-pii-" + c.ID,
				Type:        "classification_result",
				Source:      "classification_engine",
				Description: "PII type " + c.EntityType + " processed without documented processing activity",
				ResourceID:  c.DatasetID,
				ResourceRef: "classifications/" + c.ID,
				DetectedAt:  c.CreatedAt,
			})
		}
		recs = append(recs, Recommendation{
			ID:                "rec-gdpr-ropa",
			Severity:          "CRITICAL",
			Category:          "gdpr",
			Title:             "Missing Records of Processing Activities",
			Description:       "GDPR Article 30 requires maintaining records of processing activities",
			Action:            "Create RoPA entries documenting all data processing activities",
			Regulation:        "GDPR Art. 30",
			RegulationArticle: "GDPR Art. 30(1) - Each controller shall maintain a record of processing activities under its responsibility",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "0 RoPA entries found while PII data is actively being processed across multiple datasets",
			SeverityReason:    "CRITICAL: Art. 30 is a fundamental obligation; absence results in inability to demonstrate compliance to supervisory authorities",
		})
	}

	if !hasRetentionPolicy {
		evidence := []EvidenceItem{{
			ID:          "ev-gdpr-retention-absence",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No active retention policy found. Personal data storage must have defined limits.",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"policy_type": "retention", "active_policies": len(advCtx.Policies)},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-gdpr-retention",
			Severity:          "HIGH",
			Category:          "gdpr",
			Title:             "No data retention policy defined",
			Description:       "GDPR requires data minimization and storage limitation",
			Action:            "Define retention policies for personal data categories",
			Regulation:        "GDPR Art. 5(1)(e)",
			RegulationArticle: "GDPR Art. 5(1)(e) - Storage limitation: kept no longer than necessary",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "No retention policy exists while personal data is being stored without defined storage limits",
			SeverityReason:    "HIGH: Storage limitation is a core GDPR principle; indefinite retention of personal data violates Art. 5(1)(e)",
		})
	}

	if !hasAccessPolicy {
		evidence := []EvidenceItem{{
			ID:          "ev-gdpr-access-absence",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No active access control policy found. Personal data access must be restricted.",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"policy_type": "access", "active_policies": len(advCtx.Policies)},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-gdpr-access",
			Severity:          "HIGH",
			Category:          "gdpr",
			Title:             "No access control policies defined",
			Description:       "Personal data should have appropriate access restrictions",
			Action:            "Create access policies to limit who can view personal data",
			Regulation:        "GDPR Art. 25, Art. 32",
			RegulationArticle: "GDPR Art. 32(1)(b) - Ensure ongoing confidentiality of processing systems",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "Personal data is accessible without defined access control policies",
			SeverityReason:    "HIGH: Lack of access controls exposes personal data to unauthorized access, violating security of processing requirements",
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
	var evidence []EvidenceItem
	var assets []AffectedAsset
	for _, v := range advCtx.RetentionViolations {
		if v.DaysOverdue > 0 {
			overdueCount++
			if len(evidence) < 5 {
				evidence = append(evidence, EvidenceItem{
					ID:          "ev-ret-" + v.ID,
					Type:        "retention_violation",
					Source:      "retention_engine",
					Description: "Dataset " + v.DatasetID + " is " + strconv.Itoa(v.DaysOverdue) + " days past retention limit",
					ResourceID:  v.DatasetID,
					ResourceRef: "retention_violations/" + v.ID,
					DetectedAt:  v.CreatedAt,
					Metadata:    map[string]any{"days_overdue": v.DaysOverdue, "violation_type": v.ViolationType},
				})
				assets = append(assets, AffectedAsset{ID: v.DatasetID, Name: v.DatasetID, Type: "dataset"})
			}
		}
	}

	if overdueCount > 0 {
		recs = append(recs, Recommendation{
			ID:                "rec-retention-overdue",
			Severity:          "HIGH",
			Category:          "retention",
			Title:             "Data retention violations detected",
			Description:       strconv.Itoa(overdueCount) + " datasets exceed their retention period",
			Action:            "Review and remediate retention violations - archive or delete overdue data",
			Regulation:        "GDPR Art. 17, CCPA 1798.105",
			RegulationArticle: "GDPR Art. 17(1) - Right to erasure when data no longer necessary",
			AffectedCount:     overdueCount,
			Evidence:          evidence,
			AffectedAssets:    assets,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   strconv.Itoa(overdueCount) + " datasets found exceeding their defined retention period",
			SeverityReason:    "HIGH: Active retention violations mean personal data is stored beyond lawful limits, creating legal liability",
		})
	}

	return recs
}

func checkStaleDataSources(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	staleCount := 0
	var evidence []EvidenceItem
	var assets []AffectedAsset
	for _, ds := range advCtx.DataSources {
		if ds.LastScan == nil || ds.LastScan.IsZero() {
			staleCount++
			if len(evidence) < 5 {
				evidence = append(evidence, EvidenceItem{
					ID:          "ev-stale-" + ds.ID,
					Type:        "unscanned_source",
					Source:      "datasource_registry",
					Description: "Data source '" + ds.Name + "' (type: " + ds.Type + ") has never been scanned for sensitive data",
					ResourceID:  ds.ID,
					ResourceRef: "datasources/" + ds.ID,
					DetectedAt:  ds.CreatedAt,
					Metadata:    map[string]any{"source_type": ds.Type, "status": ds.Status},
				})
				assets = append(assets, AffectedAsset{ID: ds.ID, Name: ds.Name, Type: "datasource"})
			}
		}
	}

	if staleCount > 0 {
		recs = append(recs, Recommendation{
			ID:                "rec-stale-sources",
			Severity:          "MEDIUM",
			Category:          "governance",
			Title:             "Data sources not scanned",
			Description:       strconv.Itoa(staleCount) + " data sources have never been scanned for sensitive data",
			Action:            "Schedule classification scans for all data sources",
			AffectedCount:     staleCount,
			Evidence:          evidence,
			AffectedAssets:    assets,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   strconv.Itoa(staleCount) + " registered data sources have never undergone classification scanning",
			SeverityReason:    "MEDIUM: Unscanned sources may contain sensitive data without governance, creating unknown risk exposure",
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
		severityReason := "LOW: A small fraction of datasets lack labels; minimal compliance risk"
		if percentage > 50 {
			severity = "HIGH"
			severityReason = "HIGH: More than 50% of datasets lack sensitivity labels, indicating systemic governance gap"
		} else if percentage > 25 {
			severity = "MEDIUM"
			severityReason = "MEDIUM: Over 25% of datasets unlabeled; data handling decisions may not reflect actual sensitivity"
		}

		evidence := []EvidenceItem{{
			ID:          "ev-labels-gap",
			Type:        "coverage_gap",
			Source:      "label_engine",
			Description: strconv.Itoa(unlabeled) + " of " + strconv.Itoa(advCtx.TotalDatasets) + " datasets (" + strconv.FormatFloat(percentage, 'f', 1, 64) + "%) lack sensitivity labels",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"total_datasets": advCtx.TotalDatasets, "labeled": advCtx.LabeledDatasets, "unlabeled": unlabeled, "percentage_unlabeled": percentage},
		}}

		recs = append(recs, Recommendation{
			ID:                "rec-missing-labels",
			Severity:          severity,
			Category:          "governance",
			Title:             "Datasets missing sensitivity labels",
			Description:       strconv.Itoa(unlabeled) + " datasets lack sensitivity labels",
			Action:            "Assign sensitivity labels to all datasets containing classified data",
			AffectedCount:     unlabeled,
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   strconv.FormatFloat(percentage, 'f', 1, 64) + "% of datasets have no sensitivity label assigned",
			SeverityReason:    severityReason,
		})
	}

	return recs
}

func checkHighSensitivityData(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	totalHighSens := 0
	var evidence []EvidenceItem
	var assets []AffectedAsset
	seenAssets := make(map[string]bool)

	for _, c := range advCtx.Classifications {
		if HighSensitivityTypes[c.EntityType] {
			totalHighSens++
			if len(evidence) < 5 {
				evidence = append(evidence, EvidenceItem{
					ID:          "ev-highsens-" + c.ID,
					Type:        "classification_result",
					Source:      "classification_engine",
					Description: "High-sensitivity " + c.EntityType + " found in " + c.DatasetID,
					ResourceID:  c.DatasetID,
					ResourceRef: "classifications/" + c.ID,
					DetectedAt:  c.CreatedAt,
					Metadata:    map[string]any{"entity_type": c.EntityType, "confidence": c.Confidence},
				})
			}
			if !seenAssets[c.SourceID] {
				seenAssets[c.SourceID] = true
				assets = append(assets, AffectedAsset{ID: c.SourceID, Name: c.DatasetID, Type: "dataset"})
			}
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
			ID:                "rec-high-sens-policy",
			Severity:          "CRITICAL",
			Category:          "security",
			Title:             "High-sensitivity data without protection policy",
			Description:       strconv.Itoa(totalHighSens) + " instances of high-sensitivity data found without explicit protection policies",
			Action:            "Create strict access and redaction policies for high-sensitivity data types",
			Regulation:        "HIPAA, PCI-DSS, GDPR Art. 9",
			RegulationArticle: "GDPR Art. 9 - Processing of special categories of personal data",
			AffectedCount:     totalHighSens,
			Evidence:          evidence,
			AffectedAssets:    assets,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   strconv.Itoa(totalHighSens) + " high-sensitivity records (SSN, credit card, health data) across " + strconv.Itoa(len(assets)) + " datasets without dedicated protection policies",
			SeverityReason:    "CRITICAL: Unprotected high-sensitivity data (financial, health) creates severe regulatory and breach liability",
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
		evidence := []EvidenceItem{{
			ID:          "ev-ai-no-policy",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No AI governance policy defined while " + strconv.Itoa(len(advCtx.Classifications)) + " classified data records exist",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"classification_count": len(advCtx.Classifications), "ai_policy_count": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-ai-governance",
			Severity:          "MEDIUM",
			Category:          "ai",
			Title:             "No AI governance policies defined",
			Description:       "Sensitive data exists but no policies govern its use in AI/LLM systems",
			Action:            "Create AI governance policies to control what data can be sent to LLMs",
			Regulation:        "EU AI Act",
			RegulationArticle: "EU AI Act Art. 10 - Data and data governance for high-risk AI systems",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   strconv.Itoa(len(advCtx.Classifications)) + " classified data records exist without any AI usage governance policy",
			SeverityReason:    "MEDIUM: Sensitive data could be inadvertently sent to AI/LLM systems without controls",
		})
	}

	return recs
}

// checkPDPBCompliance checks India's Digital Personal Data Protection Act 2023 requirements
func checkPDPBCompliance(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	hasPII := false
	piiCount := 0
	for _, c := range advCtx.Classifications {
		if PIITypes[c.EntityType] {
			hasPII = true
			piiCount++
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
		evidence := []EvidenceItem{{
			ID:          "ev-pdpb-consent-absence",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No consent management policy found while " + strconv.Itoa(piiCount) + " personal data records are processed",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"pii_count": piiCount, "consent_policies": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-pdpb-consent",
			Severity:          "CRITICAL",
			Category:          "pdpb",
			Title:             "Missing consent management for DPDP Act",
			Description:       "India's DPDP Act requires explicit consent before processing personal data",
			Action:            "Implement consent management system with clear purpose specification",
			Regulation:        "DPDP Act 2023 Section 6",
			RegulationArticle: "DPDP Act 2023 S.6(1) - Processing based on consent given for specified purpose",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "Personal data being processed without any consent management framework in place",
			SeverityReason:    "CRITICAL: Processing without consent is a fundamental violation of DPDP Act; penalties up to INR 250 crore",
		})
	}

	if !hasDataLocalizationPolicy {
		evidence := []EvidenceItem{{
			ID:          "ev-pdpb-localization-absence",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No data localization policy defined for critical personal data",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"localization_policies": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-pdpb-localization",
			Severity:          "HIGH",
			Category:          "pdpb",
			Title:             "No data localization policy defined",
			Description:       "Critical personal data must be stored and processed within India under DPDP Act",
			Action:            "Define data localization policies for critical personal data categories",
			Regulation:        "DPDP Act 2023 Section 16",
			RegulationArticle: "DPDP Act 2023 S.16 - Transfer of personal data outside India restricted by Central Government",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "No data localization controls exist to prevent unauthorized cross-border transfers",
			SeverityReason:    "HIGH: Cross-border transfer restrictions may be imposed by government notification; non-compliance risks penalties",
		})
	}

	if !hasChildrenDataPolicy {
		evidence := []EvidenceItem{{
			ID:          "ev-pdpb-children-absence",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No children's data protection policy defined",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"children_policies": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-pdpb-children",
			Severity:          "HIGH",
			Category:          "pdpb",
			Title:             "Missing children's data protection policy",
			Description:       "DPDP Act requires verifiable parental consent for processing data of persons under 18",
			Action:            "Implement age verification and parental consent mechanisms",
			Regulation:        "DPDP Act 2023 Section 9",
			RegulationArticle: "DPDP Act 2023 S.9 - Processing personal data of children requires verifiable consent of parent/guardian",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "No mechanisms exist to identify or protect children's personal data",
			SeverityReason:    "HIGH: Processing children's data without parental consent attracts enhanced penalties under DPDP Act",
		})
	}

	if !hasRetentionPolicy {
		evidence := []EvidenceItem{{
			ID:          "ev-pdpb-retention-absence",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No retention policy defined; DPDP Act requires erasure when purpose fulfilled",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"retention_policies": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-pdpb-retention",
			Severity:          "HIGH",
			Category:          "pdpb",
			Title:             "No data retention limits defined",
			Description:       "DPDP Act requires data to be erased when purpose is fulfilled or consent withdrawn",
			Action:            "Define retention policies with automatic deletion when purpose is complete",
			Regulation:        "DPDP Act 2023 Section 8(7)",
			RegulationArticle: "DPDP Act 2023 S.8(7) - Data Fiduciary shall erase personal data upon withdrawal of consent or specified purpose",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "Personal data stored indefinitely without defined retention limits or purpose expiry",
			SeverityReason:    "HIGH: Indefinite retention violates purpose limitation principle under DPDP Act",
		})
	}

	if len(advCtx.RoPA) == 0 {
		evidence := []EvidenceItem{{
			ID:          "ev-pdpb-dpo-absence",
			Type:        "absence_of_record",
			Source:      "ropa_registry",
			Description: "No processing activity records found; Significant Data Fiduciary obligations may apply",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"ropa_count": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-pdpb-dpo",
			Severity:          "CRITICAL",
			Category:          "pdpb",
			Title:             "Significant Data Fiduciary obligations not documented",
			Description:       "Significant Data Fiduciaries must appoint a DPO and conduct Data Protection Impact Assessments",
			Action:            "Document processing activities and appoint Data Protection Officer if applicable",
			Regulation:        "DPDP Act 2023 Section 10",
			RegulationArticle: "DPDP Act 2023 S.10 - Obligations of Significant Data Fiduciary including DPO appointment and DPIA",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "No documented processing activities; unable to determine Significant Data Fiduciary status",
			SeverityReason:    "CRITICAL: Failure to comply with SDF obligations attracts highest tier penalties",
		})
	}

	return recs
}

// checkUAEPDPLCompliance checks UAE Personal Data Protection Law requirements
func checkUAEPDPLCompliance(advCtx *AdvisorContext) []Recommendation {
	var recs []Recommendation

	hasPII := false
	piiCount := 0
	for _, c := range advCtx.Classifications {
		if PIITypes[c.EntityType] {
			hasPII = true
			piiCount++
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
		evidence := []EvidenceItem{{
			ID:          "ev-uae-lawful-absence",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No lawful basis documentation while " + strconv.Itoa(piiCount) + " personal data records are processed",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"pii_count": piiCount, "lawful_basis_policies": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-uae-lawful-basis",
			Severity:          "CRITICAL",
			Category:          "uae_pdpl",
			Title:             "Missing lawful basis documentation",
			Description:       "UAE PDPL requires documented lawful basis for all personal data processing",
			Action:            "Document lawful basis (consent, contract, legal obligation, vital interests, public interest, or legitimate interests)",
			Regulation:        "UAE PDPL Art. 4",
			RegulationArticle: "UAE Federal Decree-Law No. 45/2021, Art. 4 - Lawful basis for processing personal data",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "Personal data being processed without documented lawful basis as required by UAE PDPL",
			SeverityReason:    "CRITICAL: Processing without lawful basis is a fundamental violation; administrative penalties apply",
		})
	}

	if !hasCrossBorderPolicy {
		evidence := []EvidenceItem{{
			ID:          "ev-uae-crossborder-absence",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No cross-border transfer policy defined for personal data",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"cross_border_policies": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-uae-cross-border",
			Severity:          "HIGH",
			Category:          "uae_pdpl",
			Title:             "No cross-border transfer policy defined",
			Description:       "UAE PDPL restricts transfer of personal data outside UAE without adequate protection",
			Action:            "Define cross-border transfer policies ensuring adequate protection level",
			Regulation:        "UAE PDPL Art. 22",
			RegulationArticle: "UAE Federal Decree-Law No. 45/2021, Art. 22 - Cross-border transfer conditions",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "No controls on cross-border data transfers; data may leave UAE without adequate protection",
			SeverityReason:    "HIGH: Unrestricted cross-border transfers expose organization to regulatory action",
		})
	}

	hasHighSensitivity := false
	highSensCount := 0
	for _, c := range advCtx.Classifications {
		if HighSensitivityTypes[c.EntityType] {
			hasHighSensitivity = true
			highSensCount++
		}
	}

	if hasHighSensitivity && !hasSpecialCategoryPolicy {
		evidence := []EvidenceItem{{
			ID:          "ev-uae-special-absence",
			Type:        "classification_result",
			Source:      "classification_engine",
			Description: strconv.Itoa(highSensCount) + " special category data records found without explicit protection policy",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"special_category_count": highSensCount, "special_policies": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-uae-special-category",
			Severity:          "CRITICAL",
			Category:          "uae_pdpl",
			Title:             "Special category data without explicit protection",
			Description:       "UAE PDPL requires explicit consent and additional safeguards for sensitive personal data",
			Action:            "Implement explicit consent and enhanced protection for health, biometric, genetic, and other sensitive data",
			Regulation:        "UAE PDPL Art. 7",
			RegulationArticle: "UAE Federal Decree-Law No. 45/2021, Art. 7 - Processing sensitive personal data",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   strconv.Itoa(highSensCount) + " sensitive data records require explicit consent and enhanced safeguards",
			SeverityReason:    "CRITICAL: Processing sensitive data without explicit consent is a serious violation with enhanced penalties",
		})
	}

	if !hasRetentionPolicy {
		evidence := []EvidenceItem{{
			ID:          "ev-uae-retention-absence",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No retention policy defined; data may be stored beyond necessary period",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"retention_policies": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-uae-retention",
			Severity:          "HIGH",
			Category:          "uae_pdpl",
			Title:             "No data retention policy for UAE PDPL",
			Description:       "UAE PDPL requires data to be kept only as long as necessary for the specified purpose",
			Action:            "Define retention periods aligned with processing purposes",
			Regulation:        "UAE PDPL Art. 5",
			RegulationArticle: "UAE Federal Decree-Law No. 45/2021, Art. 5 - Data minimization and storage limitation",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "No data retention limits defined; personal data stored without purpose-aligned time limits",
			SeverityReason:    "HIGH: Storage beyond necessity violates data minimization principle under UAE PDPL",
		})
	}

	if !hasAccessPolicy {
		evidence := []EvidenceItem{{
			ID:          "ev-uae-rights-absence",
			Type:        "policy_absence",
			Source:      "policy_engine",
			Description: "No access control policy to support data subject rights fulfillment",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"access_policies": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-uae-data-subject-rights",
			Severity:          "HIGH",
			Category:          "uae_pdpl",
			Title:             "Data subject rights not fully implemented",
			Description:       "UAE PDPL grants rights to access, rectification, erasure, and data portability",
			Action:            "Implement mechanisms for data subject access requests and rights fulfillment",
			Regulation:        "UAE PDPL Art. 13-18",
			RegulationArticle: "UAE Federal Decree-Law No. 45/2021, Art. 13-18 - Rights of Data Subject",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "Data subject rights mechanisms incomplete; cannot fulfill access, rectification, or erasure requests",
			SeverityReason:    "HIGH: Failure to implement data subject rights mechanisms violates core PDPL obligations",
		})
	}

	if len(advCtx.RoPA) == 0 {
		evidence := []EvidenceItem{{
			ID:          "ev-uae-records-absence",
			Type:        "absence_of_record",
			Source:      "ropa_registry",
			Description: "No records of processing activities maintained",
			DetectedAt:  advCtx.AssessmentTime,
			Metadata:    map[string]any{"ropa_count": 0},
		}}
		recs = append(recs, Recommendation{
			ID:                "rec-uae-records",
			Severity:          "HIGH",
			Category:          "uae_pdpl",
			Title:             "Missing records of processing activities",
			Description:       "UAE PDPL requires controllers to maintain records of processing activities",
			Action:            "Create and maintain records of all personal data processing activities",
			Regulation:        "UAE PDPL Art. 8",
			RegulationArticle: "UAE Federal Decree-Law No. 45/2021, Art. 8 - Record of processing activities",
			Evidence:          evidence,
			DetectedAt:        advCtx.AssessmentTime,
			EvidenceSummary:   "Zero processing activity records maintained; unable to demonstrate accountability to UAE Data Office",
			SeverityReason:    "HIGH: Record-keeping is a fundamental accountability obligation under UAE PDPL",
		})
	}

	return recs
}
