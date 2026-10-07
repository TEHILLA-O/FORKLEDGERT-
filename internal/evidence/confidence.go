package evidence

import "fmt"

// Level is the public confidence category. Percentages stay inside the engine
// until they can be explained without pretending that 84 and 85 are different.
type Level string

const (
	LevelNone     Level = "NONE"
	LevelLow      Level = "LOW"
	LevelModerate Level = "MODERATE"
	LevelHigh     Level = "HIGH"
	LevelVeryHigh Level = "VERY_HIGH"
)

// Assessment is boring math on purpose. If we cannot explain the score, we should not show it.
type Assessment struct {
	Level    Level  `json:"level"`
	Score    int    `json:"score"`
	Basis    string `json:"basis"`
	Evidence []Item `json:"evidence"`
}

// FromWeights sums evidence weights, clamps to 0..100, and maps onto a category.
// The mapping is coarse because fake precision is worse than looking unsophisticated.
func FromWeights(basis string, items []Item) Assessment {
	total := 0
	for _, item := range items {
		total += item.Weight
	}
	if total < 0 {
		total = 0
	}
	if total > 100 {
		total = 100
	}
	return Assessment{
		Level:    LevelFromScore(total),
		Score:    total,
		Basis:    basis,
		Evidence: items,
	}
}

func LevelFromScore(score int) Level {
	switch {
	case score >= 90:
		return LevelVeryHigh
	case score >= 70:
		return LevelHigh
	case score >= 40:
		return LevelModerate
	case score > 0:
		return LevelLow
	default:
		return LevelNone
	}
}

func (a Assessment) Explain() string {
	if a.Basis == "" {
		return fmt.Sprintf("%s (%d)", a.Level, a.Score)
	}
	return fmt.Sprintf("%s (%d). %s", a.Level, a.Score, a.Basis)
}

// Strong enough to act on without asking the operator to pick among candidates.
func (a Assessment) Actionable() bool {
	return a.Level == LevelHigh || a.Level == LevelVeryHigh
}
