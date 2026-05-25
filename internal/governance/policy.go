package governance

import (
	"context"
	"fmt"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
)

const (
	StrategyBaseline = "baseline"
	StrategyNSGA3    = "nsga3"
)

type Policy struct {
	ID                string                       `json:"id"`
	Source            string                       `json:"source"`
	Reason            string                       `json:"reason"`
	PreferredSelector string                       `json:"preferred_selector"`
	ObjectiveWeights  auctioncore.ObjectiveWeights `json:"objective_weights"`
	TaskProfile       auctioncore.TaskProfile      `json:"task_profile"`
}

type Decision struct {
	Policy             Policy `json:"policy"`
	AcceptedCandidates int    `json:"accepted_candidates"`
	RejectedCandidates int    `json:"rejected_candidates"`
}

type Supervisor interface {
	Decide(ctx context.Context, task auctioncore.Task, bids []auctioncore.Bid) (Decision, error)
}

type StaticSupervisor struct {
	policy Policy
}

func NewStaticSupervisor(policy Policy) (StaticSupervisor, error) {
	if err := validatePolicy(policy); err != nil {
		return StaticSupervisor{}, err
	}

	return StaticSupervisor{policy: policy}, nil
}

func PolicyByName(name string) (Policy, error) {
	switch name {
	case "", "balanced":
		return Policy{
			ID:                "balanced-efficiency",
			Source:            "static-supervisor",
			Reason:            "Favor balanced residual resources, QoS stability, and energy awareness across the four-objective scheduler.",
			PreferredSelector: StrategyNSGA3,
			ObjectiveWeights:  auctioncore.DefaultObjectiveWeights(),
			TaskProfile:       auctioncore.TaskProfileBalanced,
		}, nil
	case "capacity-first":
		return Policy{
			ID:                "capacity-first",
			Source:            "static-supervisor",
			Reason:            "Prioritize the highest aggregate residual capacity.",
			PreferredSelector: StrategyBaseline,
			ObjectiveWeights: auctioncore.ObjectiveWeights{
				CPU:    1.0,
				RAM:    1.0,
				QoS:    0.4,
				Energy: 0.4,
			},
			TaskProfile: auctioncore.TaskProfileBalanced,
		}, nil
	case "black-friday":
		return Policy{
			ID:                "black-friday-burst",
			Source:            "static-supervisor",
			Reason:            "Prioritize QoS-sensitive burst handling during high-demand events.",
			PreferredSelector: StrategyNSGA3,
			ObjectiveWeights: auctioncore.ObjectiveWeights{
				CPU:    0.95,
				RAM:    0.90,
				QoS:    1.00,
				Energy: 0.45,
			},
			TaskProfile: auctioncore.TaskProfilePerformanceBurst,
		}, nil
	case "energy-saver":
		return Policy{
			ID:                "energy-saver",
			Source:            "static-supervisor",
			Reason:            "Prioritize lower energy cost while preserving multi-objective feasibility.",
			PreferredSelector: StrategyNSGA3,
			ObjectiveWeights: auctioncore.ObjectiveWeights{
				CPU:    0.75,
				RAM:    0.75,
				QoS:    0.60,
				Energy: 1.00,
			},
			TaskProfile: auctioncore.TaskProfileEnergySaver,
		}, nil
	default:
		return Policy{}, fmt.Errorf("unsupported policy %q", name)
	}
}

func (s StaticSupervisor) Decide(_ context.Context, _ auctioncore.Task, bids []auctioncore.Bid) (Decision, error) {
	if err := validatePolicy(s.policy); err != nil {
		return Decision{}, err
	}

	accepted := 0
	for _, bid := range bids {
		if bid.Accepted {
			accepted++
		}
	}

	return Decision{
		Policy:             s.policy,
		AcceptedCandidates: accepted,
		RejectedCandidates: len(bids) - accepted,
	}, nil
}

func validatePolicy(policy Policy) error {
	if policy.ID == "" {
		return fmt.Errorf("policy id is required")
	}
	if policy.Source == "" {
		return fmt.Errorf("policy source is required")
	}
	if policy.Reason == "" {
		return fmt.Errorf("policy reason is required")
	}
	if policy.ObjectiveWeights.CPU <= 0 || policy.ObjectiveWeights.RAM <= 0 || policy.ObjectiveWeights.QoS <= 0 || policy.ObjectiveWeights.Energy <= 0 {
		return fmt.Errorf("all objective weights must be positive")
	}

	switch policy.PreferredSelector {
	case StrategyBaseline, StrategyNSGA3:
	default:
		return fmt.Errorf("unsupported preferred selector %q", policy.PreferredSelector)
	}

	switch policy.TaskProfile {
	case auctioncore.TaskProfileBalanced, auctioncore.TaskProfilePerformanceBurst, auctioncore.TaskProfileEnergySaver:
		return nil
	default:
		return fmt.Errorf("unsupported task profile %q", policy.TaskProfile)
	}
}
