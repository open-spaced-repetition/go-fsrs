package fsrs

import (
	"fmt"
	"math"
)

// Weights holds the 21 FSRS v6 weight parameters used by the scheduler.
type Weights [21]float64

// DefaultWeights returns the default FSRS v6 weight values.
func DefaultWeights() Weights {
	return Weights{
		0.212,
		1.2931,
		2.3065,
		8.2956,
		6.4133,
		0.8334,
		3.0194,
		0.001,
		1.8722,
		0.1666,
		0.796,
		1.4835,
		0.0614,
		0.2629,
		1.6483,
		0.6014,
		1.8729,
		0.5425,
		0.0912,
		0.0658,
		0.1542,
	}
}

// ConvertV5Weights converts v5 [19]float64 weights to v6 [21]float64 weights.
// W[19] is set to 0.0 (no short-term stability decay in v5) and
// W[20] is set to 0.5 (v5 used a fixed decay of -0.5).
// Returns an error if any input parameter is non-finite (NaN or Inf).
func ConvertV5Weights(v5 [19]float64) (Weights, error) {
	if err := validateFiniteWeights(v5[:]); err != nil {
		return Weights{}, err
	}

	var w Weights
	copy(w[:19], v5[:])
	w[19] = 0.0
	w[20] = fsrs5DefaultDecay
	return w, nil
}

const fsrs5DefaultDecay = 0.5

func validateFiniteWeights(weights []float64) error {
	for i, val := range weights {
		if math.IsNaN(val) || math.IsInf(val, 0) {
			return &Error{Code: ErrCodeInvalidWeightsValue, Message: fmt.Sprintf("fsrs: invalid weight W[%d]: must be finite", i)}
		}
	}
	return nil
}

// ConvertV45Weights converts FSRS v4.5 [17]float64 weights to v6 [21]float64 weights.
// Returns an error if any input parameter is non-finite (NaN or Inf), or if the converted
// weights are non-finite.
func ConvertV45Weights(v45 [17]float64) (Weights, error) {
	if err := validateFiniteWeights(v45[:]); err != nil {
		return Weights{}, err
	}

	var w Weights
	copy(w[:17], v45[:])

	// Convert difficulty parameters matching fsrs-rs model.rs:319-325
	w[4] = w[5]*2.0 + w[4]
	w[5] = math.Log(w[5]*3.0+1.0) / 3.0
	w[6] += 0.5

	w[17] = 0.0
	w[18] = 0.0
	w[19] = 0.0
	w[20] = fsrs5DefaultDecay

	if err := validateFiniteWeights(w[:]); err != nil {
		return Weights{}, err
	}

	return w, nil
}

// MigrateWeights automatically detects the version of the input weights slice based
// on its length (17, 19, or 21) and converts it to the current FSRS v6 format.
// Returns an error if the length is invalid or if any weight is non-finite (NaN/Inf).
func MigrateWeights(weights []float64) (Weights, error) {
	switch len(weights) {
	case 17:
		var v45 [17]float64
		copy(v45[:], weights)
		return ConvertV45Weights(v45)
	case 19:
		var v5 [19]float64
		copy(v5[:], weights)
		return ConvertV5Weights(v5)
	case 21:
		if err := validateFiniteWeights(weights); err != nil {
			return Weights{}, err
		}
		var w Weights
		copy(w[:], weights)
		return w, nil
	default:
		return Weights{}, ErrInvalidWeightsLength
	}
}
