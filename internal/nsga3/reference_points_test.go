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
		{NodeID: "node-a", Objectives: []float64{0.40, 0.20, 0.70, 0.65}},
		{NodeID: "node-b", Objectives: []float64{0.35, 0.30, 0.75, 0.55}},
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

	if len(preparation.Fronts) == 0 {
		t.Fatalf("expected at least one nondominated front")
	}
}

func TestOptimizerSelectProducesWinnerAndTraceForFourObjectives(t *testing.T) {
	t.Parallel()

	optimizer, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	selection, err := optimizer.Select([]Candidate{
		{NodeID: "node-cpu", Objectives: []float64{0.95, 0.10, 0.10, 0.10}},
		{NodeID: "node-balanced", Objectives: []float64{0.70, 0.70, 0.70, 0.70}},
		{NodeID: "node-ram", Objectives: []float64{0.10, 0.95, 0.10, 0.10}},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !selection.HasWinner {
		t.Fatalf("expected a winner")
	}

	if selection.Winner.NodeID == "" {
		t.Fatalf("expected a concrete winner node id")
	}

	if selection.Preparation.SelectedCandidate == nil {
		t.Fatalf("expected selected candidate trace")
	}

	if len(selection.Preparation.ObjectiveNames) != 4 {
		t.Fatalf("expected 4 objective names, got %d", len(selection.Preparation.ObjectiveNames))
	}
}
