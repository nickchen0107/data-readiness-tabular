package assessment

import (
	"errors"
	"math"
)

// Weights holds the configurable weights for each indicator.
type Weights struct {
	RowCompleteness    float64 `json:"row_completeness"`
	ColumnCompleteness float64 `json:"column_completeness"`
	FormatConsistency  float64 `json:"format_consistency"`
	DuplicateSimilar   float64 `json:"duplicate_similar"`
	TableStructure     float64 `json:"table_structure"`
	AIQueryReadiness   float64 `json:"ai_query_readiness"`
}

// DefaultWeights returns the default indicator weights.
func DefaultWeights() Weights {
	return Weights{
		RowCompleteness:    0.25,
		ColumnCompleteness: 0.25,
		FormatConsistency:  0.15,
		DuplicateSimilar:   0.10,
		TableStructure:     0.10,
		AIQueryReadiness:   0.15,
	}
}

// Sum returns the sum of all weights.
func (w Weights) Sum() float64 {
	return w.RowCompleteness +
		w.ColumnCompleteness +
		w.FormatConsistency +
		w.DuplicateSimilar +
		w.TableStructure +
		w.AIQueryReadiness
}

// IsValid checks that the weights sum to 1.0 (±0.001).
func (w Weights) IsValid() bool {
	return math.Abs(w.Sum()-1.0) < 0.001
}

// IndicatorScores holds the computed scores for all 6 indicators.
type IndicatorScores struct {
	RowCompleteness    float64
	ColumnCompleteness float64
	FormatConsistency  float64
	DuplicateSimilar   float64
	TableStructure     float64
	AIQueryReadiness   float64
}

// CalculateTotalScore computes the weighted sum of indicator scores, applies a
// row readiness adjustment (60% indicators + 40% high-readiness ratio), rounds
// to 1 decimal, and determines the grade.
// The highReadinessRatio is the fraction of rows classified as "High" readiness
// (≥80% non-empty cells). This ensures that datasets with very few high-quality
// rows receive appropriately low overall scores.
// Returns: total score, grade string ("ready"/"conditional"/"not_ready"), error.
func CalculateTotalScore(indicators IndicatorScores, weights Weights) (float64, string, error) {
	if !weights.IsValid() {
		return 0, "", errors.New("weights must sum to 1.0 (±0.001)")
	}

	total := indicators.RowCompleteness*weights.RowCompleteness +
		indicators.ColumnCompleteness*weights.ColumnCompleteness +
		indicators.FormatConsistency*weights.FormatConsistency +
		indicators.DuplicateSimilar*weights.DuplicateSimilar +
		indicators.TableStructure*weights.TableStructure +
		indicators.AIQueryReadiness*weights.AIQueryReadiness

	// Round to 1 decimal place
	total = math.Round(total*10) / 10

	// Determine grade
	var grade string
	switch {
	case total >= 80:
		grade = "ready"
	case total >= 60:
		grade = "conditional"
	default:
		grade = "not_ready"
	}

	return total, grade, nil
}

// CalculateTotalScoreWithReadiness computes the final score with row readiness adjustment.
// Formula: finalScore = indicatorScore × 0.6 + highReadinessRatio × 100 × 0.4
// This ensures datasets with very few high-quality rows get appropriately low scores.
func CalculateTotalScoreWithReadiness(indicators IndicatorScores, weights Weights, rowDist RowDistribution) (float64, string, error) {
	if !weights.IsValid() {
		return 0, "", errors.New("weights must sum to 1.0 (±0.001)")
	}

	indicatorScore := indicators.RowCompleteness*weights.RowCompleteness +
		indicators.ColumnCompleteness*weights.ColumnCompleteness +
		indicators.FormatConsistency*weights.FormatConsistency +
		indicators.DuplicateSimilar*weights.DuplicateSimilar +
		indicators.TableStructure*weights.TableStructure +
		indicators.AIQueryReadiness*weights.AIQueryReadiness

	// Calculate high readiness ratio
	totalRows := rowDist.High + rowDist.Medium + rowDist.Low
	highRatio := 0.0
	if totalRows > 0 {
		highRatio = float64(rowDist.High) / float64(totalRows)
	}

	// Final score: 60% from indicators + 40% from high readiness ratio
	total := indicatorScore*0.6 + highRatio*100*0.4

	// Round to 1 decimal place
	total = math.Round(total*10) / 10

	// Determine grade
	var grade string
	switch {
	case total >= 80:
		grade = "ready"
	case total >= 60:
		grade = "conditional"
	default:
		grade = "not_ready"
	}

	return total, grade, nil
}
