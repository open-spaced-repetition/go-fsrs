package fsrs

import "time"

// Rollback reverts a card to its pre-review state using the information stored
// in the ReviewLog. It returns an error if the log entry is a Manual rating
// (ErrManualRating) or if the rating is outside [Again, Easy] (ErrInvalidRating).
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
