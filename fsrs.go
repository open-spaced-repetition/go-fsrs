package fsrs

import (
	"errors"
	"math"
	"sort"
	"time"
)

var (
	ErrManualRating        = errors.New("fsrs: cannot rollback a manual rating")
	ErrInvalidRating       = errors.New("fsrs: invalid rating for rollback")
	ErrManualStateRequired = errors.New("fsrs: state is required for manual rating")
	ErrManualDueRequired   = errors.New("fsrs: due is required for manual rating when state is not New")
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
	result.Stability = log.Stability
	result.Difficulty = log.Difficulty
	result.ElapsedDays = log.ElapsedDays
	result.ScheduledDays = log.ScheduledDays
	if card.Reps > 0 {
		result.Reps = card.Reps - 1
	}
	switch log.State {
	case New:
		result.Due = log.Due
		result.LastReview = time.Time{}
		result.Lapses = 0
	case Learning, Relearning, Review:
		result.Due = log.Review
		result.LastReview = log.Due
		result.Lapses = card.Lapses
		if log.Rating == Again && log.State == Review && card.Lapses > 0 {
			result.Lapses = card.Lapses - 1
		}
	}
	return result, nil
}

// Reschedule replays a sequence of past reviews against card, reconstructing
// the card's memory state. It returns the full collection of scheduling
// decisions produced during replay and an optional reschedule item that
// captures any change in due date relative to the original card.
func (f *FSRS) Reschedule(card Card, reviews []ReviewHistory, opts RescheduleOptions) (RescheduleResult, error) {
	working := reviews
	if opts.SortReviews {
		working = make([]ReviewHistory, len(reviews))
		copy(working, reviews)
		sort.Slice(working, func(i, j int) bool {
			return working[i].Review.Before(working[j].Review)
		})
	}

	if opts.SkipManual {
		filtered := make([]ReviewHistory, 0, len(working))
		for _, r := range working {
			if r.Rating != Manual {
				filtered = append(filtered, r)
			}
		}
		working = filtered
	}

	var startDue time.Time
	if !opts.FirstDue.IsZero() {
		startDue = opts.FirstDue
	} else {
		startDue = card.Due
	}
	curCard := Card{Due: startDue}

	collections := make([]SchedulingInfo, 0, len(working))
	for _, review := range working {
		var item SchedulingInfo
		var err error

		if review.Rating == Manual {
			if review.State == nil {
				return RescheduleResult{}, ErrManualStateRequired
			}
			state := *review.State
			var elapsedDays uint64
			if curCard.State != New && !curCard.LastReview.IsZero() {
				elapsedDays = dateDiffInDays(curCard.LastReview, review.Review)
			}
			item, err = f.handleManualRating(curCard, state, review.Review, elapsedDays, review.Stability, review.Difficulty, review.Due)
			if err != nil {
				return RescheduleResult{}, err
			}
		} else {
			item = f.Next(curCard, review.Review, review.Rating)
		}

		collections = append(collections, item)
		curCard = item.Card
	}

	var rescheduleItem *SchedulingInfo
	if len(collections) > 0 {
		rescheduleItem = f.calculateManualRecord(card, opts.Now, collections[len(collections)-1], opts.UpdateMemoryState)
	}

	return RescheduleResult{
		Collections:    collections,
		RescheduleItem: rescheduleItem,
	}, nil
}

func (f *FSRS) handleManualRating(card Card, state State, reviewed time.Time, elapsedDays uint64, stability, difficulty float64, due time.Time) (SchedulingInfo, error) {
	if state == New {
		effectiveDue := reviewed
		if !due.IsZero() {
			effectiveDue = due
		}
		log := ReviewLog{
			Rating:         Manual,
			State:          New,
			Due:            effectiveDue,
			Stability:      card.Stability,
			Difficulty:     card.Difficulty,
			ElapsedDays:    elapsedDays,
			ScheduledDays:  card.ScheduledDays,
			RemainingSteps: card.RemainingSteps,
			Review:         reviewed,
		}
		nextCard := Card{Due: effectiveDue, LastReview: reviewed}
		if effectiveDue.After(reviewed) {
			nextCard.ScheduledDays = dateDiffInDays(reviewed, effectiveDue)
		}
		return SchedulingInfo{Card: nextCard, ReviewLog: log}, nil
	}

	if due.IsZero() {
		return SchedulingInfo{}, ErrManualDueRequired
	}

	scheduledDays := dateDiffInDays(reviewed, due)

	logDue := card.LastReview
	if logDue.IsZero() {
		logDue = card.Due
	}
	log := ReviewLog{
		Rating:         Manual,
		State:          card.State,
		Due:            logDue,
		Stability:      card.Stability,
		Difficulty:     card.Difficulty,
		ElapsedDays:    elapsedDays,
		ScheduledDays:  card.ScheduledDays,
		RemainingSteps: card.RemainingSteps,
		Review:         reviewed,
	}

	stab := stability
	if stab == 0 {
		stab = card.Stability
	}
	diff := difficulty
	if diff == 0 {
		diff = card.Difficulty
	}

	nextCard := card
	nextCard.State = state
	nextCard.Due = due
	nextCard.LastReview = reviewed
	nextCard.Stability = stab
	nextCard.Difficulty = diff
	nextCard.ElapsedDays = elapsedDays
	nextCard.ScheduledDays = scheduledDays
	nextCard.Reps = card.Reps + 1

	return SchedulingInfo{Card: nextCard, ReviewLog: log}, nil
}

func (f *FSRS) calculateManualRecord(currentCard Card, now time.Time, lastItem SchedulingInfo, updateMemory bool) *SchedulingInfo {
	rescheduleCard := lastItem.Card
	log := lastItem.ReviewLog

	if currentCard.Due.Equal(rescheduleCard.Due) {
		return nil
	}

	scheduledDays := dateDiffInDays(currentCard.Due, rescheduleCard.Due)

	curCard := currentCard
	curCard.ScheduledDays = scheduledDays

	var stab, diff float64
	if updateMemory {
		stab = rescheduleCard.Stability
		diff = rescheduleCard.Difficulty
	}

	item, err := f.handleManualRating(curCard, rescheduleCard.State, now, log.ElapsedDays, stab, diff, rescheduleCard.Due)
	if err != nil {
		return nil
	}
	return &item
}
