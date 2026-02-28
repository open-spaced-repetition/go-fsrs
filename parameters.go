package fsrs

import (
	"fmt"
	"math"
)

const (
	sMin = 0.001
	sMax = 36500.0
	dMin = 1.0
	dMax = 10.0
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
	w := DefaultWeights()
	p := Parameters{
		RequestRetention: 0.9,
		MaximumInterval:  36500,
		W:                w,
		EnableShortTerm:  true,
		EnableFuzz:       false,
	}
	p.Decay, p.Factor = p.decayAndFactor()
	return p
}

func (p *Parameters) Validate() error {
	for i, w := range p.W {
		if math.IsNaN(w) || math.IsInf(w, 0) {
			return fmt.Errorf("fsrs: invalid weight W[%d]: must be finite", i)
		}
	}

	if p.W[20] <= 0 {
		return fmt.Errorf("fsrs: invalid weight W[20]: must be > 0")
	}

	return nil
}

func defaultDecayAndFactor() (float64, float64) {
	w := DefaultWeights()
	decay := -w[20]
	factor := math.Pow(0.9, 1.0/decay) - 1.0
	return decay, factor
}

func (p *Parameters) decayAndFactor() (float64, float64) {
	if p.Validate() != nil {
		return defaultDecayAndFactor()
	}

	decay := -p.W[20]
	factor := math.Pow(0.9, 1.0/decay) - 1.0

	if math.IsNaN(factor) || math.IsInf(factor, 0) || factor == 0 {
		return defaultDecayAndFactor()
	}

	return decay, factor
}

func (p *Parameters) forgettingCurve(elapsedDays float64, stability float64) float64 {
	decay, factor := p.decayAndFactor()
	stability = constrainStability(stability)
	return math.Pow(1+factor*elapsedDays/stability, decay)
}

func (p *Parameters) initStability(r Rating) float64 {
	return constrainStability(p.W[r-1])
}

func (p *Parameters) initDifficulty(r Rating) float64 {
	return constrainDifficulty(p.W[4] - math.Exp(p.W[5]*float64(r-1)) + 1)
}

func (p *Parameters) initDifficultyRaw(r Rating) float64 {
	return p.W[4] - math.Exp(p.W[5]*float64(r-1)) + 1
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
	return math.Min(math.Max(d, dMin), dMax)
}

func constrainStability(s float64) float64 {
	return math.Min(math.Max(s, sMin), sMax)
}

func linearDamping(deltaD float64, oldD float64) float64 {
	return (10.0 - oldD) * deltaD / 9.0
}

func (p *Parameters) nextInterval(s, elapsedDays float64) float64 {
	decay, factor := p.decayAndFactor()
	s = constrainStability(s)
	newInterval := s / factor * (math.Pow(p.RequestRetention, 1/decay) - 1)
	return p.ApplyFuzz(math.Max(math.Min(math.Round(newInterval), p.MaximumInterval), 1), elapsedDays, p.EnableFuzz)
}

func (p *Parameters) nextDifficulty(d float64, r Rating) float64 {
	deltaD := -p.W[6] * float64(r-3)
	nextD := d + linearDamping(deltaD, d)
	return constrainDifficulty(p.meanReversion(p.initDifficultyRaw(Easy), nextD))
}

func (p *Parameters) shortTermStability(s float64, r Rating) float64 {
	s = constrainStability(s)
	sinc := math.Exp(p.W[17]*(float64(r-3)+p.W[18])) * math.Pow(s, -p.W[19])
	if r >= Hard && sinc < 1.0 {
		sinc = 1.0
	}
	return constrainStability(s * sinc)
}

func (p *Parameters) meanReversion(init float64, current float64) float64 {
	return p.W[7]*init + (1-p.W[7])*current
}

func (p *Parameters) nextRecallStability(d float64, s float64, r float64, rating Rating) float64 {
	d = constrainDifficulty(d)
	s = constrainStability(s)

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
	newS := s * (1 + math.Exp(p.W[8])*
		(11-d)*
		math.Pow(s, -p.W[9])*
		(math.Exp((1-r)*p.W[10])-1)*
		hardPenalty*
		easyBonus)

	return constrainStability(newS)
}

func (p *Parameters) nextForgetStability(d float64, s float64, r float64) float64 {
	d = constrainDifficulty(d)
	s = constrainStability(s)

	newS := p.W[11] *
		math.Pow(d, -p.W[12]) *
		(math.Pow(s+1, p.W[13]) - 1) *
		math.Exp((1-r)*p.W[14])
	sCeil := s / math.Exp(p.W[17]*p.W[18])
	return constrainStability(math.Min(newS, sCeil))
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
