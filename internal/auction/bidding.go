package auction

func EvaluateBid(nodeID string, capacity Resources, task Task) Bid {
	if task.CPUReqNorm > capacity.CPU || task.RAMReqNorm > capacity.RAM {
		return Bid{
			NodeID:   nodeID,
			Accepted: false,
		}
	}

	return Bid{
		NodeID:           nodeID,
		Accepted:         true,
		CPUFragmentation: capacity.CPU - task.CPUReqNorm,
		RAMFragmentation: capacity.RAM - task.RAMReqNorm,
	}
}

func (b Bid) Score() float64 {
	if !b.Accepted {
		return 0
	}

	return b.CPUFragmentation + b.RAMFragmentation
}
