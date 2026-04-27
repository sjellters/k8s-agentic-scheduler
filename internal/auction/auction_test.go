package auction

import "testing"

func TestEvaluateBidRejectsInsufficientResources(t *testing.T) {
	t.Parallel()

	bid := EvaluateBid("node-a", Resources{CPU: 0.30, RAM: 0.40}, Task{ID: "pod-a", CPUReqNorm: 0.31, RAMReqNorm: 0.20})

	if bid.Accepted {
		t.Fatalf("expected bid rejection")
	}

	if bid.CPUFragmentation != 0 || bid.RAMFragmentation != 0 {
		t.Fatalf("expected zero fragmentation for rejected bid, got CPU %.2f RAM %.2f", bid.CPUFragmentation, bid.RAMFragmentation)
	}
}

func TestEvaluateBidCalculatesFragmentation(t *testing.T) {
	t.Parallel()

	bid := EvaluateBid("node-b", Resources{CPU: 0.80, RAM: 0.90}, Task{ID: "pod-b", CPUReqNorm: 0.25, RAMReqNorm: 0.40})

	if !bid.Accepted {
		t.Fatalf("expected accepted bid")
	}

	if bid.CPUFragmentation != 0.55 {
		t.Fatalf("expected CPU fragmentation 0.55, got %.2f", bid.CPUFragmentation)
	}

	if bid.RAMFragmentation != 0.50 {
		t.Fatalf("expected RAM fragmentation 0.50, got %.2f", bid.RAMFragmentation)
	}
}

func TestSelectWinnerPrefersHighestAcceptedScore(t *testing.T) {
	t.Parallel()

	winner, ok := SelectWinner([]Bid{
		{NodeID: "node-reject", Accepted: false},
		{NodeID: "node-fit", Accepted: true, CPUFragmentation: 0.00, RAMFragmentation: 0.00},
		{NodeID: "node-best", Accepted: true, CPUFragmentation: 0.45, RAMFragmentation: 0.25},
	})

	if !ok {
		t.Fatalf("expected winner")
	}

	if winner.NodeID != "node-best" {
		t.Fatalf("expected node-best, got %s", winner.NodeID)
	}
}
