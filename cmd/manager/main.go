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
)

func main() {
	nodes := flag.String("nodes", "localhost:50051", "Comma-separated list of node addresses")
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

	result := auctionmanager.RunAuction(
		context.Background(),
		auctionmanager.NewGRPCBidRequester(time.Second),
		nodeList,
		task,
	)

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
