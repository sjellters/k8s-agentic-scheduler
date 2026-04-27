package nodeagent

import (
	"context"
	"log"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	auctionpb "github.com/diego/k8s-agentic-scheduler/proto"
)

type Server struct {
	auctionpb.UnimplementedContractNetServer
	nodeID   string
	capacity auctioncore.Resources
}

func NewServer(nodeID string, capacity auctioncore.Resources) *Server {
	return &Server{
		nodeID:   nodeID,
		capacity: capacity,
	}
}

func (s *Server) RequestBid(_ context.Context, in *auctionpb.TaskRequest) (*auctionpb.BidResponse, error) {
	log.Printf(">>> incoming auction: %s (req: CPU %.2f, RAM %.2f)", in.TaskId, in.CpuReqNorm, in.RamReqNorm)

	bid := auctioncore.EvaluateBid(s.nodeID, s.capacity, auctioncore.Task{
		ID:         in.TaskId,
		CPUReqNorm: in.CpuReqNorm,
		RAMReqNorm: in.RamReqNorm,
	})

	if !bid.Accepted {
		log.Printf("!!! [AUCTION REJECTED] Node %s has insufficient resources (Cap: CPU %.2f, RAM %.2f)", s.nodeID, s.capacity.CPU, s.capacity.RAM)
		return bidResponseFromBid(bid), nil
	}

	log.Printf("<<< [BID SUBMITTED] F1: %.4f, F3: %.4f", bid.CPUFragmentation, bid.RAMFragmentation)

	return bidResponseFromBid(bid), nil
}

func bidResponseFromBid(bid auctioncore.Bid) *auctionpb.BidResponse {
	return &auctionpb.BidResponse{
		NodeId:    bid.NodeID,
		F1CpuFrag: bid.CPUFragmentation,
		F3RamFrag: bid.RAMFragmentation,
		Accepted:  bid.Accepted,
	}
}
