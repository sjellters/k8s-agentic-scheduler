package nsga3

import "fmt"

type Optimizer struct {
	config Config
}

func New(config Config) (Optimizer, error) {
	if config.Objectives <= 0 {
		return Optimizer{}, fmt.Errorf("objectives must be positive")
	}
	if config.Divisions <= 0 {
		return Optimizer{}, fmt.Errorf("divisions must be positive")
	}

	return Optimizer{config: config}, nil
}

func (o Optimizer) Prepare(candidates []Candidate) (Preparation, error) {
	if err := validateCandidates(candidates, o.config.Objectives); err != nil {
		return Preparation{}, err
	}

	referencePoints, err := GenerateReferencePoints(o.config.Objectives, o.config.Divisions)
	if err != nil {
		return Preparation{}, err
	}

	return Preparation{
		Config:          o.config,
		Candidates:      candidates,
		ReferencePoints: referencePoints,
	}, nil
}
