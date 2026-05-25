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
	if len(config.ObjectiveNames) > 0 && len(config.ObjectiveNames) != config.Objectives {
		return Optimizer{}, fmt.Errorf("objective names count must match objectives")
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

	fronts, _ := buildFronts(candidates)
	activeReferencePoint := 0
	if len(referencePoints) > 0 {
		activeReferencePoint = selectBalancedReferencePoint(referencePoints, o.config.Objectives)
	}

	return Preparation{
		Config:               o.config,
		ObjectiveNames:       o.config.ObjectiveNames,
		Candidates:           candidates,
		ReferencePoints:      referencePoints,
		Fronts:               fronts,
		IdealPoint:           idealPoint(candidates, o.config.Objectives),
		ActiveReferencePoint: activeReferencePoint,
	}, nil
}

func (o Optimizer) Select(candidates []Candidate) (Selection, error) {
	preparation, err := o.Prepare(candidates)
	if err != nil {
		return Selection{}, err
	}

	evaluations := evaluateCandidates(candidates, preparation)
	preparation.Evaluations = evaluations
	selected, ok := chooseWinner(evaluations)
	if !ok {
		return Selection{
			Preparation: preparation,
		}, nil
	}

	preparation.SelectedCandidate = &selected

	return Selection{
		Preparation: preparation,
		Winner:      selected.Candidate,
		HasWinner:   true,
	}, nil
}
