package manager

import (
	"context"
	"sync"
	"time"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	"github.com/diego/k8s-agentic-scheduler/internal/governance"
	"github.com/diego/k8s-agentic-scheduler/internal/nsga3"
	auctionpb "github.com/diego/k8s-agentic-scheduler/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type BidRequester interface {
	RequestBid(ctx context.Context, address string, task auctioncore.Task) (auctioncore.Bid, error)
}

type GRPCBidRequester struct {
	RequestTimeout time.Duration
	DialOptions    []grpc.DialOption
}

type NodeResult struct {
	Address string
	Bid     auctioncore.Bid
	Err     error
}

type AuctionResult struct {
	Task              auctioncore.Task
	NodeProfiles      map[string]auctioncore.NodeProfile
	NodeResults       []NodeResult
	Winner            auctioncore.Bid
	HasWinner         bool
	SelectionStrategy string
	NSGA3Preparation  *nsga3.Preparation
	PolicyDecision    *governance.Decision
	DecisionTrace     DecisionTrace
}

func NewGRPCBidRequester(timeout time.Duration) GRPCBidRequester {
	return GRPCBidRequester{RequestTimeout: timeout}
}

func (r GRPCBidRequester) RequestBid(ctx context.Context, address string, task auctioncore.Task) (auctioncore.Bid, error) {
	dialOptions := make([]grpc.DialOption, 0, len(r.DialOptions)+1)
	if len(r.DialOptions) == 0 {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		dialOptions = append(dialOptions, r.DialOptions...)
	}

	conn, err := grpc.NewClient(address, dialOptions...)
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
	result, _ := RunAuctionWithSelector(ctx, requester, NewBaselineSelector(), nodes, task)
	return result
}

func RunAuctionWithSelector(ctx context.Context, requester BidRequester, selector WinnerSelector, nodes []string, task auctioncore.Task) (AuctionResult, error) {
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

	selection, err := selector.Select(ctx, task, bids)
	if err != nil {
		return AuctionResult{}, err
	}

	return AuctionResult{
		Task:              task,
		NodeProfiles:      map[string]auctioncore.NodeProfile{},
		NodeResults:       results,
		Winner:            selection.Winner,
		HasWinner:         selection.HasWinner,
		SelectionStrategy: selection.Strategy,
		NSGA3Preparation:  selection.NSGA3Preparation,
		PolicyDecision:    selection.PolicyDecision,
		DecisionTrace:     newDecisionTrace(task, map[string]auctioncore.NodeProfile{}, results, selection),
	}, nil
}

func RunAuctionWithSupervisor(
	ctx context.Context,
	requester BidRequester,
	supervisor governance.Supervisor,
	selectorFactories map[string]SelectorFactory,
	nodeProfiles map[string]auctioncore.NodeProfile,
	nodes []string,
	task auctioncore.Task,
) (AuctionResult, error) {
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

	policyDecision, err := supervisor.Decide(ctx, task, bids)
	if err != nil {
		return AuctionResult{}, err
	}

	selector, err := selectorFromPolicy(policyDecision, selectorFactories)
	if err != nil {
		return AuctionResult{}, err
	}

	return RunAuctionWithSelectorFromResults(ctx, selector, task, nodeProfiles, results)
}

func RunAuctionWithSelectorFromResults(ctx context.Context, selector WinnerSelector, task auctioncore.Task, nodeProfiles map[string]auctioncore.NodeProfile, results []NodeResult) (AuctionResult, error) {
	bids := make([]auctioncore.Bid, 0, len(results))
	for _, result := range results {
		if result.Err != nil {
			continue
		}
		bids = append(bids, result.Bid)
	}

	selection, err := selector.Select(ctx, task, bids)
	if err != nil {
		return AuctionResult{}, err
	}

	return AuctionResult{
		Task:              task,
		NodeProfiles:      nodeProfiles,
		NodeResults:       results,
		Winner:            selection.Winner,
		HasWinner:         selection.HasWinner,
		SelectionStrategy: selection.Strategy,
		NSGA3Preparation:  selection.NSGA3Preparation,
		PolicyDecision:    selection.PolicyDecision,
		DecisionTrace:     newDecisionTrace(task, nodeProfiles, results, selection),
	}, nil
}
