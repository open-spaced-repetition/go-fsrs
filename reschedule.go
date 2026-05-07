package fsrs

import (
	"fmt"
	"sort"
	"time"
)

// Reschedule replays a sequence of past reviews against card, reconstructing
// the card's memory state. It returns the full collection of scheduling
// decisions produced during replay and an optional reschedule item that
// captures any change in due date relative to the original card.
//
// For graded-only histories where only the final stability and difficulty are
// needed (without full Card/ReviewLog reconstruction), consider using
// [FSRS.MemoryState] with [ReviewHistoryToEntries] for a lighter-weight
// alternative.
//
// Returns an error if any graded review has a rating outside [Again–Easy] or a
// zero review time, or any manual review has a zero review time, a nil State
// field, or a zero Due when State is not New.
func (f *FSRS) Reschedule(card Card, reviews []ReviewHistory, opts RescheduleOptions) (RescheduleResult, error) {
	for i, r := range reviews {
		if r.Rating == Manual {
			if r.Review.IsZero() {
				return RescheduleResult{}, &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: review[%d] has zero review time", i)}
			}
			continue
		}
		if r.Rating < Again || r.Rating > Easy {
			return RescheduleResult{}, &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: review[%d] has invalid rating %d", i, r.Rating)}
		}
		if r.Review.IsZero() {
			return RescheduleResult{}, &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: review[%d] has zero review time", i)}
		}
	}
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
			item, err = f.handleManualRating(curCard, state, review.Review, review.Stability, review.Difficulty, review.Due)
			if err != nil {
				return RescheduleResult{}, err
			}
		} else {
			item, err = f.Next(curCard, review.Review, review.Rating)
			if err != nil {
				return RescheduleResult{}, err
			}
		}

		collections = append(collections, item)
		curCard = item.Card
	}

	var rescheduleItem *SchedulingInfo
	if len(collections) > 0 {
		var err error
		rescheduleItem, err = f.calculateManualRecord(card, opts.Now, collections[len(collections)-1], opts.UpdateMemoryState)
		if err != nil {
			return RescheduleResult{}, err
		}
	}

	return RescheduleResult{
		Collections:    collections,
		RescheduleItem: rescheduleItem,
	}, nil
}

func (f *FSRS) handleManualRating(card Card, state State, reviewed time.Time, stability, difficulty float64, due time.Time) (SchedulingInfo, error) {
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
	nextCard.ScheduledDays = scheduledDays
	nextCard.Reps = card.Reps + 1

	return SchedulingInfo{Card: nextCard, ReviewLog: log}, nil
}

func (f *FSRS) calculateManualRecord(currentCard Card, now time.Time, lastItem SchedulingInfo, updateMemory bool) (*SchedulingInfo, error) {
	rescheduleCard := lastItem.Card

	if currentCard.Due.Equal(rescheduleCard.Due) {
		return nil, nil
	}

	scheduledDays := dateDiffInDays(currentCard.Due, rescheduleCard.Due)

	curCard := currentCard
	curCard.ScheduledDays = scheduledDays

	var stab, diff float64
	if updateMemory {
		stab = rescheduleCard.Stability
		diff = rescheduleCard.Difficulty
	}

	item, err := f.handleManualRating(curCard, rescheduleCard.State, now, stab, diff, rescheduleCard.Due)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
