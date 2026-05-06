package fsrs

import (
	"fmt"
	"math"
)

var DefaultLearningSteps = []float64{1, 10}

var DefaultRelearningSteps = []float64{10}

type Parameters struct {
	RequestRetention float64   `json:"RequestRetention"`
	MaximumInterval  float64   `json:"MaximumInterval"`
	W                Weights   `json:"Weights"`
	Decay            float64   `json:"Decay"`
	Factor           float64   `json:"Factor"`
	EnableShortTerm  bool      `json:"EnableShortTerm"`
	EnableFuzz       bool      `json:"EnableFuzz"`
	LearningSteps    []float64 `json:"LearningSteps"`
	RelearningSteps  []float64 `json:"RelearningSteps"`
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
		LearningSteps:    DefaultLearningSteps,
		RelearningSteps:  DefaultRelearningSteps,
	}
	p.Decay, p.Factor = p.decayAndFactor()
	return p
}

// Validate checks that all parameters are within valid ranges. It verifies:
// weights are finite and W[20] > 0, RequestRetention is in (0, 1],
// MaximumInterval is in (0, 106000], and LearningSteps/RelearningSteps
// contain only finite non-negative values.
func (p *Parameters) Validate() error {
	for i, w := range p.W {
		if math.IsNaN(w) || math.IsInf(w, 0) {
			return fmt.Errorf("fsrs: invalid weight W[%d]: must be finite", i)
		}
	}

	if p.W[20] <= 0 {
		return fmt.Errorf("fsrs: invalid weight W[20]: must be > 0")
	}

	if math.IsNaN(p.RequestRetention) || math.IsInf(p.RequestRetention, 0) ||
		p.RequestRetention <= 0 || p.RequestRetention > 1 {
		return fmt.Errorf("fsrs: invalid RequestRetention: must be in (0, 1], got %v", p.RequestRetention)
	}

	if math.IsNaN(p.MaximumInterval) || math.IsInf(p.MaximumInterval, 0) ||
		p.MaximumInterval <= 0 || p.MaximumInterval > 106000 {
		return fmt.Errorf("fsrs: invalid MaximumInterval: must be in (0, 106000], got %v", p.MaximumInterval)
	}

	for i, s := range p.LearningSteps {
		if math.IsNaN(s) || math.IsInf(s, 0) || s < 0 {
			return fmt.Errorf("fsrs: invalid LearningSteps[%d]: must be finite and >= 0, got %v", i, s)
		}
	}

	for i, s := range p.RelearningSteps {
		if math.IsNaN(s) || math.IsInf(s, 0) || s < 0 {
			return fmt.Errorf("fsrs: invalid RelearningSteps[%d]: must be finite and >= 0, got %v", i, s)
		}
	}

	return nil
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(v, hi))
}

func clipParameters(p *Parameters) {
	const initSMax = 100.0
	const sMin = 0.001
	const (
		wFailStabMult = 11
		wFailStabPow  = 13
		wFailStabExp  = 14
	)

	w17W18Ceiling := 2.0
	numRelearning := len(p.RelearningSteps)
	if numRelearning > 1 {
		w11 := clamp(p.W[wFailStabMult], 0.001, 5.0)
		w13 := clamp(p.W[wFailStabPow], 0.001, 0.9)
		w14 := clamp(p.W[wFailStabExp], 0.0, 4.0)
		value := -(
			math.Log(w11) +
				math.Log(math.Pow(2.0, w13)-1.0) +
				w14*0.3) /
			float64(numRelearning)
		w17W18Ceiling = clamp(value, 0.01, 2.0)
	}

	w19Min := 0.0
	if p.EnableShortTerm {
		w19Min = 0.01
	}

	clampRanges := [21][2]float64{
		{sMin, initSMax}, {sMin, initSMax}, {sMin, initSMax}, {sMin, initSMax},
		{1.0, 10.0},
		{0.001, 4.0}, {0.001, 4.0},
		{0.001, 0.75},
		{0.0, 4.5},
		{0.0, 0.8},
		{0.001, 3.5},
		{0.001, 5.0},
		{0.001, 0.25},
		{0.001, 0.9},
		{0.0, 4.0},
		{0.0, 1.0},
		{1.0, 6.0},
		{0.0, w17W18Ceiling}, {0.0, w17W18Ceiling},
		{w19Min, 0.8},
		{0.1, 0.8},
	}

	for i := range p.W {
		p.W[i] = clamp(p.W[i], clampRanges[i][0], clampRanges[i][1])
	}
}

func (p *Parameters) ForgettingCurve(elapsedDays float64, stability float64) float64 {
	decay, factor := p.decayAndFactor()
	stability = constrainStability(stability)
	return math.Pow(1+factor*elapsedDays/stability, decay)
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
