package fsrs

import (
	"math"
	"testing"
	"time"
)

func BenchmarkBasicSchedulerFullSequence(b *testing.B) {
	f := NewFSRS(DefaultParam())
	ratings := []Rating{Good, Good, Good, Good, Good, Good, Again, Again, Good, Good, Good, Good, Good}
	for b.Loop() {
		card := NewCard()
		now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
		for _, rating := range ratings {
			schedCards, err := f.Repeat(card, now)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			record := schedCards[rating]
			card = record.Card
			now = card.Due
		}
	}
}

func BenchmarkLongTermSchedulerFullSequence(b *testing.B) {
	p := DefaultParam()
	p.EnableShortTerm = false
	f := NewFSRS(p)
	ratings := []Rating{Good, Good, Good, Good, Good, Good, Again, Again, Good, Good, Good, Good, Good}
	for b.Loop() {
		card := NewCard()
		now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
		for _, rating := range ratings {
			schedCards2, err := f.Repeat(card, now)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			record := schedCards2[rating]
			card = record.Card
			now = card.Due
		}
	}
}

func TestNewStateStepBased(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	againCard := schedulingCards[Again].Card
	if againCard.State != Learning {
		t.Errorf("Again on new card should be Learning, got=%v", againCard.State)
	}
	if againCard.ScheduledDays != 0 {
		t.Errorf("Again on new card should have ScheduledDays=0, got=%v", againCard.ScheduledDays)
	}
	if againCard.RemainingSteps != len(p.LearningSteps) {
		t.Errorf("Again should have RemainingSteps=%d, got=%d", len(p.LearningSteps), againCard.RemainingSteps)
	}

	hardCard := schedulingCards[Hard].Card
	if hardCard.State != Learning {
		t.Errorf("Hard on new card should be Learning (step-based), got=%v", hardCard.State)
	}
	if hardCard.ScheduledDays != 0 {
		t.Errorf("Hard on new card should have ScheduledDays=0, got=%v", hardCard.ScheduledDays)
	}

	goodCard := schedulingCards[Good].Card
	if goodCard.State != Learning {
		t.Errorf("Good on new card should go to Learning (step 1 of 2), got=%v", goodCard.State)
	}
	if goodCard.RemainingSteps != 1 {
		t.Errorf("Good should have RemainingSteps=1, got=%d", goodCard.RemainingSteps)
	}

	easyCard := schedulingCards[Easy].Card
	if easyCard.State != Review {
		t.Errorf("Easy on new card should graduate to Review, got=%v", easyCard.State)
	}
}

func TestLearningStateThresholdBranching(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	againCard := schedulingCards[Again].Card
	now = againCard.Due
	schedulingCards, err = fsrs.Repeat(againCard, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	againAgainCard := schedulingCards[Again].Card
	if againAgainCard.State != Learning {
		t.Errorf("Again on learning card should stay in Learning (short interval), got=%v", againAgainCard.State)
	}

	hardCard := schedulingCards[Hard].Card
	if hardCard.State != Learning {
		t.Errorf("Hard on learning card should stay in Learning, got=%v", hardCard.State)
	}

	goodCard := schedulingCards[Good].Card
	if goodCard.State != Learning {
		t.Errorf("Good on learning card should stay in Learning (has remaining step), got=%v", goodCard.State)
	}
	if goodCard.ScheduledDays != 0 {
		t.Errorf("Good on learning card should have ScheduledDays=0, got=%v", goodCard.ScheduledDays)
	}

	easyCard := schedulingCards[Easy].Card
	if easyCard.State != Review {
		t.Errorf("Easy on learning card should graduate to Review, got=%v", easyCard.State)
	}
}

func TestLearningStateEasyConstraintWhenGoodGraduates(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	againCard := schedulingCards[Again].Card
	now = againCard.Due
	schedulingCards, err = fsrs.Repeat(againCard, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	goodInterval := schedulingCards[Good].Card.ScheduledDays
	easyInterval := schedulingCards[Easy].Card.ScheduledDays

	if easyInterval <= goodInterval {
		t.Errorf("Easy interval (%v) should be > Good interval (%v)",
			easyInterval, goodInterval)
	}
}

func TestReviewStateAgainAlwaysRelearning(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	{
		rec, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = rec.Card
	}
	now = card.Due
	{
		rec, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = rec.Card
	}
	now = card.Due

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	againCard := schedulingCards[Again].Card

	if againCard.State != Relearning {
		t.Errorf("Again on review card should always go to Relearning, got=%v", againCard.State)
	}
	if againCard.ScheduledDays != 0 {
		t.Errorf("Again on review card should have ScheduledDays=0 (short-term), got=%d", againCard.ScheduledDays)
	}
	if againCard.RemainingSteps != len(p.RelearningSteps) {
		t.Errorf("Again should have RemainingSteps=%d, got=%d", len(p.RelearningSteps), againCard.RemainingSteps)
	}

	expectedDue := now.Add(minutesToDuration(p.RelearningSteps[0]))
	dueDiff := againCard.Due.Sub(expectedDue)
	if dueDiff < 0 {
		dueDiff = -dueDiff
	}
	if dueDiff > time.Second {
		t.Errorf("Again Due should match relearning step, expected=%v got=%v diff=%v",
			expectedDue, againCard.Due, dueDiff)
	}
}

func TestCustomLearningSteps(t *testing.T) {
	p := DefaultParam()
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	p.LearningSteps = []float64{1, 10, 60}
	s := p.NewBasicScheduler(card, now)
	schedulingCards := s.Preview()

	againCard := schedulingCards[Again].Card
	if againCard.State != Learning {
		t.Errorf("Again should be Learning with steps, got=%v", againCard.State)
	}
	if againCard.RemainingSteps != 3 {
		t.Errorf("Again should have RemainingSteps=3, got=%d", againCard.RemainingSteps)
	}

	goodCard := schedulingCards[Good].Card
	if goodCard.State != Learning {
		t.Errorf("Good should stay Learning with 3 steps (goes to step 2), got=%v", goodCard.State)
	}

	easyCard := schedulingCards[Easy].Card
	if easyCard.State != Review {
		t.Errorf("Easy should always graduate, got=%v", easyCard.State)
	}

	p.LearningSteps = []float64{}
	s = p.NewBasicScheduler(card, now)
	schedulingCards = s.Preview()
	goodCard = schedulingCards[Good].Card
	if goodCard.State != Review {
		t.Errorf("Good should graduate with no steps, got=%v", goodCard.State)
	}
}

func TestLearningStateRecallStabilityWhenElapsedDays(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	againCard := schedulingCards[Again].Card

	schedulingCards0, err := fsrs.Repeat(againCard, againCard.Due)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goodCardImmediate := schedulingCards0[Good].Card
	stabilityImmediate := goodCardImmediate.Stability

	twoDaysLater := now.Add(2 * 24 * time.Hour)
	schedulingCards2, err := fsrs.Repeat(againCard, twoDaysLater)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goodCardDelayed := schedulingCards2[Good].Card
	stabilityDelayed := goodCardDelayed.Stability

	if stabilityDelayed <= stabilityImmediate {
		t.Errorf("stability with ElapsedDays>0 (%.4f) should be > stability with ElapsedDays=0 (%.4f)",
			stabilityDelayed, stabilityImmediate)
	}
}

func TestLearningStateHardAndEasyWithElapsedDays(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	againCard := schedulingCards[Again].Card

	twoDaysLater := now.Add(2 * 24 * time.Hour)
	schedulingCards2, err := fsrs.Repeat(againCard, twoDaysLater)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hardCardDelayed := schedulingCards2[Hard].Card
	if hardCardDelayed.Stability <= againCard.Stability {
		t.Errorf("Hard stability with ElapsedDays=2 (%.4f) should be > initial stability (%.4f)",
			hardCardDelayed.Stability, againCard.Stability)
	}

	easyCardDelayed := schedulingCards2[Easy].Card
	if easyCardDelayed.State != Review {
		t.Errorf("Easy with ElapsedDays=2 should graduate to Review, got=%v", easyCardDelayed.State)
	}

	goodCardDelayed := schedulingCards2[Good].Card
	if easyCardDelayed.ScheduledDays <= goodCardDelayed.ScheduledDays {
		t.Errorf("Easy interval (%v) should be > Good interval (%v) with ElapsedDays>0",
			easyCardDelayed.ScheduledDays, goodCardDelayed.ScheduledDays)
	}
}

func TestThreeStepProgression(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10, 60}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	goodCard := schedulingCards[Good].Card
	if goodCard.State != Learning {
		t.Errorf("Good should be Learning at step 1, got=%v", goodCard.State)
	}
	if goodCard.RemainingSteps != 2 {
		t.Errorf("Good should have RS=2, got=%d", goodCard.RemainingSteps)
	}
	expectedDue := now.Add(minutesToDuration(10))
	if goodCard.Due.Sub(expectedDue).Abs() > time.Second {
		t.Errorf("Good delay should be 10min (step 1), Due=%v expected=%v", goodCard.Due, expectedDue)
	}

	now = goodCard.Due
	schedulingCards, err = fsrs.Repeat(goodCard, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goodCard2 := schedulingCards[Good].Card
	if goodCard2.State != Learning {
		t.Errorf("Good at RS=2 should stay Learning, got=%v", goodCard2.State)
	}
	if goodCard2.RemainingSteps != 1 {
		t.Errorf("Good at RS=2 should have RS=1, got=%d", goodCard2.RemainingSteps)
	}
	expectedDue2 := now.Add(minutesToDuration(60))
	if goodCard2.Due.Sub(expectedDue2).Abs() > time.Second {
		t.Errorf("Good delay at RS=2 should be 60min (step 2), Due=%v expected=%v", goodCard2.Due, expectedDue2)
	}

	now = goodCard2.Due
	schedulingCards, err = fsrs.Repeat(goodCard2, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goodCard3 := schedulingCards[Good].Card
	if goodCard3.State != Review {
		t.Errorf("Good at RS=1 should graduate, got=%v", goodCard3.State)
	}
}

func TestEmptyLearningStepsAllGraduate(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, rating := range []Rating{Again, Hard, Good, Easy} {
		c := schedulingCards[rating].Card
		if c.State != Review {
			t.Errorf("Rating %v with empty steps should graduate to Review, got=%v", rating, c.State)
		}
		if c.ScheduledDays == 0 {
			t.Errorf("Rating %v with empty steps should have interval > 0", rating)
		}
	}
}

func TestGoodAfterAgainAdvancesStep(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	againCard := schedulingCards[Again].Card

	now = againCard.Due
	schedulingCards, err = fsrs.Repeat(againCard, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goodCard := schedulingCards[Good].Card

	expectedDue := now.Add(minutesToDuration(10))
	if goodCard.Due.Sub(expectedDue).Abs() > time.Second {
		t.Errorf("Good after Again should have delay=10min (next step), Due=%v expected=%v", goodCard.Due, expectedDue)
	}
}

func TestNextStateSingle(t *testing.T) {
	p := DefaultParam()

	item := p.NextState(nil, 0.9, 0, Good)
	if item.Memory.Stability != p.initStability(Good) {
		t.Errorf("NextState new card stability mismatch: got=%v want=%v", item.Memory.Stability, p.initStability(Good))
	}

	current := &MemoryState{Stability: 5.0, Difficulty: 5.0}
	item = p.NextState(current, 0.9, 1, Good)
	if item.Interval < 1 {
		t.Errorf("NextState interval should be >= 1, got=%v", item.Interval)
	}

	states := p.NextStates(current, 0.9, 1)
	if states.Good != item {
		t.Errorf("NextState(Good) should equal NextStates.Good")
	}
}

func TestReviewLogFields(t *testing.T) {
	t.Run("zero Due card preserves zero in review log", func(t *testing.T) {
		fsrs := NewFSRS(DefaultParam())
		card := NewCard(time.Time{})
		now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

		record, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		log := record.ReviewLog

		if !log.Due.IsZero() {
			t.Errorf("ReviewLog.Due should be zero for new card with zero Due (pre-review), got=%v", log.Due)
		}
		if log.Stability != 0 {
			t.Errorf("ReviewLog.Stability should be 0 for new card (pre-review), got=%v", log.Stability)
		}
		if log.Difficulty != 0 {
			t.Errorf("ReviewLog.Difficulty should be 0 for new card (pre-review), got=%v", log.Difficulty)
		}
		if log.RemainingSteps != 0 {
			t.Errorf("ReviewLog.RemainingSteps should be 0 for new card (pre-review), got=%d", log.RemainingSteps)
		}
	})

	t.Run("default Due card propagates to review log", func(t *testing.T) {
		fsrs := NewFSRS(DefaultParam())
		card := NewCard()
		cardDue := card.Due
		now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

		record, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		log := record.ReviewLog

		if log.Due.IsZero() {
			t.Errorf("ReviewLog.Due should be non-zero for card with default Due, got=%v", log.Due)
		}
		if log.Due != cardDue {
			t.Errorf("ReviewLog.Due should equal card.Due (%v), got=%v", cardDue, log.Due)
		}
	})

	t.Run("second review log carries previous LastReview", func(t *testing.T) {
		fsrs := NewFSRS(DefaultParam())
		card := NewCard(time.Time{})
		now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

		record, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = record.Card
		previousLastReview := card.LastReview
		now = card.Due
		record, err = fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		log := record.ReviewLog

		if log.Due != previousLastReview {
			t.Errorf("ReviewLog.Due should be the card's last review time, got=%v want=%v", log.Due, previousLastReview)
		}
		if log.Stability == 0 {
			t.Errorf("ReviewLog.Stability should be non-zero after first review")
		}
		if log.Difficulty == 0 {
			t.Errorf("ReviewLog.Difficulty should be non-zero after first review")
		}
		if log.RemainingSteps != 1 {
			t.Errorf("ReviewLog.RemainingSteps should be 1 (Learning card with 1 step remaining), got=%d", log.RemainingSteps)
		}
	})

	t.Run("explicit Due card propagates to review log", func(t *testing.T) {
		fsrs := NewFSRS(DefaultParam())
		specificTime := time.Date(2024, 3, 10, 8, 0, 0, 0, time.UTC)
		card := NewCard(specificTime)
		now := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)

		record, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		log := record.ReviewLog

		if log.Due != specificTime {
			t.Errorf("ReviewLog.Due should equal explicit card Due (%v), got=%v", specificTime, log.Due)
		}
	})
}

func TestReviewStateAgainWithEmptyRelearningSteps(t *testing.T) {
	p := DefaultParam()
	p.RelearningSteps = []float64{}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	{
		rec, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = rec.Card
	}
	now = card.Due
	{
		rec, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = rec.Card
	}
	now = card.Due

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	againCard := schedulingCards[Again].Card

	if againCard.State != Review {
		t.Errorf("Again with empty relearning steps should go to Review, got=%v", againCard.State)
	}
	if againCard.ScheduledDays == 0 {
		t.Errorf("Again with empty relearning steps should have interval > 0")
	}
	if againCard.Lapses != 1 {
		t.Errorf("Again should increment Lapses, got=%d", againCard.Lapses)
	}
}

func TestLearningStateRemainingZero(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	card.State = Learning
	card.Stability = 2.0
	card.Difficulty = 5.0
	card.RemainingSteps = 0
	card.LastReview = time.Date(2022, 11, 29, 12, 0, 0, 0, time.UTC)
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, rating := range []Rating{Again, Hard, Good} {
		c := schedulingCards[rating].Card
		if c.State != Review {
			t.Errorf("Rating %v with RS=0 should graduate to Review, got=%v", rating, c.State)
		}
	}
}

func TestRelearningStepProgression(t *testing.T) {
	p := DefaultParam()
	p.RelearningSteps = []float64{10, 60}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	{
		rec, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = rec.Card
	}
	now = card.Due
	{
		rec, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = rec.Card
	}
	now = card.Due

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	againCard := schedulingCards[Again].Card
	if againCard.State != Relearning {
		t.Errorf("Again on review card should go to Relearning, got=%v", againCard.State)
	}
	if againCard.RemainingSteps != 2 {
		t.Errorf("Again should have RS=2, got=%d", againCard.RemainingSteps)
	}

	now = againCard.Due
	schedulingCards, err = fsrs.Repeat(againCard, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goodCard := schedulingCards[Good].Card
	if goodCard.State != Relearning {
		t.Errorf("Good at RS=2 should stay Relearning (step 1 of 2), got=%v", goodCard.State)
	}
	if goodCard.RemainingSteps != 1 {
		t.Errorf("Good at RS=2 should have RS=1, got=%d", goodCard.RemainingSteps)
	}
	expectedDue := now.Add(minutesToDuration(60))
	if goodCard.Due.Sub(expectedDue).Abs() > time.Second {
		t.Errorf("Good at RS=2 should have delay=60min, Due=%v expected=%v", goodCard.Due, expectedDue)
	}

	now = goodCard.Due
	schedulingCards, err = fsrs.Repeat(goodCard, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goodCard2 := schedulingCards[Good].Card
	if goodCard2.State != Review {
		t.Errorf("Good at RS=1 should graduate to Review, got=%v", goodCard2.State)
	}

	hardCard := schedulingCards[Hard].Card
	if hardCard.State != Relearning {
		t.Errorf("Hard at RS=1 should stay Relearning, got=%v", hardCard.State)
	}
}

func TestReviewStateZeroInterval(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	{
		rec, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = rec.Card
	}
	{
		rec, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = rec.Card
	}
	card.State = Review

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	againCard := schedulingCards[Again].Card
	if againCard.Stability == 0 {
		t.Errorf("Again with ElapsedDays=0 should use shortTermStability, got stability=0")
	}

	goodCard := schedulingCards[Good].Card
	if goodCard.Stability == 0 {
		t.Errorf("Good with ElapsedDays=0 should use shortTermStability, got stability=0")
	}
	if goodCard.State != Review {
		t.Errorf("Good should be Review, got=%v", goodCard.State)
	}
	if goodCard.ScheduledDays == 0 {
		t.Errorf("Good should have interval > 0, got=%d", goodCard.ScheduledDays)
	}

	hardCard := schedulingCards[Hard].Card
	if hardCard.Stability == 0 {
		t.Errorf("Hard with ElapsedDays=0 should use shortTermStability, got stability=0")
	}
	if hardCard.ScheduledDays > goodCard.ScheduledDays {
		t.Errorf("Hard interval (%d) should be <= Good interval (%d)", hardCard.ScheduledDays, goodCard.ScheduledDays)
	}

	easyCard := schedulingCards[Easy].Card
	if easyCard.ScheduledDays <= goodCard.ScheduledDays {
		t.Errorf("Easy interval (%d) should be > Good interval (%d)", easyCard.ScheduledDays, goodCard.ScheduledDays)
	}
}

func TestNextStateElapsedZero(t *testing.T) {
	p := DefaultParam()

	item := p.NextState(nil, 0.9, 0, Good)
	if item.Memory.Stability != p.initStability(Good) {
		t.Errorf("NextState new card: got stability=%v want=%v", item.Memory.Stability, p.initStability(Good))
	}
	if item.Memory.Difficulty != constrainDifficulty(p.initDifficulty(Good)) {
		t.Errorf("NextState new card: got difficulty=%v want=%v", item.Memory.Difficulty, constrainDifficulty(p.initDifficulty(Good)))
	}

	current := &MemoryState{Stability: 5.0, Difficulty: 5.0}
	item = p.NextState(current, 0.9, 0, Good)
	expectedS := p.shortTermStability(5.0, Good)
	if math.Abs(item.Memory.Stability-expectedS) > 1e-10 {
		t.Errorf("NextState elapsed=0: got stability=%v want=%v", item.Memory.Stability, expectedS)
	}

	itemElapsed := p.NextState(current, 0.9, 1, Good)
	if itemElapsed.Interval < 1 {
		t.Errorf("NextState elapsed=1: interval should be >= 1, got=%v", itemElapsed.Interval)
	}

	states := p.NextStates(current, 0.9, 0)
	if states.Good != item {
		t.Errorf("NextState(Good, elapsed=0) should equal NextStates.Good")
	}
}

func TestReviewLogAllRatings(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, rating := range []Rating{Again, Hard, Good, Easy} {
		log := schedulingCards[rating].ReviewLog
		if log.Rating != rating {
			t.Errorf("ReviewLog.Rating should be %v, got=%v", rating, log.Rating)
		}
		if log.State != New {
			t.Errorf("ReviewLog.State for new card should be New (0), got=%v", log.State)
		}
		if log.Stability != 0 {
			t.Errorf("ReviewLog.Stability for new card should be 0, got=%v", log.Stability)
		}
		if log.Difficulty != 0 {
			t.Errorf("ReviewLog.Difficulty for new card should be 0, got=%v", log.Difficulty)
		}
		if log.RemainingSteps != 0 {
			t.Errorf("ReviewLog.RemainingSteps for new card should be 0, got=%d", log.RemainingSteps)
		}
		if log.Review != now {
			t.Errorf("ReviewLog.Review should be %v, got=%v", now, log.Review)
		}
	}

	learningCard := schedulingCards[Good].Card
	schedulingCards2, err := fsrs.Repeat(learningCard, learningCard.Due)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, rating := range []Rating{Again, Hard, Good, Easy} {
		log := schedulingCards2[rating].ReviewLog
		if log.Rating != rating {
			t.Errorf("Learning ReviewLog.Rating should be %v, got=%v", rating, log.Rating)
		}
		if log.Stability == 0 {
			t.Errorf("Learning ReviewLog.Stability for rating %v should be non-zero", rating)
		}
		if log.Difficulty == 0 {
			t.Errorf("Learning ReviewLog.Difficulty for rating %v should be non-zero", rating)
		}
	}
}

func TestLearningStateHardRemainingStepsInvariant(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10, 60}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	againCard := schedulingCards[Again].Card
	if againCard.RemainingSteps != 3 {
		t.Errorf("Again should have RS=3, got=%d", againCard.RemainingSteps)
	}

	now = againCard.Due
	schedulingCards, err = fsrs.Repeat(againCard, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hardCard := schedulingCards[Hard].Card
	if hardCard.RemainingSteps != 3 {
		t.Errorf("Hard should keep RS unchanged (3), got=%d", hardCard.RemainingSteps)
	}
}

func TestLearningStateAgainResetsRemainingSteps(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10, 60}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goodCard := schedulingCards[Good].Card
	if goodCard.RemainingSteps != 2 {
		t.Errorf("Good should have RS=2, got=%d", goodCard.RemainingSteps)
	}

	now = goodCard.Due
	schedulingCards, err = fsrs.Repeat(goodCard, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	againCard := schedulingCards[Again].Card
	if againCard.RemainingSteps != 3 {
		t.Errorf("Again should reset RS to full length (3), got=%d", againCard.RemainingSteps)
	}
}

func TestDayScaleSteps(t *testing.T) {
	p := DefaultParam()
	p.LearningSteps = []float64{1, 10, 1440}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	goodCard := schedulingCards[Good].Card
	if goodCard.State != Learning {
		t.Errorf("Good step 1 should be Learning, got=%v", goodCard.State)
	}
	if goodCard.ScheduledDays != 0 {
		t.Errorf("Good step 1 should have ScheduledDays=0, got=%d", goodCard.ScheduledDays)
	}
	if goodCard.RemainingSteps != 2 {
		t.Errorf("Good should have RS=2, got=%d", goodCard.RemainingSteps)
	}

	now = goodCard.Due
	schedulingCards, err = fsrs.Repeat(goodCard, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goodCard2 := schedulingCards[Good].Card
	if goodCard2.State != Review {
		t.Errorf("Good step 2 (1440min) should graduate to Review, got=%v", goodCard2.State)
	}
	if goodCard2.ScheduledDays != 1 {
		t.Errorf("Good step 2 (1440min) should have ScheduledDays=1, got=%d", goodCard2.ScheduledDays)
	}
	if goodCard2.RemainingSteps != 1 {
		t.Errorf("Good step 2 should preserve RemainingSteps=1, got=%d", goodCard2.RemainingSteps)
	}
	expectedDue := now.Add(1440 * time.Minute)
	if goodCard2.Due.Sub(expectedDue).Abs() > time.Second {
		t.Errorf("Good step 2 Due should be now+1440min, due=%v expected=%v", goodCard2.Due, expectedDue)
	}

	againCard := schedulingCards[Again].Card
	if againCard.State != Learning {
		t.Errorf("Again should be Learning (step[0]=1min < 1440), got=%v", againCard.State)
	}
}

func TestDayScaleStepRelearning(t *testing.T) {
	p := DefaultParam()
	p.RelearningSteps = []float64{1440}
	fsrs := NewFSRS(p)
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)

	{
		rec, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = rec.Card
	}
	now = card.Due
	{
		rec, err := fsrs.Next(card, now, Good)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		card = rec.Card
	}
	now = card.Due

	schedulingCards, err := fsrs.Repeat(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	againCard := schedulingCards[Again].Card
	if againCard.State != Review {
		t.Errorf("Again with 1440min relearning step should graduate to Review, got=%v", againCard.State)
	}
	if againCard.ScheduledDays != 1 {
		t.Errorf("Again should have ScheduledDays=1 (1440min = 1 day), got=%d", againCard.ScheduledDays)
	}
	if againCard.Lapses != 1 {
		t.Errorf("Again should increment Lapses, got=%d", againCard.Lapses)
	}
}
