package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	"github.com/diego/k8s-agentic-scheduler/internal/governance"
	auctionmanager "github.com/diego/k8s-agentic-scheduler/internal/manager"
	"github.com/diego/k8s-agentic-scheduler/internal/nsga3"
)

func main() {
	nodes := flag.String("nodes", "localhost:50051", "Comma-separated list of node addresses")
	selectorMode := flag.String("selector", "baseline", "Winner selector strategy: baseline or nsga3")
	policyName := flag.String("policy", "", "Supervisor policy: balanced, capacity-first, black-friday, or energy-saver")
	nodeProfilesFlag := flag.String("node-profiles", "", "Comma-separated node profiles in node-id=class form")
	taskQoS := flag.Float64("task-qos", 0.5, "Normalized QoS sensitivity for the incoming task (0.0 to 1.0)")
	traceFormat := flag.String("trace-format", "text", "Decision trace format: text or json")
	flag.Parse()

	nodeList := strings.Split(*nodes, ",")

	task := auctioncore.DefaultTask("pod-xyz-123", 0.25, 0.40)
	task.QoSSensitivity = *taskQoS

	log.Printf("==========================================")
	log.Printf(" MANAGER STARTING AUCTION")
	log.Printf(" POD ID:  %s", task.ID)
	log.Printf(" REQ:     CPU %.2f | RAM %.2f", task.CPUReqNorm, task.RAMReqNorm)
	log.Printf(" QoS:     %.2f", task.QoSSensitivity)
	log.Printf("==========================================")

	requester := auctionmanager.NewGRPCBidRequester(time.Second)
	nodeProfiles, err := parseNodeProfiles(*nodeProfilesFlag)
	if err != nil {
		log.Fatalf("failed to parse node profiles: %v", err)
	}
	selectorFactories := auctionmanager.NewSelectorFactory(nsga3.DefaultConfig(), nodeProfiles)

	var (
		result auctionmanager.AuctionResult
	)

	if *policyName != "" {
		policy, policyErr := governance.PolicyByName(*policyName)
		if policyErr != nil {
			log.Fatalf("failed to resolve policy: %v", policyErr)
		}

		supervisor, supervisorErr := governance.NewStaticSupervisor(policy)
		if supervisorErr != nil {
			log.Fatalf("failed to configure supervisor: %v", supervisorErr)
		}

		result, err = auctionmanager.RunAuctionWithSupervisor(
			context.Background(),
			requester,
			supervisor,
			selectorFactories,
			nodeProfiles,
			nodeList,
			task,
		)
	} else {
		selector, selectorErr := selectorByName(*selectorMode, selectorFactories)
		if selectorErr != nil {
			log.Fatalf("failed to configure selector: %v", selectorErr)
		}

		result, err = auctionmanager.RunAuctionWithSelector(
			context.Background(),
			requester,
			selector,
			nodeList,
			task,
		)
	}
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
	logDecisionTrace(result, *traceFormat)

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

func selectorByName(name string, factories map[string]auctionmanager.SelectorFactory) (auctionmanager.WinnerSelector, error) {
	factory, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("unsupported selector %q", name)
	}

	return factory()
}

func logDecisionTrace(result auctionmanager.AuctionResult, format string) {
	switch format {
	case "text":
		for _, line := range result.DecisionTrace.Lines() {
			log.Print(" ", line)
		}
	case "json":
		payload, err := json.MarshalIndent(result.DecisionTrace, "", "  ")
		if err != nil {
			log.Printf(" failed to marshal decision trace: %v", err)
			return
		}
		log.Printf(" DECISION TRACE JSON:\n%s", string(payload))
	default:
		log.Fatalf("unsupported trace format %q", format)
	}
}

func parseNodeProfiles(raw string) (map[string]auctioncore.NodeProfile, error) {
	profiles := make(map[string]auctioncore.NodeProfile)
	if raw == "" {
		return profiles, nil
	}

	for _, entry := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(entry), "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid node profile entry %q", entry)
		}

		nodeID := strings.TrimSpace(parts[0])
		className := strings.TrimSpace(parts[1])
		if nodeID == "" || className == "" {
			return nil, fmt.Errorf("invalid node profile entry %q", entry)
		}

		class, err := parseNodeClass(className)
		if err != nil {
			return nil, err
		}
		profiles[nodeID] = auctioncore.ProfileForNodeClass(class)
	}

	return profiles, nil
}

func parseNodeClass(raw string) (auctioncore.NodeClass, error) {
	switch raw {
	case string(auctioncore.NodeClassHighPerformance):
		return auctioncore.NodeClassHighPerformance, nil
	case string(auctioncore.NodeClassBalanced):
		return auctioncore.NodeClassBalanced, nil
	case string(auctioncore.NodeClassHighEfficiency):
		return auctioncore.NodeClassHighEfficiency, nil
	default:
		return "", fmt.Errorf("unsupported node class %q", raw)
	}
}
