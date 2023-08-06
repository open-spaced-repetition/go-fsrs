package fsrs

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestExample(t *testing.T) {
	p := DefaultParam()
	p.W = Weights{1.14, 1.01, 5.44, 14.67, 5.3024, 1.5662, 1.2503, 0.0028, 1.5489, 0.1763, 0.9953, 2.7473, 0.0179, 0.3105, 0.3976, 0.0, 2.0902}
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

	wantIvlList := []uint64{0, 5, 16, 43, 106, 236, 0, 0, 12, 25, 47, 85, 147}
	if !reflect.DeepEqual(ivlList, wantIvlList) {
		t.Errorf("excepted:%v, got:%v", ivlList, wantIvlList)
	}
	wantStateList := []State{New, Learning, Review, Review, Review, Review, Review, Relearning, Relearning, Review, Review, Review, Review}
	if !reflect.DeepEqual(stateList, wantStateList) {
		t.Errorf("excepted:%v, got:%v", stateList, wantStateList)
	}
}
