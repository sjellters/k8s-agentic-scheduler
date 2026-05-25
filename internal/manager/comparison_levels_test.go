package manager

import "testing"

func TestThesisComparisonLevelsFreezeDelivery6Contract(t *testing.T) {
	t.Parallel()

	levels := ThesisComparisonLevels()
	if len(levels) != 3 {
		t.Fatalf("expected 3 thesis comparison levels, got %d", len(levels))
	}

	if levels[0].Level != ComparisonLevelBaseline || levels[0].RequiresSupervisor {
		t.Fatalf("expected baseline level without supervisor, got %+v", levels[0])
	}

	if levels[1].Level != ComparisonLevelOptimizerOnly || levels[1].SelectorStrategy != "nsga3" || levels[1].RequiresSupervisor {
		t.Fatalf("expected optimizer-without-policy-supervision nsga3 level without supervisor, got %+v", levels[1])
	}

	if levels[2].Level != ComparisonLevelOptimizerGoverned || !levels[2].RequiresSupervisor || levels[2].PolicyName != "balanced" {
		t.Fatalf("expected optimizer-with-policy-supervision level with balanced policy, got %+v", levels[2])
	}
}
