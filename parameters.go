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
		newD = constrainDifficulty(p.initDifficulty(grade))
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
