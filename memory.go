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
		states = make([]MemoryState, 0, len(history))
	}

	state := startingState
	if startingState != nil && returnAll {
		states = append(states, *state)
	}
	if state == nil {
		state = &MemoryState{}
	}

	for _, review := range history {
		if review.Rating < Again || review.Rating > Easy {
			return nil, fmt.Errorf("fsrs: invalid rating %d, must be 1-4", review.Rating)
		}
		if review.DeltaT < 0 || math.IsNaN(review.DeltaT) || math.IsInf(review.DeltaT, 0) {
			return nil, fmt.Errorf("fsrs: invalid delta_t, must be a finite non-negative number")
		}

		item := f.nextStateInner(&MemoryState{
			Stability:  state.Stability,
			Difficulty: state.Difficulty,
		}, f.RequestRetention, review.DeltaT, review.Rating, decay, factor)

		state = &item.Memory
		if returnAll {
			states = append(states, *state)
		}
	}

	if math.IsNaN(state.Stability) || math.IsInf(state.Stability, 0) {
		return nil, fmt.Errorf("fsrs: computed stability is not finite")
	}
	if math.IsNaN(state.Difficulty) || math.IsInf(state.Difficulty, 0) {
		return nil, fmt.Errorf("fsrs: computed difficulty is not finite")
	}

	if returnAll {
		return states, nil
	}
	return []MemoryState{*state}, nil
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
