package fsrs

import (
	"math"
)

type Parameters struct {
	RequestRetention float64 `json:"RequestRetention"`
	MaximumInterval  float64 `json:"MaximumInterval"`
	W                Weights `json:"Weights"`
	Decay            float64 `json:"Decay"`
	Factor           float64 `json:"Factor"`
	EnableShortTerm  bool    `json:"EnableShortTerm"`
	EnableFuzz       bool    `json:"EnableFuzz"`
	seed             string
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
		EnableFuzz:       false,
	}
}

func (p *Parameters) forgettingCurve(elapsedDays float64, stability float64) float64 {
	return math.Pow(1+p.Factor*elapsedDays/stability, p.Decay)
}

func (p *Parameters) initStability(r Rating) float64 {
	return math.Max(p.W[r-1], 0.1)
}
func (p *Parameters) initDifficulty(r Rating) float64 {
	return constrainDifficulty(p.W[4] - math.Exp(p.W[5]*float64(r-1)) + 1)
}

func (p *Parameters) ApplyFuzz(ivl float64, elapsedDays float64, enableFuzz bool) float64 {
	if !enableFuzz || ivl < 2.5 {
		return ivl
	}

	generator := Alea(p.seed)
	fuzzFactor := generator.Double()

	minIvl, maxIvl := getFuzzRange(ivl, elapsedDays, p.MaximumInterval)

	return math.Floor(fuzzFactor*float64(maxIvl-minIvl+1)) + float64(minIvl)
}

func constrainDifficulty(d float64) float64 {
	return math.Min(math.Max(d, 1), 10)
}

func (p *Parameters) nextInterval(s, elapsedDays float64) float64 {
	newInterval := s / p.Factor * (math.Pow(p.RequestRetention, 1/p.Decay) - 1)
	return p.ApplyFuzz(math.Max(math.Min(math.Round(newInterval), p.MaximumInterval), 1), elapsedDays, p.EnableFuzz)
}

func (p *Parameters) nextDifficulty(d float64, r Rating) float64 {
	nextD := d - p.W[6]*float64(r-3)
	return constrainDifficulty(p.meanReversion(p.initDifficulty(Easy), nextD))
}

func (p *Parameters) shortTermStability(s float64, r Rating) float64 {
	return s * math.Exp(p.W[17]*(float64(r-3)+p.W[18]))
}

func (p *Parameters) meanReversion(init float64, current float64) float64 {
	return p.W[7]*init + (1-p.W[7])*current
}

func (p *Parameters) nextRecallStability(d float64, s float64, r float64, rating Rating) float64 {
	var hardPenalty, easyBonus float64
	if rating == Hard {
		hardPenalty = p.W[15]
	} else {
		hardPenalty = 1
	}
	if rating == Easy {
		easyBonus = p.W[16]
	} else {
		easyBonus = 1
	}
	return s * (1 + math.Exp(p.W[8])*
		(11-d)*
		math.Pow(s, -p.W[9])*
		(math.Exp((1-r)*p.W[10])-1)*
		hardPenalty*
		easyBonus)
}

func (p *Parameters) nextForgetStability(d float64, s float64, r float64) float64 {
	return p.W[11] *
		math.Pow(d, -p.W[12]) *
		(math.Pow(s+1, p.W[13]) - 1) *
		math.Exp((1-r)*p.W[14])
}

type FuzzRange struct {
	Start  float64
	End    float64
	Factor float64
}

var FUZZ_RANGES = []FuzzRange{
	{Start: 2.5, End: 7.0, Factor: 0.15},
	{Start: 7.0, End: 20.0, Factor: 0.1},
	{Start: 20.0, End: math.Inf(1), Factor: 0.05},
}

func getFuzzRange(interval, elapsedDays, maximumInterval float64) (minIvl, maxIvl int) {
	delta := 1.0
	for _, r := range FUZZ_RANGES {
		delta += r.Factor * math.Max(math.Min(interval, r.End)-r.Start, 0.0)
	}

	interval = math.Min(interval, maximumInterval)
	minIvlFloat := math.Max(2, math.Round(interval-delta))
	maxIvlFloat := math.Min(math.Round(interval+delta), maximumInterval)

	if interval > elapsedDays {
		minIvlFloat = math.Max(minIvlFloat, elapsedDays+1)
	}
	minIvlFloat = math.Min(minIvlFloat, maxIvlFloat)

	minIvl = int(minIvlFloat)
	maxIvl = int(maxIvlFloat)

	return minIvl, maxIvl
}
