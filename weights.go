package fsrs

type Weights [21]float64

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
func ConvertV5Weights(v5 [19]float64) Weights {
	var w Weights
	copy(w[:19], v5[:])
	w[19] = 0.0
	w[20] = 0.5
	return w
}
