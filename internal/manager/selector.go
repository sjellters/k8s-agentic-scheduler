package manager

import (
	"context"
	"fmt"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	"github.com/diego/k8s-agentic-scheduler/internal/governance"
	"github.com/diego/k8s-agentic-scheduler/internal/nsga3"
)

type Selection struct {
	Winner           auctioncore.Bid
	HasWinner        bool
	Strategy         string
	EffectiveTask    auctioncore.Task
	NSGA3Preparation *nsga3.Preparation
	PolicyDecision   *governance.Decision
}

type WinnerSelector interface {
	Select(ctx context.Context, task auctioncore.Task, bids []auctioncore.Bid) (Selection, error)
}

type BaselineSelector struct{}

type NSGA3Selector struct {
	optimizer    nsga3.Optimizer
	fallback     WinnerSelector
	nodeProfiles map[string]auctioncore.NodeProfile
}

type SelectorFactory func() (WinnerSelector, error)

func NewBaselineSelector() BaselineSelector {
	return BaselineSelector{}
}

func NewNSGA3Selector(config nsga3.Config, nodeProfiles map[string]auctioncore.NodeProfile) (NSGA3Selector, error) {
	optimizer, err := nsga3.New(config)
	if err != nil {
		return NSGA3Selector{}, err
	}

	return NSGA3Selector{
		optimizer:    optimizer,
		fallback:     NewBaselineSelector(),
		nodeProfiles: nodeProfiles,
	}, nil
}

func NewSelectorFactory(config nsga3.Config, nodeProfiles map[string]auctioncore.NodeProfile) map[string]SelectorFactory {
	return map[string]SelectorFactory{
		governance.StrategyBaseline: func() (WinnerSelector, error) {
			return NewBaselineSelector(), nil
		},
		governance.StrategyNSGA3: func() (WinnerSelector, error) {
			return NewNSGA3Selector(config, nodeProfiles)
		},
	}
}

func (BaselineSelector) Select(_ context.Context, task auctioncore.Task, bids []auctioncore.Bid) (Selection, error) {
	winner, ok := auctioncore.SelectWinner(bids)

	return Selection{
		Winner:        winner,
		HasWinner:     ok,
		Strategy:      "baseline",
		EffectiveTask: task,
	}, nil
}

func (s NSGA3Selector) Select(ctx context.Context, task auctioncore.Task, bids []auctioncore.Bid) (Selection, error) {
	if s.fallback == nil {
		return Selection{}, fmt.Errorf("nsga3 selector requires a fallback selector")
	}

	candidates := nsga3.CandidatesFromBids(task, bids, s.nodeProfiles)
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
				EffectiveTask:    task,
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

func selectorFromPolicy(policy governance.Decision, factories map[string]SelectorFactory) (WinnerSelector, error) {
	factory, ok := factories[policy.Policy.PreferredSelector]
	if !ok {
		return nil, fmt.Errorf("no selector factory for preferred selector %q", policy.Policy.PreferredSelector)
	}

	selector, err := factory()
	if err != nil {
		return nil, err
	}

	return governedSelector{
		base:     selector,
		decision: policy,
	}, nil
}

type governedSelector struct {
	base     WinnerSelector
	decision governance.Decision
}

func (s governedSelector) Select(ctx context.Context, task auctioncore.Task, bids []auctioncore.Bid) (Selection, error) {
	task.ObjectiveSet = s.decision.Policy.ObjectiveWeights
	task.Profile = s.decision.Policy.TaskProfile

	selection, err := s.base.Select(ctx, task, bids)
	if err != nil {
		return Selection{}, err
	}

	decision := s.decision
	selection.PolicyDecision = &decision

	return selection, nil
}

func bidByNodeID(bids []auctioncore.Bid, nodeID string) (auctioncore.Bid, bool) {
	for _, bid := range bids {
		if bid.NodeID == nodeID && bid.Accepted {
			return bid, true
		}
	}

	return auctioncore.Bid{}, false
}
