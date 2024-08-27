package fsrs

import "math"

type Weights [19]float64

type Parameters struct {
	RequestRetention float64 `json:"RequestRetention"`
	MaximumInterval  float64 `json:"MaximumInterval"`
	W                Weights `json:"Weights"`
	Decay            float64 `json:"Decay"`
	Factor           float64 `json:"Factor"`
	EnableShortTerm  bool    `json:"EnableShortTerm"`
}

func DefaultParam() Parameters {
	var Decay = -0.5
	var Factor = math.Pow(0.9, 1/Decay) - 1
	return Parameters{
		RequestRetention: 0.9,
		MaximumInterval:  36500,
		W:                DefaultWeights(),
		Decay:            Decay,
		Factor:           Factor,
		EnableShortTerm:  true,
	}
}

func DefaultWeights() Weights {
	return Weights{0.4197,
		1.1869,
		3.0412,
		15.2441,
		7.1434,
		0.6477,
		1.0007,
		0.0674,
		1.6597,
		0.1712,
		1.1178,
		2.0225,
		0.0904,
		0.3025,
		2.1214,
		0.2498,
		2.9466,
		0.4891,
		0.6468}
}
