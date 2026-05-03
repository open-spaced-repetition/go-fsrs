package fsrs

import (
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
