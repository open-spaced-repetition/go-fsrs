package fsrs

import (
	"fmt"
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

// Repeat previews the scheduling result for all four ratings (Again, Hard, Good, Easy)
// without modifying any card state. Returns a RecordLog keyed by Rating.
// Returns an error if the card or computed results are invalid.
func (f *FSRS) Repeat(card Card, now time.Time) (RecordLog, error) {
	if err := validateCard(card, now); err != nil {
		return RecordLog{}, err
	}
	log := f.scheduler(card, now).Preview()
	for _, rating := range []Rating{Again, Hard, Good, Easy} {
		if err := validateResult(log[rating].Card); err != nil {
			return RecordLog{}, err
		}
	}
	return log, nil
}

// Next applies a single review with the given grade and returns the updated card
// and its review log. Returns an error if the grade, card, or computed result is invalid.
func (f *FSRS) Next(card Card, now time.Time, grade Rating) (SchedulingInfo, error) {
	if err := validateRating(grade); err != nil {
		return SchedulingInfo{}, err
	}
	if err := validateCard(card, now); err != nil {
		return SchedulingInfo{}, err
	}
	info := f.scheduler(card, now).Review(grade)
	if err := validateResult(info.Card); err != nil {
		return SchedulingInfo{}, err
	}
	return info, nil
}

// Retrievability returns the current retrievability (probability of recall) for
// the given card at the specified time. Returns 0 for New cards or cards with no
// LastReview. Returns an error if the card state or stability is invalid.
func (f *FSRS) Retrievability(card Card, now time.Time) (float64, error) {
	if card.State == New || card.LastReview.IsZero() {
		return 0, nil
	}
	if !isValidState(card.State) {
		return 0, &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: invalid card state: %d", card.State)}
	}
	if !isFinite(card.Stability) || card.Stability <= 0 {
		return 0, &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: invalid stability for retrievability calculation: %v", card.Stability)}
	}
	elapsedDays := math.Max(0, dateDiffRaw(card.LastReview, now))
	return f.Parameters.ForgettingCurve(elapsedDays, card.Stability), nil
}

// Deprecated: Use Retrievability instead.
func (f *FSRS) GetRetrievability(card Card, now time.Time) (float64, error) {
	return f.Retrievability(card, now)
}
