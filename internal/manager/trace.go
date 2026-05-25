package manager

import (
	"fmt"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	"github.com/diego/k8s-agentic-scheduler/internal/governance"
	"github.com/diego/k8s-agentic-scheduler/internal/nsga3"
)

type DecisionTrace struct {
	TaskID              string                       `json:"task_id"`
	TaskQoSSensitivity  float64                      `json:"task_qos_sensitivity"`
	TaskProfile         auctioncore.TaskProfile      `json:"task_profile"`
	ObjectiveWeights    auctioncore.ObjectiveWeights `json:"objective_weights"`
	Policy              *governance.Decision         `json:"policy,omitempty"`
	SelectionStrategy   string                       `json:"selection_strategy"`
	FallbackUsed        bool                         `json:"fallback_used"`
	NodeResults         []NodeTrace                  `json:"node_results"`
	BaselineEvaluations []BaselineEvaluation         `json:"baseline_evaluations"`
	NSGA3Preparation    *nsga3.Preparation           `json:"nsga3_preparation,omitempty"`
	WinnerNodeID        string                       `json:"winner_node_id,omitempty"`
	HasWinner           bool                         `json:"has_winner"`
}

type NodeTrace struct {
	Address          string  `json:"address"`
	NodeID           string  `json:"node_id,omitempty"`
	NodeClass        string  `json:"node_class,omitempty"`
	Accepted         bool    `json:"accepted"`
	CPUFragmentation float64 `json:"cpu_fragmentation,omitempty"`
	RAMFragmentation float64 `json:"ram_fragmentation,omitempty"`
	CombinedScore    float64 `json:"combined_score,omitempty"`
	RequestError     string  `json:"request_error,omitempty"`
}

type BaselineEvaluation struct {
	NodeID        string  `json:"node_id"`
	CPUResidual   float64 `json:"cpu_residual"`
	RAMResidual   float64 `json:"ram_residual"`
	CombinedScore float64 `json:"combined_score"`
}

func newDecisionTrace(task auctioncore.Task, nodeProfiles map[string]auctioncore.NodeProfile, results []NodeResult, selection Selection) DecisionTrace {
	effectiveTask := selection.EffectiveTask
	if effectiveTask.ID == "" {
		effectiveTask = task
	}

	trace := DecisionTrace{
		TaskID:              effectiveTask.ID,
		TaskQoSSensitivity:  effectiveTask.QoSSensitivity,
		TaskProfile:         effectiveTask.Profile,
		ObjectiveWeights:    effectiveTask.ObjectiveSet,
		SelectionStrategy:   selection.Strategy,
		FallbackUsed:        selection.Strategy == "nsga3-fallback-baseline",
		NodeResults:         buildNodeTrace(nodeProfiles, results),
		BaselineEvaluations: buildBaselineEvaluations(results),
		NSGA3Preparation:    selection.NSGA3Preparation,
		HasWinner:           selection.HasWinner,
	}

	if selection.HasWinner {
		trace.WinnerNodeID = selection.Winner.NodeID
	}

	if selection.PolicyDecision != nil {
		decision := *selection.PolicyDecision
		trace.Policy = &decision
	}

	return trace
}

func buildNodeTrace(nodeProfiles map[string]auctioncore.NodeProfile, results []NodeResult) []NodeTrace {
	traces := make([]NodeTrace, 0, len(results))
	for _, result := range results {
		trace := NodeTrace{
			Address: result.Address,
		}

		if result.Err != nil {
			trace.RequestError = result.Err.Error()
			traces = append(traces, trace)
			continue
		}

		trace.NodeID = result.Bid.NodeID
		trace.NodeClass = string(nodeProfiles[result.Bid.NodeID].Class)
		trace.Accepted = result.Bid.Accepted
		trace.CPUFragmentation = result.Bid.CPUFragmentation
		trace.RAMFragmentation = result.Bid.RAMFragmentation
		if result.Bid.Accepted {
			trace.CombinedScore = result.Bid.Score()
		}

		traces = append(traces, trace)
	}

	return traces
}

func buildBaselineEvaluations(results []NodeResult) []BaselineEvaluation {
	evaluations := make([]BaselineEvaluation, 0, len(results))
	for _, result := range results {
		if result.Err != nil || !result.Bid.Accepted {
			continue
		}

		evaluations = append(evaluations, BaselineEvaluation{
			NodeID:        result.Bid.NodeID,
			CPUResidual:   result.Bid.CPUFragmentation,
			RAMResidual:   result.Bid.RAMFragmentation,
			CombinedScore: result.Bid.Score(),
		})
	}

	return evaluations
}

func (t DecisionTrace) Lines() []string {
	lines := []string{
		fmt.Sprintf("TRACE TASK: %s", t.TaskID),
		fmt.Sprintf("TRACE TASK QOS: %.2f", t.TaskQoSSensitivity),
		fmt.Sprintf("TRACE TASK PROFILE: %s", t.TaskProfile),
		fmt.Sprintf("TRACE OBJECTIVE WEIGHTS: cpu=%.2f ram=%.2f qos=%.2f energy=%.2f", t.ObjectiveWeights.CPU, t.ObjectiveWeights.RAM, t.ObjectiveWeights.QoS, t.ObjectiveWeights.Energy),
		fmt.Sprintf("TRACE STRATEGY: %s", t.SelectionStrategy),
		fmt.Sprintf("TRACE FALLBACK: %t", t.FallbackUsed),
	}

	if t.Policy != nil {
		lines = append(lines,
			fmt.Sprintf("TRACE POLICY: %s (%s)", t.Policy.Policy.ID, t.Policy.Policy.PreferredSelector),
			fmt.Sprintf("TRACE POLICY SOURCE: %s", t.Policy.Policy.Source),
			fmt.Sprintf("TRACE POLICY REASON: %s", t.Policy.Policy.Reason),
			fmt.Sprintf("TRACE POLICY CANDIDATES: accepted=%d rejected=%d", t.Policy.AcceptedCandidates, t.Policy.RejectedCandidates),
		)
	}

	for _, evaluation := range t.BaselineEvaluations {
		lines = append(lines, fmt.Sprintf(
			"TRACE BASELINE: node %s | cpu %.4f | ram %.4f | score %.4f",
			evaluation.NodeID,
			evaluation.CPUResidual,
			evaluation.RAMResidual,
			evaluation.CombinedScore,
		))
	}

	if t.NSGA3Preparation != nil {
		lines = append(lines, fmt.Sprintf(
			"TRACE NSGA3: candidates=%d reference-points=%d fronts=%d objectives=%d",
			len(t.NSGA3Preparation.Candidates),
			len(t.NSGA3Preparation.ReferencePoints),
			len(t.NSGA3Preparation.Fronts),
			t.NSGA3Preparation.Config.Objectives,
		))
		lines = append(lines, fmt.Sprintf("TRACE NSGA3 OBJECTIVES: %v", t.NSGA3Preparation.ObjectiveNames))
		lines = append(lines, fmt.Sprintf("TRACE NSGA3 IDEAL POINT: %v", t.NSGA3Preparation.IdealPoint))
		for _, evaluation := range t.NSGA3Preparation.Evaluations {
			lines = append(lines, fmt.Sprintf(
				"TRACE NSGA3 CANDIDATE: node %s | raw %v | normalized %v | front %d | distance %.4f",
				evaluation.Candidate.NodeID,
				evaluation.Candidate.Objectives,
				evaluation.NormalizedObjectives,
				evaluation.FrontRank,
				evaluation.Distance,
			))
		}
	}

	if t.HasWinner {
		lines = append(lines, fmt.Sprintf("TRACE WINNER: %s", t.WinnerNodeID))
	} else {
		lines = append(lines, "TRACE WINNER: none")
	}

	return lines
}
