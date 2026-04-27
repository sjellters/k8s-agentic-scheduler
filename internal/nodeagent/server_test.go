package nodeagent

import (
	"context"
	"testing"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	auctionpb "github.com/diego/k8s-agentic-scheduler/proto"
)

func TestRequestBidRejectsInsufficientTask(t *testing.T) {
	t.Parallel()

	server := NewServer("node-a", auctioncore.Resources{CPU: 0.20, RAM: 0.20})
	response, err := server.RequestBid(context.Background(), &auctionpb.TaskRequest{
		TaskId:     "pod-a",
		CpuReqNorm: 0.30,
		RamReqNorm: 0.10,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if response.Accepted {
		t.Fatalf("expected rejected response")
	}

	if response.NodeId != "node-a" {
		t.Fatalf("expected node-a, got %s", response.NodeId)
	}
}

func TestRequestBidReturnsFragmentationForAcceptedTask(t *testing.T) {
	t.Parallel()

	server := NewServer("node-b", auctioncore.Resources{CPU: 0.80, RAM: 0.90})
	response, err := server.RequestBid(context.Background(), &auctionpb.TaskRequest{
		TaskId:     "pod-b",
		CpuReqNorm: 0.25,
		RamReqNorm: 0.40,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !response.Accepted {
		t.Fatalf("expected accepted response")
	}

	if response.F1CpuFrag != 0.55 {
		t.Fatalf("expected CPU fragmentation 0.55, got %.2f", response.F1CpuFrag)
	}

	if response.F3RamFrag != 0.50 {
		t.Fatalf("expected RAM fragmentation 0.50, got %.2f", response.F3RamFrag)
	}
}
