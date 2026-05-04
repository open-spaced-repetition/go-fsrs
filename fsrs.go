package fsrs

import (
	"errors"
	"math"
	"time"
)

var (
	ErrManualRating  = errors.New("fsrs: cannot rollback a manual rating")
	ErrInvalidRating = errors.New("fsrs: invalid rating for rollback")
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
	return f.Parameters.ForgettingCurve(elapsedDays, card.Stability)
}

func (f *FSRS) Forget(card Card, now time.Time, resetCount bool) SchedulingInfo {
	forgetCard := Card{
		Due:   now,
		State: New,
	}
	if !resetCount {
		forgetCard.Reps = card.Reps
		forgetCard.Lapses = card.Lapses
	}
	log := ReviewLog{
		Rating: Manual,
		State:  New,
		Due:    now,
		Review: now,
	}
	return SchedulingInfo{Card: forgetCard, ReviewLog: log}
}

func (f *FSRS) Rollback(card Card, log ReviewLog) (Card, error) {
	if log.Rating == Manual {
		return Card{}, ErrManualRating
	}
	if log.Rating < Again || log.Rating > Easy {
		return Card{}, ErrInvalidRating
	}
	result := card
	result.State = log.State
	result.Due = log.Due
	result.Stability = log.Stability
	result.Difficulty = log.Difficulty
	result.ElapsedDays = log.ElapsedDays
	result.ScheduledDays = log.ScheduledDays
	result.LastReview = log.Review
	if card.Reps > 0 {
		result.Reps = card.Reps - 1
	}
	if log.Rating == Again && card.Lapses > 0 {
		result.Lapses = card.Lapses - 1
	}
	return result, nil
}
