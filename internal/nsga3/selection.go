package nsga3

import (
	"math"
	"slices"
)

func buildFronts(candidates []Candidate) ([]Front, map[string]int) {
	dominationCounts := make(map[string]int, len(candidates))
	dominatesMap := make(map[string][]string, len(candidates))

	currentFront := make([]string, 0)
	for _, candidate := range candidates {
		for _, other := range candidates {
			if candidate.NodeID == other.NodeID {
				continue
			}

			if dominates(candidate, other) {
				dominatesMap[candidate.NodeID] = append(dominatesMap[candidate.NodeID], other.NodeID)
			}
			if dominates(other, candidate) {
				dominationCounts[candidate.NodeID]++
			}
		}

		if dominationCounts[candidate.NodeID] == 0 {
			currentFront = append(currentFront, candidate.NodeID)
		}
	}

	sortNodeIDs(currentFront)
	fronts := make([]Front, 0)
	frontRanks := make(map[string]int, len(candidates))
	rank := 1

	for len(currentFront) > 0 {
		fronts = append(fronts, Front{
			Rank:             rank,
			CandidateNodeIDs: slices.Clone(currentFront),
		})

		nextFront := make([]string, 0)
		for _, nodeID := range currentFront {
			frontRanks[nodeID] = rank

			for _, dominatedID := range dominatesMap[nodeID] {
				dominationCounts[dominatedID]--
				if dominationCounts[dominatedID] == 0 {
					nextFront = append(nextFront, dominatedID)
				}
			}
		}

		sortNodeIDs(nextFront)
		currentFront = nextFront
		rank++
	}

	return fronts, frontRanks
}

func dominates(left, right Candidate) bool {
	betterInAtLeastOne := false
	for index := range left.Objectives {
		if left.Objectives[index] < right.Objectives[index] {
			return false
		}
		if left.Objectives[index] > right.Objectives[index] {
			betterInAtLeastOne = true
		}
	}

	return betterInAtLeastOne
}

func idealPoint(candidates []Candidate, objectives int) []float64 {
	point := make([]float64, objectives)
	for _, candidate := range candidates {
		for index := 0; index < objectives; index++ {
			if candidate.Objectives[index] > point[index] {
				point[index] = candidate.Objectives[index]
			}
		}
	}

	return point
}

func normalizeCandidate(candidate Candidate, ideal []float64) []float64 {
	normalized := make([]float64, len(candidate.Objectives))
	for index, value := range candidate.Objectives {
		if ideal[index] == 0 {
			normalized[index] = 0
			continue
		}
		normalized[index] = value / ideal[index]
	}

	return normalized
}

func selectBalancedReferencePoint(referencePoints []ReferencePoint, objectives int) int {
	target := 1.0 / float64(objectives)
	bestIndex := 0
	bestDistance := math.MaxFloat64

	for index, point := range referencePoints {
		distance := 0.0
		for _, coordinate := range point.Coordinates {
			delta := coordinate - target
			distance += delta * delta
		}

		if distance < bestDistance {
			bestIndex = index
			bestDistance = distance
		}
	}

	return bestIndex
}

func evaluateCandidates(candidates []Candidate, preparation Preparation) []CandidateEvaluation {
	evaluations := make([]CandidateEvaluation, 0, len(candidates))
	activeReferencePoint := preparation.ReferencePoints[preparation.ActiveReferencePoint]

	frontRanks := make(map[string]int, len(preparation.Candidates))
	for _, front := range preparation.Fronts {
		for _, nodeID := range front.CandidateNodeIDs {
			frontRanks[nodeID] = front.Rank
		}
	}

	for _, candidate := range candidates {
		normalized := normalizeCandidate(candidate, preparation.IdealPoint)
		evaluations = append(evaluations, CandidateEvaluation{
			Candidate:            candidate,
			FrontRank:            frontRanks[candidate.NodeID],
			NormalizedObjectives: normalized,
			ReferencePointIndex:  preparation.ActiveReferencePoint,
			Distance:             euclideanDistance(normalized, activeReferencePoint.Coordinates),
		})
	}

	return evaluations
}

func chooseWinner(evaluations []CandidateEvaluation) (CandidateEvaluation, bool) {
	if len(evaluations) == 0 {
		return CandidateEvaluation{}, false
	}

	best := evaluations[0]
	for _, evaluation := range evaluations[1:] {
		if evaluation.FrontRank < best.FrontRank {
			best = evaluation
			continue
		}
		if evaluation.FrontRank > best.FrontRank {
			continue
		}

		if evaluation.Distance < best.Distance {
			best = evaluation
			continue
		}
		if evaluation.Distance > best.Distance {
			continue
		}

		evaluationScore := sumObjectives(evaluation.NormalizedObjectives)
		bestScore := sumObjectives(best.NormalizedObjectives)
		if evaluationScore > bestScore {
			best = evaluation
			continue
		}
		if evaluationScore < bestScore {
			continue
		}

		if evaluation.Candidate.NodeID < best.Candidate.NodeID {
			best = evaluation
		}
	}

	return best, true
}

func euclideanDistance(left, right []float64) float64 {
	total := 0.0
	for index := range left {
		delta := left[index] - right[index]
		total += delta * delta
	}

	return math.Sqrt(total)
}

func sumObjectives(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}

	return total
}

func sortNodeIDs(nodeIDs []string) {
	slices.Sort(nodeIDs)
}
