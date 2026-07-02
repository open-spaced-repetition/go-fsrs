package fsrs

import (
	"testing"
	"time"
)

func TestForget(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	ratings := []Rating{Good, Good, Good}
	for _, r := range ratings {
		schedulingCards, err := fsrs.Repeat(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = schedulingCards[r].Card
		now = card.Due
	}

	if card.State == New {
		t.Fatal("card should not be New before Forget")
	}

	t.Run("preserves counters", func(t *testing.T) {
		result := fsrs.Forget(card, now, false)
		if result.Card.State != New {
			t.Errorf("expected State=New, got=%v", result.Card.State)
		}
		if result.Card.Due != now {
			t.Errorf("expected Due=now, got=%v", result.Card.Due)
		}
		if result.Card.Reps != card.Reps {
			t.Errorf("expected Reps=%d, got=%d", card.Reps, result.Card.Reps)
		}
		if result.Card.Lapses != card.Lapses {
			t.Errorf("expected Lapses=%d, got=%d", card.Lapses, result.Card.Lapses)
		}
		if result.Card.Stability != 0 {
			t.Errorf("expected Stability=0, got=%v", result.Card.Stability)
		}
		if result.Card.Difficulty != 0 {
			t.Errorf("expected Difficulty=0, got=%v", result.Card.Difficulty)
		}
		if result.Card.ScheduledDays != 0 {
			t.Errorf("expected ScheduledDays=0, got=%d", result.Card.ScheduledDays)
		}
		if result.Card.LastReview != card.LastReview {
			t.Errorf("expected LastReview=%v, got=%v", card.LastReview, result.Card.LastReview)
		}
		if result.Card.RemainingSteps != 0 {
			t.Errorf("expected RemainingSteps=0, got=%d", result.Card.RemainingSteps)
		}
		if result.ReviewLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", result.ReviewLog.Rating)
		}
		if result.ReviewLog.State != card.State {
			t.Errorf("expected log State=%v, got=%v", card.State, result.ReviewLog.State)
		}
		if result.ReviewLog.Due != card.Due {
			t.Errorf("expected log Due=%v, got=%v", card.Due, result.ReviewLog.Due)
		}
		if result.ReviewLog.Review != now {
			t.Errorf("expected log Review=now, got=%v", result.ReviewLog.Review)
		}
		if result.ReviewLog.Stability != card.Stability {
			t.Errorf("expected log Stability=%v, got=%v", card.Stability, result.ReviewLog.Stability)
		}
		if result.ReviewLog.Difficulty != card.Difficulty {
			t.Errorf("expected log Difficulty=%v, got=%v", card.Difficulty, result.ReviewLog.Difficulty)
		}
		if result.ReviewLog.RemainingSteps != card.RemainingSteps {
			t.Errorf("expected log RemainingSteps=%d, got=%d", card.RemainingSteps, result.ReviewLog.RemainingSteps)
		}
	})

	t.Run("resets counters", func(t *testing.T) {
		result := fsrs.Forget(card, now, true)
		if result.Card.State != New {
			t.Errorf("expected State=New, got=%v", result.Card.State)
		}
		if result.Card.Due != now {
			t.Errorf("expected Due=now, got=%v", result.Card.Due)
		}
		if result.Card.Reps != 0 {
			t.Errorf("expected Reps=0, got=%d", result.Card.Reps)
		}
		if result.Card.Lapses != 0 {
			t.Errorf("expected Lapses=0, got=%d", result.Card.Lapses)
		}
		if result.Card.Stability != 0 {
			t.Errorf("expected Stability=0, got=%v", result.Card.Stability)
		}
		if result.Card.Difficulty != 0 {
			t.Errorf("expected Difficulty=0, got=%v", result.Card.Difficulty)
		}
		if result.Card.ScheduledDays != 0 {
			t.Errorf("expected ScheduledDays=0, got=%d", result.Card.ScheduledDays)
		}
		if result.Card.LastReview != card.LastReview {
			t.Errorf("expected LastReview=%v, got=%v", card.LastReview, result.Card.LastReview)
		}
		if result.Card.RemainingSteps != 0 {
			t.Errorf("expected RemainingSteps=0, got=%d", result.Card.RemainingSteps)
		}
		if result.ReviewLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", result.ReviewLog.Rating)
		}
		if result.ReviewLog.State != card.State {
			t.Errorf("expected log State=%v, got=%v", card.State, result.ReviewLog.State)
		}
		if result.ReviewLog.Due != card.Due {
			t.Errorf("expected log Due=%v, got=%v", card.Due, result.ReviewLog.Due)
		}
		if result.ReviewLog.Stability != card.Stability {
			t.Errorf("expected log Stability=%v, got=%v", card.Stability, result.ReviewLog.Stability)
		}
		if result.ReviewLog.Difficulty != card.Difficulty {
			t.Errorf("expected log Difficulty=%v, got=%v", card.Difficulty, result.ReviewLog.Difficulty)
		}
		if result.ReviewLog.RemainingSteps != card.RemainingSteps {
			t.Errorf("expected log RemainingSteps=%d, got=%d", card.RemainingSteps, result.ReviewLog.RemainingSteps)
		}
	})

	t.Run("already new", func(t *testing.T) {
		newCard := NewCard(now)
		result := fsrs.Forget(newCard, now, false)
		if result.Card.State != New {
			t.Errorf("expected State=New, got=%v", result.Card.State)
		}
		if result.Card.Due != now {
			t.Errorf("expected Due=now, got=%v", result.Card.Due)
		}
		if result.Card.Reps != 0 {
			t.Errorf("expected Reps=0 for new card, got=%d", result.Card.Reps)
		}
		if result.Card.Lapses != 0 {
			t.Errorf("expected Lapses=0, got=%d", result.Card.Lapses)
		}
		if result.Card.Stability != 0 {
			t.Errorf("expected Stability=0, got=%v", result.Card.Stability)
		}
		if result.Card.Difficulty != 0 {
			t.Errorf("expected Difficulty=0, got=%v", result.Card.Difficulty)
		}
		if !result.Card.LastReview.IsZero() {
			t.Errorf("expected LastReview zero, got=%v", result.Card.LastReview)
		}
		if result.ReviewLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", result.ReviewLog.Rating)
		}
		if result.ReviewLog.State != New {
			t.Errorf("expected log State=New, got=%v", result.ReviewLog.State)
		}
		if result.ReviewLog.Due != now {
			t.Errorf("expected log Due=now, got=%v", result.ReviewLog.Due)
		}
		if result.ReviewLog.Review != now {
			t.Errorf("expected log Review=now, got=%v", result.ReviewLog.Review)
		}
		if result.ReviewLog.Stability != 0 {
			t.Errorf("expected log Stability=0, got=%v", result.ReviewLog.Stability)
		}
		if result.ReviewLog.Difficulty != 0 {
			t.Errorf("expected log Difficulty=0, got=%v", result.ReviewLog.Difficulty)
		}
		if result.ReviewLog.ScheduledDays != 0 {
			t.Errorf("expected log ScheduledDays=0, got=%d", result.ReviewLog.ScheduledDays)
		}
		if result.ReviewLog.RemainingSteps != 0 {
			t.Errorf("expected log RemainingSteps=0, got=%d", result.ReviewLog.RemainingSteps)
		}
	})

	t.Run("review state card", func(t *testing.T) {
		schedulingCards, err := fsrs.Repeat(NewCard(), time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		reviewCard := schedulingCards[Good].Card
		if reviewCard.State != Learning && reviewCard.State != Review {
			t.Fatalf("expected Learning or Review, got=%v", reviewCard.State)
		}
		result := fsrs.Forget(reviewCard, reviewCard.Due, false)
		if result.Card.State != New {
			t.Errorf("expected State=New, got=%v", result.Card.State)
		}
		if result.Card.Due != reviewCard.Due {
			t.Errorf("expected Due=%v, got=%v", reviewCard.Due, result.Card.Due)
		}
		if result.Card.Reps != reviewCard.Reps {
			t.Errorf("expected Reps=%d, got=%d", reviewCard.Reps, result.Card.Reps)
		}
		if result.Card.Lapses != reviewCard.Lapses {
			t.Errorf("expected Lapses=%d, got=%d", reviewCard.Lapses, result.Card.Lapses)
		}
		if result.Card.Stability != 0 {
			t.Errorf("expected Stability=0, got=%v", result.Card.Stability)
		}
		if result.Card.Difficulty != 0 {
			t.Errorf("expected Difficulty=0, got=%v", result.Card.Difficulty)
		}
		if result.Card.LastReview != reviewCard.LastReview {
			t.Errorf("expected LastReview=%v, got=%v", reviewCard.LastReview, result.Card.LastReview)
		}
		if result.ReviewLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", result.ReviewLog.Rating)
		}
		if result.ReviewLog.State != reviewCard.State {
			t.Errorf("expected log State=%v, got=%v", reviewCard.State, result.ReviewLog.State)
		}
		if result.ReviewLog.Due != reviewCard.Due {
			t.Errorf("expected log Due=%v, got=%v", reviewCard.Due, result.ReviewLog.Due)
		}
		if result.ReviewLog.Review != reviewCard.Due {
			t.Errorf("expected log Review=%v, got=%v", reviewCard.Due, result.ReviewLog.Review)
		}
		if result.ReviewLog.Stability != reviewCard.Stability {
			t.Errorf("expected log Stability=%v, got=%v", reviewCard.Stability, result.ReviewLog.Stability)
		}
		if result.ReviewLog.Difficulty != reviewCard.Difficulty {
			t.Errorf("expected log Difficulty=%v, got=%v", reviewCard.Difficulty, result.ReviewLog.Difficulty)
		}
		if result.ReviewLog.RemainingSteps != reviewCard.RemainingSteps {
			t.Errorf("expected log RemainingSteps=%d, got=%d", reviewCard.RemainingSteps, result.ReviewLog.RemainingSteps)
		}
		if result.ReviewLog.ScheduledDays != 0 {
			t.Errorf("expected log ScheduledDays=0 (Due==now), got=%d", result.ReviewLog.ScheduledDays)
		}
	})

	t.Run("log captures pre-forget state", func(t *testing.T) {
		result := fsrs.Forget(card, now, false)
		if result.ReviewLog.State != card.State {
			t.Errorf("expected log State=%v, got=%v", card.State, result.ReviewLog.State)
		}
		if result.ReviewLog.Due != card.Due {
			t.Errorf("expected log Due=%v, got=%v", card.Due, result.ReviewLog.Due)
		}
		if result.ReviewLog.Stability != card.Stability {
			t.Errorf("expected log Stability=%v, got=%v", card.Stability, result.ReviewLog.Stability)
		}
		if result.ReviewLog.Difficulty != card.Difficulty {
			t.Errorf("expected log Difficulty=%v, got=%v", card.Difficulty, result.ReviewLog.Difficulty)
		}
		if result.ReviewLog.RemainingSteps != card.RemainingSteps {
			t.Errorf("expected log RemainingSteps=%d, got=%d", card.RemainingSteps, result.ReviewLog.RemainingSteps)
		}
		expectedScheduledDays := uint64(0)
		if card.State != New {
			expectedScheduledDays = dateDiffInDays(now, card.Due)
		}
		if result.ReviewLog.ScheduledDays != expectedScheduledDays {
			t.Errorf("expected log ScheduledDays=%d, got=%d", expectedScheduledDays, result.ReviewLog.ScheduledDays)
		}
		if result.Card.LastReview != card.LastReview {
			t.Errorf("expected card LastReview=%v, got=%v", card.LastReview, result.Card.LastReview)
		}
		if result.Card.Stability != 0 {
			t.Errorf("expected card Stability=0, got=%v", result.Card.Stability)
		}
		if result.Card.Difficulty != 0 {
			t.Errorf("expected card Difficulty=0, got=%v", result.Card.Difficulty)
		}
		if result.Card.ScheduledDays != 0 {
			t.Errorf("expected card ScheduledDays=0, got=%d", result.Card.ScheduledDays)
		}
		if result.Card.RemainingSteps != 0 {
			t.Errorf("expected card RemainingSteps=0, got=%d", result.Card.RemainingSteps)
		}
	})

	t.Run("log scheduled days with future due", func(t *testing.T) {
		forgetNow := now.Add(-48 * time.Hour)
		result := fsrs.Forget(card, forgetNow, false)
		expectedDays := dateDiffInDays(forgetNow, card.Due)
		if result.ReviewLog.ScheduledDays != expectedDays {
			t.Errorf("expected log ScheduledDays=%d, got=%d", expectedDays, result.ReviewLog.ScheduledDays)
		}
	})
}
