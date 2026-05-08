package fsrs

import (
	"math"
	"time"
)

// FSRS implements the Free Spaced Repetition Scheduler algorithm.
// A zero-value FSRS is not usable; always construct instances with NewFSRS.
type FSRS struct {
	Parameters
}

// NewFSRS creates a new FSRS instance with the given parameters.
// All weights are clipped to valid ranges before validation. If validation
// fails after clipping, all parameters are reset to defaults.
func NewFSRS(param Parameters) *FSRS {
	clipParameters(&param)

	if param.Validate() != nil {
		param = DefaultParam()
	}

	param.Decay, param.Factor = param.decayAndFactor()

	return &FSRS{
		Parameters: param,
	}
}

func (f *FSRS) Repeat(card Card, now time.Time) RecordLog {
	return f.scheduler(card, now).Preview()
}

func (f *FSRS) Next(card Card, now time.Time, grade Rating) SchedulingInfo {
	return f.scheduler(card, now).Review(grade)
}

func (f *FSRS) GetRetrievability(card Card, now time.Time) float64 {
	if card.State == New || card.LastReview.IsZero() {
		return 0
	}
	elapsedDays := math.Max(0, dateDiffRaw(card.LastReview, now))
	return f.Parameters.ForgettingCurve(elapsedDays, card.Stability)
}
