package fsrs

import (
	"fmt"
	"math"
	"time"
)

func isFinite(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}

func isValidState(s State) bool {
	return s >= New && s <= Relearning
}

func validateCard(card Card, now time.Time) error {
	if !isValidState(card.State) {
		return &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: invalid card state: %d", card.State)}
	}
	if card.State != New {
		if !isFinite(card.Stability) || card.Stability < sMin {
			return &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: invalid stability: %v (minimum %v for non-New cards)", card.Stability, sMin)}
		}
		if !isFinite(card.Difficulty) || card.Difficulty < dMin {
			return &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: invalid difficulty: %v (minimum %v for non-New cards)", card.Difficulty, dMin)}
		}
	}
	if !card.LastReview.IsZero() && card.LastReview.After(now) {
		return &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: last review date (%v) is after current time (%v)", card.LastReview, now)}
	}
	return nil
}

func validateRating(grade Rating) error {
	if grade < Again || grade > Easy {
		return &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: invalid grade: %d (must be Again, Hard, Good, or Easy)", grade)}
	}
	return nil
}

func validateResult(card Card) error {
	if !isFinite(card.Stability) || card.Stability < sMin {
		return &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: computed stability is invalid: %v", card.Stability)}
	}
	if !isFinite(card.Difficulty) || card.Difficulty < dMin {
		return &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: computed difficulty is invalid: %v", card.Difficulty)}
	}
	return nil
}
