package manager

import (
	"context"
	"errors"
	"testing"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	"github.com/diego/k8s-agentic-scheduler/internal/nsga3"
)

type fakeRequester struct {
	bids map[string]auctioncore.Bid
	errs map[string]error
}

func (f fakeRequester) RequestBid(_ context.Context, address string, _ auctioncore.Task) (auctioncore.Bid, error) {
	if err, ok := f.errs[address]; ok {
		return auctioncore.Bid{}, err
	}

	return f.bids[address], nil
}

func TestRunAuctionSelectsHighestAcceptedWinner(t *testing.T) {
	t.Parallel()

	result := RunAuction(context.Background(), fakeRequester{
		bids: map[string]auctioncore.Bid{
			"node-a": {NodeID: "node-a", Accepted: true, CPUFragmentation: 0.10, RAMFragmentation: 0.10},
			"node-b": {NodeID: "node-b", Accepted: true, CPUFragmentation: 0.40, RAMFragmentation: 0.20},
			"node-c": {NodeID: "node-c", Accepted: false},
		},
	}, []string{"node-a", "node-b", "node-c"}, auctioncore.Task{ID: "pod-a", CPUReqNorm: 0.25, RAMReqNorm: 0.40})

	if !result.HasWinner {
		t.Fatalf("expected a winner")
	}

	if result.Winner.NodeID != "node-b" {
		t.Fatalf("expected node-b, got %s", result.Winner.NodeID)
	}

	if len(result.NodeResults) != 3 {
		t.Fatalf("expected 3 node results, got %d", len(result.NodeResults))
	}
}

func TestRunAuctionPreservesPerNodeErrors(t *testing.T) {
	t.Parallel()

	requestErr := errors.New("request failed")
	result := RunAuction(context.Background(), fakeRequester{
		bids: map[string]auctioncore.Bid{
			"node-ok":     {NodeID: "node-ok", Accepted: true, CPUFragmentation: 0.30, RAMFragmentation: 0.30},
			"node-reject": {NodeID: "node-reject", Accepted: false},
		},
		errs: map[string]error{
			"node-error": requestErr,
		},
	}, []string{"node-error", "node-ok", "node-reject"}, auctioncore.Task{ID: "pod-b", CPUReqNorm: 0.25, RAMReqNorm: 0.40})

	if result.NodeResults[0].Err == nil {
		t.Fatalf("expected error for first node result")
	}

	if !errors.Is(result.NodeResults[0].Err, requestErr) {
		t.Fatalf("expected preserved request error")
	}

	if !result.HasWinner {
		t.Fatalf("expected a winner from successful bids")
	}

	if result.Winner.NodeID != "node-ok" {
		t.Fatalf("expected node-ok, got %s", result.Winner.NodeID)
	}
}

func TestRunAuctionWithSelectorInvokesNSGA3Skeleton(t *testing.T) {
	t.Parallel()

	selector, err := NewNSGA3Selector(nsga3.DefaultConfig())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	result, err := RunAuctionWithSelector(context.Background(), fakeRequester{
		bids: map[string]auctioncore.Bid{
			"node-a": {NodeID: "node-a", Accepted: true, CPUFragmentation: 0.20, RAMFragmentation: 0.20},
			"node-b": {NodeID: "node-b", Accepted: true, CPUFragmentation: 0.35, RAMFragmentation: 0.25},
		},
	}, selector, []string{"node-a", "node-b"}, auctioncore.Task{ID: "pod-c", CPUReqNorm: 0.25, RAMReqNorm: 0.40})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result.SelectionStrategy != "nsga3-skeleton" {
		t.Fatalf("expected nsga3-skeleton, got %s", result.SelectionStrategy)
	}

	if result.NSGA3Preparation == nil {
		t.Fatalf("expected nsga3 preparation trace")
	}

	if len(result.NSGA3Preparation.ReferencePoints) == 0 {
		t.Fatalf("expected reference points in nsga3 trace")
	}

	if result.Winner.NodeID != "node-b" {
		t.Fatalf("expected baseline fallback winner node-b, got %s", result.Winner.NodeID)
	}
}
