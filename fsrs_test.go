package fsrs

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestSimpleWorkWell(t *testing.T) {
	Example()
}

func Example() {
	p := DefaultParam()
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	schedulingCards := p.Repeat(card, now)
	schedule, _ := json.Marshal(schedulingCards)
	fmt.Println(string(schedule))

	card = schedulingCards[Good].Card
	now = card.Due
	schedulingCards = p.Repeat(card, now)
	schedule, _ = json.Marshal(schedulingCards)
	fmt.Println(string(schedule))

	card = schedulingCards[Good].Card
	now = card.Due
	schedulingCards = p.Repeat(card, now)
	schedule, _ = json.Marshal(schedulingCards)
	fmt.Println(string(schedule))

	card = schedulingCards[Again].Card
	now = card.Due
	schedulingCards = p.Repeat(card, now)
	schedule, _ = json.Marshal(schedulingCards)
	fmt.Println(string(schedule))

	card = schedulingCards[Good].Card
	now = card.Due
	schedulingCards = p.Repeat(card, now)
	schedule, _ = json.Marshal(schedulingCards)
	fmt.Println(string(schedule))
}
