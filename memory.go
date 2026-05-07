package fsrs

import (
	"fmt"
	"math"
)

func (f *FSRS) computeMemoryStates(history ReviewEntries, startingState *MemoryState, returnAll bool) ([]MemoryState, error) {
	if len(history) == 0 {
		return nil, fmt.Errorf("fsrs: history must not be empty")
	}

	decay, factor := f.decayAndFactor()

	var states []MemoryState
	if returnAll {
		states = make([]MemoryState, 0, len(history)+1)
	}

	cur := startingState
	if startingState != nil && returnAll {
		states = append(states, *cur)
	}
	if cur == nil {
		cur = &MemoryState{}
	}

	for _, review := range history {
		if review.Rating < Again || review.Rating > Easy {
			return nil, fmt.Errorf("fsrs: invalid rating %d, must be 1-4", review.Rating)
		}
		if review.DeltaT < 0 || math.IsNaN(review.DeltaT) || math.IsInf(review.DeltaT, 0) {
			return nil, fmt.Errorf("fsrs: invalid delta_t, must be a finite non-negative number")
		}

		item := f.nextStateInner(&MemoryState{
			Stability:  cur.Stability,
			Difficulty: cur.Difficulty,
		}, f.RequestRetention, review.DeltaT, review.Rating, decay, factor)

		cur = &item.Memory
		if returnAll {
			states = append(states, *cur)
		}
	}

	if math.IsNaN(cur.Stability) || math.IsInf(cur.Stability, 0) {
		return nil, fmt.Errorf("fsrs: computed stability is not finite")
	}
	if math.IsNaN(cur.Difficulty) || math.IsInf(cur.Difficulty, 0) {
		return nil, fmt.Errorf("fsrs: computed difficulty is not finite")
	}

	if returnAll {
		return states, nil
	}
	return []MemoryState{*cur}, nil
}

// MemoryState computes the final memory state by replaying the review history through the FSRS algorithm.
func (f *FSRS) MemoryState(history ReviewEntries, startingState *MemoryState) (*MemoryState, error) {
	states, err := f.computeMemoryStates(history, startingState, false)
	if err != nil {
		return nil, err
	}
	return &states[0], nil
}

// HistoricalMemoryStates returns all intermediate memory states, one per review entry.
func (f *FSRS) HistoricalMemoryStates(history ReviewEntries, startingState *MemoryState) ([]MemoryState, error) {
	return f.computeMemoryStates(history, startingState, true)
}

// MemoryStateFromSM2 converts SM-2 parameters (easeFactor, interval, sm2Retention)
// into an approximate FSRS MemoryState. This is useful when migrating from an
// SM-2-based scheduler to FSRS.
//
// easeFactor must be finite and > 1; interval must be finite and >= 0;
// sm2Retention must be finite and in (0, 1) exclusive.
func (f *FSRS) MemoryStateFromSM2(easeFactor, interval, sm2Retention float64) (*MemoryState, error) {
	if !isFinite(easeFactor) || easeFactor <= 1 {
		return nil, fmt.Errorf("fsrs: invalid easeFactor: %v (must be finite and > 1)", easeFactor)
	}
	if !isFinite(interval) || interval < 0 {
		return nil, fmt.Errorf("fsrs: invalid interval: %v (must be a finite non-negative number)", interval)
	}
	if !isFinite(sm2Retention) || sm2Retention <= 0 || sm2Retention >= 1 {
		return nil, fmt.Errorf("fsrs: invalid sm2Retention: %v (must be finite and in (0, 1))", sm2Retention)
	}

	decay, factor := f.decayAndFactor()

	stability := math.Max(interval, sMin) * factor / (math.Pow(sm2Retention, 1/decay) - 1)

	w8 := f.W[8]
	w9 := f.W[9]
	w10 := f.W[10]

	difficulty := 11 - (easeFactor-1)/
		(math.Exp(w8)*math.Pow(stability, -w9)*math.Expm1((1-sm2Retention)*w10))

	if !isFinite(stability) || !isFinite(difficulty) {
		return nil, fmt.Errorf("fsrs: computed memory state is not finite (stability=%v, difficulty=%v)", stability, difficulty)
	}

	return &MemoryState{
		Stability:  constrainStability(stability),
		Difficulty: constrainDifficulty(difficulty),
	}, nil
}

// ReviewHistoryToEntries converts a timestamp-based review history into
// DeltaT-based [ReviewEntries] suitable for use with [FSRS.MemoryState] and
// [FSRS.HistoricalMemoryStates]. The input reviews must be in chronological
// order.
//
// Returns an error if the list is empty, any review has a zero timestamp,
// any rating is [Manual] or outside the range [Again–Easy], or any computed
// DeltaT is negative (indicating out-of-order timestamps).
func ReviewHistoryToEntries(reviews []ReviewHistory) (ReviewEntries, error) {
	if len(reviews) == 0 {
		return nil, &Error{Code: ErrCodeInvalidInput, Message: "fsrs: review history must not be empty"}
	}
	entries := make(ReviewEntries, 0, len(reviews))
	for i, review := range reviews {
		if review.Review.IsZero() {
			return nil, &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: review[%d] has zero review time", i)}
		}
		if review.Rating == Manual {
			return nil, &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: review[%d] has Manual rating; use Reschedule for mixed histories", i)}
		}
		if review.Rating < Again || review.Rating > Easy {
			return nil, &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: review[%d] has invalid rating %d", i, review.Rating)}
		}

		var deltaT float64
		if i == 0 {
			deltaT = 0
		} else {
			deltaT = dateDiffRaw(reviews[i-1].Review, review.Review)
			if deltaT < 0 {
				return nil, &Error{Code: ErrCodeInvalidInput, Message: fmt.Sprintf("fsrs: review[%d] has negative delta_t (%.1f); reviews must be chronological", i, deltaT)}
			}
		}
		entries = append(entries, ReviewEntry{Rating: review.Rating, DeltaT: deltaT})
	}
	return entries, nil
}
