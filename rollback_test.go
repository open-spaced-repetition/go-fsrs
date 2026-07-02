package fsrs

import (
	"errors"
	"testing"
	"time"
)

func TestRollback(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	t.Run("restores card after good review", func(t *testing.T) {
		card := NewCard()
		schedulingCards, err := fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		result, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rolledBack, err := fsrs.Rollback(result.Card, result.ReviewLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.State != result.ReviewLog.State {
			t.Errorf("expected State=%v, got=%v", result.ReviewLog.State, rolledBack.State)
		}
		if rolledBack.Stability != result.ReviewLog.Stability {
			t.Errorf("expected Stability=%v, got=%v", result.ReviewLog.Stability, rolledBack.Stability)
		}
		if rolledBack.Difficulty != result.ReviewLog.Difficulty {
			t.Errorf("expected Difficulty=%v, got=%v", result.ReviewLog.Difficulty, rolledBack.Difficulty)
		}
		if rolledBack.ScheduledDays != result.ReviewLog.ScheduledDays {
			t.Errorf("expected ScheduledDays=%d, got=%d", result.ReviewLog.ScheduledDays, rolledBack.ScheduledDays)
		}
		if rolledBack.Due != result.ReviewLog.Due {
			t.Errorf("expected Due=%v, got=%v", result.ReviewLog.Due, rolledBack.Due)
		}
		if !rolledBack.LastReview.IsZero() {
			t.Errorf("expected LastReview zero (New state), got=%v", rolledBack.LastReview)
		}
		if rolledBack.Reps != 0 {
			t.Errorf("expected Reps=0, got=%d", rolledBack.Reps)
		}
		if rolledBack.Lapses != 0 {
			t.Errorf("expected Lapses=0, got=%d", rolledBack.Lapses)
		}
		_ = schedulingCards
	})

	t.Run("decrements lapses on again", func(t *testing.T) {
		card := NewCard()
		schedulingCards, err := fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards, err = fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards, err = fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[Again].Card
		if card.Lapses == 0 {
			t.Fatal("expected Lapses > 0 before rollback")
		}
		log := schedulingCards[Again].ReviewLog
		rolledBack, err := fsrs.Rollback(card, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Lapses != card.Lapses-1 {
			t.Errorf("expected Lapses=%d, got=%d", card.Lapses-1, rolledBack.Lapses)
		}
		if rolledBack.Reps != card.Reps-1 {
			t.Errorf("expected Reps=%d, got=%d", card.Reps-1, rolledBack.Reps)
		}
	})

	t.Run("rejects manual rating", func(t *testing.T) {
		card := NewCard()
		log := ReviewLog{Rating: Manual}
		_, err := fsrs.Rollback(card, log)
		if err == nil {
			t.Error("expected error for manual rating")
		}
		if !errors.Is(err, ErrManualRating) {
			t.Errorf("expected ErrManualRating, got=%v", err)
		}
	})

	t.Run("no underflow on zero reps", func(t *testing.T) {
		card := NewCard()
		result, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rolledBack, err := fsrs.Rollback(result.Card, result.ReviewLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Reps != 0 {
			t.Errorf("expected Reps=0 (no underflow), got=%d", rolledBack.Reps)
		}
	})

	t.Run("lapses unchanged for good rating", func(t *testing.T) {
		card := NewCard()
		result, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rolledBack, err := fsrs.Rollback(result.Card, result.ReviewLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Lapses != 0 {
			t.Errorf("expected Lapses=0 (no underflow), got=%d", rolledBack.Lapses)
		}
	})

	t.Run("restores remaining steps from log", func(t *testing.T) {
		card := NewCard()
		schedulingCards, err := fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[Again].Card
		result, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rolledBack, err := fsrs.Rollback(result.Card, result.ReviewLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.RemainingSteps != result.ReviewLog.RemainingSteps {
			t.Errorf("expected RemainingSteps=%d (from log), got=%d", result.ReviewLog.RemainingSteps, rolledBack.RemainingSteps)
		}
	})

	t.Run("no lapses decrement when again with zero lapses", func(t *testing.T) {
		card := NewCard()
		schedulingCards, err := fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[Again].Card
		log := schedulingCards[Again].ReviewLog
		rolledBack, err := fsrs.Rollback(card, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Lapses != card.Lapses {
			t.Errorf("expected Lapses=%d (unchanged), got=%d", card.Lapses, rolledBack.Lapses)
		}
	})

	t.Run("no lapses decrement for good with positive lapses", func(t *testing.T) {
		card := NewCard()
		schedulingCards, err := fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards, err = fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards, err = fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[Again].Card
		if card.Lapses == 0 {
			t.Fatal("expected Lapses > 0 before rollback")
		}
		schedulingCards2, err := fsrs.Repeat(card, now)
		card = schedulingCards2[Good].Card
		log := schedulingCards2[Good].ReviewLog
		rolledBack, err := fsrs.Rollback(card, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Lapses != card.Lapses {
			t.Errorf("expected Lapses=%d (unchanged for Good), got=%d", card.Lapses, rolledBack.Lapses)
		}
	})

	t.Run("reps zero on card with zero reps", func(t *testing.T) {
		card := Card{State: Review, Reps: 0, Lapses: 0}
		log := ReviewLog{Rating: Good, State: Review}
		rolledBack, err := fsrs.Rollback(card, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Reps != 0 {
			t.Errorf("expected Reps=0, got=%d", rolledBack.Reps)
		}
	})

	t.Run("manual rating string", func(t *testing.T) {
		if Manual.String() != "Manual" {
			t.Errorf("expected Manual.String()=Manual, got=%s", Manual.String())
		}
	})

	t.Run("again on relearning preserves lapses", func(t *testing.T) {
		card := NewCard()
		schedulingCards, err := fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards, err = fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[Good].Card
		now = card.Due
		schedulingCards, err = fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[Again].Card
		if card.State != Relearning {
			t.Fatalf("expected Relearning state, got=%v", card.State)
		}
		if card.Lapses == 0 {
			t.Fatal("expected Lapses > 0")
		}
		schedulingCards2, err := fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards2[Again].Card
		log := schedulingCards2[Again].ReviewLog
		if log.State != Relearning {
			t.Fatalf("expected log State=Relearning, got=%v", log.State)
		}
		rolledBack, err := fsrs.Rollback(card, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.Lapses != card.Lapses {
			t.Errorf("expected Lapses=%d (Again on Relearning does not decrement), got=%d", card.Lapses, rolledBack.Lapses)
		}
	})

	t.Run("rejects out of range rating", func(t *testing.T) {
		card := Card{State: Review, Reps: 1}
		log := ReviewLog{Rating: Rating(99), State: Review}
		_, err := fsrs.Rollback(card, log)
		if err == nil {
			t.Error("expected error for out-of-range rating")
		}
		if !errors.Is(err, ErrInvalidRating) {
			t.Errorf("expected ErrInvalidRating, got=%v", err)
		}
	})

	t.Run("learning state rollback", func(t *testing.T) {
		card := NewCard()
		schedulingCards, err := fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		learningCard := schedulingCards[Again].Card
		if learningCard.State != Learning {
			t.Fatalf("expected Learning state, got=%v", learningCard.State)
		}
		schedulingCards2, err := fsrs.Repeat(learningCard, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		goodResult := schedulingCards2[Good]
		rolledBack, err := fsrs.Rollback(goodResult.Card, goodResult.ReviewLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rolledBack.State != Learning {
			t.Errorf("expected State=Learning, got=%v", rolledBack.State)
		}
		if rolledBack.Stability != goodResult.ReviewLog.Stability {
			t.Errorf("expected Stability=%v, got=%v", goodResult.ReviewLog.Stability, rolledBack.Stability)
		}
		if rolledBack.Difficulty != goodResult.ReviewLog.Difficulty {
			t.Errorf("expected Difficulty=%v, got=%v", goodResult.ReviewLog.Difficulty, rolledBack.Difficulty)
		}
		if rolledBack.RemainingSteps != goodResult.ReviewLog.RemainingSteps {
			t.Errorf("expected RemainingSteps=%d, got=%d", goodResult.ReviewLog.RemainingSteps, rolledBack.RemainingSteps)
		}
		if rolledBack.Due != goodResult.ReviewLog.Review {
			t.Errorf("expected Due=%v, got=%v", goodResult.ReviewLog.Review, rolledBack.Due)
		}
		if rolledBack.LastReview != goodResult.ReviewLog.Due {
			t.Errorf("expected LastReview=%v, got=%v", goodResult.ReviewLog.Due, rolledBack.LastReview)
		}
	})
}
