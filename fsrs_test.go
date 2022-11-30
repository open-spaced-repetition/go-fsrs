package fsrs

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRepeat(t *testing.T) {
	p := DefaultParam()
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	schedulingCards := p.Repeat(card, now)
	schedule, _ := json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards[Good].Card
	now = card.Due
	schedulingCards = p.Repeat(card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards[Good].Card
	now = card.Due
	schedulingCards = p.Repeat(card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards[Again].Card
	now = card.Due
	schedulingCards = p.Repeat(card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards[Good].Card
	now = card.Due
	schedulingCards = p.Repeat(card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))
}
