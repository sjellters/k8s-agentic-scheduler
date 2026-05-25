package manager

import "github.com/diego/k8s-agentic-scheduler/internal/governance"

type ComparisonLevel string

const (
	ComparisonLevelBaseline          ComparisonLevel = "baseline"
	ComparisonLevelOptimizerOnly     ComparisonLevel = "optimizer-without-policy-supervision"
	ComparisonLevelOptimizerGoverned ComparisonLevel = "optimizer-with-policy-supervision"
)

type ComparisonSpec struct {
	Level              ComparisonLevel
	RequiresSupervisor bool
	SelectorStrategy   string
	PolicyName         string
}

func ThesisComparisonLevels() []ComparisonSpec {
	return []ComparisonSpec{
		{
			Level:              ComparisonLevelBaseline,
			RequiresSupervisor: false,
			SelectorStrategy:   governance.StrategyBaseline,
		},
		{
			Level:              ComparisonLevelOptimizerOnly,
			RequiresSupervisor: false,
			SelectorStrategy:   governance.StrategyNSGA3,
		},
		{
			Level:              ComparisonLevelOptimizerGoverned,
			RequiresSupervisor: true,
			SelectorStrategy:   governance.StrategyNSGA3,
			PolicyName:         "balanced",
		},
	}
}
