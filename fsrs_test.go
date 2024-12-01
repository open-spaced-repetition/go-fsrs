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

	wantIvlList := []uint64{0, 4, 14, 44, 125, 328, 0, 0, 7, 16, 34, 71, 142}
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
	wantStability := 48.4848
	cardStability := roundFloat(schedulingCards[Good].Card.Stability, 4)
	wantDifficulty := 7.0866
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
	wantIvlList := []float64{422, 102, 43, 22, 13, 8, 4, 2, 1, 1}
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
	wantIvlHistory := []uint64{3, 11, 35, 101, 269, 669, 12, 2, 5, 12, 26, 55, 112}
	if !reflect.DeepEqual(ivlHistory, wantIvlHistory) {
		t.Errorf("excepted:%v, got:%v", wantIvlHistory, ivlHistory)
	}
	wantSHistory := []float64{3.173, 10.7389, 34.5776, 100.7483, 269.2838, 669.3093, 11.8987, 2.236, 5.2001, 11.8993, 26.4917, 55.4949, 111.9726}
	if !reflect.DeepEqual(sHisotry, wantSHistory) {
		t.Errorf("excepted:%v, got:%v", wantSHistory, sHisotry)
	}
	wantDHistory := []float64{5.2824, 5.273, 5.2635, 5.2542, 5.2448, 5.2355, 6.7654, 7.794, 7.773, 7.7521, 7.7312, 7.7105, 7.6899}
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
	wantRetrievabilityList := []float64{0, 0.9997, 0.9091, 0.9013}
	if !reflect.DeepEqual(retrievabilityList, wantRetrievabilityList) {
		t.Errorf("excepted:%v, got:%v", wantRetrievabilityList, retrievabilityList)
	}
}
