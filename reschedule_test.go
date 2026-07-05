package fsrs

import (
	"errors"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestReschedule(t *testing.T) {
	p := DefaultParam()
	f := NewFSRS(p)

	t.Run("basic grade replay matches direct replay", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}
		card := NewCard()
		result, err := f.Reschedule(card, reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 4 {
			t.Fatalf("expected 4 collections, got %d", len(result.Collections))
		}

		curCard := Card{Due: card.Due}
		for i, review := range reviews {
			direct, err := f.Next(curCard, review.Review, review.Rating)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(result.Collections[i].Card, direct.Card) {
				t.Errorf("review %d: reschedule card mismatch\n  got:  %+v\n  want: %+v", i, result.Collections[i].Card, direct.Card)
			}
			curCard = result.Collections[i].Card
		}
	})

	t.Run("basic grade exact values", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantIntervals := []uint64{0, 2, 16, 53}
		for i, want := range wantIntervals {
			if got := result.Collections[i].Card.ScheduledDays; got != want {
				t.Errorf("review %d: scheduled_days=%d, want=%d", i, got, want)
			}
		}

		wantStability := []float64{2.3065, 2.3065, 16.1880, 52.7633}
		for i, want := range wantStability {
			if got := roundFloat(result.Collections[i].Card.Stability, 4); got != want {
				t.Errorf("review %d: stability=%.10f, want=%.4f", i, result.Collections[i].Card.Stability, want)
			}
		}

		wantDifficulty := []float64{2.1181, 2.1112, 2.1043, 2.0975}
		for i, want := range wantDifficulty {
			if got := roundFloat(result.Collections[i].Card.Difficulty, 4); got != want {
				t.Errorf("review %d: difficulty=%.10f, want=%.4f", i, result.Collections[i].Card.Difficulty, want)
			}
		}

		if result.RescheduleItem == nil {
			t.Error("expected non-nil reschedule_item")
		}
	})

	t.Run("basic grade long-term", func(t *testing.T) {
		ltParam := DefaultParam()
		ltParam.EnableShortTerm = false
		ltFsrs := NewFSRS(ltParam)

		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		result, err := ltFsrs.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantIntervals := []uint64{3, 3, 16, 53}
		for i, want := range wantIntervals {
			if got := result.Collections[i].Card.ScheduledDays; got != want {
				t.Errorf("review %d: scheduled_days=%d, want=%d", i, got, want)
			}
		}

		wantStability := []float64{2.3065, 2.3065, 16.1880, 52.7633}
		for i, want := range wantStability {
			if got := roundFloat(result.Collections[i].Card.Stability, 4); got != want {
				t.Errorf("review %d: stability=%.10f, want=%.4f", i, result.Collections[i].Card.Stability, want)
			}
		}

		if result.RescheduleItem == nil {
			t.Error("expected non-nil reschedule_item")
		}
	})

	t.Run("empty reviews", func(t *testing.T) {
		result, err := f.Reschedule(NewCard(), nil, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 0 {
			t.Errorf("expected 0 collections, got %d", len(result.Collections))
		}
		if result.RescheduleItem != nil {
			t.Error("expected nil reschedule_item")
		}
	})

	t.Run("manual rating with state New", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Good, Review: reviewTime},
			{Rating: Manual, Review: reviewTime.Add(24 * time.Hour), State: StatePtr(New)},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Fatalf("expected 2 collections, got %d", len(result.Collections))
		}

		lastCard := result.Collections[1].Card
		if lastCard.State != New {
			t.Errorf("expected State=New, got=%v", lastCard.State)
		}
		if lastCard.Stability != 0 {
			t.Errorf("expected Stability=0, got=%v", lastCard.Stability)
		}
		if lastCard.Difficulty != 0 {
			t.Errorf("expected Difficulty=0, got=%v", lastCard.Difficulty)
		}
		if lastCard.Due.IsZero() {
			t.Error("expected non-zero Due")
		}

		lastLog := result.Collections[1].ReviewLog
		if lastLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", lastLog.Rating)
		}
		if lastLog.State != New {
			t.Errorf("expected log State=New, got=%v", lastLog.State)
		}
		if !lastLog.Due.Equal(reviewTime.Add(24 * time.Hour)) {
			t.Errorf("expected log Due=reviewed time, got=%v", lastLog.Due)
		}
		if lastLog.ScheduledDays != 0 {
			t.Errorf("expected log ScheduledDays=0, got=%d", lastLog.ScheduledDays)
		}
	})

	t.Run("manual rating with state New and explicit Due", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 20, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Good, Review: reviewTime},
			{Rating: Manual, Review: reviewTime.Add(24 * time.Hour), State: StatePtr(New), Due: manualDue},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		lastCard := result.Collections[1].Card
		if !lastCard.Due.Equal(manualDue) {
			t.Errorf("expected Due=%v (explicit), got=%v", manualDue, lastCard.Due)
		}
		if lastCard.ScheduledDays != 6 {
			t.Errorf("expected ScheduledDays=6, got=%d", lastCard.ScheduledDays)
		}
	})

	t.Run("manual rating with state New and Due before Review", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		dueBeforeReview := time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Good, Review: reviewTime},
			{Rating: Manual, Review: reviewTime.Add(24 * time.Hour), State: StatePtr(New), Due: dueBeforeReview},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		lastCard := result.Collections[1].Card
		if !lastCard.Due.Equal(dueBeforeReview) {
			t.Errorf("expected Due=%v, got=%v", dueBeforeReview, lastCard.Due)
		}
		if lastCard.ScheduledDays != 0 {
			t.Errorf("expected ScheduledDays=0 when Due <= Review, got=%d", lastCard.ScheduledDays)
		}
	})

	t.Run("manual rating with state Review", func(t *testing.T) {
		review1 := time.Date(2024, 8, 12, 1, 0, 0, 0, time.UTC)
		review2 := time.Date(2024, 8, 13, 1, 0, 0, 0, time.UTC)
		review3 := time.Date(2024, 8, 14, 1, 0, 0, 0, time.UTC)
		review4 := time.Date(2024, 8, 15, 1, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 4, 17, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Easy, Review: review1},
			{Rating: Good, Review: review2},
			{Rating: Manual, Review: review3, State: StatePtr(Review), Stability: 21.79806877, Difficulty: 3.2828565, Due: manualDue},
			{Rating: Good, Review: review4},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 4 {
			t.Fatalf("expected 4 collections, got %d", len(result.Collections))
		}

		manualCard := result.Collections[2].Card
		if manualCard.State != Review {
			t.Errorf("expected State=Review, got=%v", manualCard.State)
		}
		if math.Abs(manualCard.Stability-21.79806877) > 1e-6 {
			t.Errorf("expected Stability=21.79806877, got=%v", manualCard.Stability)
		}
		if math.Abs(manualCard.Difficulty-3.2828565) > 1e-6 {
			t.Errorf("expected Difficulty=3.2828565, got=%v", manualCard.Difficulty)
		}
		if !manualCard.Due.Equal(manualDue) {
			t.Errorf("expected Due=%v, got=%v", manualDue, manualCard.Due)
		}
		if manualCard.Reps != 3 {
			t.Errorf("expected Reps=3, got=%d", manualCard.Reps)
		}
		if manualCard.ScheduledDays != 21 {
			t.Errorf("expected ScheduledDays=21, got=%d", manualCard.ScheduledDays)
		}
		if !manualCard.LastReview.Equal(review3) {
			t.Errorf("expected LastReview=%v, got=%v", review3, manualCard.LastReview)
		}

		manualLog := result.Collections[2].ReviewLog
		if manualLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", manualLog.Rating)
		}
		if manualLog.State != Review {
			t.Errorf("expected log State=Review (pre-review), got=%v", manualLog.State)
		}
		if manualLog.ScheduledDays != 13 {
			t.Errorf("expected log ScheduledDays=13, got=%d", manualLog.ScheduledDays)
		}
	})

	t.Run("manual without state returns error", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Easy, Review: reviewTime},
			{Rating: Good, Review: reviewTime.Add(24 * time.Hour)},
			{Rating: Manual, Review: reviewTime.Add(48 * time.Hour)},
			{Rating: Good, Review: reviewTime.Add(72 * time.Hour)},
		}

		_, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err == nil {
			t.Fatal("expected error for manual rating without state")
		}
		if !errors.Is(err, ErrManualStateRequired) {
			t.Errorf("expected ErrManualStateRequired, got=%v", err)
		}
	})

	t.Run("manual non-New without due returns error", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Easy, Review: reviewTime},
			{Rating: Good, Review: reviewTime.Add(24 * time.Hour)},
			{Rating: Manual, Review: reviewTime.Add(48 * time.Hour), State: StatePtr(Review)},
			{Rating: Good, Review: reviewTime.Add(72 * time.Hour)},
		}

		_, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err == nil {
			t.Fatal("expected error for manual rating without due")
		}
		if !errors.Is(err, ErrManualDueRequired) {
			t.Errorf("expected ErrManualDueRequired, got=%v", err)
		}
	})

	t.Run("reschedule item generated", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		currentCard := Card{
			Due:       time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC),
			State:     Review,
			Reps:      5,
			Stability: 50.0,
		}
		now := time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC)

		result, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			UpdateMemoryState: true,
			Now:               now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item")
		}
		if result.RescheduleItem.ReviewLog.Rating != Manual {
			t.Errorf("expected log Rating=Manual, got=%v", result.RescheduleItem.ReviewLog.Rating)
		}
		if !result.RescheduleItem.Card.LastReview.Equal(now) {
			t.Errorf("expected card LastReview=now, got=%v", result.RescheduleItem.Card.LastReview)
		}
		if result.RescheduleItem.Card.Reps != currentCard.Reps+1 {
			t.Errorf("expected card Reps=%d, got=%d", currentCard.Reps+1, result.RescheduleItem.Card.Reps)
		}
	})

	t.Run("reschedule item null when due matches", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		lastCard := result.Collections[len(result.Collections)-1].Card

		currentCard := lastCard
		result2, err := f.Reschedule(currentCard, reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result2.RescheduleItem != nil {
			t.Error("expected nil reschedule_item when due matches")
		}
	})

	t.Run("skip manual filters Manual ratings", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Good, Review: reviewTime},
			{Rating: Manual, Review: reviewTime.Add(24 * time.Hour), State: StatePtr(New)},
			{Rating: Good, Review: reviewTime.Add(48 * time.Hour)},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{SkipManual: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Errorf("expected 2 collections (Manual filtered), got %d", len(result.Collections))
		}
	})

	t.Run("skip manual with all-Manual reviews yields empty collections", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Manual, Review: r1, State: StatePtr(New)},
			{Rating: Manual, Review: r2, State: StatePtr(New)},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{SkipManual: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 0 {
			t.Errorf("expected 0 collections, got %d", len(result.Collections))
		}
		if result.RescheduleItem != nil {
			t.Error("expected nil RescheduleItem with empty collections")
		}
	})

	t.Run("sort reviews by review time", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		sortedReviews := make([]ReviewHistory, len(reviewTimes))
		sortedTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		for i, rt := range sortedTimes {
			sortedReviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		resultSorted, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{SortReviews: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		resultPreSorted, err := f.Reschedule(NewCard(), sortedReviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for i := range resultSorted.Collections {
			if !reflect.DeepEqual(resultSorted.Collections[i].Card, resultPreSorted.Collections[i].Card) {
				t.Errorf("review %d: sorted result doesn't match pre-sorted", i)
			}
		}
	})

	t.Run("first card override", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		firstDue := time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC)
		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{FirstDue: firstDue})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 4 {
			t.Fatalf("expected 4 collections, got %d", len(result.Collections))
		}
	})

	t.Run("forget scenario", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		firstDue := time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC)

		var currentCard Card
		var historyCards []Card
		for _, review := range reviews {
				item, err := f.Next(currentCard, review.Review, review.Rating)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			currentCard = item.Card
			historyCards = append(historyCards, currentCard)
		}

		forgetItem := f.Forget(currentCard, time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC), false)
		currentCard = forgetItem.Card

		result, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			FirstDue:          firstDue,
			UpdateMemoryState: true,
			Now:               time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item")
		}
		lastHistory := historyCards[len(historyCards)-1]
		if !result.RescheduleItem.Card.Due.Equal(lastHistory.Due) {
			t.Errorf("expected card Due=%v, got=%v", lastHistory.Due, result.RescheduleItem.Card.Due)
		}
		if math.Abs(result.RescheduleItem.Card.Stability-lastHistory.Stability) > 1e-10 {
			t.Errorf("expected card Stability=%v, got=%v", lastHistory.Stability, result.RescheduleItem.Card.Stability)
		}
		if math.Abs(result.RescheduleItem.Card.Difficulty-lastHistory.Difficulty) > 1e-10 {
			t.Errorf("expected card Difficulty=%v, got=%v", lastHistory.Difficulty, result.RescheduleItem.Card.Difficulty)
		}
	})

	t.Run("manual rating without stability uses card values", func(t *testing.T) {
		review1 := time.Date(2024, 8, 12, 1, 0, 0, 0, time.UTC)
		review2 := time.Date(2024, 8, 13, 1, 0, 0, 0, time.UTC)
		review3 := time.Date(2024, 8, 14, 1, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 4, 17, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Easy, Review: review1},
			{Rating: Good, Review: review2},
			{Rating: Manual, Review: review3, State: StatePtr(Review), Due: manualDue},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		manualCard := result.Collections[2].Card
		prevCard := result.Collections[1].Card
		if manualCard.Stability != prevCard.Stability {
			t.Errorf("expected Stability=%v (from previous card), got=%v", prevCard.Stability, manualCard.Stability)
		}
		if manualCard.Difficulty != prevCard.Difficulty {
			t.Errorf("expected Difficulty=%v (from previous card), got=%v", prevCard.Difficulty, manualCard.Difficulty)
		}
	})

	t.Run("sort reviews does not mutate caller slice", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}
		originalOrder := make([]time.Time, len(reviews))
		for i, r := range reviews {
			originalOrder[i] = r.Review
		}

		_, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{SortReviews: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for i, r := range reviews {
			if !r.Review.Equal(originalOrder[i]) {
				t.Errorf("caller slice mutated at index %d: got=%v, want=%v", i, r.Review, originalOrder[i])
			}
		}
	})

	t.Run("sort reviews with skip manual combined", func(t *testing.T) {
		reviews := []ReviewHistory{
			{Rating: Good, Review: time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC)},
			{Rating: Manual, Review: time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC), State: StatePtr(New)},
			{Rating: Good, Review: time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{SortReviews: true, SkipManual: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Fatalf("expected 2 collections, got %d", len(result.Collections))
		}
		firstReview := result.Collections[0].ReviewLog.Review
		secondReview := result.Collections[1].ReviewLog.Review
		if !firstReview.Before(secondReview) {
			t.Errorf("expected sorted order: first=%v should be before second=%v", firstReview, secondReview)
		}
	})

	t.Run("single review", func(t *testing.T) {
		reviewTime := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{{Rating: Good, Review: reviewTime}}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 1 {
			t.Fatalf("expected 1 collection, got %d", len(result.Collections))
		}
		if result.Collections[0].Card.State == New {
			t.Errorf("expected non-New state after Good review, got=%v", result.Collections[0].Card.State)
		}
	})

	t.Run("all manual reviews", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)
		reviews := []ReviewHistory{
			{Rating: Manual, Review: r1, State: StatePtr(New)},
			{Rating: Manual, Review: r2, State: StatePtr(New)},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Collections) != 2 {
			t.Fatalf("expected 2 collections, got %d", len(result.Collections))
		}
		for i, c := range result.Collections {
			if c.Card.State != New {
				t.Errorf("collection %d: expected State=New, got=%v", i, c.Card.State)
			}
		}
	})

	t.Run("manual rating with Learning state", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 14, 10, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Good, Review: r1},
			{Rating: Manual, Review: r2, State: StatePtr(Learning), Stability: 2.5, Difficulty: 4.0, Due: manualDue},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		manualCard := result.Collections[1].Card
		if manualCard.State != Learning {
			t.Errorf("expected State=Learning, got=%v", manualCard.State)
		}
		if math.Abs(manualCard.Stability-2.5) > 1e-6 {
			t.Errorf("expected Stability=2.5, got=%v", manualCard.Stability)
		}
		if !manualCard.Due.Equal(manualDue) {
			t.Errorf("expected Due=%v, got=%v", manualDue, manualCard.Due)
		}
	})

	t.Run("manual rating with Relearning state", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)
		r3 := time.Date(2024, 9, 15, 0, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 15, 10, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Good, Review: r1},
			{Rating: Manual, Review: r2, State: StatePtr(Review), Stability: 10.0, Difficulty: 5.0, Due: time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC)},
			{Rating: Manual, Review: r3, State: StatePtr(Relearning), Stability: 5.0, Difficulty: 6.0, Due: manualDue},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		relearnCard := result.Collections[2].Card
		if relearnCard.State != Relearning {
			t.Errorf("expected State=Relearning, got=%v", relearnCard.State)
		}
		if math.Abs(relearnCard.Stability-5.0) > 1e-6 {
			t.Errorf("expected Stability=5.0, got=%v", relearnCard.Stability)
		}
	})

	t.Run("update memory state false preserves stability", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		now := time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC)
		currentCard := Card{Due: now, State: Review, Reps: 3, Stability: 10.0, Difficulty: 5.0}

		result, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			UpdateMemoryState: false,
			Now:               now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item")
		}
		if result.RescheduleItem.Card.Stability != currentCard.Stability {
			t.Errorf("with UpdateMemoryState=false: expected Stability=%v (unchanged), got=%v", currentCard.Stability, result.RescheduleItem.Card.Stability)
		}
		if result.RescheduleItem.Card.Difficulty != currentCard.Difficulty {
			t.Errorf("with UpdateMemoryState=false: expected Difficulty=%v (unchanged), got=%v", currentCard.Difficulty, result.RescheduleItem.Card.Difficulty)
		}
	})

	t.Run("reschedule item for manual-last replay with non-New state", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)
		manualDue := time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Good, Review: r1},
			{Rating: Manual, Review: r2, State: StatePtr(Review), Stability: 10.0, Difficulty: 5.0, Due: manualDue},
		}

		now := time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC)
		currentCard := Card{Due: now, State: Review, Reps: 3, Stability: 1.0, Difficulty: 1.0}

		result, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			UpdateMemoryState: true,
			Now:               now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item for manual-last replay")
		}
		if result.RescheduleItem.Card.State != Review {
			t.Errorf("expected State=Review, got=%v", result.RescheduleItem.Card.State)
		}
		if math.Abs(result.RescheduleItem.Card.Stability-10.0) > 1e-6 {
			t.Errorf("with UpdateMemoryState: expected Stability=10.0, got=%v", result.RescheduleItem.Card.Stability)
		}
		if math.Abs(result.RescheduleItem.Card.Difficulty-5.0) > 1e-6 {
			t.Errorf("with UpdateMemoryState: expected Difficulty=5.0, got=%v", result.RescheduleItem.Card.Difficulty)
		}
	})

	t.Run("reschedule item for manual-last New state", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		r2 := time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Good, Review: r1},
			{Rating: Manual, Review: r2, State: StatePtr(New)},
		}

		now := time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC)
		currentCard := Card{Due: now, State: Review, Reps: 3, Stability: 10.0, Difficulty: 5.0}

		result, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			Now: now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item when current card Due differs from manual-last New card")
		}
		if result.RescheduleItem.Card.State != New {
			t.Errorf("expected State=New, got=%v", result.RescheduleItem.Card.State)
		}
	})

	t.Run("manual due before review produces zero scheduled days", func(t *testing.T) {
		r1 := time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC)
		dueBeforeReview := time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC)

		reviews := []ReviewHistory{
			{Rating: Good, Review: r1},
			{Rating: Manual, Review: time.Date(2024, 9, 14, 0, 0, 0, 0, time.UTC), State: StatePtr(Review), Stability: 5.0, Difficulty: 4.0, Due: dueBeforeReview},
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		manualCard := result.Collections[1].Card
		if manualCard.ScheduledDays != 0 {
			t.Errorf("expected ScheduledDays=0 when due < review, got=%d", manualCard.ScheduledDays)
		}
	})

	t.Run("reschedule item with update memory state", func(t *testing.T) {
		reviewTimes := []time.Time{
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 13, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 17, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 9, 28, 0, 0, 0, 0, time.UTC),
		}
		reviews := make([]ReviewHistory, len(reviewTimes))
		for i, rt := range reviewTimes {
			reviews[i] = ReviewHistory{Rating: Good, Review: rt}
		}

		result, err := f.Reschedule(NewCard(), reviews, RescheduleOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		lastCard := result.Collections[len(result.Collections)-1].Card

		now := time.Date(2024, 10, 27, 0, 0, 0, 0, time.UTC)
		currentCard := Card{Due: now, State: Review, Reps: 3, Stability: 10.0, Difficulty: 5.0}

		result2, err := f.Reschedule(currentCard, reviews, RescheduleOptions{
			UpdateMemoryState: true,
			Now:               now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result2.RescheduleItem == nil {
			t.Fatal("expected non-nil reschedule_item")
		}

		if math.Abs(result2.RescheduleItem.Card.Stability-lastCard.Stability) > 1e-10 {
			t.Errorf("with UpdateMemoryState: expected Stability=%v, got=%v", lastCard.Stability, result2.RescheduleItem.Card.Stability)
		}
		if math.Abs(result2.RescheduleItem.Card.Difficulty-lastCard.Difficulty) > 1e-10 {
			t.Errorf("with UpdateMemoryState: expected Difficulty=%v, got=%v", lastCard.Difficulty, result2.RescheduleItem.Card.Difficulty)
		}
	})
}
