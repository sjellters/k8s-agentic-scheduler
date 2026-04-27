package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	auctioncore "github.com/diego/k8s-agentic-scheduler/internal/auction"
	agentserver "github.com/diego/k8s-agentic-scheduler/internal/nodeagent"
	auctionpb "github.com/diego/k8s-agentic-scheduler/proto"
	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("port", 50051, "The server port")
	nodeID := flag.String("id", "node-1", "The Node ID")
	cpu := flag.Float64("cpu", 1.0, "Node CPU capacity (0.0 - 1.0)")
	ram := flag.Float64("ram", 1.0, "Node RAM capacity (0.0 - 1.0)")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	auctionpb.RegisterContractNetServer(s, agentserver.NewServer(*nodeID, auctioncore.Resources{
		CPU: *cpu,
		RAM: *ram,
	}))

	log.Printf("==========================================")
	log.Printf(" NODE AGENT ACTIVE")
	log.Printf(" ID:   %s", *nodeID)
	log.Printf(" PORT: %d", *port)
	log.Printf(" CAP:  CPU %.2f | RAM %.2f", *cpu, *ram)
	log.Printf("==========================================")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
