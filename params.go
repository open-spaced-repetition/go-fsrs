package fsrs

type Weights [17]float64

type Parameters struct {
	RequestRetention float64 `json:"RequestRetention"`
	MaximumInterval  float64 `json:"MaximumInterval"`
	W                Weights `json:"Weights"`
}

func DefaultParam() Parameters {
	return Parameters{
		RequestRetention: 0.9,
		MaximumInterval:  36500,
		W:                DefaultWeights(),
	}
}

func DefaultWeights() Weights {
	return Weights{0.4, 0.6, 2.4, 5.8, 4.93, 0.94, 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61}
}
