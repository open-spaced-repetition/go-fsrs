package fsrs

import (
	"math"
)

func (p *Parameters) forgettingCurve(elapsedDays float64, stability float64) float64 {
	return math.Pow(1+p.Factor*elapsedDays/stability, p.Decay)
}

func (p *Parameters) initStability(r Rating) float64 {
	return math.Max(p.W[r-1], 0.1)
}
func (p *Parameters) initDifficulty(r Rating) float64 {
	return constrainDifficulty(p.W[4] - math.Exp(p.W[5]*float64(r-1)) + 1)
}

func constrainDifficulty(d float64) float64 {
	return math.Min(math.Max(d, 1), 10)
}

func (p *Parameters) nextInterval(s float64) float64 {
	newInterval := s / p.Factor * (math.Pow(p.RequestRetention, 1/p.Decay) - 1)
	return math.Max(math.Min(math.Round(newInterval), p.MaximumInterval), 1)
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
