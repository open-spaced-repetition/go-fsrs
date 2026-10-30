package fsrs

import (
	"errors"
	"math"
	"testing"
	"time"
)

// maxScheduledDaysForMigration is the largest valid scheduled interval: the
// maximum allowed MaximumInterval.
const maxScheduledDaysForMigration uint64 = 36500

func isFiniteValue(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// clampMemoryForMigration mirrors the memory clamp in the "Migrating from v3"
// documented migration recipe. Keep it in sync with the guide.
func clampMemoryForMigration(v, lo, hi float64) float64 {
	if math.IsNaN(v) {
		return lo
	}
	return math.Max(lo, math.Min(v, hi))
}

// clampCardMemory mirrors the memory-state part of the documented migration
// recipe. New cards are left untouched because the scheduler ignores their
// memory state.
func clampCardMemory(card Card) Card {
	if card.State == New {
		return card
	}
	card.Stability = clampMemoryForMigration(card.Stability, SMin, SMax)
	card.Difficulty = clampMemoryForMigration(card.Difficulty, DMin, DMax)
	return card
}

// migrateCard mirrors the full documented migration recipe, including the
// scheduling-metadata reset.
func migrateCard(card Card, now time.Time) Card {
	card = clampCardMemory(card)
	if card.ScheduledDays > maxScheduledDaysForMigration {
		card.ScheduledDays = 0
		card.Due = now
	}
	return card
}

// needsMigration mirrors the documented detection snippet.
func needsMigration(card Card) bool {
	if card.State != New {
		if !isFiniteValue(card.Stability) || card.Stability < SMin || card.Stability > SMax ||
			!isFiniteValue(card.Difficulty) || card.Difficulty < DMin || card.Difficulty > DMax {
			return true
		}
	}
	return card.ScheduledDays > maxScheduledDaysForMigration
}

func assertInvalidInput(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected an error, got nil", label)
	}
	var fsrsErr *Error
	if !errors.As(err, &fsrsErr) {
		t.Fatalf("%s: expected *Error, got %T: %v", label, err, err)
	}
	if fsrsErr.Code != ErrCodeInvalidInput {
		t.Fatalf("%s: expected ErrCodeInvalidInput, got %v", label, fsrsErr.Code)
	}
}

func v3CorruptCard(now time.Time) Card {
	return Card{
		Due:           now.AddDate(0, 0, -3),
		Stability:     0,
		Difficulty:    5,
		ScheduledDays: uint64(1) << 63,
		Reps:          10,
		Lapses:        3,
		State:         Review,
		LastReview:    now.AddDate(0, 0, -3),
	}
}

func assertSchedulable(t *testing.T, f *FSRS, card Card, now time.Time) SchedulingInfo {
	t.Helper()

	info, err := f.Next(card, now, Good)
	if err != nil {
		t.Fatalf("Next after migration: %v", err)
	}
	if !isFiniteValue(info.Card.Stability) || info.Card.Stability < SMin || info.Card.Stability > SMax {
		t.Fatalf("Next stability out of bounds: %v", info.Card.Stability)
	}
	if !isFiniteValue(info.Card.Difficulty) || info.Card.Difficulty < DMin || info.Card.Difficulty > DMax {
		t.Fatalf("Next difficulty out of bounds: %v", info.Card.Difficulty)
	}
	if info.Card.State == Review && info.Card.ScheduledDays == 0 {
		t.Fatal("review result has zero scheduled days")
	}
	if info.Card.Due.IsZero() {
		t.Fatal("Next returned a zero due date")
	}

	if _, err := f.Repeat(card, now); err != nil {
		t.Fatalf("Repeat after migration: %v", err)
	}

	retrievability, err := f.Retrievability(card, now)
	if err != nil {
		t.Fatalf("Retrievability after migration: %v", err)
	}
	if !isFiniteValue(retrievability) || retrievability < 0 || retrievability > 1 {
		t.Fatalf("Retrievability out of range: %v", retrievability)
	}

	if _, err := f.Forget(card, now, false); err != nil {
		t.Fatalf("Forget after migration: %v", err)
	}

	return info
}

func TestExportedBounds(t *testing.T) {
	if SMin != 0.001 {
		t.Errorf("SMin = %v, want 0.001", SMin)
	}
	if SMax != 36500.0 {
		t.Errorf("SMax = %v, want 36500", SMax)
	}
	if DMin != 1.0 {
		t.Errorf("DMin = %v, want 1", DMin)
	}
	if DMax != 10.0 {
		t.Errorf("DMax = %v, want 10", DMax)
	}
	if SMin >= SMax {
		t.Errorf("SMin (%v) must be less than SMax (%v)", SMin, SMax)
	}
	if DMin >= DMax {
		t.Errorf("DMin (%v) must be less than DMax (%v)", DMin, DMax)
	}
}

func TestV3CardMigrationRecipe(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	f := NewFSRS(DefaultParam())

	t.Run("corrupt card is rejected before migration", func(t *testing.T) {
		card := v3CorruptCard(now)

		_, err := f.Next(card, now, Good)
		assertInvalidInput(t, "Next", err)

		_, err = f.Repeat(card, now)
		assertInvalidInput(t, "Repeat", err)

		_, err = f.Retrievability(card, now)
		assertInvalidInput(t, "Retrievability", err)

		_, err = f.Forget(card, now, false)
		assertInvalidInput(t, "Forget", err)
	})

	// This subtest documents why the memory clamp alone is insufficient:
	// buildLog echoes the input ScheduledDays, so the returned log is only
	// safe to persist after the full recipe resets ScheduledDays and Due.
	t.Run("memory clamp schedules but keeps old scheduled days in the log", func(t *testing.T) {
		card := v3CorruptCard(now)
		if !needsMigration(card) {
			t.Fatal("expected corrupt card to need migration")
		}

		clamped := clampCardMemory(card)
		info := assertSchedulable(t, f, clamped, now)

		if info.Card.ScheduledDays < 1 {
			t.Fatalf("Next scheduled days = %d, want >= 1", info.Card.ScheduledDays)
		}
		if info.ReviewLog.ScheduledDays != uint64(1)<<63 {
			t.Fatalf("ReviewLog.ScheduledDays = %d, want %d", info.ReviewLog.ScheduledDays, uint64(1)<<63)
		}
	})

	t.Run("full recipe resets scheduled days and the first log", func(t *testing.T) {
		card := v3CorruptCard(now)
		migrated := migrateCard(card, now)
		if needsMigration(migrated) {
			t.Fatalf("migrated card still needs migration: %+v", migrated)
		}

		info := assertSchedulable(t, f, migrated, now)
		if info.Card.ScheduledDays < 1 || info.Card.ScheduledDays > maxScheduledDaysForMigration {
			t.Fatalf("Next scheduled days = %d, want in [1, %d]", info.Card.ScheduledDays, maxScheduledDaysForMigration)
		}
		if info.ReviewLog.ScheduledDays != 0 {
			t.Fatalf("ReviewLog.ScheduledDays = %d, want 0", info.ReviewLog.ScheduledDays)
		}
	})

	t.Run("recipe is idempotent", func(t *testing.T) {
		migrated := migrateCard(v3CorruptCard(now), now)
		again := migrateCard(migrated, now)
		if again != migrated {
			t.Fatalf("migration is not idempotent:\nfirst:  %+v\nsecond: %+v", migrated, again)
		}
	})

	t.Run("valid card is unchanged", func(t *testing.T) {
		card := Card{
			Due:           now.AddDate(0, 0, 10),
			Stability:     5,
			Difficulty:    5,
			ScheduledDays: 10,
			Reps:          3,
			State:         Review,
			LastReview:    now.AddDate(0, 0, -1),
		}
		if needsMigration(card) {
			t.Fatal("valid card detected as needing migration")
		}
		if got := migrateCard(card, now); got != card {
			t.Fatalf("valid card changed:\nbefore: %+v\nafter:  %+v", card, got)
		}
	})

	t.Run("New card is untouched", func(t *testing.T) {
		card := NewCard(now)
		card.Stability = math.NaN()
		card.Difficulty = math.NaN()

		if needsMigration(card) {
			t.Fatal("New card detected as needing memory migration")
		}
		migrated := migrateCard(card, now)
		if !math.IsNaN(migrated.Stability) || !math.IsNaN(migrated.Difficulty) {
			t.Fatalf("New card memory state changed: %+v", migrated)
		}
		if _, err := f.Next(migrated, now, Good); err != nil {
			t.Fatalf("Next on New card: %v", err)
		}
	})

	t.Run("scheduled days boundary", func(t *testing.T) {
		due := now.AddDate(0, 0, 10)
		atLimit := Card{
			Due:           due,
			Stability:     5,
			Difficulty:    5,
			ScheduledDays: maxScheduledDaysForMigration,
			State:         Review,
			LastReview:    now.AddDate(0, 0, -1),
		}
		if needsMigration(atLimit) {
			t.Fatal("card at the scheduled-days limit needs migration")
		}
		if got := migrateCard(atLimit, now); got != atLimit {
			t.Fatalf("card at the scheduled-days limit changed:\nbefore: %+v\nafter:  %+v", atLimit, got)
		}

		aboveLimit := atLimit
		aboveLimit.ScheduledDays = maxScheduledDaysForMigration + 1
		if !needsMigration(aboveLimit) {
			t.Fatal("card above the scheduled-days limit does not need migration")
		}
		got := migrateCard(aboveLimit, now)
		if got.ScheduledDays != 0 {
			t.Errorf("ScheduledDays = %d, want 0", got.ScheduledDays)
		}
		if !got.Due.Equal(now) {
			t.Errorf("Due = %v, want %v", got.Due, now)
		}
	})

	t.Run("New card keeps memory but resets corrupt metadata", func(t *testing.T) {
		card := NewCard(now)
		card.Stability = math.NaN()
		card.Difficulty = math.NaN()
		card.ScheduledDays = uint64(1) << 63
		card.Due = now.AddDate(0, 0, 10)

		if !needsMigration(card) {
			t.Fatal("New card with corrupt scheduled days does not need migration")
		}
		migrated := migrateCard(card, now)
		if !math.IsNaN(migrated.Stability) || !math.IsNaN(migrated.Difficulty) {
			t.Fatalf("New card memory state changed: %+v", migrated)
		}
		if migrated.ScheduledDays != 0 {
			t.Errorf("ScheduledDays = %d, want 0", migrated.ScheduledDays)
		}
		if !migrated.Due.Equal(now) {
			t.Errorf("Due = %v, want %v", migrated.Due, now)
		}
		if _, err := f.Next(migrated, now, Good); err != nil {
			t.Fatalf("Next on New card: %v", err)
		}
	})

	t.Run("non-finite values", func(t *testing.T) {
		cases := []struct {
			name           string
			stability      float64
			difficulty     float64
			wantStability  float64
			wantDifficulty float64
		}{
			{name: "NaN", stability: math.NaN(), difficulty: math.NaN(), wantStability: SMin, wantDifficulty: DMin},
			{name: "positive infinity", stability: math.Inf(1), difficulty: math.Inf(1), wantStability: SMax, wantDifficulty: DMax},
			{name: "negative infinity", stability: math.Inf(-1), difficulty: math.Inf(-1), wantStability: SMin, wantDifficulty: DMin},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				card := v3CorruptCard(now)
				card.Stability = tc.stability
				card.Difficulty = tc.difficulty
				if !needsMigration(card) {
					t.Fatal("expected card to need migration")
				}

				migrated := migrateCard(card, now)
				if migrated.Stability != tc.wantStability {
					t.Errorf("Stability = %v, want %v", migrated.Stability, tc.wantStability)
				}
				if migrated.Difficulty != tc.wantDifficulty {
					t.Errorf("Difficulty = %v, want %v", migrated.Difficulty, tc.wantDifficulty)
				}
				assertSchedulable(t, f, migrated, now)
			})
		}
	})

	t.Run("below-minimum values in non-New states", func(t *testing.T) {
		for _, state := range []State{Learning, Review, Relearning} {
			t.Run(state.String(), func(t *testing.T) {
				card := Card{
					Due:        now.AddDate(0, 0, -1),
					Stability:  0.0005,
					Difficulty: 0.5,
					State:      state,
					LastReview: now.AddDate(0, 0, -1),
					Reps:       5,
				}
				if !needsMigration(card) {
					t.Fatal("expected card to need migration")
				}

				migrated := migrateCard(card, now)
				if migrated.Stability != SMin {
					t.Errorf("Stability = %v, want %v", migrated.Stability, SMin)
				}
				if migrated.Difficulty != DMin {
					t.Errorf("Difficulty = %v, want %v", migrated.Difficulty, DMin)
				}
				assertSchedulable(t, f, migrated, now)
			})
		}
	})

	t.Run("long-term scheduler", func(t *testing.T) {
		p := DefaultParam()
		p.EnableShortTerm = false
		longTerm := NewFSRS(p)

		migrated := migrateCard(v3CorruptCard(now), now)
		assertSchedulable(t, longTerm, migrated, now)
	})
}

func TestNeedsMigration(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		card Card
		want bool
	}{
		{name: "valid new card", card: NewCard(now), want: false},
		{name: "valid review card", card: Card{State: Review, Stability: 5, Difficulty: 5, ScheduledDays: 10, LastReview: now.AddDate(0, 0, -1)}, want: false},
		{name: "stability at minimum", card: Card{State: Review, Stability: SMin, Difficulty: 5, ScheduledDays: 10}, want: false},
		{name: "stability at maximum", card: Card{State: Review, Stability: SMax, Difficulty: 5, ScheduledDays: 10}, want: false},
		{name: "difficulty at minimum", card: Card{State: Review, Stability: 5, Difficulty: DMin, ScheduledDays: 10}, want: false},
		{name: "difficulty at maximum", card: Card{State: Review, Stability: 5, Difficulty: DMax, ScheduledDays: 10}, want: false},
		{name: "zero stability", card: Card{State: Review, Stability: 0, Difficulty: 5, ScheduledDays: 10}, want: true},
		{name: "zero difficulty", card: Card{State: Review, Stability: 5, Difficulty: 0, ScheduledDays: 10}, want: true},
		{name: "stability above maximum", card: Card{State: Review, Stability: SMax + 1, Difficulty: 5, ScheduledDays: 10}, want: true},
		{name: "difficulty above maximum", card: Card{State: Review, Stability: 5, Difficulty: DMax + 1, ScheduledDays: 10}, want: true},
		{name: "NaN stability", card: Card{State: Review, Stability: math.NaN(), Difficulty: 5, ScheduledDays: 10}, want: true},
		{name: "infinite difficulty", card: Card{State: Review, Stability: 5, Difficulty: math.Inf(1), ScheduledDays: 10}, want: true},
		{name: "scheduled days at limit", card: Card{State: Review, Stability: 5, Difficulty: 5, ScheduledDays: maxScheduledDaysForMigration}, want: false},
		{name: "scheduled days above limit", card: Card{State: Review, Stability: 5, Difficulty: 5, ScheduledDays: maxScheduledDaysForMigration + 1}, want: true},
		{name: "corrupt scheduled days", card: Card{State: Review, Stability: 5, Difficulty: 5, ScheduledDays: uint64(1) << 63}, want: true},
		{name: "New card memory ignored", card: Card{State: New, Stability: 0, Difficulty: 0}, want: false},
		{name: "New card with corrupt scheduled days", card: Card{State: New, Stability: 0, Difficulty: 0, ScheduledDays: uint64(1) << 63}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := needsMigration(tc.card); got != tc.want {
				t.Errorf("needsMigration(%+v) = %v, want %v", tc.card, got, tc.want)
			}
		})
	}
}
