package fsrs

import (
	"fmt"
	"math"
	"time"
)

const (
	sMin = 0.001
	sMax = 36500.0
	dMin = 1.0
	dMax = 10.0
)

var DefaultLearningSteps = []float64{1, 10}

var DefaultRelearningSteps = []float64{10}

func dateDiffInDays(last, cur time.Time) uint64 {
	lr := last.UTC()
	utc1 := time.Date(lr.Year(), lr.Month(), lr.Day(), 0, 0, 0, 0, time.UTC)
	n := cur.UTC()
	utc2 := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	hours := utc2.Sub(utc1).Hours()
	if hours < 0 {
		return 0
	}
	return uint64(math.Floor(hours / 24))
}

func dateDiffRaw(last, cur time.Time) float64 {
	return math.Floor(cur.Sub(last).Hours() / 24)
}

type Parameters struct {
	RequestRetention  float64   `json:"RequestRetention"`
	MaximumInterval   float64   `json:"MaximumInterval"`
	W                 Weights   `json:"Weights"`
	Decay             float64   `json:"Decay"`
	Factor            float64   `json:"Factor"`
	EnableShortTerm   bool      `json:"EnableShortTerm"`
	EnableFuzz        bool      `json:"EnableFuzz"`
	LearningSteps     []float64 `json:"LearningSteps"`
	RelearningSteps   []float64 `json:"RelearningSteps"`
	seed              string
}

func DefaultParam() Parameters {
	w := DefaultWeights()
	p := Parameters{
		RequestRetention:  0.9,
		MaximumInterval:   36500,
		W:                 w,
		EnableShortTerm:   true,
		EnableFuzz:        false,
		LearningSteps:     DefaultLearningSteps,
		RelearningSteps:   DefaultRelearningSteps,
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

func (p *Parameters) ForgettingCurve(elapsedDays float64, stability float64) float64 {
	decay, factor := p.decayAndFactor()
	stability = constrainStability(stability)
	return math.Pow(1+factor*elapsedDays/stability, decay)
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
	return min(max(d, dMin), dMax)
}

func constrainStability(s float64) float64 {
	return min(max(s, sMin), sMax)
}

func linearDamping(deltaD float64, oldD float64) float64 {
	return (10.0 - oldD) * deltaD / 9.0
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

func (p *Parameters) NextState(current *MemoryState, desiredRetention float64, daysElapsed uint64, grade Rating) ItemState {
	decay, factor := p.decayAndFactor()
	return p.nextStateInner(current, desiredRetention, float64(daysElapsed), grade, decay, factor)
}

func (p *Parameters) NextStates(current *MemoryState, desiredRetention float64, daysElapsed uint64) NextStates {
	decay, factor := p.decayAndFactor()
	elapsed := float64(daysElapsed)
	return NextStates{
		Again: p.nextStateInner(current, desiredRetention, elapsed, Again, decay, factor),
		Hard:  p.nextStateInner(current, desiredRetention, elapsed, Hard, decay, factor),
		Good:  p.nextStateInner(current, desiredRetention, elapsed, Good, decay, factor),
		Easy:  p.nextStateInner(current, desiredRetention, elapsed, Easy, decay, factor),
	}
}

func (p *Parameters) nextStateInner(current *MemoryState, desiredRetention, elapsed float64, grade Rating, decay, factor float64) ItemState {
	var newS, newD float64

	if current == nil || current.Stability == 0 {
		newS = p.initStability(grade)
		newD = p.initDifficulty(grade)
	} else {
		newD = p.nextDifficulty(current.Difficulty, grade)
		if elapsed == 0 && p.EnableShortTerm {
			newS = p.shortTermStability(current.Stability, grade)
		} else {
			retrievability := p.ForgettingCurve(elapsed, current.Stability)
			if grade == Again {
				newS = p.nextForgetStability(current.Difficulty, current.Stability, retrievability)
			} else {
				newS = p.nextRecallStability(current.Difficulty, current.Stability, retrievability, grade)
			}
		}
	}

	newS = constrainStability(newS)
	interval := newS / factor * (math.Pow(desiredRetention, 1/decay) - 1)

	return ItemState{
		Memory:   MemoryState{Stability: newS, Difficulty: newD},
		Interval: interval,
	}
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
		delta += r.Factor * max(min(interval, r.End)-r.Start, 0.0)
	}

	interval = min(interval, maximumInterval)
	minIvlFloat := max(2.0, math.Round(interval-delta))
	maxIvlFloat := min(math.Round(interval+delta), maximumInterval)

	if interval > elapsedDays {
		minIvlFloat = max(minIvlFloat, elapsedDays+1)
	}
	minIvlFloat = min(minIvlFloat, maxIvlFloat)

	minIvl = int(minIvlFloat)
	maxIvl = int(maxIvlFloat)

	return minIvl, maxIvl
}

func (p *Parameters) againDelayMinutes(steps []float64) float64 {
	if len(steps) == 0 {
		return 0
	}
	return steps[0]
}

func (p *Parameters) hardDelayMinutes(steps []float64) float64 {
	if len(steps) == 0 {
		return 0
	}
	first := steps[0]
	if len(steps) == 1 {
		return math.Round(first * 1.5)
	}
	return math.Round((first + steps[1]) / 2)
}

func (p *Parameters) goodDelayMinutes(steps []float64, remaining int) (float64, bool) {
	nextIdx := len(steps) - remaining + 1
	if nextIdx < 0 || nextIdx >= len(steps) {
		return 0, false
	}
	return steps[nextIdx], true
}
