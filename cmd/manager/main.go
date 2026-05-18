package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	auctionmanager "github.com/diego/k8s-agentic-scheduler/internal/manager"
	"github.com/diego/k8s-agentic-scheduler/internal/nsga3"
)

func main() {
	nodes := flag.String("nodes", "localhost:50051", "Comma-separated list of node addresses")
	selectorMode := flag.String("selector", "baseline", "Winner selector strategy: baseline or nsga3")
	flag.Parse()

	nodeList := strings.Split(*nodes, ",")

	task := auctioncore.Task{
		ID:         "pod-xyz-123",
		CPUReqNorm: 0.25,
		RAMReqNorm: 0.40,
	}

	log.Printf("==========================================")
	log.Printf(" MANAGER STARTING AUCTION")
	log.Printf(" POD ID:  %s", task.ID)
	log.Printf(" REQ:     CPU %.2f | RAM %.2f", task.CPUReqNorm, task.RAMReqNorm)
	log.Printf("==========================================")

	var selector auctionmanager.WinnerSelector
	switch *selectorMode {
	case "baseline":
		selector = auctionmanager.NewBaselineSelector()
	case "nsga3":
		nsga3Selector, err := auctionmanager.NewNSGA3Selector(nsga3.DefaultConfig())
		if err != nil {
			log.Fatalf("failed to configure nsga3 selector: %v", err)
		}
		selector = nsga3Selector
	default:
		log.Fatalf("unsupported selector %q", *selectorMode)
	}

	result, err := auctionmanager.RunAuctionWithSelector(
		context.Background(),
		auctionmanager.NewGRPCBidRequester(time.Second),
		selector,
		nodeList,
		task,
	)
	if err != nil {
		log.Fatalf("failed to run auction: %v", err)
	}

	for _, nodeResult := range result.NodeResults {
		if nodeResult.Err != nil {
			log.Printf("[ERROR] %s: %v", nodeResult.Address, nodeResult.Err)
			continue
		}

		bid := nodeResult.Bid
		if !bid.Accepted {
			fmt.Printf(" [-] NODE: %-10s | STATUS: INSUFFICIENT RESOURCES\n", bid.NodeID)
			continue
		}

		fmt.Printf(" [+] NODE: %-10s | STATUS: BID SUBMITTED (F1: %.4f, F3: %.4f)\n", bid.NodeID, bid.CPUFragmentation, bid.RAMFragmentation)
	}

	log.Printf(" SELECTION STRATEGY: %s", result.SelectionStrategy)
	logBaselineTrace(result)
	if result.NSGA3Preparation != nil {
		preparation := result.NSGA3Preparation
		log.Printf(
			" NSGA3 TRACE: %d candidates | %d reference points | %d fronts | %d objectives",
			len(preparation.Candidates),
			len(preparation.ReferencePoints),
			len(preparation.Fronts),
			preparation.Config.Objectives,
		)
		log.Printf(" IDEAL POINT: %s", formatValues(preparation.IdealPoint))
		log.Printf(" ACTIVE REFERENCE POINT: %v", preparation.ReferencePoints[preparation.ActiveReferencePoint].Coordinates)
		for _, evaluation := range preparation.Evaluations {
			log.Printf(
				" NSGA3 CANDIDATE: node %s | raw %s | normalized %s | front %d | distance %.4f | normalized-sum %.4f",
				evaluation.Candidate.NodeID,
				formatValues(evaluation.Candidate.Objectives),
				formatValues(evaluation.NormalizedObjectives),
				evaluation.FrontRank,
				evaluation.Distance,
				sumValues(evaluation.NormalizedObjectives),
			)
		}
		if preparation.SelectedCandidate != nil {
			log.Printf(
				" NSGA3 WINNER TRACE: node %s | front %d | distance %.4f | balanced over pure sum",
				preparation.SelectedCandidate.Candidate.NodeID,
				preparation.SelectedCandidate.FrontRank,
				preparation.SelectedCandidate.Distance,
			)
		}
	}

	fmt.Println("------------------------------------------")
	if result.HasWinner {
		fmt.Printf(" >>> AUCTION SUCCESSFUL <<<\n")
		fmt.Printf(" WINNER: %s\n", result.Winner.NodeID)
		fmt.Printf(" OPTIMAL SCORE: %.4f\n", result.Winner.Score())
	} else {
		fmt.Printf(" >>> AUCTION FAILED: NO SUITABLE NODES FOUND <<<\n")
	}
	fmt.Println("------------------------------------------")
}

func logBaselineTrace(result auctionmanager.AuctionResult) {
	highestScore := 0.0
	highestNodes := make([]string, 0)
	firstAccepted := true

	for _, nodeResult := range result.NodeResults {
		if nodeResult.Err != nil || !nodeResult.Bid.Accepted {
			continue
		}

		bid := nodeResult.Bid
		score := bid.Score()
		log.Printf(
			" BASELINE CANDIDATE: node %s | F1 %.4f + F3 %.4f = score %.4f",
			bid.NodeID,
			bid.CPUFragmentation,
			bid.RAMFragmentation,
			score,
		)

		if firstAccepted || score > highestScore {
			highestScore = score
			highestNodes = []string{bid.NodeID}
			firstAccepted = false
			continue
		}

		if score == highestScore {
			highestNodes = append(highestNodes, bid.NodeID)
		}
	}

	if len(highestNodes) > 1 {
		log.Printf(
			" BASELINE TIE: score %.4f shared by %s | winner resolved by first highest score encountered",
			highestScore,
			strings.Join(highestNodes, ", "),
		)
	}
}

func formatValues(values []float64) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%.4f", value))
	}

	return "[" + strings.Join(parts, ", ") + "]"
}

func sumValues(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}

	return total
}
