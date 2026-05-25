package manager

import (
	"context"
	"errors"
	"net"
	"testing"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	"github.com/diego/k8s-agentic-scheduler/internal/governance"
	"github.com/diego/k8s-agentic-scheduler/internal/nodeagent"
	"github.com/diego/k8s-agentic-scheduler/internal/nsga3"
	auctionpb "github.com/diego/k8s-agentic-scheduler/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type fakeRequester struct {
	bids map[string]auctioncore.Bid
	errs map[string]error
}

func (f fakeRequester) RequestBid(_ context.Context, address string, _ auctioncore.Task) (auctioncore.Bid, error) {
	if err, ok := f.errs[address]; ok {
		return auctioncore.Bid{}, err
	}

	return f.bids[address], nil
}

func TestRunAuctionSelectsHighestAcceptedWinner(t *testing.T) {
	t.Parallel()

	result := RunAuction(context.Background(), fakeRequester{
		bids: map[string]auctioncore.Bid{
			"node-a": {NodeID: "node-a", Accepted: true, CPUFragmentation: 0.10, RAMFragmentation: 0.10},
			"node-b": {NodeID: "node-b", Accepted: true, CPUFragmentation: 0.40, RAMFragmentation: 0.20},
			"node-c": {NodeID: "node-c", Accepted: false},
		},
	}, []string{"node-a", "node-b", "node-c"}, auctioncore.DefaultTask("pod-a", 0.25, 0.40))

	if !result.HasWinner {
		t.Fatalf("expected a winner")
	}

	if result.Winner.NodeID != "node-b" {
		t.Fatalf("expected node-b, got %s", result.Winner.NodeID)
	}

	if len(result.NodeResults) != 3 {
		t.Fatalf("expected 3 node results, got %d", len(result.NodeResults))
	}

	if len(result.DecisionTrace.BaselineEvaluations) != 2 {
		t.Fatalf("expected 2 baseline evaluations, got %d", len(result.DecisionTrace.BaselineEvaluations))
	}
}

func TestRunAuctionPreservesPerNodeErrors(t *testing.T) {
	t.Parallel()

	requestErr := errors.New("request failed")
	result := RunAuction(context.Background(), fakeRequester{
		bids: map[string]auctioncore.Bid{
			"node-ok":     {NodeID: "node-ok", Accepted: true, CPUFragmentation: 0.30, RAMFragmentation: 0.30},
			"node-reject": {NodeID: "node-reject", Accepted: false},
		},
		errs: map[string]error{
			"node-error": requestErr,
		},
	}, []string{"node-error", "node-ok", "node-reject"}, auctioncore.DefaultTask("pod-b", 0.25, 0.40))

	if result.NodeResults[0].Err == nil {
		t.Fatalf("expected error for first node result")
	}

	if !errors.Is(result.NodeResults[0].Err, requestErr) {
		t.Fatalf("expected preserved request error")
	}

	if !result.HasWinner {
		t.Fatalf("expected a winner from successful bids")
	}

	if result.Winner.NodeID != "node-ok" {
		t.Fatalf("expected node-ok, got %s", result.Winner.NodeID)
	}

	if result.DecisionTrace.NodeResults[0].RequestError == "" {
		t.Fatalf("expected request error to be preserved in decision trace")
	}
}

func TestRunAuctionWithSelectorInvokesNSGA3Skeleton(t *testing.T) {
	t.Parallel()

	selector, err := NewNSGA3Selector(nsga3.DefaultConfig(), map[string]auctioncore.NodeProfile{
		"node-a": auctioncore.ProfileForNodeClass(auctioncore.NodeClassHighEfficiency),
		"node-b": auctioncore.ProfileForNodeClass(auctioncore.NodeClassBalanced),
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	result, err := RunAuctionWithSelector(context.Background(), fakeRequester{
		bids: map[string]auctioncore.Bid{
			"node-a": {NodeID: "node-a", Accepted: true, CPUFragmentation: 0.20, RAMFragmentation: 0.20},
			"node-b": {NodeID: "node-b", Accepted: true, CPUFragmentation: 0.35, RAMFragmentation: 0.25},
		},
	}, selector, []string{"node-a", "node-b"}, auctioncore.DefaultTask("pod-c", 0.25, 0.40))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result.SelectionStrategy != "nsga3-first-pass" {
		t.Fatalf("expected nsga3-first-pass, got %s", result.SelectionStrategy)
	}

	if result.NSGA3Preparation == nil {
		t.Fatalf("expected nsga3 preparation trace")
	}

	if len(result.NSGA3Preparation.ReferencePoints) == 0 {
		t.Fatalf("expected reference points in nsga3 trace")
	}

	if result.NSGA3Preparation.SelectedCandidate == nil {
		t.Fatalf("expected selected candidate trace")
	}

	if len(result.NSGA3Preparation.Evaluations) != 2 {
		t.Fatalf("expected 2 candidate evaluations, got %d", len(result.NSGA3Preparation.Evaluations))
	}

	if result.DecisionTrace.NSGA3Preparation == nil {
		t.Fatalf("expected nsga3 preparation in decision trace")
	}

	if len(result.NSGA3Preparation.ObjectiveNames) != 4 {
		t.Fatalf("expected 4 objective names, got %d", len(result.NSGA3Preparation.ObjectiveNames))
	}
}

func TestRunAuctionWithSupervisorAppliesPolicySelector(t *testing.T) {
	t.Parallel()

	policy, err := governance.PolicyByName("balanced")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	supervisor, err := governance.NewStaticSupervisor(policy)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	result, err := RunAuctionWithSupervisor(
		context.Background(),
		fakeRequester{
			bids: map[string]auctioncore.Bid{
				"node-a": {NodeID: "node-a", Accepted: true, CPUFragmentation: 0.90, RAMFragmentation: 0.20},
				"node-b": {NodeID: "node-b", Accepted: true, CPUFragmentation: 0.70, RAMFragmentation: 0.70},
				"node-c": {NodeID: "node-c", Accepted: true, CPUFragmentation: 0.20, RAMFragmentation: 0.90},
			},
		},
		supervisor,
		NewSelectorFactory(nsga3.DefaultConfig(), map[string]auctioncore.NodeProfile{
			"node-a": auctioncore.ProfileForNodeClass(auctioncore.NodeClassHighPerformance),
			"node-b": auctioncore.ProfileForNodeClass(auctioncore.NodeClassBalanced),
			"node-c": auctioncore.ProfileForNodeClass(auctioncore.NodeClassHighEfficiency),
		}),
		map[string]auctioncore.NodeProfile{
			"node-a": auctioncore.ProfileForNodeClass(auctioncore.NodeClassHighPerformance),
			"node-b": auctioncore.ProfileForNodeClass(auctioncore.NodeClassBalanced),
			"node-c": auctioncore.ProfileForNodeClass(auctioncore.NodeClassHighEfficiency),
		},
		[]string{"node-a", "node-b", "node-c"},
		auctioncore.DefaultTask("pod-d", 0.25, 0.40),
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result.SelectionStrategy != "nsga3-first-pass" {
		t.Fatalf("expected nsga3-first-pass, got %s", result.SelectionStrategy)
	}

	if result.PolicyDecision == nil {
		t.Fatalf("expected applied policy decision")
	}

	if result.PolicyDecision.Policy.ID != "balanced-efficiency" {
		t.Fatalf("expected balanced-efficiency policy, got %s", result.PolicyDecision.Policy.ID)
	}

	if result.DecisionTrace.Policy == nil {
		t.Fatalf("expected decision trace policy")
	}

	if result.DecisionTrace.ObjectiveWeights.QoS != 1.0 {
		t.Fatalf("expected qos weight 1.0, got %.2f", result.DecisionTrace.ObjectiveWeights.QoS)
	}
}

func TestRunAuctionWithSupervisorTraceReflectsEffectivePolicyTask(t *testing.T) {
	t.Parallel()

	policy, err := governance.PolicyByName("black-friday")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	supervisor, err := governance.NewStaticSupervisor(policy)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	nodeProfiles := map[string]auctioncore.NodeProfile{
		"node-a": auctioncore.ProfileForNodeClass(auctioncore.NodeClassHighPerformance),
		"node-b": auctioncore.ProfileForNodeClass(auctioncore.NodeClassBalanced),
	}

	result, err := RunAuctionWithSupervisor(
		context.Background(),
		fakeRequester{
			bids: map[string]auctioncore.Bid{
				"node-a": {NodeID: "node-a", Accepted: true, CPUFragmentation: 0.60, RAMFragmentation: 0.45},
				"node-b": {NodeID: "node-b", Accepted: true, CPUFragmentation: 0.55, RAMFragmentation: 0.70},
			},
		},
		supervisor,
		NewSelectorFactory(nsga3.DefaultConfig(), nodeProfiles),
		nodeProfiles,
		[]string{"node-a", "node-b"},
		auctioncore.DefaultTask("pod-policy-trace", 0.25, 0.40),
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if result.DecisionTrace.TaskProfile != auctioncore.TaskProfilePerformanceBurst {
		t.Fatalf("expected performance-burst profile in trace, got %s", result.DecisionTrace.TaskProfile)
	}

	if result.DecisionTrace.ObjectiveWeights.QoS != 1.00 {
		t.Fatalf("expected qos weight 1.00 in trace, got %.2f", result.DecisionTrace.ObjectiveWeights.QoS)
	}

	if result.DecisionTrace.ObjectiveWeights.Energy != 0.45 {
		t.Fatalf("expected energy weight 0.45 in trace, got %.2f", result.DecisionTrace.ObjectiveWeights.Energy)
	}
}

func TestGovernedAuctionSmokePathWithGRPCNodeAgent(t *testing.T) {
	t.Parallel()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	auctionpb.RegisterContractNetServer(server, nodeagent.NewServer("node-alpha", auctioncore.Resources{
		CPU: 0.90,
		RAM: 0.90,
	}))

	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	requester := GRPCBidRequester{
		RequestTimeout: 0,
		DialOptions: []grpc.DialOption{
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
				return listener.Dial()
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
	}

	policy, err := governance.PolicyByName("balanced")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	supervisor, err := governance.NewStaticSupervisor(policy)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	nodeProfiles := map[string]auctioncore.NodeProfile{
		"node-alpha": auctioncore.ProfileForNodeClass(auctioncore.NodeClassBalanced),
	}

	result, err := RunAuctionWithSupervisor(
		context.Background(),
		requester,
		supervisor,
		NewSelectorFactory(nsga3.DefaultConfig(), nodeProfiles),
		nodeProfiles,
		[]string{"passthrough:///bufnet"},
		auctioncore.DefaultTask("pod-smoke", 0.25, 0.40),
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(result.NodeResults) != 1 {
		t.Fatalf("expected 1 node result, got %d", len(result.NodeResults))
	}

	if result.NodeResults[0].Err != nil {
		t.Fatalf("expected nil node error, got %v", result.NodeResults[0].Err)
	}

	if !result.NodeResults[0].Bid.Accepted {
		t.Fatalf("expected accepted bid, got %+v", result.NodeResults[0].Bid)
	}

	if !result.HasWinner {
		t.Fatalf("expected a winner, got strategy=%s trace=%+v", result.SelectionStrategy, result.DecisionTrace)
	}

	if result.DecisionTrace.Policy == nil {
		t.Fatalf("expected policy in decision trace")
	}

	if result.DecisionTrace.Policy.Policy.ID != "balanced-efficiency" {
		t.Fatalf("expected balanced-efficiency policy, got %s", result.DecisionTrace.Policy.Policy.ID)
	}

	if result.DecisionTrace.SelectionStrategy != "nsga3-first-pass" {
		t.Fatalf("expected nsga3-first-pass trace strategy, got %s", result.DecisionTrace.SelectionStrategy)
	}

	if len(result.DecisionTrace.Lines()) == 0 {
		t.Fatalf("expected visible trace lines")
	}
}
