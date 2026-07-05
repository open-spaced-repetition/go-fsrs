package fsrs

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestBasicSchedulerExample(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	var intervalList []uint64
	var stateList []State
	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var ratings = []Rating{Good, Good, Good, Good, Good, Good, Again, Again, Good, Good, Good, Good, Good}
	var rating Rating
	var revlog ReviewLog

	for i := range ratings {
		rating = ratings[i]
		card = schedulingCards[rating].Card
		intervalList = append(intervalList, card.ScheduledDays)
		revlog = schedulingCards[rating].ReviewLog
		stateList = append(stateList, revlog.State)
		now = card.Due
		schedulingCards, err = fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	wantIntervalList := []uint64{0, 2, 11, 46, 163, 498, 0, 0, 2, 4, 7, 12, 21}
	if !reflect.DeepEqual(intervalList, wantIntervalList) {
		t.Errorf("expected:%v, got:%v", wantIntervalList, intervalList)
	}
	wantStateList := []State{New, Learning, Review, Review, Review, Review, Review, Relearning, Relearning, Review, Review, Review, Review}
	if !reflect.DeepEqual(stateList, wantStateList) {
		t.Errorf("expected:%v, got:%v", wantStateList, stateList)
	}
}

func TestBasicSchedulerMemoState(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var ratings = []Rating{Again, Good, Good, Good, Good, Good}
	var intervalList = []uint64{0, 0, 1, 3, 8, 21}
	var rating Rating
	for i := range ratings {
		rating = ratings[i]
		card = schedulingCards[rating].Card
		now = now.Add(time.Duration(intervalList[i]) * 24 * time.Hour)
		schedulingCards, err = fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	wantStability := 54.0393
	cardStability := roundFloat(schedulingCards[Good].Card.Stability, 4)
	wantDifficulty := 6.3464
	cardDifficulty := roundFloat(schedulingCards[Good].Card.Difficulty, 4)
	if !reflect.DeepEqual(wantStability, cardStability) {
		t.Errorf("expected:%v, got:%v", wantStability, cardStability)
	}

	if !reflect.DeepEqual(wantDifficulty, cardDifficulty) {
		t.Errorf("expected:%v, got:%v", wantDifficulty, cardDifficulty)
	}
}

func TestNextStatesMemoryStateShortTerm(t *testing.T) {
	p := DefaultParam()
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

	gotS := roundFloat(current.Stability, 4)
	wantS := 53.6269
	if gotS != wantS {
		t.Errorf("short-term stability: got %.4f, want %.4f", gotS, wantS)
	}
	gotD := roundFloat(current.Difficulty, 4)
	wantD := 6.3575
	if gotD != wantD {
		t.Errorf("short-term difficulty: got %.4f, want %.4f", gotD, wantD)
	}

	states := p.NextStates(current, 0.9, 1)
	if states.Good.Interval < 1 {
		t.Errorf("Good interval should be >= 1 day, got=%v", states.Good.Interval)
	}
	if states.Again.Interval < 1 {
		t.Errorf("Again interval should be >= 1 day, got=%v", states.Again.Interval)
	}
	if states.Hard.Interval >= states.Good.Interval {
		t.Errorf("Hard interval (%v) should be < Good interval (%v)", states.Hard.Interval, states.Good.Interval)
	}
	if states.Easy.Interval <= states.Good.Interval {
		t.Errorf("Easy interval (%v) should be > Good interval (%v)", states.Easy.Interval, states.Good.Interval)
	}
}

func TestNextStatesMemoryStateAllRatingsShortTerm(t *testing.T) {
	p := DefaultParam()
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

	gotS := roundFloat(current.Stability, 4)
	wantS := 51.0810
	if gotS != wantS {
		t.Errorf("all-ratings stability: got %.4f, want %.4f", gotS, wantS)
	}
	gotD := roundFloat(current.Difficulty, 4)
	wantD := 6.7493
	if gotD != wantD {
		t.Errorf("all-ratings difficulty: got %.4f, want %.4f", gotD, wantD)
	}

	states := p.NextStates(current, 0.9, 1)
	if states.Hard.Interval >= states.Good.Interval {
		t.Errorf("Hard interval (%v) should be < Good interval (%v)", states.Hard.Interval, states.Good.Interval)
	}
	if states.Easy.Interval <= states.Good.Interval {
		t.Errorf("Easy interval (%v) should be > Good interval (%v)", states.Easy.Interval, states.Good.Interval)
	}
}

func TestNextStatesMemoryStateLongTerm(t *testing.T) {
	p := DefaultParam()
	p.W[17] = 0
	p.W[18] = 0
	p.W[19] = 0
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

	gotS := roundFloat(current.Stability, 4)
	wantS := 53.3351
	if gotS != wantS {
		t.Errorf("long-term stability: got %.4f, want %.4f", gotS, wantS)
	}
	gotD := roundFloat(current.Difficulty, 4)
	wantD := 6.3575
	if gotD != wantD {
		t.Errorf("long-term difficulty: got %.4f, want %.4f", gotD, wantD)
	}
}

func TestLongTermScheduler(t *testing.T) {
	p := DefaultParam()
	p.EnableShortTerm = false
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	ratings := []Rating{Good, Good, Good, Good, Good, Good, Again, Again, Good, Good, Good, Good, Good}
	intervalHistory := []uint64{}
	stabilityHistory := []float64{}
	difficultyHistory := []float64{}
	for _, rating := range ratings {
		schedCards, err := fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		record := schedCards[rating]
		next, err := fsrs.Next(card, now, rating)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(record.Card, next.Card) {
			t.Errorf("expected:%v, got:%v", record.Card, next.Card)
		}

		card = record.Card
		intervalHistory = append(intervalHistory, (card.ScheduledDays))
		stabilityHistory = append(stabilityHistory, roundFloat(card.Stability, 4))
		difficultyHistory = append(difficultyHistory, roundFloat(card.Difficulty, 4))
		now = card.Due
	}
	wantIntervalHistory := []uint64{3, 14, 57, 196, 586, 1559, 10, 1, 3, 5, 9, 15, 25}
	if !reflect.DeepEqual(intervalHistory, wantIntervalHistory) {
		t.Errorf("expected:%v, got:%v", wantIntervalHistory, intervalHistory)
	}
	wantStabilityHistory := []float64{2.3065, 13.8269, 56.9567, 196.2353, 586.4835, 1559.3567, 9.8832, 1.3527, 2.392, 4.8562, 8.7624, 15.186, 25.1362}
	if !reflect.DeepEqual(stabilityHistory, wantStabilityHistory) {
		t.Errorf("expected:%v, got:%v", wantStabilityHistory, stabilityHistory)
	}
	wantDifficultyHistory := []float64{2.1181, 2.1112, 2.1043, 2.0975, 2.0906, 2.0837, 7.3832, 9.1251, 9.1112, 9.0973, 9.0835, 9.0696, 9.0558}
	if !reflect.DeepEqual(difficultyHistory, wantDifficultyHistory) {
		t.Errorf("expected:%v, got:%v", wantDifficultyHistory, difficultyHistory)
	}
}

func BenchmarkRepeat(b *testing.B) {
	f := NewFSRS(DefaultParam())
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	for b.Loop() {
		_, _ = f.Repeat(card, now)
	}
}

func BenchmarkNext(b *testing.B) {
	f := NewFSRS(DefaultParam())
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	for b.Loop() {
		_, _ = f.Next(card, now, Good)
		_, _ = f.Next(card, now, Hard)
		_, _ = f.Next(card, now, Easy)
		_, _ = f.Next(card, now, Again)
	}
}

func BenchmarkRetrievability(b *testing.B) {
	f := NewFSRS(DefaultParam())
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	{
		rec, err := f.Next(card, now, Good)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		card = rec.Card
	}
	for b.Loop() {
		_, _ = f.Retrievability(card, now)
	}
}

func TestNewFSRSFallbacksToDefaultOnInvalidInput(t *testing.T) {
	p := DefaultParam()
	p.W[0] = math.NaN()

	fsrs := NewFSRS(p)
	defaults := DefaultParam()
	if !reflect.DeepEqual(fsrs.W, defaults.W) {
		t.Fatalf("expected default weights fallback, got=%v", fsrs.W)
	}
	if fsrs.RequestRetention != defaults.RequestRetention {
		t.Errorf("expected default RequestRetention=%v, got=%v", defaults.RequestRetention, fsrs.RequestRetention)
	}
	if fsrs.MaximumInterval != defaults.MaximumInterval {
		t.Errorf("expected default MaximumInterval=%v, got=%v", defaults.MaximumInterval, fsrs.MaximumInterval)
	}
}

func TestNewFSRSFallbacksOnInvalidRetention(t *testing.T) {
	p := DefaultParam()
	p.RequestRetention = 0

	fsrs := NewFSRS(p)
	defaults := DefaultParam()
	if fsrs.RequestRetention != defaults.RequestRetention {
		t.Errorf("expected default RequestRetention after fallback, got=%v", fsrs.RequestRetention)
	}
}

func TestNewFSRSFallbacksOnInvalidMaximumInterval(t *testing.T) {
	p := DefaultParam()
	p.MaximumInterval = -1

	fsrs := NewFSRS(p)
	defaults := DefaultParam()
	if fsrs.MaximumInterval != defaults.MaximumInterval {
		t.Errorf("expected default MaximumInterval after fallback, got=%v", fsrs.MaximumInterval)
	}
}
