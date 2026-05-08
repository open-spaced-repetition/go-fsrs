package fsrs

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
)

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func TestBasicSchedulerExample(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	var ivlList []uint64
	var stateList []State
	schedulingCards := fsrs.Repeat(card, now)

	var ratings = []Rating{Good, Good, Good, Good, Good, Good, Again, Again, Good, Good, Good, Good, Good}
	var rating Rating
	var revlog ReviewLog

	for i := range ratings {
		rating = ratings[i]
		card = schedulingCards[rating].Card
		ivlList = append(ivlList, card.ScheduledDays)
		revlog = schedulingCards[rating].ReviewLog
		stateList = append(stateList, revlog.State)
		now = card.Due
		schedulingCards = fsrs.Repeat(card, now)
	}

	wantIvlList := []uint64{0, 2, 11, 46, 163, 498, 0, 0, 2, 4, 7, 12, 21}
	if !reflect.DeepEqual(ivlList, wantIvlList) {
		t.Errorf("expected:%v, got:%v", wantIvlList, ivlList)
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
	schedulingCards := fsrs.Repeat(card, now)
	var ratings = []Rating{Again, Good, Good, Good, Good, Good}
	var ivlList = []uint64{0, 0, 1, 3, 8, 21}
	var rating Rating
	for i := range ratings {
		rating = ratings[i]
		card = schedulingCards[rating].Card
		now = now.Add(time.Duration(ivlList[i]) * 24 * time.Hour)
		schedulingCards = fsrs.Repeat(card, now)
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

func TestNextInterval(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	var ivlList []float64
	for i := 1; i <= 10; i++ {
		fsrs.RequestRetention = float64(i) / 10
		ivlList = append(ivlList, fsrs.nextInterval(1, 0))
	}
	wantIvlList := []float64{36500, 34793, 2508, 387, 90, 27, 9, 3, 1, 1}
	if !reflect.DeepEqual(ivlList, wantIvlList) {
		t.Errorf("expected:%v, got:%v", wantIvlList, ivlList)
	}
}

func TestNextIntervalRaw(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)

	ivl := fsrs.nextIntervalRaw(0.212)
	if ivl >= 1.0 {
		t.Errorf("nextIntervalRaw(0.212) should return < 1 day, got=%v", ivl)
	}
	if ivl <= 0 {
		t.Errorf("nextIntervalRaw(0.212) should return > 0, got=%v", ivl)
	}

	ivl = fsrs.nextIntervalRaw(1.0)
	if ivl < 0.99 {
		t.Errorf("nextIntervalRaw(1.0) should return ~1 day, got=%v", ivl)
	}

	ivl = fsrs.nextIntervalRaw(0.001)
	if ivl >= 0.5 {
		t.Errorf("nextIntervalRaw(0.001) should return < 0.5 days, got=%v", ivl)
	}

	ivl = fsrs.nextIntervalRaw(sMin)
	if ivl <= 0 || math.IsNaN(ivl) || math.IsInf(ivl, 0) {
		t.Errorf("nextIntervalRaw(sMin) should return finite positive, got=%v", ivl)
	}

	ivl = fsrs.nextIntervalRaw(sMax)
	if ivl <= 0 || math.IsNaN(ivl) || math.IsInf(ivl, 0) {
		t.Errorf("nextIntervalRaw(sMax) should return finite positive, got=%v", ivl)
	}

	decay, factor := p.decayAndFactor()
	boundaryStab := 0.5 * factor / (math.Pow(p.RequestRetention, 1/decay) - 1)
	ivl = fsrs.nextIntervalRaw(boundaryStab)
	if math.Abs(ivl-0.5) > 1e-9 {
		t.Errorf("nextIntervalRaw at boundary stability should ≈ 0.5, got=%v", ivl)
	}
}

func TestLongTermScheduler(t *testing.T) {
	p := DefaultParam()
	p.EnableShortTerm = false
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	ratings := []Rating{Good, Good, Good, Good, Good, Good, Again, Again, Good, Good, Good, Good, Good}
	ivlHistory := []uint64{}
	sHistory := []float64{}
	dHistory := []float64{}
	for _, rating := range ratings {
		record := fsrs.Repeat(card, now)[rating]
		next := fsrs.Next(card, now, rating)
		if !reflect.DeepEqual(record.Card, next.Card) {
			t.Errorf("expected:%v, got:%v", record.Card, next.Card)
		}

		card = record.Card
		ivlHistory = append(ivlHistory, (card.ScheduledDays))
		sHistory = append(sHistory, roundFloat(card.Stability, 4))
		dHistory = append(dHistory, roundFloat(card.Difficulty, 4))
		now = card.Due
	}
	wantIvlHistory := []uint64{3, 14, 57, 196, 586, 1559, 10, 1, 3, 5, 9, 15, 25}
	if !reflect.DeepEqual(ivlHistory, wantIvlHistory) {
		t.Errorf("expected:%v, got:%v", wantIvlHistory, ivlHistory)
	}
	wantSHistory := []float64{2.3065, 13.8269, 56.9567, 196.2353, 586.4835, 1559.3567, 9.8832, 1.3527, 2.392, 4.8562, 8.7624, 15.186, 25.1362}
	if !reflect.DeepEqual(sHistory, wantSHistory) {
		t.Errorf("expected:%v, got:%v", wantSHistory, sHistory)
	}
	wantDHistory := []float64{2.1181, 2.1112, 2.1043, 2.0975, 2.0906, 2.0837, 7.3832, 9.1251, 9.1112, 9.0973, 9.0835, 9.0696, 9.0558}
	if !reflect.DeepEqual(dHistory, wantDHistory) {
		t.Errorf("expected:%v, got:%v", wantDHistory, dHistory)
	}
}

func TestDateDiffInDays(t *testing.T) {
	tests := []struct {
		name     string
		last     time.Time
		now      time.Time
		expected uint64
	}{
		{"same day different hours", time.Date(2032, 1, 15, 12, 30, 0, 0, time.UTC), time.Date(2032, 1, 15, 23, 59, 0, 0, time.UTC), 0},
		{"next day", time.Date(2032, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2032, 1, 16, 0, 0, 0, 0, time.UTC), 1},
		{"end of month", time.Date(2032, 1, 31, 12, 0, 0, 0, time.UTC), time.Date(2032, 2, 1, 0, 0, 0, 0, time.UTC), 1},
		{"year boundary", time.Date(2032, 12, 31, 18, 0, 0, 0, time.UTC), time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC), 1},
		{"large span", time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC), 366},
		{"CET midnight crosses UTC day", time.Date(2032, 1, 15, 0, 30, 0, 0, time.FixedZone("CET", 3600)), time.Date(2032, 1, 15, 12, 0, 0, 0, time.FixedZone("CET", 3600)), 1},
		{"CET evening to next UTC day", time.Date(2032, 1, 14, 23, 30, 0, 0, time.FixedZone("CET", 3600)), time.Date(2032, 1, 15, 1, 0, 0, 0, time.FixedZone("CET", 3600)), 1},
		{"different local days but UTC boundary matters", time.Date(2032, 1, 15, 23, 0, 0, 0, time.UTC), time.Date(2032, 1, 16, 1, 0, 0, 0, time.FixedZone("EST", -18000)), 1},
		{"multiple days", time.Date(2032, 1, 10, 0, 0, 0, 0, time.UTC), time.Date(2032, 1, 20, 0, 0, 0, 0, time.UTC), 10},
		{"reversed dates returns 0", time.Date(2032, 1, 20, 0, 0, 0, 0, time.UTC), time.Date(2032, 1, 10, 0, 0, 0, 0, time.UTC), 0},
		{"reversed same day returns 0", time.Date(2032, 1, 15, 18, 0, 0, 0, time.UTC), time.Date(2032, 1, 15, 6, 0, 0, 0, time.UTC), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dateDiffInDays(tt.last, tt.now)
			if got != tt.expected {
				t.Errorf("dateDiffInDays(%v, %v) = %d, want %d", tt.last, tt.now, got, tt.expected)
			}
		})
	}
}

func TestGetRetrievability(t *testing.T) {
	retrievabilityList := []float64{}
	fsrs := NewFSRS(DefaultParam())
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	retrievabilityList = append(retrievabilityList, roundFloat(fsrs.GetRetrievability(card, now), 4))
	for range 3 {
		card = fsrs.Next(card, now, Good).Card
		now = card.Due
		retrievabilityList = append(retrievabilityList, roundFloat(fsrs.GetRetrievability(card, now), 4))
	}
	wantRetrievabilityList := []float64{0, 1, 0.9095, 0.8998}
	if !reflect.DeepEqual(retrievabilityList, wantRetrievabilityList) {
		t.Errorf("expected:%v, got:%v", wantRetrievabilityList, retrievabilityList)
	}
}

func TestDateDiffRawMidnightCrossover(t *testing.T) {
	last := time.Date(2032, 1, 15, 23, 0, 0, 0, time.UTC)
	now := time.Date(2032, 1, 16, 1, 0, 0, 0, time.UTC)

	got := dateDiffRaw(last, now)
	if got != 0 {
		t.Errorf("dateDiffRaw(midnight crossover) = %v, want 0 (only 2h elapsed)", got)
	}

	gotCal := dateDiffInDays(last, now)
	if gotCal != 1 {
		t.Errorf("dateDiffInDays(midnight crossover) = %d, want 1 (calendar day)", gotCal)
	}
}

func TestDateDiffRawFullDay(t *testing.T) {
	last := time.Date(2032, 1, 15, 10, 0, 0, 0, time.UTC)
	now := time.Date(2032, 1, 16, 12, 0, 0, 0, time.UTC)

	got := dateDiffRaw(last, now)
	if got != 1 {
		t.Errorf("dateDiffRaw(26h) = %v, want 1", got)
	}
}

func TestGetRetrievabilityRawVsCalendar(t *testing.T) {
	f := NewFSRS(DefaultParam())
	card := Card{
		Stability:  5.0,
		Difficulty: 5.0,
		State:      Review,
		LastReview: time.Date(2032, 1, 15, 23, 0, 0, 0, time.UTC),
	}

	now := time.Date(2032, 1, 16, 1, 0, 0, 0, time.UTC)

	r := f.GetRetrievability(card, now)
	if r != 1.0 {
		t.Errorf("GetRetrievability(midnight crossover, 2h) = %v, want 1.0 (t=0, no decay)", r)
	}
}

func TestGetRetrievabilityReversedDates(t *testing.T) {
	f := NewFSRS(DefaultParam())
	card := Card{
		Stability:  5.0,
		Difficulty: 5.0,
		State:      Review,
		LastReview: time.Date(2032, 1, 16, 12, 0, 0, 0, time.UTC),
	}

	now := time.Date(2032, 1, 15, 12, 0, 0, 0, time.UTC)

	r := f.GetRetrievability(card, now)
	if r != 1.0 {
		t.Errorf("GetRetrievability(reversed dates) = %v, want 1.0 (t clamped to 0)", r)
	}
}

func TestDecayAndFactorDerivedFromW20(t *testing.T) {
	p := DefaultParam()
	p.W[20] = 0.2
	p.Decay = -0.5
	p.Factor = math.Pow(0.9, 1.0/p.Decay) - 1.0

	got := p.ForgettingCurve(1, 1)
	wantDecay := -p.W[20]
	wantFactor := math.Pow(0.9, 1.0/wantDecay) - 1.0
	want := math.Pow(1+wantFactor, wantDecay)

	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("ForgettingCurve should use W[20], got=%v want=%v", got, want)
	}
}

func TestInvalidW20ValidationAndFallback(t *testing.T) {
	p := DefaultParam()
	p.W[20] = 0
	if err := p.Validate(); err == nil {
		t.Fatal("expected validation error for invalid W[20]")
	}

	gotDecay, gotFactor := p.decayAndFactor()
	wantDecay, wantFactor := defaultDecayAndFactor()
	if gotDecay != wantDecay || gotFactor != wantFactor {
		t.Fatalf("decayAndFactor should fallback to defaults, got=(%v,%v) want=(%v,%v)", gotDecay, gotFactor, wantDecay, wantFactor)
	}

	pDefault := DefaultParam()
	if got, want := p.nextInterval(1, 0), pDefault.nextInterval(1, 0); got != want {
		t.Fatalf("nextInterval should fallback safely, got=%v want=%v", got, want)
	}
}

func TestStabilityIsClamped(t *testing.T) {
	p := DefaultParam()
	p.W[0] = 1e9

	if got := p.initStability(Again); got != sMax {
		t.Fatalf("initStability should clamp to sMax, got=%v", got)
	}

	if got := p.shortTermStability(1e9, Good); got != sMax {
		t.Fatalf("shortTermStability should clamp to sMax, got=%v", got)
	}
}

func BenchmarkRepeat(b *testing.B) {
	f := NewFSRS(DefaultParam())
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	for b.Loop() {
		f.Repeat(card, now)
	}
}

func BenchmarkNext(b *testing.B) {
	f := NewFSRS(DefaultParam())
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	for b.Loop() {
		f.Next(card, now, Good)
		f.Next(card, now, Hard)
		f.Next(card, now, Easy)
		f.Next(card, now, Again)
	}
}

func BenchmarkGetRetrievability(b *testing.B) {
	f := NewFSRS(DefaultParam())
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	card = f.Next(card, now, Good).Card
	for b.Loop() {
		f.GetRetrievability(card, now)
	}
}

func BenchmarkBasicSchedulerFullSequence(b *testing.B) {
	f := NewFSRS(DefaultParam())
	ratings := []Rating{Good, Good, Good, Good, Good, Good, Again, Again, Good, Good, Good, Good, Good}
	for b.Loop() {
		card := NewCard()
		now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
		for _, rating := range ratings {
			record := f.Repeat(card, now)[rating]
			card = record.Card
			now = card.Due
		}
	}
}

func BenchmarkLongTermSchedulerFullSequence(b *testing.B) {
	p := DefaultParam()
	p.EnableShortTerm = false
	f := NewFSRS(p)
	ratings := []Rating{Good, Good, Good, Good, Good, Good, Again, Again, Good, Good, Good, Good, Good}
	for b.Loop() {
		card := NewCard()
		now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
		for _, rating := range ratings {
			record := f.Repeat(card, now)[rating]
			card = record.Card
			now = card.Due
		}
	}
}

func BenchmarkNextStates(b *testing.B) {
	p := DefaultParam()
	current := &MemoryState{Stability: 51.344814, Difficulty: 7.005062}
	for b.Loop() {
		p.NextStates(current, 0.9, 21)
	}
}

func BenchmarkNextStates_NewCard(b *testing.B) {
	p := DefaultParam()
	for b.Loop() {
		p.NextStates(nil, 0.9, 0)
	}
}

func BenchmarkNextState(b *testing.B) {
	p := DefaultParam()
	current := &MemoryState{Stability: 51.344814, Difficulty: 7.005062}
	for b.Loop() {
		p.NextState(current, 0.9, 21, Good)
	}
}

func BenchmarkCurrentRetrievability(b *testing.B) {
	p := DefaultParam()
	for b.Loop() {
		p.ForgettingCurve(21, 51.344814)
	}
}

func TestNewFSRSFallbacksToDefaultWeightsOnInvalidInput(t *testing.T) {
	p := DefaultParam()
	p.W[0] = math.NaN()

	fsrs := NewFSRS(p)
	if !reflect.DeepEqual(fsrs.W, DefaultWeights()) {
		t.Fatalf("expected default weights fallback, got=%v", fsrs.W)
	}
}

func TestNewStateStepBased(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)

	againCard := schedulingCards[Again].Card
	if againCard.State != Learning {
		t.Errorf("Again on new card should be Learning, got=%v", againCard.State)
	}
	if againCard.ScheduledDays != 0 {
		t.Errorf("Again on new card should have ScheduledDays=0, got=%v", againCard.ScheduledDays)
	}
	if againCard.RemainingSteps != len(p.LearningSteps) {
		t.Errorf("Again should have RemainingSteps=%d, got=%d", len(p.LearningSteps), againCard.RemainingSteps)
	}

	hardCard := schedulingCards[Hard].Card
	if hardCard.State != Learning {
		t.Errorf("Hard on new card should be Learning (step-based), got=%v", hardCard.State)
	}
	if hardCard.ScheduledDays != 0 {
		t.Errorf("Hard on new card should have ScheduledDays=0, got=%v", hardCard.ScheduledDays)
	}

	goodCard := schedulingCards[Good].Card
	if goodCard.State != Learning {
		t.Errorf("Good on new card should go to Learning (step 1 of 2), got=%v", goodCard.State)
	}
	if goodCard.RemainingSteps != 1 {
		t.Errorf("Good should have RemainingSteps=1, got=%d", goodCard.RemainingSteps)
	}

	easyCard := schedulingCards[Easy].Card
	if easyCard.State != Review {
		t.Errorf("Easy on new card should graduate to Review, got=%v", easyCard.State)
	}
}

func TestLearningStateThresholdBranching(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)

	againCard := schedulingCards[Again].Card
	now = againCard.Due
	schedulingCards = fsrs.Repeat(againCard, now)

	againAgainCard := schedulingCards[Again].Card
	if againAgainCard.State != Learning {
		t.Errorf("Again on learning card should stay in Learning (short interval), got=%v", againAgainCard.State)
	}

	hardCard := schedulingCards[Hard].Card
	if hardCard.State != Learning {
		t.Errorf("Hard on learning card should stay in Learning, got=%v", hardCard.State)
	}

	goodCard := schedulingCards[Good].Card
	if goodCard.State != Learning {
		t.Errorf("Good on learning card should stay in Learning (has remaining step), got=%v", goodCard.State)
	}
	if goodCard.ScheduledDays != 0 {
		t.Errorf("Good on learning card should have ScheduledDays=0, got=%v", goodCard.ScheduledDays)
	}

	easyCard := schedulingCards[Easy].Card
	if easyCard.State != Review {
		t.Errorf("Easy on learning card should graduate to Review, got=%v", easyCard.State)
	}
}

func TestLearningStateEasyConstraintWhenGoodGraduates(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)
	againCard := schedulingCards[Again].Card
	now = againCard.Due
	schedulingCards = fsrs.Repeat(againCard, now)

	goodInterval := schedulingCards[Good].Card.ScheduledDays
	easyInterval := schedulingCards[Easy].Card.ScheduledDays

	if easyInterval <= goodInterval {
		t.Errorf("Easy interval (%v) should be > Good interval (%v)",
			easyInterval, goodInterval)
	}
}

func TestReviewStateAgainAlwaysRelearning(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	card = fsrs.Next(card, now, Good).Card
	now = card.Due
	card = fsrs.Next(card, now, Good).Card
	now = card.Due

	schedulingCards := fsrs.Repeat(card, now)
	againCard := schedulingCards[Again].Card

	if againCard.State != Relearning {
		t.Errorf("Again on review card should always go to Relearning, got=%v", againCard.State)
	}
	if againCard.ScheduledDays != 0 {
		t.Errorf("Again on review card should have ScheduledDays=0 (short-term), got=%d", againCard.ScheduledDays)
	}
	if againCard.RemainingSteps != len(p.RelearningSteps) {
		t.Errorf("Again should have RemainingSteps=%d, got=%d", len(p.RelearningSteps), againCard.RemainingSteps)
	}

	expectedDue := now.Add(minutesToDuration(p.RelearningSteps[0]))
	dueDiff := againCard.Due.Sub(expectedDue)
	if dueDiff < 0 {
		dueDiff = -dueDiff
	}
	if dueDiff > time.Second {
		t.Errorf("Again Due should match relearning step, expected=%v got=%v diff=%v",
			expectedDue, againCard.Due, dueDiff)
	}
}

func TestCustomLearningSteps(t *testing.T) {
	p := DefaultParam()
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	p.LearningSteps = []float64{1, 10, 60}
	s := p.NewBasicScheduler(card, now)
	schedulingCards := s.Preview()

	againCard := schedulingCards[Again].Card
	if againCard.State != Learning {
		t.Errorf("Again should be Learning with steps, got=%v", againCard.State)
	}
	if againCard.RemainingSteps != 3 {
		t.Errorf("Again should have RemainingSteps=3, got=%d", againCard.RemainingSteps)
	}

	goodCard := schedulingCards[Good].Card
	if goodCard.State != Learning {
		t.Errorf("Good should stay Learning with 3 steps (goes to step 2), got=%v", goodCard.State)
	}

	easyCard := schedulingCards[Easy].Card
	if easyCard.State != Review {
		t.Errorf("Easy should always graduate, got=%v", easyCard.State)
	}

	p.LearningSteps = []float64{}
	s = p.NewBasicScheduler(card, now)
	schedulingCards = s.Preview()
	goodCard = schedulingCards[Good].Card
	if goodCard.State != Review {
		t.Errorf("Good should graduate with no steps, got=%v", goodCard.State)
	}
}

func TestLearningStateRecallStabilityWhenElapsedDays(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)
	againCard := schedulingCards[Again].Card

	schedulingCards0 := fsrs.Repeat(againCard, againCard.Due)
	goodCardImmediate := schedulingCards0[Good].Card
	stabilityImmediate := goodCardImmediate.Stability

	twoDaysLater := now.Add(2 * 24 * time.Hour)
	schedulingCards2 := fsrs.Repeat(againCard, twoDaysLater)
	goodCardDelayed := schedulingCards2[Good].Card
	stabilityDelayed := goodCardDelayed.Stability

	if stabilityDelayed <= stabilityImmediate {
		t.Errorf("stability with ElapsedDays>0 (%.4f) should be > stability with ElapsedDays=0 (%.4f)",
			stabilityDelayed, stabilityImmediate)
	}
}

func TestLearningStateHardAndEasyWithElapsedDays(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)
	againCard := schedulingCards[Again].Card

	twoDaysLater := now.Add(2 * 24 * time.Hour)
	schedulingCards2 := fsrs.Repeat(againCard, twoDaysLater)

	hardCardDelayed := schedulingCards2[Hard].Card
	if hardCardDelayed.Stability <= againCard.Stability {
		t.Errorf("Hard stability with ElapsedDays=2 (%.4f) should be > initial stability (%.4f)",
			hardCardDelayed.Stability, againCard.Stability)
	}

	easyCardDelayed := schedulingCards2[Easy].Card
	if easyCardDelayed.State != Review {
		t.Errorf("Easy with ElapsedDays=2 should graduate to Review, got=%v", easyCardDelayed.State)
	}

	goodCardDelayed := schedulingCards2[Good].Card
	if easyCardDelayed.ScheduledDays <= goodCardDelayed.ScheduledDays {
		t.Errorf("Easy interval (%v) should be > Good interval (%v) with ElapsedDays>0",
			easyCardDelayed.ScheduledDays, goodCardDelayed.ScheduledDays)
	}
}

func TestThreeStepProgression(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10, 60}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)

	goodCard := schedulingCards[Good].Card
	if goodCard.State != Learning {
		t.Errorf("Good should be Learning at step 1, got=%v", goodCard.State)
	}
	if goodCard.RemainingSteps != 2 {
		t.Errorf("Good should have RS=2, got=%d", goodCard.RemainingSteps)
	}
	expectedDue := now.Add(minutesToDuration(10))
	if goodCard.Due.Sub(expectedDue).Abs() > time.Second {
		t.Errorf("Good delay should be 10min (step 1), Due=%v expected=%v", goodCard.Due, expectedDue)
	}

	now = goodCard.Due
	schedulingCards = fsrs.Repeat(goodCard, now)
	goodCard2 := schedulingCards[Good].Card
	if goodCard2.State != Learning {
		t.Errorf("Good at RS=2 should stay Learning, got=%v", goodCard2.State)
	}
	if goodCard2.RemainingSteps != 1 {
		t.Errorf("Good at RS=2 should have RS=1, got=%d", goodCard2.RemainingSteps)
	}
	expectedDue2 := now.Add(minutesToDuration(60))
	if goodCard2.Due.Sub(expectedDue2).Abs() > time.Second {
		t.Errorf("Good delay at RS=2 should be 60min (step 2), Due=%v expected=%v", goodCard2.Due, expectedDue2)
	}

	now = goodCard2.Due
	schedulingCards = fsrs.Repeat(goodCard2, now)
	goodCard3 := schedulingCards[Good].Card
	if goodCard3.State != Review {
		t.Errorf("Good at RS=1 should graduate, got=%v", goodCard3.State)
	}
}

func TestEmptyLearningStepsAllGraduate(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)
	for _, rating := range []Rating{Again, Hard, Good, Easy} {
		c := schedulingCards[rating].Card
		if c.State != Review {
			t.Errorf("Rating %v with empty steps should graduate to Review, got=%v", rating, c.State)
		}
		if c.ScheduledDays == 0 {
			t.Errorf("Rating %v with empty steps should have interval > 0", rating)
		}
	}
}

func TestHardInterpolation(t *testing.T) {
	p := DefaultParam()

	p.LearningSteps = []float64{1}
	hard := p.hardDelayMinutes(p.LearningSteps)
	if hard != 2 {
		t.Errorf("Hard with 1 step [1]: expected 2, got=%v", hard)
	}

	p.LearningSteps = []float64{1, 10}
	hard = p.hardDelayMinutes(p.LearningSteps)
	if hard != 6 {
		t.Errorf("Hard with 2 steps [1,10]: expected 6, got=%v", hard)
	}

	p.LearningSteps = []float64{1, 10, 60}
	hard = p.hardDelayMinutes(p.LearningSteps)
	if hard != 6 {
		t.Errorf("Hard with 3 steps [1,10,60]: expected 6, got=%v", hard)
	}
}

func TestGoodAfterAgainAdvancesStep(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)
	againCard := schedulingCards[Again].Card

	now = againCard.Due
	schedulingCards = fsrs.Repeat(againCard, now)
	goodCard := schedulingCards[Good].Card

	expectedDue := now.Add(minutesToDuration(10))
	if goodCard.Due.Sub(expectedDue).Abs() > time.Second {
		t.Errorf("Good after Again should have delay=10min (next step), Due=%v expected=%v", goodCard.Due, expectedDue)
	}
}

func TestNextStateSingle(t *testing.T) {
	p := DefaultParam()

	item := p.NextState(nil, 0.9, 0, Good)
	if item.Memory.Stability != p.initStability(Good) {
		t.Errorf("NextState new card stability mismatch: got=%v want=%v", item.Memory.Stability, p.initStability(Good))
	}

	current := &MemoryState{Stability: 5.0, Difficulty: 5.0}
	item = p.NextState(current, 0.9, 1, Good)
	if item.Interval < 1 {
		t.Errorf("NextState interval should be >= 1, got=%v", item.Interval)
	}

	states := p.NextStates(current, 0.9, 1)
	if states.Good != item {
		t.Errorf("NextState(Good) should equal NextStates.Good")
	}
}

func TestReviewLogFields(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	record := fsrs.Next(card, now, Good)
	log := record.ReviewLog

	if !log.Due.IsZero() {
		t.Errorf("ReviewLog.Due should be zero for new card (pre-review), got=%v", log.Due)
	}
	if log.Stability != 0 {
		t.Errorf("ReviewLog.Stability should be 0 for new card (pre-review), got=%v", log.Stability)
	}
	if log.Difficulty != 0 {
		t.Errorf("ReviewLog.Difficulty should be 0 for new card (pre-review), got=%v", log.Difficulty)
	}
	if log.RemainingSteps != 0 {
		t.Errorf("ReviewLog.RemainingSteps should be 0 for new card (pre-review), got=%d", log.RemainingSteps)
	}

	card = record.Card
	previousLastReview := card.LastReview
	now = card.Due
	record = fsrs.Next(card, now, Good)
	log = record.ReviewLog

	if log.Due != previousLastReview {
		t.Errorf("ReviewLog.Due should be the card's last review time, got=%v want=%v", log.Due, previousLastReview)
	}
	if log.Stability == 0 {
		t.Errorf("ReviewLog.Stability should be non-zero after first review")
	}
	if log.Difficulty == 0 {
		t.Errorf("ReviewLog.Difficulty should be non-zero after first review")
	}
	if log.RemainingSteps != 1 {
		t.Errorf("ReviewLog.RemainingSteps should be 1 (Learning card with 1 step remaining), got=%d", log.RemainingSteps)
	}
}

func TestReviewStateAgainWithEmptyRelearningSteps(t *testing.T) {
	p := DefaultParam()
	p.RelearningSteps = []float64{}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	card = fsrs.Next(card, now, Good).Card
	now = card.Due
	card = fsrs.Next(card, now, Good).Card
	now = card.Due

	schedulingCards := fsrs.Repeat(card, now)
	againCard := schedulingCards[Again].Card

	if againCard.State != Review {
		t.Errorf("Again with empty relearning steps should go to Review, got=%v", againCard.State)
	}
	if againCard.ScheduledDays == 0 {
		t.Errorf("Again with empty relearning steps should have interval > 0")
	}
	if againCard.Lapses != 1 {
		t.Errorf("Again should increment Lapses, got=%d", againCard.Lapses)
	}
}

func TestLearningStateRemainingZero(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	card.State = Learning
	card.Stability = 2.0
	card.Difficulty = 5.0
	card.RemainingSteps = 0
	card.LastReview = time.Date(2022, 11, 29, 12, 0, 0, 0, time.UTC)
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)
	for _, rating := range []Rating{Again, Hard, Good} {
		c := schedulingCards[rating].Card
		if c.State != Review {
			t.Errorf("Rating %v with RS=0 should graduate to Review, got=%v", rating, c.State)
		}
	}
}

func TestRelearningStepProgression(t *testing.T) {
	p := DefaultParam()
	p.RelearningSteps = []float64{10, 60}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	card = fsrs.Next(card, now, Good).Card
	now = card.Due
	card = fsrs.Next(card, now, Good).Card
	now = card.Due

	schedulingCards := fsrs.Repeat(card, now)
	againCard := schedulingCards[Again].Card
	if againCard.State != Relearning {
		t.Errorf("Again on review card should go to Relearning, got=%v", againCard.State)
	}
	if againCard.RemainingSteps != 2 {
		t.Errorf("Again should have RS=2, got=%d", againCard.RemainingSteps)
	}

	now = againCard.Due
	schedulingCards = fsrs.Repeat(againCard, now)
	goodCard := schedulingCards[Good].Card
	if goodCard.State != Relearning {
		t.Errorf("Good at RS=2 should stay Relearning (step 1 of 2), got=%v", goodCard.State)
	}
	if goodCard.RemainingSteps != 1 {
		t.Errorf("Good at RS=2 should have RS=1, got=%d", goodCard.RemainingSteps)
	}
	expectedDue := now.Add(minutesToDuration(60))
	if goodCard.Due.Sub(expectedDue).Abs() > time.Second {
		t.Errorf("Good at RS=2 should have delay=60min, Due=%v expected=%v", goodCard.Due, expectedDue)
	}

	now = goodCard.Due
	schedulingCards = fsrs.Repeat(goodCard, now)
	goodCard2 := schedulingCards[Good].Card
	if goodCard2.State != Review {
		t.Errorf("Good at RS=1 should graduate to Review, got=%v", goodCard2.State)
	}

	hardCard := schedulingCards[Hard].Card
	if hardCard.State != Relearning {
		t.Errorf("Hard at RS=1 should stay Relearning, got=%v", hardCard.State)
	}
}

func TestReviewStateZeroInterval(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	card = fsrs.Next(card, now, Good).Card
	card = fsrs.Next(card, now, Good).Card
	card.State = Review

	schedulingCards := fsrs.Repeat(card, now)
	againCard := schedulingCards[Again].Card
	if againCard.Stability == 0 {
		t.Errorf("Again with ElapsedDays=0 should use shortTermStability, got stability=0")
	}

	goodCard := schedulingCards[Good].Card
	if goodCard.Stability == 0 {
		t.Errorf("Good with ElapsedDays=0 should use shortTermStability, got stability=0")
	}
	if goodCard.State != Review {
		t.Errorf("Good should be Review, got=%v", goodCard.State)
	}
	if goodCard.ScheduledDays == 0 {
		t.Errorf("Good should have interval > 0, got=%d", goodCard.ScheduledDays)
	}

	hardCard := schedulingCards[Hard].Card
	if hardCard.Stability == 0 {
		t.Errorf("Hard with ElapsedDays=0 should use shortTermStability, got stability=0")
	}
	if hardCard.ScheduledDays > goodCard.ScheduledDays {
		t.Errorf("Hard interval (%d) should be <= Good interval (%d)", hardCard.ScheduledDays, goodCard.ScheduledDays)
	}

	easyCard := schedulingCards[Easy].Card
	if easyCard.ScheduledDays <= goodCard.ScheduledDays {
		t.Errorf("Easy interval (%d) should be > Good interval (%d)", easyCard.ScheduledDays, goodCard.ScheduledDays)
	}
}

func TestGoodDelayMinutesBoundary(t *testing.T) {
	p := DefaultParam()

	p.LearningSteps = []float64{1, 10}
	delay, ok := p.goodDelayMinutes(p.LearningSteps, 2)
	if !ok || delay != 10 {
		t.Errorf("goodDelayMinutes([1,10], 2): expected (10, true), got=(%v, %v)", delay, ok)
	}

	delay, ok = p.goodDelayMinutes(p.LearningSteps, 1)
	if ok {
		t.Errorf("goodDelayMinutes([1,10], 1): expected graduation (0, false), got=(%v, %v)", delay, ok)
	}

	delay, ok = p.goodDelayMinutes(p.LearningSteps, 0)
	if ok {
		t.Errorf("goodDelayMinutes([1,10], 0): expected (0, false), got=(%v, %v)", delay, ok)
	}

	delay, ok = p.goodDelayMinutes([]float64{}, 1)
	if ok {
		t.Errorf("goodDelayMinutes([], 1): expected (0, false), got=(%v, %v)", delay, ok)
	}

	p.LearningSteps = []float64{1}
	delay, ok = p.goodDelayMinutes(p.LearningSteps, 1)
	if ok {
		t.Errorf("goodDelayMinutes([1], 1): expected graduation (0, false), got=(%v, %v)", delay, ok)
	}

	p.LearningSteps = []float64{1, 10, 60}
	delay, ok = p.goodDelayMinutes(p.LearningSteps, 3)
	if !ok || delay != 10 {
		t.Errorf("goodDelayMinutes([1,10,60], 3): expected (10, true), got=(%v, %v)", delay, ok)
	}

	delay, ok = p.goodDelayMinutes(p.LearningSteps, 2)
	if !ok || delay != 60 {
		t.Errorf("goodDelayMinutes([1,10,60], 2): expected (60, true), got=(%v, %v)", delay, ok)
	}

	delay, ok = p.goodDelayMinutes(p.LearningSteps, 1)
	if ok {
		t.Errorf("goodDelayMinutes([1,10,60], 1): expected graduation (0, false), got=(%v, %v)", delay, ok)
	}
}

func TestNextStateElapsedZero(t *testing.T) {
	p := DefaultParam()

	item := p.NextState(nil, 0.9, 0, Good)
	if item.Memory.Stability != p.initStability(Good) {
		t.Errorf("NextState new card: got stability=%v want=%v", item.Memory.Stability, p.initStability(Good))
	}
	if item.Memory.Difficulty != p.initDifficulty(Good) {
		t.Errorf("NextState new card: got difficulty=%v want=%v", item.Memory.Difficulty, p.initDifficulty(Good))
	}

	current := &MemoryState{Stability: 5.0, Difficulty: 5.0}
	item = p.NextState(current, 0.9, 0, Good)
	expectedS := p.shortTermStability(5.0, Good)
	if math.Abs(item.Memory.Stability-expectedS) > 1e-10 {
		t.Errorf("NextState elapsed=0: got stability=%v want=%v", item.Memory.Stability, expectedS)
	}

	itemElapsed := p.NextState(current, 0.9, 1, Good)
	if itemElapsed.Interval < 1 {
		t.Errorf("NextState elapsed=1: interval should be >= 1, got=%v", itemElapsed.Interval)
	}

	states := p.NextStates(current, 0.9, 0)
	if states.Good != item {
		t.Errorf("NextState(Good, elapsed=0) should equal NextStates.Good")
	}
}

func TestReviewLogAllRatings(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)
	for _, rating := range []Rating{Again, Hard, Good, Easy} {
		log := schedulingCards[rating].ReviewLog
		if log.Rating != rating {
			t.Errorf("ReviewLog.Rating should be %v, got=%v", rating, log.Rating)
		}
		if log.State != New {
			t.Errorf("ReviewLog.State for new card should be New (0), got=%v", log.State)
		}
		if log.Stability != 0 {
			t.Errorf("ReviewLog.Stability for new card should be 0, got=%v", log.Stability)
		}
		if log.Difficulty != 0 {
			t.Errorf("ReviewLog.Difficulty for new card should be 0, got=%v", log.Difficulty)
		}
		if log.RemainingSteps != 0 {
			t.Errorf("ReviewLog.RemainingSteps for new card should be 0, got=%d", log.RemainingSteps)
		}
		if log.Review != now {
			t.Errorf("ReviewLog.Review should be %v, got=%v", now, log.Review)
		}
	}

	learningCard := schedulingCards[Good].Card
	schedulingCards2 := fsrs.Repeat(learningCard, learningCard.Due)
	for _, rating := range []Rating{Again, Hard, Good, Easy} {
		log := schedulingCards2[rating].ReviewLog
		if log.Rating != rating {
			t.Errorf("Learning ReviewLog.Rating should be %v, got=%v", rating, log.Rating)
		}
		if log.Stability == 0 {
			t.Errorf("Learning ReviewLog.Stability for rating %v should be non-zero", rating)
		}
		if log.Difficulty == 0 {
			t.Errorf("Learning ReviewLog.Difficulty for rating %v should be non-zero", rating)
		}
	}
}

func TestLearningStateHardRemainingStepsInvariant(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10, 60}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)
	againCard := schedulingCards[Again].Card
	if againCard.RemainingSteps != 3 {
		t.Errorf("Again should have RS=3, got=%d", againCard.RemainingSteps)
	}

	now = againCard.Due
	schedulingCards = fsrs.Repeat(againCard, now)
	hardCard := schedulingCards[Hard].Card
	if hardCard.RemainingSteps != 3 {
		t.Errorf("Hard should keep RS unchanged (3), got=%d", hardCard.RemainingSteps)
	}
}

func TestLearningStateAgainResetsRemainingSteps(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10, 60}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)
	goodCard := schedulingCards[Good].Card
	if goodCard.RemainingSteps != 2 {
		t.Errorf("Good should have RS=2, got=%d", goodCard.RemainingSteps)
	}

	now = goodCard.Due
	schedulingCards = fsrs.Repeat(goodCard, now)
	againCard := schedulingCards[Again].Card
	if againCard.RemainingSteps != 3 {
		t.Errorf("Again should reset RS to full length (3), got=%d", againCard.RemainingSteps)
	}
}

func TestHardDelayMinutesIntegration(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)
	hardCard := schedulingCards[Hard].Card
	if hardCard.State != Learning {
		t.Errorf("Hard should be Learning, got=%v", hardCard.State)
	}
	expectedHardDelay := p.hardDelayMinutes(p.LearningSteps)
	expectedDue := now.Add(minutesToDuration(expectedHardDelay))
	if hardCard.Due.Sub(expectedDue).Abs() > time.Second {
		t.Errorf("Hard Due should match hardDelayMinutes: due=%v expected=%v", hardCard.Due, expectedDue)
	}
	if hardCard.ScheduledDays != 0 {
		t.Errorf("Hard on new card should have ScheduledDays=0, got=%d", hardCard.ScheduledDays)
	}

	againCard := schedulingCards[Again].Card
	now = againCard.Due
	schedulingCards = fsrs.Repeat(againCard, now)
	hardCard2 := schedulingCards[Hard].Card
	if hardCard2.Due.Sub(now.Add(minutesToDuration(expectedHardDelay))).Abs() > time.Second {
		t.Errorf("Hard in learningState should also match hardDelayMinutes: due=%v", hardCard2.Due)
	}
}

func TestDayScaleSteps(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10, 1440}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := fsrs.Repeat(card, now)

	goodCard := schedulingCards[Good].Card
	if goodCard.State != Learning {
		t.Errorf("Good step 1 should be Learning, got=%v", goodCard.State)
	}
	if goodCard.ScheduledDays != 0 {
		t.Errorf("Good step 1 should have ScheduledDays=0, got=%d", goodCard.ScheduledDays)
	}
	if goodCard.RemainingSteps != 2 {
		t.Errorf("Good should have RS=2, got=%d", goodCard.RemainingSteps)
	}

	now = goodCard.Due
	schedulingCards = fsrs.Repeat(goodCard, now)
	goodCard2 := schedulingCards[Good].Card
	if goodCard2.State != Review {
		t.Errorf("Good step 2 (1440min) should graduate to Review, got=%v", goodCard2.State)
	}
	if goodCard2.ScheduledDays != 1 {
		t.Errorf("Good step 2 (1440min) should have ScheduledDays=1, got=%d", goodCard2.ScheduledDays)
	}
	if goodCard2.RemainingSteps != 1 {
		t.Errorf("Good step 2 should preserve RemainingSteps=1, got=%d", goodCard2.RemainingSteps)
	}
	expectedDue := now.Add(1440 * time.Minute)
	if goodCard2.Due.Sub(expectedDue).Abs() > time.Second {
		t.Errorf("Good step 2 Due should be now+1440min, due=%v expected=%v", goodCard2.Due, expectedDue)
	}

	againCard := schedulingCards[Again].Card
	if againCard.State != Learning {
		t.Errorf("Again should be Learning (step[0]=1min < 1440), got=%v", againCard.State)
	}
}

func TestDayScaleStepRelearning(t *testing.T) {
	p := DefaultParam()
	p.RelearningSteps = []float64{1440}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	card = fsrs.Next(card, now, Good).Card
	now = card.Due
	card = fsrs.Next(card, now, Good).Card
	now = card.Due

	schedulingCards := fsrs.Repeat(card, now)
	againCard := schedulingCards[Again].Card
	if againCard.State != Review {
		t.Errorf("Again with 1440min relearning step should graduate to Review, got=%v", againCard.State)
	}
	if againCard.ScheduledDays != 1 {
		t.Errorf("Again should have ScheduledDays=1 (1440min = 1 day), got=%d", againCard.ScheduledDays)
	}
	if againCard.Lapses != 1 {
		t.Errorf("Again should increment Lapses, got=%d", againCard.Lapses)
	}
}

func TestForget(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	ratings := []Rating{Good, Good, Good}
	for _, r := range ratings {
		schedulingCards := fsrs.Repeat(card, now)
		card = schedulingCards[r].Card
		now = card.Due
	}

	if card.State == New {
		t.Fatal("card should not be New before Forget")
	}

	t.Run("preserves counters", func(t *testing.T) {
		result := fsrs.Forget(card, now, false)
		if result.Card.State != New {
			t.Errorf("expected State=New, got=%v", result.Card.State)
		}
		if result.Card.Due != now {
			t.Errorf("expected Due=now, got=%v", result.Card.Due)
		}
		if result.Card.Reps != card.Reps {
			t.Errorf("expected Reps=%d, got=%d", card.Reps, result.Card.Reps)
		}
		if result.Card.Lapses != card.Lapses {
			t.Errorf("expected Lapses=%d, got=%d", card.Lapses, result.Card.Lapses)
		}
		if result.Card.Stability != 0 {
			t.Errorf("expected Stability=0, got=%v", result.Card.Stability)
		}
		if result.Card.Difficulty != 0 {
			t.Errorf("expected Difficulty=0, got=%v", result.Card.Difficulty)
		}
		if result.Card.ScheduledDays != 0 {
			t.Errorf("expected ScheduledDays=0, got=%d", result.Card.ScheduledDays)
		}
		if !result.Card.LastReview.IsZero() {
			t.Errorf("expected LastReview zero, got=%v", result.Card.LastReview)
		}
		if result.Card.RemainingSteps != 0 {
			t.Errorf("expected RemainingSteps=0, got=%d", result.Card.RemainingSteps)
		}
		if result.ReviewLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", result.ReviewLog.Rating)
		}
		if result.ReviewLog.State != New {
			t.Errorf("expected log State=New, got=%v", result.ReviewLog.State)
		}
		if result.ReviewLog.Due != now {
			t.Errorf("expected log Due=now, got=%v", result.ReviewLog.Due)
		}
		if result.ReviewLog.Review != now {
			t.Errorf("expected log Review=now, got=%v", result.ReviewLog.Review)
		}
		if result.ReviewLog.ScheduledDays != 0 {
			t.Errorf("expected log ScheduledDays=0, got=%d", result.ReviewLog.ScheduledDays)
		}
		if result.ReviewLog.Stability != 0 {
			t.Errorf("expected log Stability=0, got=%v", result.ReviewLog.Stability)
		}
		if result.ReviewLog.Difficulty != 0 {
			t.Errorf("expected log Difficulty=0, got=%v", result.ReviewLog.Difficulty)
		}
		if result.ReviewLog.RemainingSteps != 0 {
			t.Errorf("expected log RemainingSteps=0, got=%d", result.ReviewLog.RemainingSteps)
		}
	})

	t.Run("resets counters", func(t *testing.T) {
		result := fsrs.Forget(card, now, true)
		if result.Card.State != New {
			t.Errorf("expected State=New, got=%v", result.Card.State)
		}
		if result.Card.Due != now {
			t.Errorf("expected Due=now, got=%v", result.Card.Due)
		}
		if result.Card.Reps != 0 {
			t.Errorf("expected Reps=0, got=%d", result.Card.Reps)
		}
		if result.Card.Lapses != 0 {
			t.Errorf("expected Lapses=0, got=%d", result.Card.Lapses)
		}
		if result.Card.Stability != 0 {
			t.Errorf("expected Stability=0, got=%v", result.Card.Stability)
		}
		if result.Card.Difficulty != 0 {
			t.Errorf("expected Difficulty=0, got=%v", result.Card.Difficulty)
		}
		if result.Card.ScheduledDays != 0 {
			t.Errorf("expected ScheduledDays=0, got=%d", result.Card.ScheduledDays)
		}
		if !result.Card.LastReview.IsZero() {
			t.Errorf("expected LastReview zero, got=%v", result.Card.LastReview)
		}
		if result.Card.RemainingSteps != 0 {
			t.Errorf("expected RemainingSteps=0, got=%d", result.Card.RemainingSteps)
		}
	})

	t.Run("already new", func(t *testing.T) {
		newCard := NewCard()
		result := fsrs.Forget(newCard, now, false)
		if result.Card.State != New {
			t.Errorf("expected State=New, got=%v", result.Card.State)
		}
		if result.Card.Due != now {
			t.Errorf("expected Due=now, got=%v", result.Card.Due)
		}
		if result.Card.Reps != 0 {
			t.Errorf("expected Reps=0 for new card, got=%d", result.Card.Reps)
		}
		if result.Card.Lapses != 0 {
			t.Errorf("expected Lapses=0, got=%d", result.Card.Lapses)
		}
		if result.Card.Stability != 0 {
			t.Errorf("expected Stability=0, got=%v", result.Card.Stability)
		}
		if result.Card.Difficulty != 0 {
			t.Errorf("expected Difficulty=0, got=%v", result.Card.Difficulty)
		}
		if !result.Card.LastReview.IsZero() {
			t.Errorf("expected LastReview zero, got=%v", result.Card.LastReview)
		}
		if result.ReviewLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", result.ReviewLog.Rating)
		}
		if result.ReviewLog.State != New {
			t.Errorf("expected log State=New, got=%v", result.ReviewLog.State)
		}
		if result.ReviewLog.Due != now {
			t.Errorf("expected log Due=now, got=%v", result.ReviewLog.Due)
		}
		if result.ReviewLog.Review != now {
			t.Errorf("expected log Review=now, got=%v", result.ReviewLog.Review)
		}
	})

	t.Run("review state card", func(t *testing.T) {
		schedulingCards := fsrs.Repeat(NewCard(), time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC))
		reviewCard := schedulingCards[Good].Card
		if reviewCard.State != Learning && reviewCard.State != Review {
			t.Fatalf("expected Learning or Review, got=%v", reviewCard.State)
		}
		result := fsrs.Forget(reviewCard, reviewCard.Due, false)
		if result.Card.State != New {
			t.Errorf("expected State=New, got=%v", result.Card.State)
		}
		if result.Card.Due != reviewCard.Due {
			t.Errorf("expected Due=%v, got=%v", reviewCard.Due, result.Card.Due)
		}
		if result.Card.Reps != reviewCard.Reps {
			t.Errorf("expected Reps=%d, got=%d", reviewCard.Reps, result.Card.Reps)
		}
		if result.Card.Lapses != reviewCard.Lapses {
			t.Errorf("expected Lapses=%d, got=%d", reviewCard.Lapses, result.Card.Lapses)
		}
		if result.Card.Stability != 0 {
			t.Errorf("expected Stability=0, got=%v", result.Card.Stability)
		}
		if result.Card.Difficulty != 0 {
			t.Errorf("expected Difficulty=0, got=%v", result.Card.Difficulty)
		}
		if !result.Card.LastReview.IsZero() {
			t.Errorf("expected LastReview zero, got=%v", result.Card.LastReview)
		}
		if result.ReviewLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", result.ReviewLog.Rating)
		}
		if result.ReviewLog.State != New {
			t.Errorf("expected log State=New, got=%v", result.ReviewLog.State)
		}
		if result.ReviewLog.Due != reviewCard.Due {
			t.Errorf("expected log Due=%v, got=%v", reviewCard.Due, result.ReviewLog.Due)
		}
		if result.ReviewLog.Review != reviewCard.Due {
			t.Errorf("expected log Review=%v, got=%v", reviewCard.Due, result.ReviewLog.Review)
		}
	})
}

func TestRollback(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	t.Run("restores card after good review", func(t *testing.T) {
		card := NewCard()
		schedulingCards := fsrs.Repeat(card, now)
		result := fsrs.Next(card, now, Good)
		rolledBack, err := fsrs.Rollback(result.Card, result.ReviewLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.State != result.ReviewLog.State {
			t.Errorf("expected State=%v, got=%v", result.ReviewLog.State, rolledBack.State)
		}
		if rolledBack.Stability != result.ReviewLog.Stability {
			t.Errorf("expected Stability=%v, got=%v", result.ReviewLog.Stability, rolledBack.Stability)
		}
		if rolledBack.Difficulty != result.ReviewLog.Difficulty {
			t.Errorf("expected Difficulty=%v, got=%v", result.ReviewLog.Difficulty, rolledBack.Difficulty)
		}
		if rolledBack.ScheduledDays != result.ReviewLog.ScheduledDays {
			t.Errorf("expected ScheduledDays=%d, got=%d", result.ReviewLog.ScheduledDays, rolledBack.ScheduledDays)
		}
		if rolledBack.Due != result.ReviewLog.Due {
			t.Errorf("expected Due=%v, got=%v", result.ReviewLog.Due, rolledBack.Due)
		}
		if !rolledBack.LastReview.IsZero() {
			t.Errorf("expected LastReview zero (New state), got=%v", rolledBack.LastReview)
		}
		if rolledBack.Reps != 0 {
			t.Errorf("expected Reps=0, got=%d", rolledBack.Reps)
		}
		if rolledBack.Lapses != 0 {
			t.Errorf("expected Lapses=0, got=%d", rolledBack.Lapses)
		}
		_ = schedulingCards
	})

	t.Run("decrements lapses on again", func(t *testing.T) {
		card := NewCard()
		schedulingCards := fsrs.Repeat(card, now)
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards = fsrs.Repeat(card, now)
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards = fsrs.Repeat(card, now)
		card = schedulingCards[Again].Card
		if card.Lapses == 0 {
			t.Fatal("expected Lapses > 0 before rollback")
		}
		log := schedulingCards[Again].ReviewLog
		rolledBack, err := fsrs.Rollback(card, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Lapses != card.Lapses-1 {
			t.Errorf("expected Lapses=%d, got=%d", card.Lapses-1, rolledBack.Lapses)
		}
		if rolledBack.Reps != card.Reps-1 {
			t.Errorf("expected Reps=%d, got=%d", card.Reps-1, rolledBack.Reps)
		}
	})

	t.Run("rejects manual rating", func(t *testing.T) {
		card := NewCard()
		log := ReviewLog{Rating: Manual}
		_, err := fsrs.Rollback(card, log)
		if err == nil {
			t.Error("expected error for manual rating")
		}
		if !errors.Is(err, ErrManualRating) {
			t.Errorf("expected ErrManualRating, got=%v", err)
		}
	})

	t.Run("no underflow on zero reps", func(t *testing.T) {
		card := NewCard()
		result := fsrs.Next(card, now, Good)
		rolledBack, err := fsrs.Rollback(result.Card, result.ReviewLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Reps != 0 {
			t.Errorf("expected Reps=0 (no underflow), got=%d", rolledBack.Reps)
		}
	})

	t.Run("lapses unchanged for good rating", func(t *testing.T) {
		card := NewCard()
		result := fsrs.Next(card, now, Good)
		rolledBack, err := fsrs.Rollback(result.Card, result.ReviewLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Lapses != 0 {
			t.Errorf("expected Lapses=0 (no underflow), got=%d", rolledBack.Lapses)
		}
	})

	t.Run("preserves remaining steps from post-review card", func(t *testing.T) {
		card := NewCard()
		schedulingCards := fsrs.Repeat(card, now)
		card = schedulingCards[Again].Card
		result := fsrs.Next(card, now, Good)
		rolledBack, err := fsrs.Rollback(result.Card, result.ReviewLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.RemainingSteps != result.Card.RemainingSteps {
			t.Errorf("expected RemainingSteps=%d (from post-review card), got=%d", result.Card.RemainingSteps, rolledBack.RemainingSteps)
		}
	})

	t.Run("no lapses decrement when again with zero lapses", func(t *testing.T) {
		card := NewCard()
		schedulingCards := fsrs.Repeat(card, now)
		card = schedulingCards[Again].Card
		log := schedulingCards[Again].ReviewLog
		rolledBack, err := fsrs.Rollback(card, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Lapses != card.Lapses {
			t.Errorf("expected Lapses=%d (unchanged), got=%d", card.Lapses, rolledBack.Lapses)
		}
	})

	t.Run("no lapses decrement for good with positive lapses", func(t *testing.T) {
		card := NewCard()
		schedulingCards := fsrs.Repeat(card, now)
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards = fsrs.Repeat(card, now)
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards = fsrs.Repeat(card, now)
		card = schedulingCards[Again].Card
		if card.Lapses == 0 {
			t.Fatal("expected Lapses > 0 before rollback")
		}
		schedulingCards2 := fsrs.Repeat(card, now)
		card = schedulingCards2[Good].Card
		log := schedulingCards2[Good].ReviewLog
		rolledBack, err := fsrs.Rollback(card, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Lapses != card.Lapses {
			t.Errorf("expected Lapses=%d (unchanged for Good), got=%d", card.Lapses, rolledBack.Lapses)
		}
	})

	t.Run("reps zero on card with zero reps", func(t *testing.T) {
		card := Card{State: Review, Reps: 0, Lapses: 0}
		log := ReviewLog{Rating: Good, State: Review}
		rolledBack, err := fsrs.Rollback(card, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Reps != 0 {
			t.Errorf("expected Reps=0, got=%d", rolledBack.Reps)
		}
	})

	t.Run("manual rating string", func(t *testing.T) {
		if Manual.String() != "Manual" {
			t.Errorf("expected Manual.String()=Manual, got=%s", Manual.String())
		}
	})

	t.Run("again on relearning preserves lapses", func(t *testing.T) {
		card := NewCard()
		schedulingCards := fsrs.Repeat(card, now)
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards = fsrs.Repeat(card, now)
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards = fsrs.Repeat(card, now)
		card = schedulingCards[Again].Card
		if card.State != Relearning {
			t.Fatalf("expected Relearning state, got=%v", card.State)
		}
		if card.Lapses == 0 {
			t.Fatal("expected Lapses > 0")
		}
		schedulingCards2 := fsrs.Repeat(card, now)
		card = schedulingCards2[Again].Card
		log := schedulingCards2[Again].ReviewLog
		if log.State != Relearning {
			t.Fatalf("expected log State=Relearning, got=%v", log.State)
		}
		rolledBack, err := fsrs.Rollback(card, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Lapses != card.Lapses {
			t.Errorf("expected Lapses=%d (Again on Relearning does not decrement), got=%d", card.Lapses, rolledBack.Lapses)
		}
	})

	t.Run("rejects out of range rating", func(t *testing.T) {
		card := Card{State: Review, Reps: 1}
		log := ReviewLog{Rating: Rating(99), State: Review}
		_, err := fsrs.Rollback(card, log)
		if err == nil {
			t.Error("expected error for out-of-range rating")
		}
		if !errors.Is(err, ErrInvalidRating) {
			t.Errorf("expected ErrInvalidRating, got=%v", err)
		}
	})
}

func TestReschedule(t *testing.T) {
	p := DefaultParam()
	f := NewFSRS(p)

	t.Run("basic grade replay matches direct replay", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}
		card := NewCard()
		result, err := f.Reschedule(card, reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 4 {
			t.Fatalf("expected 4 collections, got %d", len(result.Collections))
		}

		curCard := Card{Due: card.Due}
		for i, review := range reviews {
			direct := f.Next(curCard, review.Review, review.Rating)
			if !reflect.DeepEqual(result.Collections[i].Card, direct.Card) {
				t.Errorf("review %d: reschedule card mismatch\n  got:  %+v\n  want: %+v", i, result.Collections[i].Card, direct.Card)
			}
			curCard = result.Collections[i].Card
		}
	})

	t.Run("basic grade exact values", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantIvl := []uint64{0, 2, 16, 53}
		for i, want := range wantIvl {
			if got := result.Collections[i].Card.ScheduledDays; got != want {
				t.Errorf("review %d: scheduled_days=%d, want=%d", i, got, want)
			}
		}

		wantStab := []float64{2.3065, 2.3065, 16.1880, 52.7633}
		for i, want := range wantStab {
			if got := roundFloat(result.Collections[i].Card.Stability, 4); got != want {
				t.Errorf("review %d: stability=%.10f, want=%.4f", i, result.Collections[i].Card.Stability, want)
			}
		}

		wantDiff := []float64{2.1181, 2.1112, 2.1043, 2.0975}
		for i, want := range wantDiff {
			if got := roundFloat(result.Collections[i].Card.Difficulty, 4); got != want {
				t.Errorf("review %d: difficulty=%.10f, want=%.4f", i, result.Collections[i].Card.Difficulty, want)
			}
		}

		if result.RescheduleItem == nil {
			t.Error("expected non-nil reschedule_item")
		}
	})

	t.Run("basic grade long-term", func(t *testing.T) {
		ltParam := DefaultParam()
		ltParam.EnableShortTerm = false
		ltFsrs := NewFSRS(ltParam)

		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		result, err := ltFsrs.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantIvl := []uint64{3, 3, 16, 53}
		for i, want := range wantIvl {
			if got := result.Collections[i].Card.ScheduledDays; got != want {
				t.Errorf("review %d: scheduled_days=%d, want=%d", i, got, want)
			}
		}

		wantStab := []float64{2.3065, 2.3065, 16.1880, 52.7633}
		for i, want := range wantStab {
			if got := roundFloat(result.Collections[i].Card.Stability, 4); got != want {
				t.Errorf("review %d: stability=%.10f, want=%.4f", i, result.Collections[i].Card.Stability, want)
			}
		}

		if result.RescheduleItem == nil {
			t.Error("expected non-nil reschedule_item")
		}
	})

	t.Run("empty reviews", func(t *testing.T) {
		result, err := f.Reschedule(NewCard(), nil, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 0 {
			t.Errorf("expected 0 collections, got %d", len(result.Collections))
		}
		if result.RescheduleItem != nil {
			t.Error("expected nil reschedule_item")
		}
	})

	t.Run("manual rating with state New", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Good, Review: reviewTime},
			{Rating: Manual, Review: reviewTime.Add(24 * time.Hour), State: StatePtr(New)},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Fatalf("expected 2 collections, got %d", len(result.Collections))
		}

		lastCard := result.Collections[1].Card
		if lastCard.State != New {
			t.Errorf("expected State=New, got=%v", lastCard.State)
		}
		if lastCard.Stability != 0 {
			t.Errorf("expected Stability=0, got=%v", lastCard.Stability)
		}
		if lastCard.Difficulty != 0 {
			t.Errorf("expected Difficulty=0, got=%v", lastCard.Difficulty)
		}
		if lastCard.Due.IsZero() {
			t.Error("expected non-zero Due")
		}

		lastLog := result.Collections[1].ReviewLog
		if lastLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", lastLog.Rating)
		}
		if lastLog.State != New {
			t.Errorf("expected log State=New, got=%v", lastLog.State)
		}
		if !lastLog.Due.Equal(reviewTime.Add(24 * time.Hour)) {
			t.Errorf("expected log Due=reviewed time, got=%v", lastLog.Due)
		}
		if lastLog.ScheduledDays != 0 {
			t.Errorf("expected log ScheduledDays=0, got=%d", lastLog.ScheduledDays)
		}
	})

	t.Run("manual rating with state New and explicit Due", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 20, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Good, Review: reviewTime},
			{Rating: Manual, Review: reviewTime.Add(24 * time.Hour), State: StatePtr(New), Due: manualDue},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		lastCard := result.Collections[1].Card
		if !lastCard.Due.Equal(manualDue) {
			t.Errorf("expected Due=%v (explicit), got=%v", manualDue, lastCard.Due)
		}
		if lastCard.ScheduledDays != 6 {
			t.Errorf("expected ScheduledDays=6, got=%d", lastCard.ScheduledDays)
		}
	})

	t.Run("manual rating with state New and Due before Review", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		dueBeforeReview := time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Good, Review: reviewTime},
			{Rating: Manual, Review: reviewTime.Add(24 * time.Hour), State: StatePtr(New), Due: dueBeforeReview},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		lastCard := result.Collections[1].Card
		if !lastCard.Due.Equal(dueBeforeReview) {
			t.Errorf("expected Due=%v, got=%v", dueBeforeReview, lastCard.Due)
		}
		if lastCard.ScheduledDays != 0 {
			t.Errorf("expected ScheduledDays=0 when Due <= Review, got=%d", lastCard.ScheduledDays)
		}
	})

	t.Run("manual rating with state Review", func(t *testing.T) {
		review1 := time.Date(2024, 8, 12, 1, 0, 0, 0, time.UTC)
		review2 := time.Date(2024, 8, 13, 1, 0, 0, 0, time.UTC)
		review3 := time.Date(2024, 8, 14, 1, 0, 0, 0, time.UTC)
		review4 := time.Date(2024, 8, 15, 1, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 4, 17, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Easy, Review: review1},
			{Rating: Good, Review: review2},
			{Rating: Manual, Review: review3, State: StatePtr(Review), Stability: 21.79806877, Difficulty: 3.2828565, Due: manualDue},
			{Rating: Good, Review: review4},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 4 {
			t.Fatalf("expected 4 collections, got %d", len(result.Collections))
		}

		manualCard := result.Collections[2].Card
		if manualCard.State != Review {
			t.Errorf("expected State=Review, got=%v", manualCard.State)
		}
		if math.Abs(manualCard.Stability-21.79806877) > 1e-6 {
			t.Errorf("expected Stability=21.79806877, got=%v", manualCard.Stability)
		}
		if math.Abs(manualCard.Difficulty-3.2828565) > 1e-6 {
			t.Errorf("expected Difficulty=3.2828565, got=%v", manualCard.Difficulty)
		}
		if !manualCard.Due.Equal(manualDue) {
			t.Errorf("expected Due=%v, got=%v", manualDue, manualCard.Due)
		}
		if manualCard.Reps != 3 {
			t.Errorf("expected Reps=3, got=%d", manualCard.Reps)
		}
		if manualCard.ScheduledDays != 21 {
			t.Errorf("expected ScheduledDays=21, got=%d", manualCard.ScheduledDays)
		}
		if !manualCard.LastReview.Equal(review3) {
			t.Errorf("expected LastReview=%v, got=%v", review3, manualCard.LastReview)
		}

		manualLog := result.Collections[2].ReviewLog
		if manualLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", manualLog.Rating)
		}
		if manualLog.State != Review {
			t.Errorf("expected log State=Review (pre-review), got=%v", manualLog.State)
		}
		if manualLog.ScheduledDays != 13 {
			t.Errorf("expected log ScheduledDays=13, got=%d", manualLog.ScheduledDays)
		}
	})

	t.Run("manual without state returns error", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Easy, Review: reviewTime},
			{Rating: Good, Review: reviewTime.Add(24 * time.Hour)},
			{Rating: Manual, Review: reviewTime.Add(48 * time.Hour)},
			{Rating: Good, Review: reviewTime.Add(72 * time.Hour)},
		}

		_, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err == nil {
			t.Fatal("expected error for manual rating without state")
		}
		if !errors.Is(err, ErrManualStateRequired) {
			t.Errorf("expected ErrManualStateRequired, got=%v", err)
		}
	})

	t.Run("manual non-New without due returns error", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Easy, Review: reviewTime},
			{Rating: Good, Review: reviewTime.Add(24 * time.Hour)},
			{Rating: Manual, Review: reviewTime.Add(48 * time.Hour), State: StatePtr(Review)},
			{Rating: Good, Review: reviewTime.Add(72 * time.Hour)},
		}

		_, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err == nil {
			t.Fatal("expected error for manual rating without due")
		}
		if !errors.Is(err, ErrManualDueRequired) {
			t.Errorf("expected ErrManualDueRequired, got=%v", err)
		}
	})

	t.Run("reschedule item generated", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		currentCard := Card{
			Due:       time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC),
			State:     Review,
			Reps:      5,
			Stability: 50.0,
		}
		now := time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC)

		result, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			UpdateMemoryState: true,
			Now:               now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item")
		}
		if result.RescheduleItem.ReviewLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", result.RescheduleItem.ReviewLog.Rating)
		}
		if !result.RescheduleItem.Card.LastReview.Equal(now) {
			t.Errorf("expected card LastReview=now, got=%v", result.RescheduleItem.Card.LastReview)
		}
		if result.RescheduleItem.Card.Reps != currentCard.Reps+1 {
			t.Errorf("expected card Reps=%d, got=%d", currentCard.Reps+1, result.RescheduleItem.Card.Reps)
		}
	})

	t.Run("reschedule item null when due matches", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		lastCard := result.Collections[len(result.Collections)-1].Card

		currentCard := lastCard
		result2, err := f.Reschedule(currentCard, reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result2.RescheduleItem != nil {
			t.Error("expected nil reschedule_item when due matches")
		}
	})

	t.Run("skip manual filters Manual ratings", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Good, Review: reviewTime},
			{Rating: Manual, Review: reviewTime.Add(24 * time.Hour), State: StatePtr(New)},
			{Rating: Good, Review: reviewTime.Add(48 * time.Hour)},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{SkipManual: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Errorf("expected 2 collections (Manual filtered), got %d", len(result.Collections))
		}
	})

	t.Run("skip manual with all-Manual reviews yields empty collections", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Manual, Review: r1, State: StatePtr(New)},
			{Rating: Manual, Review: r2, State: StatePtr(New)},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{SkipManual: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 0 {
			t.Errorf("expected 0 collections, got %d", len(result.Collections))
		}
		if result.RescheduleItem != nil {
			t.Error("expected nil RescheduleItem with empty collections")
		}
	})

	t.Run("sort reviews by review time", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		sortedReviews := make([]ReviewHistory, len(reviewTimes))
		sortedTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		for i, rt := range sortedTimes {
			sortedReviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		resultSorted, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{SortReviews: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resultPreSorted, err := f.Reschedule(NewCard(), sortedReviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for i := range resultSorted.Collections {
			if !reflect.DeepEqual(resultSorted.Collections[i].Card, resultPreSorted.Collections[i].Card) {
				t.Errorf("review %d: sorted result doesn't match pre-sorted", i)
			}
		}
	})

	t.Run("first card override", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		firstDue := time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC)
		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{FirstDue: firstDue})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 4 {
			t.Fatalf("expected 4 collections, got %d", len(result.Collections))
		}
	})

	t.Run("forget scenario", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		firstDue := time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC)

		var currentCard Card
		var historyCards []Card
		for _, review := range reviews {
			item := f.Next(currentCard, review.Review, review.Rating)
			currentCard = item.Card
			historyCards = append(historyCards, currentCard)
		}

		forgetItem := f.Forget(currentCard, time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC), false)
		currentCard = forgetItem.Card

		result, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			FirstDue:          firstDue,
			UpdateMemoryState: true,
			Now:               time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item")
		}
		lastHistory := historyCards[len(historyCards)-1]
		if !result.RescheduleItem.Card.Due.Equal(lastHistory.Due) {
			t.Errorf("expected card Due=%v, got=%v", lastHistory.Due, result.RescheduleItem.Card.Due)
		}
		if math.Abs(result.RescheduleItem.Card.Stability-lastHistory.Stability) > 1e-10 {
			t.Errorf("expected card Stability=%v, got=%v", lastHistory.Stability, result.RescheduleItem.Card.Stability)
		}
		if math.Abs(result.RescheduleItem.Card.Difficulty-lastHistory.Difficulty) > 1e-10 {
			t.Errorf("expected card Difficulty=%v, got=%v", lastHistory.Difficulty, result.RescheduleItem.Card.Difficulty)
		}
	})

	t.Run("manual rating without stability uses card values", func(t *testing.T) {
		review1 := time.Date(2024, 8, 12, 1, 0, 0, 0, time.UTC)
		review2 := time.Date(2024, 8, 13, 1, 0, 0, 0, time.UTC)
		review3 := time.Date(2024, 8, 14, 1, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 4, 17, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Easy, Review: review1},
			{Rating: Good, Review: review2},
			{Rating: Manual, Review: review3, State: StatePtr(Review), Due: manualDue},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		manualCard := result.Collections[2].Card
		prevCard := result.Collections[1].Card
		if manualCard.Stability != prevCard.Stability {
			t.Errorf("expected Stability=%v (from previous card), got=%v", prevCard.Stability, manualCard.Stability)
		}
		if manualCard.Difficulty != prevCard.Difficulty {
			t.Errorf("expected Difficulty=%v (from previous card), got=%v", prevCard.Difficulty, manualCard.Difficulty)
		}
	})

	t.Run("sort reviews does not mutate caller slice", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}
		originalOrder := make([]time.Time, len(reviews))
		for i, r := range reviews {
			originalOrder[i] = r.Review
		}

		_, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{SortReviews: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for i, r := range reviews {
			if !r.Review.Equal(originalOrder[i]) {
				t.Errorf("caller slice mutated at index %d: got=%v, want=%v", i, r.Review, originalOrder[i])
			}
		}
	})

	t.Run("sort reviews with skip manual combined", func(t *testing.T) {
		reviews := []ReviewHistory{
			{Rating: Good, Review: time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC)},
			{Rating: Manual, Review: time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC), State: StatePtr(New)},
			{Rating: Good, Review: time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{SortReviews: true, SkipManual: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Fatalf("expected 2 collections, got %d", len(result.Collections))
		}
		firstReview := result.Collections[0].ReviewLog.Review
		secondReview := result.Collections[1].ReviewLog.Review
		if !firstReview.Before(secondReview) {
			t.Errorf("expected sorted order: first=%v should be before second=%v", firstReview, secondReview)
		}
	})

	t.Run("single review", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{{Rating: Good, Review: reviewTime}}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 1 {
			t.Fatalf("expected 1 collection, got %d", len(result.Collections))
		}
		if result.Collections[0].Card.State == New {
			t.Errorf("expected non-New state after Good review, got=%v", result.Collections[0].Card.State)
		}
	})

	t.Run("all manual reviews", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Manual, Review: r1, State: StatePtr(New)},
			{Rating: Manual, Review: r2, State: StatePtr(New)},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Fatalf("expected 2 collections, got %d", len(result.Collections))
		}
		for i, c := range result.Collections {
			if c.Card.State != New {
				t.Errorf("collection %d: expected State=New, got=%v", i, c.Card.State)
			}
		}
	})

	t.Run("manual rating with Learning state", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 14, 10, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Good, Review: r1},
			{Rating: Manual, Review: r2, State: StatePtr(Learning), Stability: 2.5, Difficulty: 4.0, Due: manualDue},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		manualCard := result.Collections[1].Card
		if manualCard.State != Learning {
			t.Errorf("expected State=Learning, got=%v", manualCard.State)
		}
		if math.Abs(manualCard.Stability-2.5) > 1e-6 {
			t.Errorf("expected Stability=2.5, got=%v", manualCard.Stability)
		}
		if !manualCard.Due.Equal(manualDue) {
			t.Errorf("expected Due=%v, got=%v", manualDue, manualCard.Due)
		}
	})

	t.Run("manual rating with Relearning state", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)
		r3 := time.Date(2024, 9, 15, 0, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 15, 10, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Good, Review: r1},
			{Rating: Manual, Review: r2, State: StatePtr(Review), Stability: 10.0, Difficulty: 5.0, Due: time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC)},
			{Rating: Manual, Review: r3, State: StatePtr(Relearning), Stability: 5.0, Difficulty: 6.0, Due: manualDue},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		relearnCard := result.Collections[2].Card
		if relearnCard.State != Relearning {
			t.Errorf("expected State=Relearning, got=%v", relearnCard.State)
		}
		if math.Abs(relearnCard.Stability-5.0) > 1e-6 {
			t.Errorf("expected Stability=5.0, got=%v", relearnCard.Stability)
		}
	})

	t.Run("update memory state false preserves stability", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		now := time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC)
		currentCard := Card{Due: now, State: Review, Reps: 3, Stability: 10.0, Difficulty: 5.0}

		result, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			UpdateMemoryState: false,
			Now:               now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item")
		}
		if result.RescheduleItem.Card.Stability != currentCard.Stability {
			t.Errorf("with UpdateMemoryState=false: expected Stability=%v (unchanged), got=%v", currentCard.Stability, result.RescheduleItem.Card.Stability)
		}
		if result.RescheduleItem.Card.Difficulty != currentCard.Difficulty {
			t.Errorf("with UpdateMemoryState=false: expected Difficulty=%v (unchanged), got=%v", currentCard.Difficulty, result.RescheduleItem.Card.Difficulty)
		}
	})

	t.Run("reschedule item for manual-last replay with non-New state", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Good, Review: r1},
			{Rating: Manual, Review: r2, State: StatePtr(Review), Stability: 10.0, Difficulty: 5.0, Due: manualDue},
		}

		now := time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC)
		currentCard := Card{Due: now, State: Review, Reps: 3, Stability: 1.0, Difficulty: 1.0}

		result, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			UpdateMemoryState: true,
			Now:               now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item for manual-last replay")
		}
		if result.RescheduleItem.Card.State != Review {
			t.Errorf("expected State=Review, got=%v", result.RescheduleItem.Card.State)
		}
		if math.Abs(result.RescheduleItem.Card.Stability-10.0) > 1e-6 {
			t.Errorf("with UpdateMemoryState: expected Stability=10.0, got=%v", result.RescheduleItem.Card.Stability)
		}
		if math.Abs(result.RescheduleItem.Card.Difficulty-5.0) > 1e-6 {
			t.Errorf("with UpdateMemoryState: expected Difficulty=5.0, got=%v", result.RescheduleItem.Card.Difficulty)
		}
	})

	t.Run("reschedule item for manual-last New state", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Good, Review: r1},
			{Rating: Manual, Review: r2, State: StatePtr(New)},
		}

		now := time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC)
		currentCard := Card{Due: now, State: Review, Reps: 3, Stability: 10.0, Difficulty: 5.0}

		result, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			Now: now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item when current card Due differs from manual-last New card")
		}
		if result.RescheduleItem.Card.State != New {
			t.Errorf("expected State=New, got=%v", result.RescheduleItem.Card.State)
		}
	})

	t.Run("manual due before review produces zero scheduled days", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		dueBeforeReview := time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Good, Review: r1},
			{Rating: Manual, Review: time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC), State: StatePtr(Review), Stability: 5.0, Difficulty: 4.0, Due: dueBeforeReview},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		manualCard := result.Collections[1].Card
		if manualCard.ScheduledDays != 0 {
			t.Errorf("expected ScheduledDays=0 when due < review, got=%d", manualCard.ScheduledDays)
		}
	})

	t.Run("reschedule item with update memory state", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		lastCard := result.Collections[len(result.Collections)-1].Card

		now := time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC)
		currentCard := Card{Due: now, State: Review, Reps: 3, Stability: 10.0, Difficulty: 5.0}

		result2, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			UpdateMemoryState: true,
			Now:               now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result2.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item")
		}

		if math.Abs(result2.RescheduleItem.Card.Stability-lastCard.Stability) > 1e-10 {
			t.Errorf("with UpdateMemoryState: expected Stability=%v, got=%v", lastCard.Stability, result2.RescheduleItem.Card.Stability)
		}
		if math.Abs(result2.RescheduleItem.Card.Difficulty-lastCard.Difficulty) > 1e-10 {
			t.Errorf("with UpdateMemoryState: expected Difficulty=%v, got=%v", lastCard.Difficulty, result2.RescheduleItem.Card.Difficulty)
		}
	})
}

func TestApplyFuzz(t *testing.T) {
	p := DefaultParam()

	t.Run("returns input when fuzz disabled", func(t *testing.T) {
		p.EnableFuzz = false
		got := p.ApplyFuzz(5.0, 0, false)
		if got != 5.0 {
			t.Errorf("expected 5.0, got=%v", got)
		}
	})

	t.Run("returns input when interval below 2.5", func(t *testing.T) {
		got := p.ApplyFuzz(2.3, 0, true)
		if got != 2.3 {
			t.Errorf("expected 2.3, got=%v", got)
		}
	})

	t.Run("fuzz result within expected range at boundary 2.5", func(t *testing.T) {
		p.EnableFuzz = true
		p.seed = "test-seed-2.5"
		minIvl, maxIvl := getFuzzRange(3, 0, p.MaximumInterval)
		got := p.ApplyFuzz(2.5, 0, true)
		if got < float64(minIvl) || got > float64(maxIvl) {
			t.Errorf("expected result in [%d, %d], got=%v", minIvl, maxIvl, got)
		}
	})

	t.Run("fuzz result within expected range for interval 3", func(t *testing.T) {
		p.EnableFuzz = true
		p.seed = "test-seed-3"
		minIvl, maxIvl := getFuzzRange(3, 0, p.MaximumInterval)
		got := p.ApplyFuzz(3.0, 0, true)
		if got < float64(minIvl) || got > float64(maxIvl) {
			t.Errorf("expected result in [%d, %d], got=%v", minIvl, maxIvl, got)
		}
	})

	t.Run("fuzz result clamped by maximum interval", func(t *testing.T) {
		p.EnableFuzz = true
		p.MaximumInterval = 5
		p.seed = "test-seed-max"
		got := p.ApplyFuzz(100.0, 0, true)
		if got > 5 {
			t.Errorf("expected result <= 5 (MaximumInterval), got=%v", got)
		}
	})

	t.Run("fuzz result respects elapsed days floor", func(t *testing.T) {
		p.EnableFuzz = true
		p.MaximumInterval = 36500
		p.seed = "test-seed-elapsed"
		ivl := 5.0
		elapsedDays := 4.0
		minIvl, maxIvl := getFuzzRange(ivl, elapsedDays, p.MaximumInterval)
		got := p.ApplyFuzz(ivl, elapsedDays, true)
		if got < float64(minIvl) || got > float64(maxIvl) {
			t.Errorf("expected result in [%d, %d], got=%v", minIvl, maxIvl, got)
		}
	})

	t.Run("fuzz is deterministic for same seed", func(t *testing.T) {
		p.EnableFuzz = true
		p.seed = "deterministic-seed"
		first := p.ApplyFuzz(10.0, 0, true)
		p.seed = "deterministic-seed"
		second := p.ApplyFuzz(10.0, 0, true)
		if first != second {
			t.Errorf("expected deterministic results: first=%v, second=%v", first, second)
		}
	})

	t.Run("fuzz range at large interval", func(t *testing.T) {
		p.EnableFuzz = true
		p.MaximumInterval = 36500
		p.seed = "test-seed-large"
		minIvl, maxIvl := getFuzzRange(100.0, 0, p.MaximumInterval)
		got := p.ApplyFuzz(100.0, 0, true)
		if got < float64(minIvl) || got > float64(maxIvl) {
			t.Errorf("expected result in [%d, %d], got=%v", minIvl, maxIvl, got)
		}
		if maxIvl > 36500 {
			t.Errorf("expected maxIvl <= 36500, got=%d", maxIvl)
		}
	})
}

func TestGetFuzzRange(t *testing.T) {
	t.Run("small interval uses minimal delta", func(t *testing.T) {
		minIvl, maxIvl := getFuzzRange(3, 0, 36500)
		if minIvl > maxIvl {
			t.Errorf("minIvl > maxIvl: %d > %d", minIvl, maxIvl)
		}
		if minIvl < 2 {
			t.Errorf("expected minIvl >= 2, got=%d", minIvl)
		}
	})

	t.Run("clamped by maximum interval", func(t *testing.T) {
		_, maxIvl := getFuzzRange(100000, 0, 10)
		if maxIvl > 10 {
			t.Errorf("expected maxIvl <= 10, got=%d", maxIvl)
		}
	})

	t.Run("elapsed days floor when interval exceeds elapsed", func(t *testing.T) {
		minIvl, _ := getFuzzRange(5, 4, 36500)
		if minIvl < 5 {
			t.Errorf("expected minIvl >= 5 (elapsedDays+1), got=%d", minIvl)
		}
	})

	t.Run("min never exceeds max", func(t *testing.T) {
		for _, tc := range []struct {
			ivl    float64
			elapsed float64
			maxIvl float64
		}{
			{2.5, 0, 36500},
			{7.0, 0, 36500},
			{20.0, 0, 36500},
			{5.0, 10, 36500},
			{3.0, 0, 2},
		} {
			minIvl, maxIvl := getFuzzRange(tc.ivl, tc.elapsed, tc.maxIvl)
			if minIvl > maxIvl {
				t.Errorf("ivl=%v elapsed=%v maxIvl=%v: minIvl=%d > maxIvl=%d", tc.ivl, tc.elapsed, tc.maxIvl, minIvl, maxIvl)
			}
		}
	})
}
