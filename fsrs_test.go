package fsrs

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRepeat(t *testing.T) {
	w := DefaultWeights()
	card := Card{
		Due:           time.Now(),
		Stability:     0,
		Difficulty:    0,
		ElapsedDays:   0,
		ScheduledDays: 0,
		Reps:          0,
		Lapses:        0,
		Leeched:       false,
		State:         New,
		LastReview:    time.Now(),
		ReviewLogs:    []*ReviewLog{},
	}
	schedulingCards := w.Repeat(&card, time.Now())
	schedule, _ := json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards.Good
	schedulingCards = w.Repeat(&card, time.Now())
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))

	card = schedulingCards.Good
	schedulingCards = w.Repeat(&card, time.Now().Add(time.Duration(10*float64(24*time.Hour))))
	schedule, _ = json.Marshal(schedulingCards)
	t.Logf(string(schedule))
}
