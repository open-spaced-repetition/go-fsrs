package fsrs

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"
)

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func TestExample(t *testing.T) {
	p := DefaultParam()
	p.W = Weights{1.0171, 1.8296, 4.4145, 10.9355, 5.0965, 1.3322, 1.017, 0.0, 1.6243, 0.1369, 1.0321,
		2.1866, 0.0661, 0.336, 1.7766, 0.1693, 2.9244}
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	var ivlList []uint64
	var stateList []State
	schedulingCards := p.Repeat(card, now)
	schedule, _ := json.MarshalIndent(schedulingCards, "", "    ")
	fmt.Println(string(schedule))

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
		schedulingCards = p.Repeat(card, now)
		schedule, _ = json.MarshalIndent(schedulingCards, "", "    ")
		fmt.Println(string(schedule))
	}

	fmt.Println(ivlList)
	fmt.Println(stateList)

	wantIvlList := []uint64{0, 4, 15, 49, 143, 379, 0, 0, 15, 37, 85, 184, 376}
	if !reflect.DeepEqual(ivlList, wantIvlList) {
		t.Errorf("excepted:%v, got:%v", wantIvlList, ivlList)
	}
	wantStateList := []State{New, Learning, Review, Review, Review, Review, Review, Relearning, Relearning, Review, Review, Review, Review}
	if !reflect.DeepEqual(stateList, wantStateList) {
		t.Errorf("excepted:%v, got:%v", wantStateList, stateList)
	}
}

func TestMemoState(t *testing.T) {
	p := DefaultParam()
	p.W = Weights{1.0171, 1.8296, 4.4145, 10.9355, 5.0965, 1.3322, 1.017, 0.0, 1.6243, 0.1369, 1.0321,
		2.1866, 0.0661, 0.336, 1.7766, 0.1693, 2.9244}
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards := p.Repeat(card, now)
	var ratings = []Rating{Again, Good, Good, Good, Good, Good}
	var ivlList = []uint64{0, 0, 1, 3, 8, 21}
	var rating Rating
	for i := 0; i < len(ratings); i++ {
		rating = ratings[i]
		card = schedulingCards[rating].Card
		now = now.Add(time.Duration(ivlList[i]) * 24 * time.Hour)
		schedulingCards = p.Repeat(card, now)
	}
	wantStability := 43.0554
	cardStability := roundFloat(schedulingCards[Good].Card.Stability, 4)
	wantDifficulty := 7.7609
	cardDifficulty := roundFloat(schedulingCards[Good].Card.Difficulty, 4)

	if !reflect.DeepEqual(wantStability, cardStability) {
		t.Errorf("excepted:%v, got:%v", wantStability, cardStability)
	}

	if !reflect.DeepEqual(wantDifficulty, cardDifficulty) {
		t.Errorf("excepted:%v, got:%v", wantDifficulty, cardDifficulty)
	}
}
