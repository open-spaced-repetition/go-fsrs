package fsrs

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestNextInputValidation(t *testing.T) {
	f := NewFSRS(DefaultParam())
	card := NewCard(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	now := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	t.Run("Manual grade returns error", func(t *testing.T) {
		_, err := f.Next(card, now, Manual)
		if err == nil {
			t.Fatal("expected error for Manual grade")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("out of range grade returns error", func(t *testing.T) {
		_, err := f.Next(card, now, Rating(5))
		if err == nil {
			t.Fatal("expected error for grade 5")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("invalid card state returns error", func(t *testing.T) {
		badCard := card
		badCard.State = State(42)
		_, err := f.Next(badCard, now, Good)
		if err == nil {
			t.Fatal("expected error for invalid state")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("NaN stability for non-New card returns error", func(t *testing.T) {
		badCard := card
		badCard.State = Review
		badCard.Stability = math.NaN()
		badCard.Difficulty = 5.0
		_, err := f.Next(badCard, now, Good)
		if err == nil {
			t.Fatal("expected error for NaN stability")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("zero stability for non-New card returns error", func(t *testing.T) {
		badCard := card
		badCard.State = Review
		badCard.Stability = 0
		badCard.Difficulty = 5.0
		_, err := f.Next(badCard, now, Good)
		if err == nil {
			t.Fatal("expected error for zero stability on Review card")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("Inf difficulty for non-New card returns error", func(t *testing.T) {
		badCard := card
		badCard.State = Learning
		badCard.Stability = 1.0
		badCard.Difficulty = math.Inf(1)
		_, err := f.Next(badCard, now, Good)
		if err == nil {
			t.Fatal("expected error for Inf difficulty")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("difficulty below dMin for non-New card returns error", func(t *testing.T) {
		badCard := card
		badCard.State = Review
		badCard.Stability = 5.0
		badCard.Difficulty = 0.5
		_, err := f.Next(badCard, now, Good)
		if err == nil {
			t.Fatal("expected error for difficulty below dMin")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("LastReview after now returns error", func(t *testing.T) {
		badCard := card
		badCard.State = Review
		badCard.Stability = 10.0
		badCard.Difficulty = 5.0
		badCard.LastReview = time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
		_, err := f.Next(badCard, now, Good)
		if err == nil {
			t.Fatal("expected error for future LastReview")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("valid New card succeeds", func(t *testing.T) {
		_, err := f.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid Review card succeeds", func(t *testing.T) {
		reviewCard := card
		reviewCard.State = Review
		reviewCard.Stability = 10.0
		reviewCard.Difficulty = 5.0
		reviewCard.LastReview = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err := f.Next(reviewCard, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid Learning card succeeds", func(t *testing.T) {
		learningCard := card
		learningCard.State = Learning
		learningCard.Stability = 2.0
		learningCard.Difficulty = 5.0
		learningCard.LastReview = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err := f.Next(learningCard, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid Relearning card succeeds", func(t *testing.T) {
		relearningCard := card
		relearningCard.State = Relearning
		relearningCard.Stability = 2.0
		relearningCard.Difficulty = 5.0
		relearningCard.LastReview = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err := f.Next(relearningCard, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("negative stability for non-New card returns error", func(t *testing.T) {
		badCard := card
		badCard.State = Review
		badCard.Stability = -5.0
		badCard.Difficulty = 5.0
		_, err := f.Next(badCard, now, Good)
		if err == nil {
			t.Fatal("expected error for negative stability")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("stability in (0, sMin) for non-New card returns error", func(t *testing.T) {
		badCard := card
		badCard.State = Review
		badCard.Stability = 0.0005
		badCard.Difficulty = 5.0
		_, err := f.Next(badCard, now, Good)
		if err == nil {
			t.Fatal("expected error for stability below sMin")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("negative difficulty for non-New card returns error", func(t *testing.T) {
		badCard := card
		badCard.State = Review
		badCard.Stability = 5.0
		badCard.Difficulty = -1.0
		_, err := f.Next(badCard, now, Good)
		if err == nil {
			t.Fatal("expected error for negative difficulty")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("stability at sMin boundary succeeds", func(t *testing.T) {
		boundaryCard := card
		boundaryCard.State = Review
		boundaryCard.Stability = sMin
		boundaryCard.Difficulty = 5.0
		boundaryCard.LastReview = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err := f.Next(boundaryCard, now, Good)
		if err != nil {
			t.Fatalf("unexpected error at sMin boundary: %v", err)
		}
	})

	t.Run("difficulty at dMin boundary succeeds", func(t *testing.T) {
		boundaryCard := card
		boundaryCard.State = Review
		boundaryCard.Stability = 5.0
		boundaryCard.Difficulty = dMin
		boundaryCard.LastReview = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err := f.Next(boundaryCard, now, Good)
		if err != nil {
			t.Fatalf("unexpected error at dMin boundary: %v", err)
		}
	})

	t.Run("LastReview equal to now succeeds", func(t *testing.T) {
		eqCard := card
		eqCard.State = Review
		eqCard.Stability = 10.0
		eqCard.Difficulty = 5.0
		eqCard.LastReview = now
		_, err := f.Next(eqCard, now, Good)
		if err != nil {
			t.Fatalf("unexpected error when LastReview == now: %v", err)
		}
	})

	t.Run("Again grade boundary succeeds", func(t *testing.T) {
		_, err := f.Next(card, now, Again)
		if err != nil {
			t.Fatalf("unexpected error for Again: %v", err)
		}
	})

	t.Run("Easy grade boundary succeeds", func(t *testing.T) {
		_, err := f.Next(card, now, Easy)
		if err != nil {
			t.Fatalf("unexpected error for Easy: %v", err)
		}
	})
}

func TestRepeatInputValidation(t *testing.T) {
	f := NewFSRS(DefaultParam())
	card := NewCard(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	now := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	t.Run("invalid card state returns error", func(t *testing.T) {
		badCard := card
		badCard.State = State(99)
		_, err := f.Repeat(badCard, now)
		if err == nil {
			t.Fatal("expected error for invalid state")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("valid New card succeeds", func(t *testing.T) {
		_, err := f.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid Review card succeeds", func(t *testing.T) {
		reviewCard := card
		reviewCard.State = Review
		reviewCard.Stability = 10.0
		reviewCard.Difficulty = 5.0
		reviewCard.LastReview = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err := f.Repeat(reviewCard, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid Learning card succeeds", func(t *testing.T) {
		learningCard := card
		learningCard.State = Learning
		learningCard.Stability = 2.0
		learningCard.Difficulty = 5.0
		learningCard.LastReview = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err := f.Repeat(learningCard, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid Relearning card succeeds", func(t *testing.T) {
		relearningCard := card
		relearningCard.State = Relearning
		relearningCard.Stability = 2.0
		relearningCard.Difficulty = 5.0
		relearningCard.LastReview = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		_, err := f.Repeat(relearningCard, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("NaN stability for non-New card returns error", func(t *testing.T) {
		badCard := card
		badCard.State = Review
		badCard.Stability = math.NaN()
		badCard.Difficulty = 5.0
		_, err := f.Repeat(badCard, now)
		if err == nil {
			t.Fatal("expected error for NaN stability")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("zero stability for non-New card returns error", func(t *testing.T) {
		badCard := card
		badCard.State = Review
		badCard.Stability = 0
		badCard.Difficulty = 5.0
		_, err := f.Repeat(badCard, now)
		if err == nil {
			t.Fatal("expected error for zero stability")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("difficulty below dMin for non-New card returns error", func(t *testing.T) {
		badCard := card
		badCard.State = Review
		badCard.Stability = 5.0
		badCard.Difficulty = 0.5
		_, err := f.Repeat(badCard, now)
		if err == nil {
			t.Fatal("expected error for difficulty below dMin")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})
}

func TestRetrievabilityInputValidation(t *testing.T) {
	f := NewFSRS(DefaultParam())
	now := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	t.Run("New card returns 0 without error", func(t *testing.T) {
		card := NewCard(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
		r, err := f.Retrievability(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r != 0 {
			t.Errorf("expected 0, got=%v", r)
		}
	})

	t.Run("card with zero LastReview returns 0 without error", func(t *testing.T) {
		card := Card{State: Review, Stability: 10.0, Difficulty: 5.0}
		r, err := f.Retrievability(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r != 0 {
			t.Errorf("expected 0, got=%v", r)
		}
	})

	t.Run("invalid state returns error", func(t *testing.T) {
		card := Card{State: State(99), Stability: 10.0, Difficulty: 5.0, LastReview: now.Add(-24 * time.Hour)}
		_, err := f.Retrievability(card, now)
		if err == nil {
			t.Fatal("expected error for invalid state")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("NaN stability returns error", func(t *testing.T) {
		card := Card{State: Review, Stability: math.NaN(), Difficulty: 5.0, LastReview: now.Add(-24 * time.Hour)}
		_, err := f.Retrievability(card, now)
		if err == nil {
			t.Fatal("expected error for NaN stability")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("negative stability returns error", func(t *testing.T) {
		card := Card{State: Review, Stability: -1.0, Difficulty: 5.0, LastReview: now.Add(-24 * time.Hour)}
		_, err := f.Retrievability(card, now)
		if err == nil {
			t.Fatal("expected error for negative stability")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("valid Review card succeeds", func(t *testing.T) {
		card := Card{State: Review, Stability: 10.0, Difficulty: 5.0, LastReview: now.Add(-24 * time.Hour)}
		r, err := f.Retrievability(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r <= 0 || r > 1 {
			t.Errorf("expected retrievability in (0,1], got=%v", r)
		}
	})

	t.Run("zero stability returns error", func(t *testing.T) {
		card := Card{State: Review, Stability: 0, Difficulty: 5.0, LastReview: now.Add(-24 * time.Hour)}
		_, err := f.Retrievability(card, now)
		if err == nil {
			t.Fatal("expected error for zero stability")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("valid Learning card succeeds", func(t *testing.T) {
		card := Card{State: Learning, Stability: 2.0, Difficulty: 5.0, LastReview: now.Add(-24 * time.Hour)}
		r, err := f.Retrievability(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r <= 0 || r > 1 {
			t.Errorf("expected retrievability in (0,1], got=%v", r)
		}
	})

	t.Run("valid Relearning card succeeds", func(t *testing.T) {
		card := Card{State: Relearning, Stability: 2.0, Difficulty: 5.0, LastReview: now.Add(-24 * time.Hour)}
		r, err := f.Retrievability(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r <= 0 || r > 1 {
			t.Errorf("expected retrievability in (0,1], got=%v", r)
		}
	})
}

func TestValidateResult(t *testing.T) {
	t.Run("valid card passes", func(t *testing.T) {
		card := Card{Stability: 5.0, Difficulty: 5.0}
		if err := validateResult(card); err != nil {
			t.Errorf("expected nil, got=%v", err)
		}
	})

	t.Run("stability below sMin returns error", func(t *testing.T) {
		card := Card{Stability: 0.0001, Difficulty: 5.0}
		err := validateResult(card)
		if err == nil {
			t.Fatal("expected error for stability below sMin")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("NaN stability returns error", func(t *testing.T) {
		card := Card{Stability: math.NaN(), Difficulty: 5.0}
		err := validateResult(card)
		if err == nil {
			t.Fatal("expected error for NaN stability")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("difficulty below dMin returns error", func(t *testing.T) {
		card := Card{Stability: 5.0, Difficulty: 0.5}
		err := validateResult(card)
		if err == nil {
			t.Fatal("expected error for difficulty below dMin")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("NaN difficulty returns error", func(t *testing.T) {
		card := Card{Stability: 5.0, Difficulty: math.NaN()}
		err := validateResult(card)
		if err == nil {
			t.Fatal("expected error for NaN difficulty")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("stability at sMin boundary passes", func(t *testing.T) {
		card := Card{Stability: sMin, Difficulty: 5.0}
		if err := validateResult(card); err != nil {
			t.Errorf("expected nil at boundary, got=%v", err)
		}
	})

	t.Run("difficulty at dMin boundary passes", func(t *testing.T) {
		card := Card{Stability: 5.0, Difficulty: dMin}
		if err := validateResult(card); err != nil {
			t.Errorf("expected nil at boundary, got=%v", err)
		}
	})
}

func TestValidateCardDirect(t *testing.T) {
	now := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	t.Run("valid New card", func(t *testing.T) {
		card := Card{State: New}
		if err := validateCard(card, now); err != nil {
			t.Errorf("expected nil, got=%v", err)
		}
	})

	t.Run("valid Review card", func(t *testing.T) {
		card := Card{State: Review, Stability: 5.0, Difficulty: 5.0}
		if err := validateCard(card, now); err != nil {
			t.Errorf("expected nil, got=%v", err)
		}
	})

	t.Run("New card skips stability check", func(t *testing.T) {
		card := Card{State: New, Stability: 0, Difficulty: 0}
		if err := validateCard(card, now); err != nil {
			t.Errorf("expected nil for New card with zero fields, got=%v", err)
		}
	})

	t.Run("invalid state", func(t *testing.T) {
		card := Card{State: State(42)}
		err := validateCard(card, now)
		if err == nil {
			t.Fatal("expected error for invalid state")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) {
			t.Fatalf("expected *Error, got %T", err)
		}
		if fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", fsrsErr.Code)
		}
	})

	t.Run("NaN stability non-New", func(t *testing.T) {
		card := Card{State: Review, Stability: math.NaN(), Difficulty: 5.0}
		err := validateCard(card, now)
		if err == nil {
			t.Fatal("expected error")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", err)
		}
	})

	t.Run("stability below sMin non-New", func(t *testing.T) {
		card := Card{State: Review, Stability: 0.0005, Difficulty: 5.0}
		err := validateCard(card, now)
		if err == nil {
			t.Fatal("expected error")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", err)
		}
	})

	t.Run("stability at sMin non-New passes", func(t *testing.T) {
		card := Card{State: Review, Stability: sMin, Difficulty: 5.0}
		if err := validateCard(card, now); err != nil {
			t.Errorf("expected nil at sMin boundary, got=%v", err)
		}
	})

	t.Run("NaN difficulty non-New", func(t *testing.T) {
		card := Card{State: Review, Stability: 5.0, Difficulty: math.NaN()}
		err := validateCard(card, now)
		if err == nil {
			t.Fatal("expected error")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", err)
		}
	})

	t.Run("difficulty below dMin non-New", func(t *testing.T) {
		card := Card{State: Review, Stability: 5.0, Difficulty: 0.5}
		err := validateCard(card, now)
		if err == nil {
			t.Fatal("expected error")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", err)
		}
	})

	t.Run("difficulty at dMin non-New passes", func(t *testing.T) {
		card := Card{State: Review, Stability: 5.0, Difficulty: dMin}
		if err := validateCard(card, now); err != nil {
			t.Errorf("expected nil at dMin boundary, got=%v", err)
		}
	})

	t.Run("LastReview after now", func(t *testing.T) {
		card := Card{State: Review, Stability: 5.0, Difficulty: 5.0, LastReview: now.Add(1 * time.Hour)}
		err := validateCard(card, now)
		if err == nil {
			t.Fatal("expected error for future LastReview")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", err)
		}
	})

	t.Run("LastReview equal to now passes", func(t *testing.T) {
		card := Card{State: Review, Stability: 5.0, Difficulty: 5.0, LastReview: now}
		if err := validateCard(card, now); err != nil {
			t.Errorf("expected nil when LastReview == now, got=%v", err)
		}
	})

	t.Run("zero LastReview passes", func(t *testing.T) {
		card := Card{State: Review, Stability: 5.0, Difficulty: 5.0}
		if err := validateCard(card, now); err != nil {
			t.Errorf("expected nil for zero LastReview, got=%v", err)
		}
	})
}

func TestValidateRatingDirect(t *testing.T) {
	t.Run("Again passes", func(t *testing.T) {
		if err := validateRating(Again); err != nil {
			t.Errorf("expected nil for Again, got=%v", err)
		}
	})

	t.Run("Easy passes", func(t *testing.T) {
		if err := validateRating(Easy); err != nil {
			t.Errorf("expected nil for Easy, got=%v", err)
		}
	})

	t.Run("Good passes", func(t *testing.T) {
		if err := validateRating(Good); err != nil {
			t.Errorf("expected nil for Good, got=%v", err)
		}
	})

	t.Run("Hard passes", func(t *testing.T) {
		if err := validateRating(Hard); err != nil {
			t.Errorf("expected nil for Hard, got=%v", err)
		}
	})

	t.Run("Manual returns error", func(t *testing.T) {
		err := validateRating(Manual)
		if err == nil {
			t.Fatal("expected error for Manual")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", err)
		}
	})

	t.Run("Rating(5) returns error", func(t *testing.T) {
		err := validateRating(Rating(5))
		if err == nil {
			t.Fatal("expected error for Rating(5)")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", err)
		}
	})

	t.Run("Rating(0) returns error", func(t *testing.T) {
		err := validateRating(Rating(0))
		if err == nil {
			t.Fatal("expected error for Rating(0)")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidInput {
			t.Errorf("expected ErrCodeInvalidInput, got=%v", err)
		}
	})
}
