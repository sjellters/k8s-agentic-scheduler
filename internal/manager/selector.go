package manager

import (
	"context"
	"fmt"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	"github.com/diego/k8s-agentic-scheduler/internal/nsga3"
)

type Selection struct {
	Winner           auctioncore.Bid
	HasWinner        bool
	Strategy         string
	NSGA3Preparation *nsga3.Preparation
}

type WinnerSelector interface {
	Select(ctx context.Context, task auctioncore.Task, bids []auctioncore.Bid) (Selection, error)
}

type BaselineSelector struct{}

type NSGA3Selector struct {
	optimizer nsga3.Optimizer
	fallback  WinnerSelector
}

func NewBaselineSelector() BaselineSelector {
	return BaselineSelector{}
}

func NewNSGA3Selector(config nsga3.Config) (NSGA3Selector, error) {
	optimizer, err := nsga3.New(config)
	if err != nil {
		return NSGA3Selector{}, err
	}

	return NSGA3Selector{
		optimizer: optimizer,
		fallback:  NewBaselineSelector(),
	}, nil
}

func (BaselineSelector) Select(_ context.Context, _ auctioncore.Task, bids []auctioncore.Bid) (Selection, error) {
	winner, ok := auctioncore.SelectWinner(bids)

	return Selection{
		Winner:    winner,
		HasWinner: ok,
		Strategy:  "baseline",
	}, nil
}

func (s NSGA3Selector) Select(ctx context.Context, task auctioncore.Task, bids []auctioncore.Bid) (Selection, error) {
	if s.fallback == nil {
		return Selection{}, fmt.Errorf("nsga3 selector requires a fallback selector")
	}

	candidates := nsga3.CandidatesFromBids(bids)
	nsga3Selection, err := s.optimizer.Select(candidates)
	if err != nil {
		return Selection{}, err
	}

	if nsga3Selection.HasWinner {
		winnerBid, ok := bidByNodeID(bids, nsga3Selection.Winner.NodeID)
		if ok {
			return Selection{
				Winner:           winnerBid,
				HasWinner:        true,
				Strategy:         "nsga3-first-pass",
				NSGA3Preparation: &nsga3Selection.Preparation,
			}, nil
		}
	}

	fallbackSelection, err := s.fallback.Select(ctx, task, bids)
	if err != nil {
		return Selection{}, err
	}

	fallbackSelection.Strategy = "nsga3-fallback-baseline"
	fallbackSelection.NSGA3Preparation = &nsga3Selection.Preparation

	return fallbackSelection, nil
}

func bidByNodeID(bids []auctioncore.Bid, nodeID string) (auctioncore.Bid, bool) {
	for _, bid := range bids {
		if bid.NodeID == nodeID && bid.Accepted {
			return bid, true
		}
	}

	return auctioncore.Bid{}, false
}
