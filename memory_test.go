package fsrs

import (
	"math"
	"testing"
	"time"
)

func TestMemoryStateMatchesScheduler(t *testing.T) {
	p := DefaultParam()
	f := NewFSRS(p)

	ratings := []Rating{Again, Good, Good, Good, Good, Good}
	deltaTs := []uint64{0, 0, 1, 3, 8, 21}

	var current *MemoryState
	for i, rating := range ratings {
		states := p.NextStates(current, 0.9, deltaTs[i])
		switch rating {
		case Again:
			current = &states.Again.Memory
		case Hard:
			current = &states.Hard.Memory
		case Good:
			current = &states.Good.Memory
		case Easy:
			current = &states.Easy.Memory
		}
	}

	history := ReviewEntries{
		{Rating: Again, DeltaT: 0},
		{Rating: Good, DeltaT: 0},
		{Rating: Good, DeltaT: 1},
		{Rating: Good, DeltaT: 3},
		{Rating: Good, DeltaT: 8},
		{Rating: Good, DeltaT: 21},
	}

	result, err := f.MemoryState(history, nil)
	if err != nil {
		t.Fatalf("MemoryState returned error: %v", err)
	}

	if math.Abs(result.Stability-current.Stability) > 1e-10 {
		t.Errorf("stability mismatch: MemoryState=%.10f, NextStates=%.10f", result.Stability, current.Stability)
	}
	if math.Abs(result.Difficulty-current.Difficulty) > 1e-10 {
		t.Errorf("difficulty mismatch: MemoryState=%.10f, NextStates=%.10f", result.Difficulty, current.Difficulty)
	}
}

func TestMemoryStateAllRatings(t *testing.T) {
	p := DefaultParam()
	f := NewFSRS(p)

	ratings := []Rating{Again, Hard, Good, Easy, Good, Good}
	deltaTs := []uint64{0, 0, 1, 3, 8, 21}

	var current *MemoryState
	for i, rating := range ratings {
		states := p.NextStates(current, 0.9, deltaTs[i])
		switch rating {
		case Again:
			current = &states.Again.Memory
		case Hard:
			current = &states.Hard.Memory
		case Good:
			current = &states.Good.Memory
		case Easy:
			current = &states.Easy.Memory
		}
	}

	history := ReviewEntries{
		{Rating: Again, DeltaT: 0},
		{Rating: Hard, DeltaT: 0},
		{Rating: Good, DeltaT: 1},
		{Rating: Easy, DeltaT: 3},
		{Rating: Good, DeltaT: 8},
		{Rating: Good, DeltaT: 21},
	}

	result, err := f.MemoryState(history, nil)
	if err != nil {
		t.Fatalf("MemoryState returned error: %v", err)
	}

	if result.Stability < sMin || result.Stability > sMax {
		t.Errorf("stability %.4f out of range [%.4f, %.4f]", result.Stability, sMin, sMax)
	}
	if result.Difficulty < dMin || result.Difficulty > dMax {
		t.Errorf("difficulty %.4f out of range [%.4f, %.4f]", result.Difficulty, dMin, dMax)
	}

	if math.Abs(result.Stability-current.Stability) > 1e-10 {
		t.Errorf("stability mismatch: MemoryState=%.10f, NextStates=%.10f", result.Stability, current.Stability)
	}
	if math.Abs(result.Difficulty-current.Difficulty) > 1e-10 {
		t.Errorf("difficulty mismatch: MemoryState=%.10f, NextStates=%.10f", result.Difficulty, current.Difficulty)
	}
}

func TestMemoryStateLongTerm(t *testing.T) {
	p := DefaultParam()
	p.W[17] = 0
	p.W[18] = 0
	p.W[19] = 0
	f := NewFSRS(p)

	ratings := []Rating{Again, Good, Good, Good, Good, Good}
	deltaTs := []uint64{0, 0, 1, 3, 8, 21}

	var current *MemoryState
	for i, rating := range ratings {
		states := f.Parameters.NextStates(current, 0.9, deltaTs[i])
		switch rating {
		case Again:
			current = &states.Again.Memory
		case Good:
			current = &states.Good.Memory
		}
	}

	history := ReviewEntries{
		{Rating: Again, DeltaT: 0},
		{Rating: Good, DeltaT: 0},
		{Rating: Good, DeltaT: 1},
		{Rating: Good, DeltaT: 3},
		{Rating: Good, DeltaT: 8},
		{Rating: Good, DeltaT: 21},
	}

	result, err := f.MemoryState(history, nil)
	if err != nil {
		t.Fatalf("MemoryState returned error: %v", err)
	}

	if math.Abs(result.Stability-current.Stability) > 1e-10 {
		t.Errorf("stability mismatch: MemoryState=%.10f, NextStates=%.10f", result.Stability, current.Stability)
	}
}

func TestHistoricalMemoryStates(t *testing.T) {
	p := DefaultParam()
	f := NewFSRS(p)

	history := ReviewEntries{
		{Rating: Again, DeltaT: 0},
		{Rating: Good, DeltaT: 1},
		{Rating: Good, DeltaT: 3},
	}

	states, err := f.HistoricalMemoryStates(history, nil)
	if err != nil {
		t.Fatalf("HistoricalMemoryStates returned error: %v", err)
	}

	if len(states) != len(history) {
		t.Fatalf("expected %d states, got %d", len(history), len(states))
	}

	for i, s := range states {
		if s.Stability < sMin || s.Stability > sMax {
			t.Errorf("state[%d] stability %.4f out of range", i, s.Stability)
		}
		if s.Difficulty < dMin || s.Difficulty > dMax {
			t.Errorf("state[%d] difficulty %.4f out of range", i, s.Difficulty)
		}
	}

	final, _ := f.MemoryState(history, nil)
	last := states[len(states)-1]
	if math.Abs(last.Stability-final.Stability) > 1e-10 {
		t.Errorf("last historical state stability %.10f != MemoryState stability %.10f", last.Stability, final.Stability)
	}
	if math.Abs(last.Difficulty-final.Difficulty) > 1e-10 {
		t.Errorf("last historical state difficulty %.10f != MemoryState difficulty %.10f", last.Difficulty, final.Difficulty)
	}

	var current *MemoryState
	for i, review := range history {
		ns := p.NextStates(current, 0.9, uint64(review.DeltaT))
		switch review.Rating {
		case Again:
			current = &ns.Again.Memory
		case Hard:
			current = &ns.Hard.Memory
		case Good:
			current = &ns.Good.Memory
		case Easy:
			current = &ns.Easy.Memory
		}
		if math.Abs(states[i].Stability-current.Stability) > 1e-10 {
			t.Errorf("state[%d] stability %.10f != NextStates %.10f", i, states[i].Stability, current.Stability)
		}
		if math.Abs(states[i].Difficulty-current.Difficulty) > 1e-10 {
			t.Errorf("state[%d] difficulty %.10f != NextStates %.10f", i, states[i].Difficulty, current.Difficulty)
		}
	}
}

func TestHistoricalMemoryStatesWithStartingState(t *testing.T) {
	p := DefaultParam()
	f := NewFSRS(p)

	starting := &MemoryState{Stability: 5.0, Difficulty: 5.0}
	history := ReviewEntries{
		{Rating: Good, DeltaT: 1},
		{Rating: Good, DeltaT: 3},
	}

	states, err := f.HistoricalMemoryStates(history, starting)
	if err != nil {
		t.Fatalf("HistoricalMemoryStates returned error: %v", err)
	}

	if len(states) != len(history)+1 {
		t.Fatalf("expected %d states (starting + reviews), got %d", len(history)+1, len(states))
	}

	if states[0].Stability != starting.Stability || states[0].Difficulty != starting.Difficulty {
		t.Errorf("first state should be the starting state: got (%.4f, %.4f), want (%.4f, %.4f)",
			states[0].Stability, states[0].Difficulty, starting.Stability, starting.Difficulty)
	}
}

func TestMemoryStateWithStartingState(t *testing.T) {
	p := DefaultParam()
	f := NewFSRS(p)

	starting := &MemoryState{Stability: 5.0, Difficulty: 5.0}
	history := ReviewEntries{{Rating: Good, DeltaT: 1}}

	resultNil, _ := f.MemoryState(history, nil)
	resultWithStart, _ := f.MemoryState(history, starting)

	if math.Abs(resultNil.Stability-resultWithStart.Stability) < 1e-10 {
		t.Error("results should differ when starting state is provided")
	}

	ns := p.NextStates(starting, 0.9, 1)
	expected := ns.Good.Memory
	if math.Abs(resultWithStart.Stability-expected.Stability) > 1e-10 {
		t.Errorf("stability mismatch: got %.10f, expected %.10f", resultWithStart.Stability, expected.Stability)
	}
	if math.Abs(resultWithStart.Difficulty-expected.Difficulty) > 1e-10 {
		t.Errorf("difficulty mismatch: got %.10f, expected %.10f", resultWithStart.Difficulty, expected.Difficulty)
	}
}

func TestMemoryStateMatchesRepeatSchedule(t *testing.T) {
	p := DefaultParam()
	p.EnableShortTerm = false
	f := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	ratings := []Rating{Good, Hard, Good, Easy, Again, Good}
	var history ReviewEntries

	for _, rating := range ratings {
		elapsed := float64(dateDiffInDays(card.LastReview, now))
		if card.State == New {
			elapsed = 0
		}
		history = append(history, ReviewEntry{Rating: rating, DeltaT: elapsed})
		{
			rec, err := f.Next(card, now, rating)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			card = rec.Card
		}
		now = card.Due
	}

	result, err := f.MemoryState(history, nil)
	if err != nil {
		t.Fatalf("MemoryState returned error: %v", err)
	}

	if math.Abs(result.Stability-card.Stability) > 1e-10 {
		t.Errorf("stability mismatch: MemoryState=%.10f, scheduler=%.10f", result.Stability, card.Stability)
	}
	if math.Abs(result.Difficulty-card.Difficulty) > 1e-10 {
		t.Errorf("difficulty mismatch: MemoryState=%.10f, scheduler=%.10f", result.Difficulty, card.Difficulty)
	}
}

func TestMemoryStateInvalidInput(t *testing.T) {
	f := NewFSRS(DefaultParam())

	_, err := f.MemoryState(ReviewEntries{}, nil)
	if err == nil {
		t.Error("empty history should return error")
	}

	_, err = f.MemoryState(ReviewEntries{{Rating: 0, DeltaT: 1}}, nil)
	if err == nil {
		t.Error("rating 0 should return error")
	}

	_, err = f.MemoryState(ReviewEntries{{Rating: 5, DeltaT: 1}}, nil)
	if err == nil {
		t.Error("rating 5 should return error")
	}

	_, err = f.MemoryState(ReviewEntries{{Rating: Good, DeltaT: -1}}, nil)
	if err == nil {
		t.Error("negative delta_t should return error")
	}

	_, err = f.MemoryState(ReviewEntries{{Rating: Good, DeltaT: math.NaN()}}, nil)
	if err == nil {
		t.Error("NaN delta_t should return error")
	}

	_, err = f.MemoryState(ReviewEntries{{Rating: Good, DeltaT: math.Inf(1)}}, nil)
	if err == nil {
		t.Error("Inf delta_t should return error")
	}

	_, err = f.HistoricalMemoryStates(ReviewEntries{}, nil)
	if err == nil {
		t.Error("empty history should return error")
	}
}

func TestMemoryStateSingleReviewFromNew(t *testing.T) {
	p := DefaultParam()
	f := NewFSRS(p)

	history := ReviewEntries{{Rating: Good, DeltaT: 0}}
	result, err := f.MemoryState(history, nil)
	if err != nil {
		t.Fatalf("MemoryState returned error: %v", err)
	}

	ns := p.NextStates(nil, 0.9, 0)
	expected := ns.Good.Memory
	if math.Abs(result.Stability-expected.Stability) > 1e-10 {
		t.Errorf("stability mismatch: got %.10f, expected %.10f", result.Stability, expected.Stability)
	}
	if math.Abs(result.Difficulty-expected.Difficulty) > 1e-10 {
		t.Errorf("difficulty mismatch: got %.10f, expected %.10f", result.Difficulty, expected.Difficulty)
	}
}

func TestMemoryStateFromSM2(t *testing.T) {
	f := NewFSRS(DefaultParam())

	cases := []struct {
		ease      float64
		interval  float64
		retention float64
		wantStab  float64
		wantDiff  float64
	}{
		{2.5, 10.0, 0.9, 10.0, 6.9140563},
		{2.5, 10.0, 0.8, 3.01572, 9.393428},
		{2.5, 10.0, 0.95, 24.841097, 1.2974405},
		{1.3, 20.0, 0.9, 20.0, 10.0},
	}

	for i, tc := range cases {
		result, err := f.MemoryStateFromSM2(tc.ease, tc.interval, tc.retention)
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
			continue
		}
		if math.Abs(result.Stability-tc.wantStab) > 1e-5 {
			t.Errorf("case %d: stability mismatch: got %.10f, want %.10f", i, result.Stability, tc.wantStab)
		}
		if math.Abs(result.Difficulty-tc.wantDiff) > 1e-5 {
			t.Errorf("case %d: difficulty mismatch: got %.10f, want %.10f", i, result.Difficulty, tc.wantDiff)
		}
	}
}

func TestMemoryStateFromSM2E2E(t *testing.T) {
	f := NewFSRS(DefaultParam())

	ease := 2.0
	interval := 15.0

	state, err := f.MemoryStateFromSM2(ease, interval, 0.9)
	if err != nil {
		t.Fatalf("MemoryStateFromSM2 returned error: %v", err)
	}

	ns := f.NextStates(state, 0.9, uint64(interval))
	fsrsFactor := ns.Good.Memory.Stability / interval

	if math.Abs(fsrsFactor-ease) >= 0.01 {
		t.Errorf("e2e sanity check failed: fsrs factor=%.6f, sm2 ease=%.6f (diff=%.6f)", fsrsFactor, ease, math.Abs(fsrsFactor-ease))
	}
}

func TestMemoryStateFromSM2InvalidInput(t *testing.T) {
	f := NewFSRS(DefaultParam())

	cases := []struct {
		name      string
		ease      float64
		interval  float64
		retention float64
	}{
		{"retention 0", 2.5, 10.0, 0},
		{"retention 1", 2.5, 10.0, 1},
		{"retention negative", 2.5, 10.0, -0.1},
		{"retention > 1", 2.5, 10.0, 1.1},
		{"ease <= 1", 1.0, 10.0, 0.9},
		{"ease NaN", math.NaN(), 10.0, 0.9},
		{"ease Inf", math.Inf(1), 10.0, 0.9},
		{"interval NaN", 2.5, math.NaN(), 0.9},
		{"interval negative", 2.5, -1, 0.9},
		{"retention NaN", 2.5, 10.0, math.NaN()},
		{"retention Inf", 2.5, 10.0, math.Inf(1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.MemoryStateFromSM2(tc.ease, tc.interval, tc.retention)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestMemoryStateFromSM2Ranges(t *testing.T) {
	f := NewFSRS(DefaultParam())

	cases := []struct {
		ease      float64
		interval  float64
		retention float64
	}{
		{1.3, 1.0, 0.9},
		{3.0, 100.0, 0.85},
		{2.0, 0.5, 0.95},
		{1.5, 365.0, 0.8},
		{2.5, 10.0, 0.7},
	}

	for i, tc := range cases {
		result, err := f.MemoryStateFromSM2(tc.ease, tc.interval, tc.retention)
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
			continue
		}
		if result.Stability < sMin {
			t.Errorf("case %d: stability %.10f below sMin (%v)", i, result.Stability, sMin)
		}
		if result.Difficulty < dMin || result.Difficulty > dMax {
			t.Errorf("case %d: difficulty %.10f outside [%.1f, %.1f]", i, result.Difficulty, dMin, dMax)
		}
	}
}

func BenchmarkMemoryState(b *testing.B) {
	f := NewFSRS(DefaultParam())
	history := ReviewEntries{
		{Rating: Again, DeltaT: 0},
		{Rating: Good, DeltaT: 1},
		{Rating: Good, DeltaT: 3},
		{Rating: Good, DeltaT: 8},
		{Rating: Good, DeltaT: 21},
	}
	for b.Loop() {
		f.MemoryState(history, nil)
	}
}
