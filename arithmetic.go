package fsrs

import (
	"math"
)

const (
	sMin = 0.001
	sMax = 36500.0
	dMin = 1.0
	dMax = 10.0
)

func constrainDifficulty(d float64) float64 {
	return min(max(d, dMin), dMax)
}

func constrainStability(s float64) float64 {
	return min(max(s, sMin), sMax)
}

func linearDamping(deltaD float64, oldD float64) float64 {
	return (10.0 - oldD) * deltaD / 9.0
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

func (p *Parameters) initStability(r Rating) float64 {
	return min(max(p.W[r-1], 0.1), sMax)
}

func (p *Parameters) initDifficulty(r Rating) float64 {
	return constrainDifficulty(p.W[4] - math.Exp(p.W[5]*float64(r-1)) + 1)
}

func (p *Parameters) initDifficultyRaw(r Rating) float64 {
	return p.W[4] - math.Exp(p.W[5]*float64(r-1)) + 1
}

func (p *Parameters) nextInterval(s, elapsedDays float64) float64 {
	decay, factor := p.decayAndFactor()
	s = constrainStability(s)
	newInterval := s / factor * (math.Pow(p.RequestRetention, 1/decay) - 1)
	return p.ApplyFuzz(max(min(math.Round(newInterval), p.MaximumInterval), 1), elapsedDays, p.EnableFuzz)
}

func (p *Parameters) nextIntervalRaw(s float64) float64 {
	decay, factor := p.decayAndFactor()
	s = constrainStability(s)
	return s / factor * (math.Pow(p.RequestRetention, 1/decay) - 1)
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
	var sCeil float64
	if p.EnableShortTerm {
		sCeil = s / math.Exp(p.W[17]*p.W[18])
	} else {
		sCeil = s
	}
	return constrainStability(min(newS, sCeil))
}
