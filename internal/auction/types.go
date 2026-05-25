package auction

type Resources struct {
	CPU float64
	RAM float64
}

type TaskProfile string

const (
	TaskProfileBalanced         TaskProfile = "balanced"
	TaskProfilePerformanceBurst TaskProfile = "performance-burst"
	TaskProfileEnergySaver      TaskProfile = "energy-saver"
)

type ObjectiveWeights struct {
	CPU    float64
	RAM    float64
	QoS    float64
	Energy float64
}

type Task struct {
	ID             string
	CPUReqNorm     float64
	RAMReqNorm     float64
	QoSSensitivity float64
	ObjectiveSet   ObjectiveWeights
	Profile        TaskProfile
}

type Bid struct {
	NodeID           string
	Accepted         bool
	CPUFragmentation float64
	RAMFragmentation float64
}

type NodeClass string

const (
	NodeClassHighPerformance NodeClass = "high-performance"
	NodeClassBalanced        NodeClass = "balanced"
	NodeClassHighEfficiency  NodeClass = "high-efficiency"
)

type NodeProfile struct {
	Class         NodeClass
	QoSBias       float64
	EnergyPenalty float64
}

func DefaultObjectiveWeights() ObjectiveWeights {
	return ObjectiveWeights{
		CPU:    1.0,
		RAM:    1.0,
		QoS:    1.0,
		Energy: 1.0,
	}
}

func DefaultTask(id string, cpuReqNorm, ramReqNorm float64) Task {
	return Task{
		ID:             id,
		CPUReqNorm:     cpuReqNorm,
		RAMReqNorm:     ramReqNorm,
		QoSSensitivity: 0.5,
		ObjectiveSet:   DefaultObjectiveWeights(),
		Profile:        TaskProfileBalanced,
	}
}

func (t Task) NormalizedLoad() float64 {
	return (t.CPUReqNorm + t.RAMReqNorm) / 2
}

func DefaultNodeProfile() NodeProfile {
	return ProfileForNodeClass(NodeClassBalanced)
}

func ProfileForNodeClass(class NodeClass) NodeProfile {
	switch class {
	case NodeClassHighPerformance:
		return NodeProfile{
			Class:         NodeClassHighPerformance,
			QoSBias:       0.10,
			EnergyPenalty: 0.85,
		}
	case NodeClassHighEfficiency:
		return NodeProfile{
			Class:         NodeClassHighEfficiency,
			QoSBias:       0.35,
			EnergyPenalty: 0.30,
		}
	default:
		return NodeProfile{
			Class:         NodeClassBalanced,
			QoSBias:       0.20,
			EnergyPenalty: 0.55,
		}
	}
}
