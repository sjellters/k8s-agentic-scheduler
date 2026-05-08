package nsga3

import "testing"

func TestGenerateReferencePointsForTwoObjectives(t *testing.T) {
	t.Parallel()

	points, err := GenerateReferencePoints(2, 2)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(points) != 3 {
		t.Fatalf("expected 3 reference points, got %d", len(points))
	}
}

func TestOptimizerPrepareBuildsPreparation(t *testing.T) {
	t.Parallel()

	optimizer, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	preparation, err := optimizer.Prepare([]Candidate{
		{NodeID: "node-a", Objectives: []float64{0.40, 0.20}},
		{NodeID: "node-b", Objectives: []float64{0.35, 0.30}},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(preparation.ReferencePoints) == 0 {
		t.Fatalf("expected reference points to be generated")
	}

	if len(preparation.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(preparation.Candidates))
	}
}
