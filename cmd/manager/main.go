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

	if result.NSGA3Preparation != nil {
		log.Printf(
			" NSGA3 SKELETON: %d candidates | %d reference points | %d objectives",
			len(result.NSGA3Preparation.Candidates),
			len(result.NSGA3Preparation.ReferencePoints),
			result.NSGA3Preparation.Config.Objectives,
		)
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
