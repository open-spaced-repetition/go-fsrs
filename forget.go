package fsrs

import "time"

// Forget resets a card to the New state. When resetCount is false, the card's
// Reps and Lapses counters are preserved; otherwise they are zeroed.
// The returned SchedulingInfo contains the reset card and a Manual review log
// that captures the card's pre-forget state (State, Due, Stability, Difficulty,
// ScheduledDays, RemainingSteps). The card's LastReview is preserved.
func (f *FSRS) Forget(card Card, now time.Time, resetCount bool) SchedulingInfo {
	scheduledDays := uint64(0)
	if card.State != New {
		scheduledDays = dateDiffInDays(now, card.Due)
	}
	forgetLog := ReviewLog{
		Rating:         Manual,
		State:          card.State,
		Due:            card.Due,
		Stability:      card.Stability,
		Difficulty:     card.Difficulty,
		ScheduledDays:  scheduledDays,
		RemainingSteps: card.RemainingSteps,
		Review:         now,
	}
	forgetCard := Card{
		Due:        now,
		State:      New,
		LastReview: card.LastReview,
	}
	if !resetCount {
		forgetCard.Reps = card.Reps
		forgetCard.Lapses = card.Lapses
	}
	return SchedulingInfo{Card: forgetCard, ReviewLog: forgetLog}
}
