package fsrs

import (
	"math"
	"time"
)

type FSRS struct {
	Parameters
}

func NewFSRS(param Parameters) *FSRS {
	if param.Validate() != nil {
		param.W = DefaultWeights()
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
	return f.Parameters.forgettingCurve(elapsedDays, card.Stability)
}
