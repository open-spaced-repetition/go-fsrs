package fsrs

type Weights [13]float64

type Parameters struct {
	RequestRetention float64
	MaximumInterval  float64
	EasyBonus        float64
	HardFactor       float64
	W                Weights
}

func DefaultParam() Parameters {
	return Parameters{
		RequestRetention: 0.9,
		MaximumInterval:  36500,
		EasyBonus:        1.3,
		HardFactor:       1.2,
		W:                DefaultWeights(),
	}
}

func DefaultWeights() Weights {
	return Weights{1, 1, 5, -0.5, -0.5, 0.2, 1.4, -0.12, 0.8, 2, -0.2, 0.2, 1}
}
