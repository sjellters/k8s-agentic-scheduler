package nsga3

type Config struct {
	Objectives     int
	Divisions      int
	ObjectiveNames []string
}

type Candidate struct {
	NodeID     string
	Objectives []float64
}

type ReferencePoint struct {
	Coordinates []float64
}

type Front struct {
	Rank             int
	CandidateNodeIDs []string
}

type CandidateEvaluation struct {
	Candidate            Candidate
	FrontRank            int
	NormalizedObjectives []float64
	ReferencePointIndex  int
	Distance             float64
}

type Preparation struct {
	Config               Config
	ObjectiveNames       []string
	Candidates           []Candidate
	ReferencePoints      []ReferencePoint
	Fronts               []Front
	IdealPoint           []float64
	ActiveReferencePoint int
	Evaluations          []CandidateEvaluation
	SelectedCandidate    *CandidateEvaluation
}

type Selection struct {
	Preparation Preparation
	Winner      Candidate
	HasWinner   bool
}

func DefaultConfig() Config {
	return Config{
		Objectives: 4,
		Divisions:  4,
		ObjectiveNames: []string{
			"cpu_residual",
			"memory_residual",
			"qos_convenience",
			"energy_convenience",
		},
	}
}
