package nsga3

import (
	"fmt"
	"math"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
)

func CandidateFromBid(task auctioncore.Task, bid auctioncore.Bid, profile auctioncore.NodeProfile) (Candidate, bool) {
	if !bid.Accepted {
		return Candidate{}, false
	}

	resourcePressure := clamp01((1 - bid.CPUFragmentation + 1 - bid.RAMFragmentation) / 2)
	qosPenalty := clamp01(task.QoSSensitivity * ((resourcePressure + profile.QoSBias) / 2))
	energyPenalty := clamp01(profile.EnergyPenalty * task.NormalizedLoad())

	return Candidate{
		NodeID: bid.NodeID,
		Objectives: []float64{
			clamp01(bid.CPUFragmentation * task.ObjectiveSet.CPU),
			clamp01(bid.RAMFragmentation * task.ObjectiveSet.RAM),
			clamp01((1 - qosPenalty) * task.ObjectiveSet.QoS),
			clamp01((1 - energyPenalty) * task.ObjectiveSet.Energy),
		},
	}, true
}

func CandidatesFromBids(task auctioncore.Task, bids []auctioncore.Bid, profiles map[string]auctioncore.NodeProfile) []Candidate {
	candidates := make([]Candidate, 0, len(bids))
	for _, bid := range bids {
		profile := profileForNode(profiles, bid.NodeID)
		candidate, ok := CandidateFromBid(task, bid, profile)
		if !ok {
			continue
		}
		candidates = append(candidates, candidate)
	}

	return candidates
}

func validateCandidates(candidates []Candidate, objectives int) error {
	for _, candidate := range candidates {
		if len(candidate.Objectives) != objectives {
			return fmt.Errorf("candidate %s has %d objectives, expected %d", candidate.NodeID, len(candidate.Objectives), objectives)
		}
	}

	return nil
}

func profileForNode(profiles map[string]auctioncore.NodeProfile, nodeID string) auctioncore.NodeProfile {
	if profile, ok := profiles[nodeID]; ok {
		return profile
	}

	return auctioncore.DefaultNodeProfile()
}

func clamp01(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}
