package fsrs

import (
	"testing"
	"time"
)

func TestDateDiffInDays(t *testing.T) {
	tests := []struct {
		name     string
		last     time.Time
		now      time.Time
		expected uint64
	}{
		{"same day different hours", time.Date(2032, 1, 15, 12, 30, 0, 0, time.UTC), time.Date(2032, 1, 15, 23, 59, 0, 0, time.UTC), 0},
		{"next day", time.Date(2032, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2032, 1, 16, 0, 0, 0, 0, time.UTC), 1},
		{"end of month", time.Date(2032, 1, 31, 12, 0, 0, 0, time.UTC), time.Date(2032, 2, 1, 0, 0, 0, 0, time.UTC), 1},
		{"year boundary", time.Date(2032, 12, 31, 18, 0, 0, 0, time.UTC), time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC), 1},
		{"large span", time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2033, 1, 1, 0, 0, 0, 0, time.UTC), 366},
		{"CET midnight crosses UTC day", time.Date(2032, 1, 15, 0, 30, 0, 0, time.FixedZone("CET", 3600)), time.Date(2032, 1, 15, 12, 0, 0, 0, time.FixedZone("CET", 3600)), 1},
		{"CET evening to next UTC day", time.Date(2032, 1, 14, 23, 30, 0, 0, time.FixedZone("CET", 3600)), time.Date(2032, 1, 15, 1, 0, 0, 0, time.FixedZone("CET", 3600)), 1},
		{"different local days but UTC boundary matters", time.Date(2032, 1, 15, 23, 0, 0, 0, time.UTC), time.Date(2032, 1, 16, 1, 0, 0, 0, time.FixedZone("EST", -18000)), 1},
		{"multiple days", time.Date(2032, 1, 10, 0, 0, 0, 0, time.UTC), time.Date(2032, 1, 20, 0, 0, 0, 0, time.UTC), 10},
		{"reversed dates returns 0", time.Date(2032, 1, 20, 0, 0, 0, 0, time.UTC), time.Date(2032, 1, 10, 0, 0, 0, 0, time.UTC), 0},
		{"reversed same day returns 0", time.Date(2032, 1, 15, 18, 0, 0, 0, time.UTC), time.Date(2032, 1, 15, 6, 0, 0, 0, time.UTC), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dateDiffInDays(tt.last, tt.now)
			if got != tt.expected {
				t.Errorf("dateDiffInDays(%v, %v) = %d, want %d", tt.last, tt.now, got, tt.expected)
			}
		})
	}
}

func TestDateDiffRawMidnightCrossover(t *testing.T) {
	last := time.Date(2032, 1, 15, 23, 0, 0, 0, time.UTC)
	now := time.Date(2032, 1, 16, 1, 0, 0, 0, time.UTC)

	got := dateDiffRaw(last, now)
	if got != 0 {
		t.Errorf("dateDiffRaw(midnight crossover) = %v, want 0 (only 2h elapsed)", got)
	}

	gotCal := dateDiffInDays(last, now)
	if gotCal != 1 {
		t.Errorf("dateDiffInDays(midnight crossover) = %d, want 1 (calendar day)", gotCal)
	}
}

func TestDateDiffRawFullDay(t *testing.T) {
	last := time.Date(2032, 1, 15, 10, 0, 0, 0, time.UTC)
	now := time.Date(2032, 1, 16, 12, 0, 0, 0, time.UTC)

	got := dateDiffRaw(last, now)
	if got != 1 {
		t.Errorf("dateDiffRaw(26h) = %v, want 1", got)
	}
}

func TestRetrievabilityRawVsCalendar(t *testing.T) {
	f := NewFSRS(DefaultParam())
	card := Card{
		Stability:  5.0,
		Difficulty: 5.0,
		State:      Review,
		LastReview: time.Date(2032, 1, 15, 23, 0, 0, 0, time.UTC),
	}

	now := time.Date(2032, 1, 16, 1, 0, 0, 0, time.UTC)

	r, err := f.Retrievability(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r != 1.0 {
		t.Errorf("Retrievability(midnight crossover, 2h) = %v, want 1.0 (t=0, no decay)", r)
	}
}

func TestRetrievabilityReversedDates(t *testing.T) {
	f := NewFSRS(DefaultParam())
	card := Card{
		Stability:  5.0,
		Difficulty: 5.0,
		State:      Review,
		LastReview: time.Date(2032, 1, 16, 12, 0, 0, 0, time.UTC),
	}

	now := time.Date(2032, 1, 15, 12, 0, 0, 0, time.UTC)

	r, _ := f.Retrievability(card, now)
	if r != 1.0 {
		t.Errorf("Retrievability(reversed dates) = %v, want 1.0 (t clamped to 0)", r)
	}
}

func TestHardInterpolation(t *testing.T) {
	p := DefaultParam()

	p.LearningSteps = []float64{1}
	hard := hardDelayMinutes(p.LearningSteps)
	if hard != 2 {
		t.Errorf("Hard with 1 step [1]: expected 2, got=%v", hard)
	}

	p.LearningSteps = []float64{1, 10}
	hard = hardDelayMinutes(p.LearningSteps)
	if hard != 6 {
		t.Errorf("Hard with 2 steps [1,10]: expected 6, got=%v", hard)
	}

	p.LearningSteps = []float64{1, 10, 60}
	hard = hardDelayMinutes(p.LearningSteps)
	if hard != 6 {
		t.Errorf("Hard with 3 steps [1,10,60]: expected 6, got=%v", hard)
	}
}

func TestGoodDelayMinutesBoundary(t *testing.T) {
	p := DefaultParam()

	p.LearningSteps = []float64{1, 10}
	delay, ok := goodDelayMinutes(p.LearningSteps, 2)
	if !ok || delay != 10 {
		t.Errorf("goodDelayMinutes([1,10], 2): expected (10, true), got=(%v, %v)", delay, ok)
	}

	delay, ok = goodDelayMinutes(p.LearningSteps, 1)
	if ok {
		t.Errorf("goodDelayMinutes([1,10], 1): expected graduation (0, false), got=(%v, %v)", delay, ok)
	}

	delay, ok = goodDelayMinutes(p.LearningSteps, 0)
	if ok {
		t.Errorf("goodDelayMinutes([1,10], 0): expected (0, false), got=(%v, %v)", delay, ok)
	}

	delay, ok = goodDelayMinutes([]float64{}, 1)
	if ok {
		t.Errorf("goodDelayMinutes([], 1): expected (0, false), got=(%v, %v)", delay, ok)
	}

	p.LearningSteps = []float64{1}
	delay, ok = goodDelayMinutes(p.LearningSteps, 1)
	if ok {
		t.Errorf("goodDelayMinutes([1], 1): expected graduation (0, false), got=(%v, %v)", delay, ok)
	}

	p.LearningSteps = []float64{1, 10, 60}
	delay, ok = goodDelayMinutes(p.LearningSteps, 3)
	if !ok || delay != 10 {
		t.Errorf("goodDelayMinutes([1,10,60], 3): expected (10, true), got=(%v, %v)", delay, ok)
	}

	delay, ok = goodDelayMinutes(p.LearningSteps, 2)
	if !ok || delay != 60 {
		t.Errorf("goodDelayMinutes([1,10,60], 2): expected (60, true), got=(%v, %v)", delay, ok)
	}

	delay, ok = goodDelayMinutes(p.LearningSteps, 1)
	if ok {
		t.Errorf("goodDelayMinutes([1,10,60], 1): expected graduation (0, false), got=(%v, %v)", delay, ok)
	}
}

func TestHardDelayMinutesIntegration(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hardCard := schedulingCards[Hard].Card
	if hardCard.State != Learning {
		t.Errorf("Hard should be Learning, got=%v", hardCard.State)
	}
	expectedHardDelay := hardDelayMinutes(p.LearningSteps)
	expectedDue := now.Add(minutesToDuration(expectedHardDelay))
	if hardCard.Due.Sub(expectedDue).Abs() > time.Second {
		t.Errorf("Hard Due should match hardDelayMinutes: due=%v expected=%v", hardCard.Due, expectedDue)
	}
	if hardCard.ScheduledDays != 0 {
		t.Errorf("Hard on new card should have ScheduledDays=0, got=%d", hardCard.ScheduledDays)
	}

	againCard := schedulingCards[Again].Card
	now = againCard.Due
	schedulingCards, err = fsrs.Repeat(againCard, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hardCard2 := schedulingCards[Hard].Card
	if hardCard2.Due.Sub(now.Add(minutesToDuration(expectedHardDelay))).Abs() > time.Second {
		t.Errorf("Hard in learningState should also match hardDelayMinutes: due=%v", hardCard2.Due)
	}
}
