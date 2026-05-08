package nsga3

import (
	"fmt"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
)

func CandidateFromBid(bid auctioncore.Bid) (Candidate, bool) {
	if !bid.Accepted {
		return Candidate{}, false
	}

	return Candidate{
		NodeID: bid.NodeID,
		Objectives: []float64{
			bid.CPUFragmentation,
			bid.RAMFragmentation,
		},
	}, true
}

func CandidatesFromBids(bids []auctioncore.Bid) []Candidate {
	candidates := make([]Candidate, 0, len(bids))
	for _, bid := range bids {
		candidate, ok := CandidateFromBid(bid)
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
