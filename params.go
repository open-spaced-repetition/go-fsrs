package fsrs

import "math"

type Weights [17]float64

type Parameters struct {
	RequestRetention float64 `json:"RequestRetention"`
	MaximumInterval  float64 `json:"MaximumInterval"`
	W                Weights `json:"Weights"`
	Decay            float64 `json:"Decay"`
	Factor           float64 `json:"Factor"`
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
	}
}

func DefaultWeights() Weights {
	return Weights{0.5701, 1.4436, 4.1386, 10.9355, 5.1443, 1.2006, 0.8627, 0.0362, 1.629, 0.1342, 1.0166, 2.1174,
		0.0839, 0.3204, 1.4676, 0.219, 2.8237}
}
