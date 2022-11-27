package fsrs

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRepeat(t *testing.T) {
	p := DefaultParam()
	card := NewCard()
	now := time.Now()
	schedulingCards := p.Repeat(&card, now)
	schedule, _ := json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards.Good
	now = card.Due
	schedulingCards = p.Repeat(&card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards.Good
	now = card.Due
	schedulingCards = p.Repeat(&card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards.Again
	now = card.Due
	schedulingCards = p.Repeat(&card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards.Good
	now = card.Due
	schedulingCards = p.Repeat(&card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))
}
