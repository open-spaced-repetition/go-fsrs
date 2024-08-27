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
