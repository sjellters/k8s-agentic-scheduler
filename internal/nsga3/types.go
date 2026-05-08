package nsga3

type Config struct {
	Objectives int
	Divisions  int
}

type Candidate struct {
	NodeID     string
	Objectives []float64
}

type ReferencePoint struct {
	Coordinates []float64
}

type Preparation struct {
	Config          Config
	Candidates      []Candidate
	ReferencePoints []ReferencePoint
}

func DefaultConfig() Config {
	return Config{
		Objectives: 2,
		Divisions:  4,
	}
}
