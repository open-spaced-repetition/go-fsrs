package fsrs

import (
	"math"
	"reflect"
	"testing"
	"time"
)

var testWeights = Weights{
	0.4197,
	1.1869,
	3.0412,
	15.2441,
	7.1434,
	0.6477,
	1.0007,
	0.0674,
	1.6597,
	0.1712,
	1.1178,
	2.0225,
	0.0904,
	0.3025,
	2.1214,
	0.2498,
	2.9466,
	0.4891,
	0.6468,
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func TestBasicSchedulerExample(t *testing.T) {
	p := DefaultParam()
	p.W = testWeights
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

	wantIvlList := []uint64{0, 4, 17, 62, 198, 563, 0, 0, 9, 27, 74, 190, 457}
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
	p.W = testWeights
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
	wantStability := 71.4554
	cardStability := roundFloat(schedulingCards[Good].Card.Stability, 4)
	wantDifficulty := 5.0976
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
		ivlList = append(ivlList, fsrs.nextInterval(1))
	}
	wantIvlList := []float64{422, 102, 43, 22, 13, 8, 4, 2, 1, 1}
	if !reflect.DeepEqual(ivlList, wantIvlList) {
		t.Errorf("excepted:%v, got:%v", wantIvlList, ivlList)
	}
}

func TestLongTermScheduler(t *testing.T) {
	p := DefaultParam()
	p.W = testWeights
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
	wantIvlHistory := []uint64{3, 13, 48, 155, 445, 1158, 17, 3, 9, 27, 74, 190, 457}
	if !reflect.DeepEqual(ivlHistory, wantIvlHistory) {
		t.Errorf("excepted:%v, got:%v", wantIvlHistory, ivlHistory)
	}
	wantSHistory := []float64{3.0412, 13.0913, 48.1585, 154.9373, 445.0556, 1158.0778, 16.6306, 2.9888, 9.4633, 26.9474, 73.9723, 189.7037, 457.4379}
	if !reflect.DeepEqual(sHisotry, wantSHistory) {
		t.Errorf("excepted:%v, got:%v", wantSHistory, sHisotry)
	}
	wantDHistory := []float64{4.4909, 4.2666, 4.0575, 3.8624, 3.6804, 3.5108, 5.219, 6.8122, 6.4314, 6.0763, 5.7452, 5.4363, 5.1483}
	if !reflect.DeepEqual(dHistory, wantDHistory) {
		t.Errorf("excepted:%v, got:%v", wantDHistory, dHistory)
	}
}
