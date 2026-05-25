# k8s-agentic-scheduler

A distributed Kubernetes scheduling MVP built in Go. Current implementation focuses on a gRPC auction baseline for container placement across heterogeneous nodes.

## Architecture

- **Hot-Path (implemented):** A Go scheduling baseline that uses the **Contract Net Protocol (CNP)** for resource auctions and baseline winner selection based on remaining normalized CPU/RAM capacity.
- **Optimization layer (first working pass):** A native **NSGA-III** package now performs candidate preparation, nondominated front construction, balanced reference-point guidance, and a first optimizer-driven winner selection pass across four objectives: CPU residual, memory residual, QoS convenience, and energy convenience.
- **Cold-Path (implemented boundary):** An out-of-band **Supervisor** injects a scheduling policy and records a structured decision trace outside the critical scheduling loop.

## Thesis Comparison Levels

The Delivery 6 thesis-facing comparison contract is frozen around three levels:

1. **Baseline**: aggregate residual CPU/RAM winner selection.
2. **Optimizer without policy supervision**: direct NSGA-III first-pass selection.
3. **Optimizer with policy supervision**: supervisor-governed NSGA-III selection with explicit policy injection and structured trace retention.

## Project Structure

The repository is organized following standard Go patterns:

- `cmd/`: Application entry points. Each subdirectory corresponds to a standalone binary.
    - `cmd/manager/`: The **Manager Agent**. Orchestrates pod scheduling, manages the auction process, and can select a winner with either the baseline scorer or the NSGA-III first pass.
    - `cmd/node_agent/`: The **Node Agent**. Runs on worker nodes, evaluates local resource availability, and participates in auctions.
- `internal/`: Shared internal packages for implemented and planned scheduler logic.
    - `internal/auction/`: Current auction-domain logic for bid evaluation, task profiles, node classes, and baseline winner selection.
    - `internal/governance/`: Supervisor-side policy injection and decision metadata.
    - `internal/nsga3/`: Native NSGA-III implementation with candidate preparation, nondominated fronts, reference points, and selection trace data.
- `proto/`: Protocol Buffers definitions and generated Go code.
    - `auction.proto`: Defines the `ContractNet` gRPC service for bidding.

## Execution Flow

The system operates as a distributed auction-based scheduler:

1.  **Deployment**: Several `node_agent` instances are started on worker nodes.
2.  **Auction Initialization**: The `manager` is invoked with a target pod request.
3.  **Request for Bids (gRPC)**: The Manager broadcasts a `TaskRequest` concurrently to all Node Agents.
4.  **Bidding Logic**: Each Node Agent evaluates capacity, computes fragmentation ($f_1$, $f_3$), and returns a `BidResponse`.
5.  **Baseline Winner Selection**: The Manager can keep the highest combined remaining CPU/RAM score as an explicit comparison path.
6.  **Policy Injection**: Before final award, the Supervisor can choose which selection strategy governs the accepted bids and how the four objective weights are prioritized.
7.  **NSGA-III First Pass**: The Manager can transform accepted bids into four-objective optimizer candidates, build nondominated fronts, pick the balanced reference point, and select the winner from the best front.
8.  **Decision Trace**: The Manager records policy, bids, selector evidence, and the winning node in a structured trace; Kubernetes API binding is not implemented in this MVP yet.

## Development

### Prerequisites
- Go 1.26.1
- Protocol Buffers Compiler (`protoc`)
- Go gRPC plugins (`protoc-gen-go`, `protoc-gen-go-grpc`)

### Getting Started
```bash
# Generate gRPC code
export PATH=$PATH:$(go env GOPATH)/bin
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/auction.proto

# Start a node agent
go run cmd/node_agent/main.go --port 50051 --id node-alpha

# Run manager
go run cmd/manager/main.go --nodes "localhost:50051"

# Run manager with the NSGA-III first-pass trace
go run cmd/manager/main.go --nodes "localhost:50051" --selector nsga3

# Run manager with supervisor-driven policy injection and JSON trace output
go run cmd/manager/main.go --nodes "localhost:50051" --policy balanced --trace-format json

# Run manager with a node-class profile and QoS-sensitive policy
go run cmd/manager/main.go --nodes "localhost:50051" --node-profiles "node-alpha=high-performance" --task-qos 0.8 --policy black-friday --trace-format json

# Run the Delivery 6 smoke test for the governed gRPC auction path
go test ./internal/manager -run TestGovernedAuctionSmokePathWithGRPCNodeAgent
```
