package fsrs

import (
	"reflect"
	"testing"
	"time"
)

func TestNewCard(t *testing.T) {
	t.Run("default sets Due to current time", func(t *testing.T) {
		before := time.Now()
		card := NewCard()
		after := time.Now()
		if card.Due.Before(before) || card.Due.After(after) {
			t.Errorf("expected Due between %v and %v, got=%v", before, after, card.Due)
		}
	})

	t.Run("explicit time sets Due to provided value", func(t *testing.T) {
		specificTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
		card := NewCard(specificTime)
		if !card.Due.Equal(specificTime) {
			t.Errorf("expected Due=%v, got=%v", specificTime, card.Due)
		}
		if card.Stability != 0 || card.Difficulty != 0 {
			t.Errorf("expected zero Stability/Difficulty, got s=%v d=%v", card.Stability, card.Difficulty)
		}
		if card.State != 0 || card.Reps != 0 || card.Lapses != 0 {
			t.Errorf("expected zero State/Reps/Lapses, got state=%v reps=%d lapses=%d", card.State, card.Reps, card.Lapses)
		}
	})

	t.Run("multiple variadic args uses only first", func(t *testing.T) {
		first := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		second := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		card := NewCard(first, second)
		if !card.Due.Equal(first) {
			t.Errorf("expected Due=%v (first arg), got=%v", first, card.Due)
		}
	})

	t.Run("zero time sets Due to zero", func(t *testing.T) {
		card := NewCard(time.Time{})
		if !card.Due.IsZero() {
			t.Errorf("expected zero Due, got=%v", card.Due)
		}
	})

	t.Run("other fields remain zero", func(t *testing.T) {
		card := NewCard()
		if card.Stability != 0 || card.Difficulty != 0 {
			t.Errorf("expected zero Stability/Difficulty, got s=%v d=%v", card.Stability, card.Difficulty)
		}
		if card.State != 0 || card.Reps != 0 || card.Lapses != 0 {
			t.Errorf("expected zero State/Reps/Lapses, got state=%v reps=%d lapses=%d", card.State, card.Reps, card.Lapses)
		}
	})
}

func TestDefaultLearningSteps(t *testing.T) {
	steps := DefaultLearningSteps()
	if want := []float64{1, 10}; !reflect.DeepEqual(steps, want) {
		t.Errorf("expected %v, got %v", want, steps)
	}
	steps[0] = 999
	if fresh := DefaultLearningSteps(); fresh[0] == 999 {
		t.Error("mutating returned slice affected future calls")
	}
}

func TestDefaultRelearningSteps(t *testing.T) {
	steps := DefaultRelearningSteps()
	if want := []float64{10}; !reflect.DeepEqual(steps, want) {
		t.Errorf("expected %v, got %v", want, steps)
	}
	steps[0] = 999
	if fresh := DefaultRelearningSteps(); fresh[0] == 999 {
		t.Error("mutating returned slice affected future calls")
	}
}
