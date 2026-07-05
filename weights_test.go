package fsrs

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestConvertV45WeightsParity(t *testing.T) {
	v45 := [17]float64{
		0.4, 0.6, 2.4, 5.8, 4.93, 0.94, 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61,
	}
	expected := [21]float64{
		0.4, 0.6, 2.4, 5.8, 6.81, 0.44675013, 1.36, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61, 0.0, 0.0, 0.0, 0.5,
	}

	w, err := ConvertV45Weights(v45)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const tolerance = 1e-6
	for i := 0; i < 21; i++ {
		if math.Abs(w[i]-expected[i]) > tolerance {
			t.Errorf("W[%d]: expected ≈ %v, got %v", i, expected[i], w[i])
		}
	}
}

func TestDefaultWeights(t *testing.T) {
	w := DefaultWeights()
	if len(w) != 21 {
		t.Fatalf("expected 21 weights, got %d", len(w))
	}
	for i, val := range w {
		if math.IsNaN(val) || math.IsInf(val, 0) {
			t.Errorf("W[%d]: expected finite, got %v", i, val)
		}
	}
}

func TestConvertV45WeightsInvalidInput(t *testing.T) {
	t.Run("NaN input", func(t *testing.T) {
		v45 := [17]float64{
			0.4, 0.6, 2.4, 5.8, 4.93, math.NaN(), 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61,
		}
		_, err := ConvertV45Weights(v45)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidWeightsValue {
			t.Errorf("expected ErrCodeInvalidWeightsValue, got=%v", err)
		}
		if !errors.Is(err, ErrInvalidWeightsValue) {
			t.Errorf("expected errors.Is(err, ErrInvalidWeightsValue), got=%v", err)
		}
	})

	t.Run("Inf input", func(t *testing.T) {
		v45 := [17]float64{
			0.4, 0.6, 2.4, 5.8, 4.93, 0.94, math.Inf(1), 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61,
		}
		_, err := ConvertV45Weights(v45)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidWeightsValue {
			t.Errorf("expected ErrCodeInvalidWeightsValue, got=%v", err)
		}
		if !errors.Is(err, ErrInvalidWeightsValue) {
			t.Errorf("expected errors.Is(err, ErrInvalidWeightsValue), got=%v", err)
		}
	})

	t.Run("finite input producing NaN in conversion", func(t *testing.T) {
		v45 := [17]float64{
			0.4, 0.6, 2.4, 5.8, 4.93, -1000.0, 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61,
		}
		_, err := ConvertV45Weights(v45)
		if err == nil {
			t.Fatal("expected error for conversion-produced NaN, got nil")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidWeightsValue {
			t.Errorf("expected ErrCodeInvalidWeightsValue, got=%v", err)
		}
		if !errors.Is(err, ErrInvalidWeightsValue) {
			t.Errorf("expected errors.Is(err, ErrInvalidWeightsValue), got=%v", err)
		}
	})

	t.Run("all-zero input", func(t *testing.T) {
		v45 := [17]float64{}
		w, err := ConvertV45Weights(v45)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w[4] != 0.0 {
			t.Errorf("expected W[4] = 0.0, got %v", w[4])
		}
		if w[5] != 0.0 {
			t.Errorf("expected W[5] = 0.0, got %v", w[5])
		}
		if w[6] != 0.5 {
			t.Errorf("expected W[6] = 0.5, got %v", w[6])
		}
		if w[20] != 0.5 {
			t.Errorf("expected W[20] = 0.5, got %v", w[20])
		}
	})
}

func TestConvertV45WeightsIndices(t *testing.T) {
	v45 := [17]float64{
		0.4, 0.6, 2.4, 5.8, 4.93, 0.94, 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61,
	}

	w, err := ConvertV45Weights(v45)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// W[0..3] unchanged
	for i := 0; i < 4; i++ {
		if w[i] != v45[i] {
			t.Errorf("W[%d]: expected unchanged %v, got %v", i, v45[i], w[i])
		}
	}

	// W[4] = W[5] * 2.0 + W[4]  (order: reads old W[5] before it is overwritten)
	expectedW4 := v45[5]*2.0 + v45[4]
	if math.Abs(w[4]-expectedW4) > 1e-6 {
		t.Errorf("W[4]: expected ≈ %v, got %v", expectedW4, w[4])
	}

	// W[5] = ln(W[5] * 3.0 + 1.0) / 3.0
	expectedW5 := math.Log(v45[5]*3.0+1.0) / 3.0
	if math.Abs(w[5]-expectedW5) > 1e-6 {
		t.Errorf("W[5]: expected ≈ %v, got %v", expectedW5, w[5])
	}

	// W[6] = W[6] + 0.5
	expectedW6 := v45[6] + 0.5
	if math.Abs(w[6]-expectedW6) > 1e-6 {
		t.Errorf("W[6]: expected ≈ %v, got %v", expectedW6, w[6])
	}

	// W[7..16] unchanged
	for i := 7; i < 17; i++ {
		if w[i] != v45[i] {
			t.Errorf("W[%d]: expected unchanged %v, got %v", i, v45[i], w[i])
		}
	}

	// W[17..19] = 0.0
	for i := 17; i < 20; i++ {
		if w[i] != 0.0 {
			t.Errorf("W[%d]: expected 0.0, got %v", i, w[i])
		}
	}

	// W[20] = fsrs5DefaultDecay (0.5)
	if w[20] != 0.5 {
		t.Errorf("W[20]: expected 0.5, got %v", w[20])
	}
}

func TestConvertV45WeightsProducesValidFSRS(t *testing.T) {
	v45 := [17]float64{
		0.4, 0.6, 2.4, 5.8, 4.93, 0.94, 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61,
	}

	w, err := ConvertV45Weights(v45)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify converted weights produce a valid FSRS instance
	p := DefaultParam()
	p.W = w
	if err := p.Validate(); err != nil {
		t.Errorf("converted weights should pass Validate: %v", err)
	}

	f := NewFSRS(p)
	card := NewCard()
	now := time.Now()

	// Verify Repeat works with converted weights
	result, err := f.Repeat(card, now)
	if err != nil {
		t.Errorf("Repeat should work with converted weights: %v", err)
	}
	if len(result) != 4 {
		t.Errorf("expected 4 scheduling cards, got %d", len(result))
	}

	// Verify Retrievability works with converted weights
	_, err = f.Retrievability(card, now)
	if err != nil {
		t.Errorf("Retrievability should work with converted weights: %v", err)
	}
}

func TestMigrateWeights(t *testing.T) {
	t.Run("length 17 parity", func(t *testing.T) {
		v17 := []float64{
			0.4, 0.6, 2.4, 5.8, 4.93, 0.94, 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61,
		}
		w, err := MigrateWeights(v17)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(w) != 21 {
			t.Fatalf("expected length 21, got %d", len(w))
		}
		if math.Abs(w[4]-6.81) > 1e-6 {
			t.Errorf("expected W[4] ≈ 6.81, got %v", w[4])
		}
	})

	t.Run("length 19 migration", func(t *testing.T) {
		v19 := []float64{
			0.4, 0.6, 2.4, 5.8, 4.93, 0.94, 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61, 1.0, 2.0,
		}
		w, err := MigrateWeights(v19)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if w[19] != 0.0 {
			t.Errorf("expected W[19] = 0.0, got %v", w[19])
		}
		if w[20] != 0.5 {
			t.Errorf("expected W[20] = 0.5, got %v", w[20])
		}
	})

	t.Run("length 21 migration", func(t *testing.T) {
		v21 := make([]float64, 21)
		for i := 0; i < 21; i++ {
			v21[i] = float64(i) + 0.1
		}
		w, err := MigrateWeights(v21)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for i := 0; i < 21; i++ {
			if w[i] != v21[i] {
				t.Errorf("expected W[%d] = %v, got %v", i, v21[i], w[i])
			}
		}
	})

	t.Run("invalid length", func(t *testing.T) {
		_, err := MigrateWeights([]float64{1.0, 2.0})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidWeightsLength {
			t.Errorf("expected ErrCodeInvalidWeightsLength, got=%v", err)
		}
		if !errors.Is(err, ErrInvalidWeightsLength) {
			t.Errorf("expected errors.Is(err, ErrInvalidWeightsLength), got=%v", err)
		}
	})

	t.Run("NaN in slice", func(t *testing.T) {
		v19 := []float64{
			0.4, 0.6, 2.4, 5.8, 4.93, 0.94, 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61, 1.0, math.NaN(),
		}
		_, err := MigrateWeights(v19)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidWeightsValue {
			t.Errorf("expected ErrCodeInvalidWeightsValue, got=%v", err)
		}
		if !errors.Is(err, ErrInvalidWeightsValue) {
			t.Errorf("expected errors.Is(err, ErrInvalidWeightsValue), got=%v", err)
		}
	})

	t.Run("Inf in length 19 slice", func(t *testing.T) {
		v19 := []float64{
			0.4, 0.6, 2.4, 5.8, 4.93, 0.94, 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61, 1.0, math.Inf(1),
		}
		_, err := MigrateWeights(v19)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidWeightsValue {
			t.Errorf("expected ErrCodeInvalidWeightsValue, got=%v", err)
		}
		if !errors.Is(err, ErrInvalidWeightsValue) {
			t.Errorf("expected errors.Is(err, ErrInvalidWeightsValue), got=%v", err)
		}
	})

	t.Run("NaN in length 21 slice", func(t *testing.T) {
		v21 := make([]float64, 21)
		for i := range v21 {
			v21[i] = float64(i) + 0.1
		}
		v21[20] = math.NaN()
		_, err := MigrateWeights(v21)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidWeightsValue {
			t.Errorf("expected ErrCodeInvalidWeightsValue, got=%v", err)
		}
		if !errors.Is(err, ErrInvalidWeightsValue) {
			t.Errorf("expected errors.Is(err, ErrInvalidWeightsValue), got=%v", err)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		_, err := MigrateWeights([]float64{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var fsrsErr *Error
		if !errors.As(err, &fsrsErr) || fsrsErr.Code != ErrCodeInvalidWeightsLength {
			t.Errorf("expected ErrCodeInvalidWeightsLength, got=%v", err)
		}
		if !errors.Is(err, ErrInvalidWeightsLength) {
			t.Errorf("expected errors.Is(err, ErrInvalidWeightsLength), got=%v", err)
		}
	})

	t.Run("sentinels are distinguishable", func(t *testing.T) {
		_, lengthErr := MigrateWeights([]float64{})
		_, valueErr := MigrateWeights([]float64{
			0.4, 0.6, 2.4, 5.8, 4.93, 0.94, 0.86, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, math.NaN(),
		})

		if errors.Is(lengthErr, ErrInvalidWeightsValue) {
			t.Error("length error should NOT match ErrInvalidWeightsValue")
		}
		if errors.Is(valueErr, ErrInvalidWeightsLength) {
			t.Error("value error should NOT match ErrInvalidWeightsLength")
		}
	})
}

func TestConvertV5Weights(t *testing.T) {
	v5 := [19]float64{
		0.4, 0.6, 2.4, 5.8, 6.81, 0.44675013, 1.36, 0.01, 1.49, 0.14, 0.94, 2.18, 0.05, 0.34, 1.26, 0.29, 2.61, 1.0, 2.0,
	}
	w := ConvertV5Weights(v5)
	for i := 0; i < 19; i++ {
		if w[i] != v5[i] {
			t.Errorf("W[%d]: expected %v, got %v", i, v5[i], w[i])
		}
	}
	if w[19] != 0.0 {
		t.Errorf("W[19]: expected 0.0, got %v", w[19])
	}
	if w[20] != 0.5 {
		t.Errorf("W[20]: expected 0.5, got %v", w[20])
	}
}
