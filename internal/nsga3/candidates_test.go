package nsga3

import (
	"testing"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
)

func TestCandidateFromBidAcceptsAcceptedBids(t *testing.T) {
	t.Parallel()

	task := auctioncore.DefaultTask("pod-a", 0.25, 0.40)
	task.QoSSensitivity = 0.8

	candidate, ok := CandidateFromBid(task, auctioncore.Bid{
		NodeID:           "node-a",
		Accepted:         true,
		CPUFragmentation: 0.40,
		RAMFragmentation: 0.25,
	}, auctioncore.ProfileForNodeClass(auctioncore.NodeClassHighPerformance))
	if !ok {
		t.Fatalf("expected accepted bid to produce a candidate")
	}

	if candidate.NodeID != "node-a" {
		t.Fatalf("expected node-a, got %s", candidate.NodeID)
	}

	if len(candidate.Objectives) != 4 {
		t.Fatalf("expected 4 objectives, got %d", len(candidate.Objectives))
	}
}

func TestCandidatesFromBidsFiltersRejectedBids(t *testing.T) {
	t.Parallel()

	task := auctioncore.DefaultTask("pod-a", 0.25, 0.40)
	candidates := CandidatesFromBids(task, []auctioncore.Bid{
		{NodeID: "node-reject", Accepted: false},
		{NodeID: "node-ok", Accepted: true, CPUFragmentation: 0.20, RAMFragmentation: 0.30},
	}, map[string]auctioncore.NodeProfile{
		"node-ok": auctioncore.ProfileForNodeClass(auctioncore.NodeClassBalanced),
	})

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	if candidates[0].NodeID != "node-ok" {
		t.Fatalf("expected node-ok, got %s", candidates[0].NodeID)
	}
}
