package auction

type Resources struct {
	CPU float64
	RAM float64
}

type Task struct {
	ID         string
	CPUReqNorm float64
	RAMReqNorm float64
}

type Bid struct {
	NodeID           string
	Accepted         bool
	CPUFragmentation float64
	RAMFragmentation float64
}
