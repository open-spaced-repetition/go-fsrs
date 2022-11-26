package fsrs

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRepeat(t *testing.T) {
	w := DefaultWeights()
	card := NewCard()
	now := time.Now()
	schedulingCards := w.Repeat(&card, now)
	schedule, _ := json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards.Good
	interval := card.ScheduledDays
	now = now.Add(time.Duration(float64(interval) * float64(24*time.Hour)))
	schedulingCards = w.Repeat(&card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards.Good
	interval = card.ScheduledDays
	now = now.Add(time.Duration(float64(interval) * float64(24*time.Hour)))
	schedulingCards = w.Repeat(&card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards.Again
	interval = card.ScheduledDays
	now = now.Add(time.Duration(float64(interval) * float64(24*time.Hour)))
	schedulingCards = w.Repeat(&card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards.Good
	interval = card.ScheduledDays
	now = now.Add(time.Duration(float64(interval) * float64(24*time.Hour)))
	schedulingCards = w.Repeat(&card, now)
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))
}
