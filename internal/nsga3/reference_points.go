package nsga3

import "fmt"

func GenerateReferencePoints(objectives, divisions int) ([]ReferencePoint, error) {
	if objectives <= 0 {
		return nil, fmt.Errorf("objectives must be positive")
	}
	if divisions <= 0 {
		return nil, fmt.Errorf("divisions must be positive")
	}

	referencePoints := make([]ReferencePoint, 0)
	current := make([]float64, objectives)

	var build func(index, remaining int)
	build = func(index, remaining int) {
		if index == objectives-1 {
			current[index] = float64(remaining) / float64(divisions)

			coordinates := make([]float64, len(current))
			copy(coordinates, current)

			referencePoints = append(referencePoints, ReferencePoint{
				Coordinates: coordinates,
			})
			return
		}

		for value := 0; value <= remaining; value++ {
			current[index] = float64(value) / float64(divisions)
			build(index+1, remaining-value)
		}
	}

	build(0, divisions)

	return referencePoints, nil
}
