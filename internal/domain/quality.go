package domain

import (
	"fmt"
	"math"
	"regexp"
	"time"
)

type QualityAssessment struct {
	DatasetID    string         `json:"dataset_id"`
	Overall      float64        `json:"overall"`
	Completeness float64        `json:"completeness"`
	Accuracy     float64        `json:"accuracy"`
	Consistency  float64        `json:"consistency"`
	Timeliness   float64        `json:"timeliness"`
	Uniqueness   float64        `json:"uniqueness"`
	Issues       []QualityIssue `json:"issues"`
	Trend        string         `json:"trend"`
}

type QualityIssue struct {
	Type     string `json:"type"`
	Column   string `json:"column,omitempty"`
	Severity string `json:"severity"`
	Count    int    `json:"count"`
	Message  string `json:"message"`
}

type DatasetStats struct {
	TotalRows       int
	NullCounts      map[string]int
	UniqueRatios    map[string]float64
	LastUpdated     time.Time
	DuplicateRows   int
	ColumnTypes     map[string]string
	ColumnSamples   map[string][]string
	FormatViolations map[string]int
	OutlierCounts   map[string]int
}

// Common format patterns for validation
var formatPatterns = map[string]*regexp.Regexp{
	"email":    regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
	"phone":    regexp.MustCompile(`^[\+]?[(]?[0-9]{3}[)]?[-\s\.]?[0-9]{3}[-\s\.]?[0-9]{4,6}$`),
	"date":     regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`),
	"datetime": regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`),
	"uuid":     regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`),
	"url":      regexp.MustCompile(`^https?://[^\s]+$`),
	"ipv4":     regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`),
	"zip":      regexp.MustCompile(`^\d{5}(-\d{4})?$`),
}

// AssessQuality calculates quality scores for a dataset
func AssessQuality(stats DatasetStats) *QualityAssessment {
	assessment := &QualityAssessment{
		Issues: []QualityIssue{},
	}

	// Completeness: 1 - (null_count / total_rows)
	totalNulls := 0
	totalFields := len(stats.NullCounts)
	for col, nulls := range stats.NullCounts {
		totalNulls += nulls
		nullRate := float64(nulls) / float64(stats.TotalRows)
		if nullRate > 0.1 {
			severity := "medium"
			if nullRate > 0.3 {
				severity = "high"
			} else if nullRate > 0.5 {
				severity = "critical"
			}
			assessment.Issues = append(assessment.Issues, QualityIssue{
				Type:     "missing_values",
				Column:   col,
				Severity: severity,
				Count:    nulls,
				Message:  formatPercent(nullRate*100) + "% null values detected",
			})
		}
	}
	if totalFields > 0 && stats.TotalRows > 0 {
		assessment.Completeness = 1 - (float64(totalNulls) / float64(stats.TotalRows*totalFields))
	} else {
		assessment.Completeness = 1.0
	}

	// Accuracy: Based on format validation and outlier detection
	assessment.Accuracy = calculateAccuracy(stats, assessment)

	// Consistency: Based on format consistency within columns
	assessment.Consistency = calculateConsistency(stats, assessment)

	// Timeliness: Based on last update
	hoursSinceUpdate := time.Since(stats.LastUpdated).Hours()
	if hoursSinceUpdate < 24 {
		assessment.Timeliness = 1.0
	} else if hoursSinceUpdate < 168 { // 1 week
		assessment.Timeliness = 0.8
	} else if hoursSinceUpdate < 720 { // 1 month
		assessment.Timeliness = 0.5
	} else {
		assessment.Timeliness = 0.2
		assessment.Issues = append(assessment.Issues, QualityIssue{
			Type:     "stale_data",
			Severity: "medium",
			Message:  "Data not updated in over 30 days",
		})
	}

	// Uniqueness: 1 - (duplicates / total)
	if stats.TotalRows > 0 {
		assessment.Uniqueness = 1 - (float64(stats.DuplicateRows) / float64(stats.TotalRows))
		if assessment.Uniqueness < 0.9 {
			severity := "medium"
			if assessment.Uniqueness < 0.7 {
				severity = "high"
			}
			assessment.Issues = append(assessment.Issues, QualityIssue{
				Type:     "duplicates",
				Severity: severity,
				Count:    stats.DuplicateRows,
				Message:  formatPercent((1-assessment.Uniqueness)*100) + "% duplicate rows detected",
			})
		}
	} else {
		assessment.Uniqueness = 1.0
	}

	// Overall: Weighted average
	assessment.Overall = (assessment.Completeness*0.25 +
		assessment.Accuracy*0.25 +
		assessment.Consistency*0.2 +
		assessment.Timeliness*0.15 +
		assessment.Uniqueness*0.15)

	// Round to 2 decimal places
	assessment.Overall = math.Round(assessment.Overall*100) / 100
	assessment.Completeness = math.Round(assessment.Completeness*100) / 100
	assessment.Accuracy = math.Round(assessment.Accuracy*100) / 100
	assessment.Consistency = math.Round(assessment.Consistency*100) / 100
	assessment.Timeliness = math.Round(assessment.Timeliness*100) / 100
	assessment.Uniqueness = math.Round(assessment.Uniqueness*100) / 100

	// Determine trend
	if assessment.Overall >= 0.9 {
		assessment.Trend = "excellent"
	} else if assessment.Overall >= 0.7 {
		assessment.Trend = "good"
	} else if assessment.Overall >= 0.5 {
		assessment.Trend = "needs_improvement"
	} else {
		assessment.Trend = "poor"
	}

	return assessment
}

// calculateAccuracy computes accuracy based on format validation and outliers
func calculateAccuracy(stats DatasetStats, assessment *QualityAssessment) float64 {
	if stats.TotalRows == 0 {
		return 1.0
	}

	totalViolations := 0
	totalChecked := 0

	// Check format violations
	for col, violations := range stats.FormatViolations {
		totalViolations += violations
		totalChecked += stats.TotalRows
		if violations > 0 {
			violationRate := float64(violations) / float64(stats.TotalRows)
			if violationRate > 0.05 {
				severity := "low"
				if violationRate > 0.1 {
					severity = "medium"
				} else if violationRate > 0.2 {
					severity = "high"
				}
				assessment.Issues = append(assessment.Issues, QualityIssue{
					Type:     "format_violation",
					Column:   col,
					Severity: severity,
					Count:    violations,
					Message:  formatPercent(violationRate*100) + "% values don't match expected format",
				})
			}
		}
	}

	// Check outliers
	for col, outliers := range stats.OutlierCounts {
		totalViolations += outliers
		totalChecked += stats.TotalRows
		if outliers > 0 {
			outlierRate := float64(outliers) / float64(stats.TotalRows)
			if outlierRate > 0.02 {
				assessment.Issues = append(assessment.Issues, QualityIssue{
					Type:     "outliers",
					Column:   col,
					Severity: "low",
					Count:    outliers,
					Message:  formatPercent(outlierRate*100) + "% potential outliers detected",
				})
			}
		}
	}

	// Validate samples against known patterns
	for col, samples := range stats.ColumnSamples {
		colType := stats.ColumnTypes[col]
		if pattern, ok := formatPatterns[colType]; ok {
			invalidCount := 0
			for _, sample := range samples {
				if sample != "" && !pattern.MatchString(sample) {
					invalidCount++
				}
			}
			if len(samples) > 0 {
				invalidRate := float64(invalidCount) / float64(len(samples))
				totalViolations += int(invalidRate * float64(stats.TotalRows))
				totalChecked += stats.TotalRows
			}
		}
	}

	if totalChecked == 0 {
		return 0.95 // Default when no validation possible
	}

	return 1 - (float64(totalViolations) / float64(totalChecked))
}

// calculateConsistency computes consistency based on format uniformity
func calculateConsistency(stats DatasetStats, assessment *QualityAssessment) float64 {
	if stats.TotalRows == 0 || len(stats.ColumnSamples) == 0 {
		return 1.0
	}

	totalConsistency := 0.0
	columnsChecked := 0

	for col, samples := range stats.ColumnSamples {
		if len(samples) < 2 {
			continue
		}

		// Check format consistency within column
		formatCounts := make(map[string]int)
		for _, sample := range samples {
			format := detectFormat(sample)
			formatCounts[format]++
		}

		// Find dominant format
		maxCount := 0
		for _, count := range formatCounts {
			if count > maxCount {
				maxCount = count
			}
		}

		columnConsistency := float64(maxCount) / float64(len(samples))
		totalConsistency += columnConsistency
		columnsChecked++

		if columnConsistency < 0.9 {
			assessment.Issues = append(assessment.Issues, QualityIssue{
				Type:     "inconsistent_format",
				Column:   col,
				Severity: "medium",
				Message:  formatPercent((1-columnConsistency)*100) + "% values have inconsistent format",
			})
		}
	}

	if columnsChecked == 0 {
		return 0.95
	}

	return totalConsistency / float64(columnsChecked)
}

// detectFormat identifies the format of a value
func detectFormat(value string) string {
	if value == "" {
		return "empty"
	}
	for name, pattern := range formatPatterns {
		if pattern.MatchString(value) {
			return name
		}
	}
	// Check if numeric
	if regexp.MustCompile(`^-?\d+\.?\d*$`).MatchString(value) {
		return "numeric"
	}
	return "text"
}

func formatPercent(p float64) string {
	return fmt.Sprintf("%.1f", math.Round(p*10)/10)
}

// DetectROT identifies Redundant, Obsolete, Trivial data
func DetectROT(stats DatasetStats, lastAccess time.Time, sizeBytes int64) (string, float64, string) {
	// Obsolete: Not accessed in 6+ months
	if time.Since(lastAccess).Hours() > 4320 { // 180 days
		return "obsolete", 0.9, "Data not accessed in over 6 months"
	}

	// Trivial: Very small or empty
	if stats.TotalRows == 0 || sizeBytes < 1024 {
		return "trivial", 0.95, "Empty or near-empty dataset"
	}

	// Redundant: High duplicate ratio
	if stats.TotalRows > 0 && float64(stats.DuplicateRows)/float64(stats.TotalRows) > 0.5 {
		return "redundant", 0.8, "Over 50% duplicate data"
	}

	return "", 0, ""
}
