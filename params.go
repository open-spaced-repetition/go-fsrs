package fsrs

type Weights [13]float64

type Parameters struct {
	RequestRetention float64 `json:"RequestRetention"`
	MaximumInterval  float64 `json:"MaximumInterval"`
	EasyBonus        float64 `json:"EasyBonus"`
	HardFactor       float64 `json:"HardFactor"`
	W                Weights `json:"Weights"`
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
