package governance

import (
	"context"
	"testing"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
)

func TestPolicyByNameBalanced(t *testing.T) {
	t.Parallel()

	policy, err := PolicyByName("balanced")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if policy.PreferredSelector != StrategyNSGA3 {
		t.Fatalf("expected nsga3 preferred selector, got %s", policy.PreferredSelector)
	}

	if policy.ObjectiveWeights.Energy != 1.0 {
		t.Fatalf("expected energy weight 1.0, got %.2f", policy.ObjectiveWeights.Energy)
	}
}

func TestStaticSupervisorCountsAcceptedAndRejectedBids(t *testing.T) {
	t.Parallel()

	supervisor, err := NewStaticSupervisor(Policy{
		ID:                "balanced-efficiency",
		Source:            "test",
		Reason:            "Test policy.",
		PreferredSelector: StrategyNSGA3,
		ObjectiveWeights:  auctioncore.DefaultObjectiveWeights(),
		TaskProfile:       auctioncore.TaskProfileBalanced,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	decision, err := supervisor.Decide(context.Background(), auctioncore.DefaultTask("pod-a", 0.25, 0.40), []auctioncore.Bid{
		{NodeID: "node-a", Accepted: true},
		{NodeID: "node-b", Accepted: false},
		{NodeID: "node-c", Accepted: true},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if decision.AcceptedCandidates != 2 {
		t.Fatalf("expected 2 accepted candidates, got %d", decision.AcceptedCandidates)
	}

	if decision.RejectedCandidates != 1 {
		t.Fatalf("expected 1 rejected candidate, got %d", decision.RejectedCandidates)
	}
}
