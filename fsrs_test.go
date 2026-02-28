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

	for i := 0; i < len(ratings); i++ {
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
		t.Errorf("excepted:%v, got:%v", wantIvlList, ivlList)
	}
	wantStateList := []State{New, Learning, Review, Review, Review, Review, Review, Relearning, Relearning, Review, Review, Review, Review}
	if !reflect.DeepEqual(stateList, wantStateList) {
		t.Errorf("excepted:%v, got:%v", wantStateList, stateList)
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
	for i := 0; i < len(ratings); i++ {
		rating = ratings[i]
		card = schedulingCards[rating].Card
		now = now.Add(time.Duration(ivlList[i]) * 24 * time.Hour)
		schedulingCards = fsrs.Repeat(card, now)
	}
	wantStability := 53.7719
	cardStability := roundFloat(schedulingCards[Good].Card.Stability, 4)
	wantDifficulty := 6.3464
	cardDifficulty := roundFloat(schedulingCards[Good].Card.Difficulty, 4)
	if !reflect.DeepEqual(wantStability, cardStability) {
		t.Errorf("excepted:%v, got:%v", wantStability, cardStability)
	}

	if !reflect.DeepEqual(wantDifficulty, cardDifficulty) {
		t.Errorf("excepted:%v, got:%v", wantDifficulty, cardDifficulty)
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
		t.Errorf("excepted:%v, got:%v", wantIvlList, ivlList)
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
	sHisotry := []float64{}
	dHistory := []float64{}
	for _, rating := range ratings {
		record := fsrs.Repeat(card, now)[rating]
		next := fsrs.Next(card, now, rating)
		if !reflect.DeepEqual(record.Card, next.Card) {
			t.Errorf("excepted:%v, got:%v", record.Card, next.Card)
		}

		card = record.Card
		ivlHistory = append(ivlHistory, (card.ScheduledDays))
		sHisotry = append(sHisotry, roundFloat(card.Stability, 4))
		dHistory = append(dHistory, roundFloat(card.Difficulty, 4))
		now = card.Due
	}
	wantIvlHistory := []uint64{3, 14, 57, 196, 586, 1559, 10, 1, 3, 5, 9, 15, 25}
	if !reflect.DeepEqual(ivlHistory, wantIvlHistory) {
		t.Errorf("excepted:%v, got:%v", wantIvlHistory, ivlHistory)
	}
	wantSHistory := []float64{2.3065, 13.8269, 56.9567, 196.2353, 586.4835, 1559.3567, 9.8832, 1.3527, 2.392, 4.8562, 8.7624, 15.186, 25.1362}
	if !reflect.DeepEqual(sHisotry, wantSHistory) {
		t.Errorf("excepted:%v, got:%v", wantSHistory, sHisotry)
	}
	wantDHistory := []float64{2.1181, 2.1112, 2.1043, 2.0975, 2.0906, 2.0837, 7.3832, 9.1251, 9.1112, 9.0973, 9.0835, 9.0696, 9.0558}
	if !reflect.DeepEqual(dHistory, wantDHistory) {
		t.Errorf("excepted:%v, got:%v", wantDHistory, dHistory)
	}
}

func TestGetRetrievability(t *testing.T) {
	retrievabilityList := []float64{}
	fsrs := NewFSRS(DefaultParam())
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	retrievabilityList = append(retrievabilityList, roundFloat(fsrs.GetRetrievability(card, now), 4))
	for i := 0; i < 3; i++ {
		card = fsrs.Next(card, now, Good).Card
		now = card.Due
		retrievabilityList = append(retrievabilityList, roundFloat(fsrs.GetRetrievability(card, now), 4))
	}
	wantRetrievabilityList := []float64{0, 0.9995, 0.9095, 0.8998}
	if !reflect.DeepEqual(retrievabilityList, wantRetrievabilityList) {
		t.Errorf("excepted:%v, got:%v", wantRetrievabilityList, retrievabilityList)
	}
}

func TestDecayAndFactorDerivedFromW20(t *testing.T) {
	p := DefaultParam()
	p.W[20] = 0.2
	p.Decay = -0.5
	p.Factor = math.Pow(0.9, 1.0/p.Decay) - 1.0

	got := p.forgettingCurve(1, 1)
	wantDecay := -p.W[20]
	wantFactor := math.Pow(0.9, 1.0/wantDecay) - 1.0
	want := math.Pow(1+wantFactor, wantDecay)

	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("forgettingCurve should use W[20], got=%v want=%v", got, want)
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

func TestNewFSRSFallbacksToDefaultWeightsOnInvalidInput(t *testing.T) {
	p := DefaultParam()
	p.W[0] = math.NaN()

	fsrs := NewFSRS(p)
	if !reflect.DeepEqual(fsrs.W, DefaultWeights()) {
		t.Fatalf("expected default weights fallback, got=%v", fsrs.W)
	}
}
