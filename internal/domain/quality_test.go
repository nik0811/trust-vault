package domain

import (
	"testing"
	"time"
)

func TestAssessQuality(t *testing.T) {
	tests := []struct {
		name          string
		stats         DatasetStats
		minOverall    float64
		maxOverall    float64
		expectIssues  bool
	}{
		{
			name: "perfect data",
			stats: DatasetStats{
				TotalRows:        1000,
				NullCounts:       map[string]int{"col1": 0, "col2": 0},
				UniqueRatios:     map[string]float64{"col1": 1.0},
				LastUpdated:      time.Now(),
				DuplicateRows:    0,
				ColumnTypes:      map[string]string{},
				ColumnSamples:    map[string][]string{},
				FormatViolations: map[string]int{},
				OutlierCounts:    map[string]int{},
			},
			minOverall:   0.85,
			maxOverall:   1.0,
			expectIssues: false,
		},
		{
			name: "high null rate",
			stats: DatasetStats{
				TotalRows:        1000,
				NullCounts:       map[string]int{"col1": 500, "col2": 200}, // 50% and 20% nulls
				UniqueRatios:     map[string]float64{"col1": 0.5},
				LastUpdated:      time.Now(),
				DuplicateRows:    0,
				ColumnTypes:      map[string]string{},
				ColumnSamples:    map[string][]string{},
				FormatViolations: map[string]int{},
				OutlierCounts:    map[string]int{},
			},
			minOverall:   0.5,
			maxOverall:   0.95,
			expectIssues: true,
		},
		{
			name: "stale data",
			stats: DatasetStats{
				TotalRows:        1000,
				NullCounts:       map[string]int{"col1": 0},
				UniqueRatios:     map[string]float64{"col1": 1.0},
				LastUpdated:      time.Now().AddDate(0, -2, 0), // 2 months ago
				DuplicateRows:    0,
				ColumnTypes:      map[string]string{},
				ColumnSamples:    map[string][]string{},
				FormatViolations: map[string]int{},
				OutlierCounts:    map[string]int{},
			},
			minOverall:   0.6,
			maxOverall:   0.95, // Updated: with real accuracy/consistency, overall is higher
			expectIssues: true,
		},
		{
			name: "many duplicates",
			stats: DatasetStats{
				TotalRows:        1000,
				NullCounts:       map[string]int{"col1": 0},
				UniqueRatios:     map[string]float64{"col1": 0.5},
				LastUpdated:      time.Now(),
				DuplicateRows:    200, // 20% duplicates
				ColumnTypes:      map[string]string{},
				ColumnSamples:    map[string][]string{},
				FormatViolations: map[string]int{},
				OutlierCounts:    map[string]int{},
			},
			minOverall:   0.7,
			maxOverall:   1.0, // Updated: with real accuracy/consistency, overall is higher
			expectIssues: true,
		},
		{
			name: "empty dataset",
			stats: DatasetStats{
				TotalRows:        0,
				NullCounts:       map[string]int{},
				UniqueRatios:     map[string]float64{},
				LastUpdated:      time.Now(),
				DuplicateRows:    0,
				ColumnTypes:      map[string]string{},
				ColumnSamples:    map[string][]string{},
				FormatViolations: map[string]int{},
				OutlierCounts:    map[string]int{},
			},
			minOverall:   0.8,
			maxOverall:   1.0,
			expectIssues: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AssessQuality(tt.stats)

			if result.Overall < tt.minOverall || result.Overall > tt.maxOverall {
				t.Errorf("Overall = %.2f, want between %.2f and %.2f", result.Overall, tt.minOverall, tt.maxOverall)
			}

			hasIssues := len(result.Issues) > 0
			if hasIssues != tt.expectIssues {
				t.Errorf("Has issues = %v, want %v. Issues: %v", hasIssues, tt.expectIssues, result.Issues)
			}

			// Verify all dimensions are between 0 and 1
			dimensions := []float64{result.Completeness, result.Accuracy, result.Consistency, result.Timeliness, result.Uniqueness}
			for i, d := range dimensions {
				if d < 0 || d > 1 {
					t.Errorf("Dimension %d = %.2f, should be between 0 and 1", i, d)
				}
			}
		})
	}
}

func TestDetectROT(t *testing.T) {
	tests := []struct {
		name             string
		stats            DatasetStats
		lastAccess       time.Time
		sizeBytes        int64
		expectedCategory string
		expectROT        bool
	}{
		{
			name: "fresh active data",
			stats: DatasetStats{
				TotalRows:     1000,
				DuplicateRows: 10,
			},
			lastAccess:       time.Now().Add(-24 * time.Hour), // Yesterday
			sizeBytes:        1024 * 1024,                     // 1MB
			expectedCategory: "",
			expectROT:        false,
		},
		{
			name: "obsolete - not accessed",
			stats: DatasetStats{
				TotalRows:     1000,
				DuplicateRows: 0,
			},
			lastAccess:       time.Now().AddDate(0, -7, 0), // 7 months ago
			sizeBytes:        1024 * 1024,
			expectedCategory: "obsolete",
			expectROT:        true,
		},
		{
			name: "trivial - empty",
			stats: DatasetStats{
				TotalRows:     0,
				DuplicateRows: 0,
			},
			lastAccess:       time.Now(),
			sizeBytes:        100, // Very small
			expectedCategory: "trivial",
			expectROT:        true,
		},
		{
			name: "redundant - high duplicates",
			stats: DatasetStats{
				TotalRows:     1000,
				DuplicateRows: 600, // 60% duplicates
			},
			lastAccess:       time.Now(),
			sizeBytes:        1024 * 1024,
			expectedCategory: "redundant",
			expectROT:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category, score, reason := DetectROT(tt.stats, tt.lastAccess, tt.sizeBytes)

			isROT := category != ""
			if isROT != tt.expectROT {
				t.Errorf("Is ROT = %v, want %v", isROT, tt.expectROT)
			}

			if tt.expectROT {
				if category != tt.expectedCategory {
					t.Errorf("Category = %s, want %s", category, tt.expectedCategory)
				}
				if score <= 0 {
					t.Errorf("Score should be > 0 for ROT data, got %.2f", score)
				}
				if reason == "" {
					t.Error("Reason should not be empty for ROT data")
				}
			}
		})
	}
}
