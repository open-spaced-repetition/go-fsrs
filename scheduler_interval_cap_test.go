package fsrs

import (
	"fmt"
	"testing"
	"time"
)

func saturatedReviewCard(now time.Time) Card {
	return Card{
		State:      Review,
		Stability:  SMax,
		Difficulty: DMax,
		Reps:       3,
		Lapses:     1,
		LastReview: now.AddDate(0, 0, -1),
		Due:        now,
	}
}

func normalReviewCard(now time.Time) Card {
	return Card{
		State:      Review,
		Stability:  10,
		Difficulty: 5,
		Reps:       3,
		Lapses:     1,
		LastReview: now.AddDate(0, 0, -1),
		Due:        now,
	}
}

func TestScheduledDaysRespectMaximumInterval(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	ratings := []Rating{Again, Hard, Good, Easy}

	for _, shortTerm := range []bool{true, false} {
		for _, fuzz := range []bool{false, true} {
			for _, maximumInterval := range []float64{1, 5, 30, 36500} {
				name := fmt.Sprintf("shortTerm=%v/fuzz=%v/max=%v", shortTerm, fuzz, maximumInterval)
				t.Run(name, func(t *testing.T) {
					p := DefaultParam()
					p.EnableShortTerm = shortTerm
					p.EnableFuzz = fuzz
					p.MaximumInterval = maximumInterval
					f := NewFSRS(p)

					log, err := f.Repeat(saturatedReviewCard(now), now)
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}

					for _, rating := range ratings {
						card := log[rating].Card
						if card.ScheduledDays > uint64(maximumInterval) {
							t.Errorf("rating %v: ScheduledDays=%d exceeds MaximumInterval=%v", rating, card.ScheduledDays, maximumInterval)
						}
						if card.State != Review {
							if card.ScheduledDays != 0 {
								t.Errorf("rating %v: non-Review outcome has ScheduledDays=%d, want 0", rating, card.ScheduledDays)
							}
							continue
						}
						if card.ScheduledDays < 1 {
							t.Errorf("rating %v: Review outcome has ScheduledDays=%d, want >= 1", rating, card.ScheduledDays)
						}
						wantDue := now.Add(daysToDuration(float64(card.ScheduledDays), maximumInterval))
						if !card.Due.Equal(wantDue) {
							t.Errorf("rating %v: Due=%v, want %v for ScheduledDays=%d", rating, card.Due, wantDue, card.ScheduledDays)
						}
					}

					if fuzz {
						return
					}
					hard := log[Hard].Card.ScheduledDays
					good := log[Good].Card.ScheduledDays
					easy := log[Easy].Card.ScheduledDays
					cap := uint64(maximumInterval)
					if hard != cap || good != cap || easy != cap {
						t.Errorf("saturated card should tie at the cap: hard=%d good=%d easy=%d cap=%d", hard, good, easy, cap)
					}
				})
			}
		}
	}
}

func TestScheduledDaysBelowCapUnchanged(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		shortTerm bool
		want      map[Rating]uint64
	}{
		{
			shortTerm: true,
			want:      map[Rating]uint64{Again: 0, Hard: 12, Good: 13, Easy: 16},
		},
		{
			shortTerm: false,
			want:      map[Rating]uint64{Again: 1, Hard: 12, Good: 13, Easy: 16},
		},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("shortTerm=%v", tt.shortTerm)
		t.Run(name, func(t *testing.T) {
			p := DefaultParam()
			p.EnableShortTerm = tt.shortTerm
			f := NewFSRS(p)

			card := normalReviewCard(now)
			log, err := f.Repeat(card, now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for rating, want := range tt.want {
				got := log[rating].Card.ScheduledDays
				if got != want {
					t.Errorf("rating %v: ScheduledDays=%d, want %d", rating, got, want)
				}
				if log[rating].ReviewLog.ScheduledDays != card.ScheduledDays {
					t.Errorf("rating %v: ReviewLog.ScheduledDays=%d, want input value %d", rating, log[rating].ReviewLog.ScheduledDays, card.ScheduledDays)
				}
			}
		})
	}
}

func TestScheduledDaysPartialSaturation(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	for _, shortTerm := range []bool{true, false} {
		name := fmt.Sprintf("shortTerm=%v", shortTerm)
		t.Run(name, func(t *testing.T) {
			card := normalReviewCard(now)

			baseParam := DefaultParam()
			baseParam.EnableShortTerm = shortTerm
			baseParam.MaximumInterval = 36500
			base, err := NewFSRS(baseParam).Repeat(card, now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			goodBase := base[Good].Card.ScheduledDays
			easyBase := base[Easy].Card.ScheduledDays
			if easyBase <= goodBase+1 {
				t.Fatalf("test setup: base intervals too close (good=%d easy=%d)", goodBase, easyBase)
			}

			maximumInterval := float64(easyBase - 1)
			p := DefaultParam()
			p.EnableShortTerm = shortTerm
			p.MaximumInterval = maximumInterval
			log, err := NewFSRS(p).Repeat(card, now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			hard := log[Hard].Card.ScheduledDays
			good := log[Good].Card.ScheduledDays
			easy := log[Easy].Card.ScheduledDays

			if good != goodBase {
				t.Errorf("Good should stay below the cap: got=%d want=%d", good, goodBase)
			}
			if hard >= good {
				t.Errorf("expected Hard < Good, got hard=%d good=%d", hard, good)
			}
			if easy != uint64(maximumInterval) {
				t.Errorf("Easy should tie at the cap: got=%d cap=%v", easy, maximumInterval)
			}
			if got := log[Easy].Card.ScheduledDays; got > uint64(maximumInterval) {
				t.Errorf("Easy exceeds cap: %d > %v", got, maximumInterval)
			}
		})
	}
}

func TestSubDayMaximumIntervalKeepsReviewFloor(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	for _, shortTerm := range []bool{true, false} {
		for _, maximumInterval := range []float64{0.5, 0.999} {
			name := fmt.Sprintf("shortTerm=%v/max=%v", shortTerm, maximumInterval)
			t.Run(name, func(t *testing.T) {
				p := DefaultParam()
				p.EnableShortTerm = shortTerm
				p.MaximumInterval = maximumInterval
				f := NewFSRS(p)

				log, err := f.Repeat(saturatedReviewCard(now), now)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				for _, rating := range []Rating{Again, Hard, Good, Easy} {
					card := log[rating].Card
					if card.State == Review && card.ScheduledDays < 1 {
						t.Errorf("rating %v: Review outcome has ScheduledDays=%d with MaximumInterval=%v, want >= 1", rating, card.ScheduledDays, maximumInterval)
					}
				}
			})
		}
	}
}

func TestNewCardScheduledDaysRespectMaximumInterval(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	// Basic-scheduler New cards graduate through graduateToReview, and
	// nextInterval already enforces the cap there. The long-term scheduler
	// puts New cards through the same monotonic ordering chain as a review,
	// which is where the post-adjustment cap matters.
	for _, maximumInterval := range []float64{1, 5, 30, 36500} {
		name := fmt.Sprintf("max=%v", maximumInterval)
		t.Run(name, func(t *testing.T) {
			p := DefaultParam()
			p.EnableShortTerm = false
			p.LearningSteps = nil
			p.RelearningSteps = nil
			p.MaximumInterval = maximumInterval
			f := NewFSRS(p)

			log, err := f.Repeat(NewCard(now), now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, rating := range []Rating{Again, Hard, Good, Easy} {
				card := log[rating].Card
				if card.ScheduledDays > uint64(maximumInterval) {
					t.Errorf("rating %v: ScheduledDays=%d exceeds MaximumInterval=%v", rating, card.ScheduledDays, maximumInterval)
				}
				if card.State == Review && card.ScheduledDays < 1 {
					t.Errorf("rating %v: Review outcome has ScheduledDays=%d, want >= 1", rating, card.ScheduledDays)
				}
			}
		})
	}
}
