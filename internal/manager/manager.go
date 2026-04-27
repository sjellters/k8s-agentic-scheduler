package manager

import (
	"context"
	"sync"
	"time"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	auctionpb "github.com/diego/k8s-agentic-scheduler/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type BidRequester interface {
	RequestBid(ctx context.Context, address string, task auctioncore.Task) (auctioncore.Bid, error)
}

type GRPCBidRequester struct {
	RequestTimeout time.Duration
}

type NodeResult struct {
	Address string
	Bid     auctioncore.Bid
	Err     error
}

type AuctionResult struct {
	Task        auctioncore.Task
	NodeResults []NodeResult
	Winner      auctioncore.Bid
	HasWinner   bool
}

func NewGRPCBidRequester(timeout time.Duration) GRPCBidRequester {
	return GRPCBidRequester{RequestTimeout: timeout}
}

func (r GRPCBidRequester) RequestBid(ctx context.Context, address string, task auctioncore.Task) (auctioncore.Bid, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return auctioncore.Bid{}, err
	}
	defer conn.Close()

	requestCtx := ctx
	cancel := func() {}
	if r.RequestTimeout > 0 {
		requestCtx, cancel = context.WithTimeout(ctx, r.RequestTimeout)
	}
	defer cancel()

	client := auctionpb.NewContractNetClient(conn)
	response, err := client.RequestBid(requestCtx, &auctionpb.TaskRequest{
		TaskId:     task.ID,
		CpuReqNorm: task.CPUReqNorm,
		RamReqNorm: task.RAMReqNorm,
	})
	if err != nil {
		return auctioncore.Bid{}, err
	}

	return auctioncore.Bid{
		NodeID:           response.NodeId,
		Accepted:         response.Accepted,
		CPUFragmentation: response.F1CpuFrag,
		RAMFragmentation: response.F3RamFrag,
	}, nil
}

func RunAuction(ctx context.Context, requester BidRequester, nodes []string, task auctioncore.Task) AuctionResult {
	results := make([]NodeResult, len(nodes))

	var wg sync.WaitGroup
	for index, address := range nodes {
		wg.Add(1)

		go func(i int, nodeAddress string) {
			defer wg.Done()

			bid, err := requester.RequestBid(ctx, nodeAddress, task)
			results[i] = NodeResult{
				Address: nodeAddress,
				Bid:     bid,
				Err:     err,
			}
		}(index, address)
	}

	wg.Wait()

	bids := make([]auctioncore.Bid, 0, len(results))
	for _, result := range results {
		if result.Err != nil {
			continue
		}
		bids = append(bids, result.Bid)
	}

	winner, ok := auctioncore.SelectWinner(bids)

	return AuctionResult{
		Task:        task,
		NodeResults: results,
		Winner:      winner,
		HasWinner:   ok,
	}
}
