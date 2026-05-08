package fsrs

import "time"

// Forget resets a card to the New state. When resetCount is false, the card's
// Reps and Lapses counters are preserved; otherwise they are zeroed.
// The returned SchedulingInfo contains the reset card and a Manual review log entry.
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
