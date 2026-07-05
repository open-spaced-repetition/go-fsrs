package fsrs

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestClipParameters(t *testing.T) {
	t.Run("default weights unchanged after clipping", func(t *testing.T) {
		p := DefaultParam()
		original := p.W
		clipParameters(&p)
		for i := range p.W {
			if p.W[i] != original[i] {
				t.Errorf("W[%d]: clipping changed default weight from %v to %v", i, original[i], p.W[i])
			}
		}
	})

	t.Run("extreme weights are clipped to range", func(t *testing.T) {
		p := DefaultParam()
		p.W[0] = -999
		p.W[1] = -1
		p.W[2] = 200
		p.W[3] = 0
		p.W[4] = 0.5
		p.W[5] = -5
		p.W[6] = 10
		p.W[7] = 2.0
		p.W[8] = 100.0
		p.W[9] = -1.0
		p.W[10] = 10.0
		p.W[11] = -0.5
		p.W[12] = 1.0
		p.W[13] = -1.0
		p.W[14] = -0.5
		p.W[15] = -1.0
		p.W[16] = 0.5
		p.W[20] = 5.0
		clipParameters(&p)
		expected := [21][2]float64{
			{0.001, 100.0},
			{0.001, 100.0},
			{0.001, 100.0},
			{0.001, 100.0},
			{1.0, 10.0},
			{0.001, 4.0},
			{0.001, 4.0},
			{0.001, 0.75},
			{0.0, 4.5},
			{0.0, 0.8},
			{0.001, 3.5},
			{0.001, 5.0},
			{0.001, 0.25},
			{0.001, 0.9},
			{0.0, 4.0},
			{0.0, 1.0},
			{1.0, 6.0},
			{0.0, 2.0},
			{0.0, 2.0},
			{0.01, 0.8},
			{0.1, 0.8},
		}
		for i := range p.W {
			if p.W[i] < expected[i][0] || p.W[i] > expected[i][1] {
				t.Errorf("W[%d]=%v outside range [%v, %v]", i, p.W[i], expected[i][0], expected[i][1])
			}
		}
		if p.W[0] != 0.001 {
			t.Errorf("W[0]: expected 0.001, got=%v", p.W[0])
		}
		if p.W[4] != 1.0 {
			t.Errorf("W[4]: expected 1.0 (min), got=%v", p.W[4])
		}
		if p.W[8] != 4.5 {
			t.Errorf("W[8]: expected 4.5 (max), got=%v", p.W[8])
		}
		if p.W[20] != 0.8 {
			t.Errorf("W[20]: expected 0.8 (max), got=%v", p.W[20])
		}
	})

	t.Run("W20 lower bound raised to 0.1", func(t *testing.T) {
		p := DefaultParam()
		p.W[20] = 0
		clipParameters(&p)
		if p.W[20] != 0.1 {
			t.Errorf("W[20]: expected 0.1 (min), got=%v", p.W[20])
		}
	})

	t.Run("W19 upper bound clipped to 0.8", func(t *testing.T) {
		p := DefaultParam()
		p.W[19] = 1.5
		clipParameters(&p)
		if p.W[19] != 0.8 {
			t.Errorf("W[19]: expected 0.8 (max), got=%v", p.W[19])
		}
	})

	t.Run("W17 W18 dynamic ceiling with multiple relearning steps", func(t *testing.T) {
		p := DefaultParam()
		p.RelearningSteps = []float64{10, 20}
		originalW17 := p.W[17]
		originalW18 := p.W[18]
		clipParameters(&p)
		if p.W[17] > 2.0 {
			t.Errorf("W[17]: expected <= 2.0, got=%v", p.W[17])
		}
		if p.W[18] > 2.0 {
			t.Errorf("W[18]: expected <= 2.0, got=%v", p.W[18])
		}
		if p.W[17] > originalW17+1e-8 || p.W[18] > originalW18+1e-8 {
			t.Errorf("W[17]/W[18] should not increase: W[17]=%v (was %v), W[18]=%v (was %v)", p.W[17], originalW17, p.W[18], originalW18)
		}
	})

	t.Run("W19 lower bound depends on EnableShortTerm", func(t *testing.T) {
		p := DefaultParam()
		p.EnableShortTerm = true
		p.W[19] = 0.001
		clipParameters(&p)
		if p.W[19] < 0.01 {
			t.Errorf("W[19]: expected >= 0.01 with EnableShortTerm, got=%v", p.W[19])
		}

		p2 := DefaultParam()
		p2.EnableShortTerm = false
		p2.W[19] = 0.001
		clipParameters(&p2)
		if p2.W[19] != 0.001 {
			t.Errorf("W[19]: expected 0.001 without EnableShortTerm, got=%v", p2.W[19])
		}
	})

	t.Run("single relearning step uses default ceiling", func(t *testing.T) {
		p := DefaultParam()
		p.RelearningSteps = []float64{10}
		p.W[17] = 1.5
		p.W[18] = 1.5
		clipParameters(&p)
		if p.W[17] != 1.5 {
			t.Errorf("W[17]: expected unchanged 1.5, got=%v", p.W[17])
		}
		if p.W[18] != 1.5 {
			t.Errorf("W[18]: expected unchanged 1.5, got=%v", p.W[18])
		}
	})

	t.Run("zero relearning steps uses default ceiling", func(t *testing.T) {
		p := DefaultParam()
		p.RelearningSteps = []float64{}
		p.W[17] = 1.5
		p.W[18] = 1.5
		clipParameters(&p)
		if p.W[17] != 1.5 {
			t.Errorf("W[17]: expected unchanged 1.5, got=%v", p.W[17])
		}
	})

	t.Run("dynamic ceiling lower clamp at 0.01", func(t *testing.T) {
		p := DefaultParam()
		p.RelearningSteps = []float64{10, 20}
		p.W[11] = 5.0
		p.W[13] = 0.9
		p.W[14] = 4.0
		p.W[17] = 1.5
		p.W[18] = 1.5
		clipParameters(&p)
		if p.W[17] < 0.01-1e-10 || p.W[18] < 0.01-1e-10 {
			t.Errorf("W[17]/W[18]: expected >= 0.01 (clamped ceiling), got=%v/%v", p.W[17], p.W[18])
		}
		if p.W[17] > 2.0+1e-10 || p.W[18] > 2.0+1e-10 {
			t.Errorf("W[17]/W[18]: expected <= 2.0, got=%v/%v", p.W[17], p.W[18])
		}
	})
}

func TestValidateRequestRetention(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		ok    bool
	}{
		{"valid 0.9", 0.9, true},
		{"valid 0.5", 0.5, true},
		{"valid 1.0", 1.0, true},
		{"valid 0.01", 0.01, true},
		{"zero", 0, false},
		{"negative", -0.5, false},
		{"above 1", 1.5, false},
		{"NaN", math.NaN(), false},
		{"Inf", math.Inf(1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := DefaultParam()
			p.RequestRetention = tt.value
			err := p.Validate()
			if (err == nil) != tt.ok {
				t.Errorf("RequestRetention=%v: expected ok=%v, got err=%v", tt.value, tt.ok, err)
			}
		})
	}
}

func TestValidateMaximumInterval(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		ok    bool
	}{
		{"valid 36500", 36500, true},
		{"valid 1", 1, true},
		{"zero", 0, false},
		{"negative", -10, false},
		{"above max", 36501, false},
		{"NaN", math.NaN(), false},
		{"Inf", math.Inf(1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := DefaultParam()
			p.MaximumInterval = tt.value
			err := p.Validate()
			if (err == nil) != tt.ok {
				t.Errorf("MaximumInterval=%v: expected ok=%v, got err=%v", tt.value, tt.ok, err)
			}
		})
	}
}

func TestValidateLearningSteps(t *testing.T) {
	tests := []struct {
		name  string
		value []float64
		ok    bool
	}{
		{"valid default", []float64{1, 10}, true},
		{"valid empty", []float64{}, true},
		{"valid zero step", []float64{0}, true},
		{"negative step", []float64{-1}, false},
		{"NaN step", []float64{math.NaN()}, false},
		{"Inf step", []float64{math.Inf(1)}, false},
		{"mixed valid and invalid", []float64{1, -5}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := DefaultParam()
			p.LearningSteps = tt.value
			err := p.Validate()
			if (err == nil) != tt.ok {
				t.Errorf("LearningSteps=%v: expected ok=%v, got err=%v", tt.value, tt.ok, err)
			}
		})
	}
}

func TestValidateRelearningSteps(t *testing.T) {
	tests := []struct {
		name  string
		value []float64
		ok    bool
	}{
		{"valid default", []float64{10}, true},
		{"valid empty", []float64{}, true},
		{"valid zero step", []float64{0}, true},
		{"negative step", []float64{-5}, false},
		{"NaN step", []float64{math.NaN()}, false},
		{"Inf step", []float64{math.Inf(1)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := DefaultParam()
			p.RelearningSteps = tt.value
			err := p.Validate()
			if (err == nil) != tt.ok {
				t.Errorf("RelearningSteps=%v: expected ok=%v, got err=%v", tt.value, tt.ok, err)
			}
		})
	}
}

func TestNewFSRSClipsWeights(t *testing.T) {
	p := DefaultParam()
	p.W[0] = -999
	p.W[20] = 5.0
	f := NewFSRS(p)
	if f.W[0] != 0.001 {
		t.Errorf("W[0]: expected 0.001 after NewFSRS, got=%v", f.W[0])
	}
	if f.W[20] != 0.8 {
		t.Errorf("W[20]: expected 0.8 after NewFSRS, got=%v", f.W[20])
	}
}

func TestNewFSRSDynamicCeiling(t *testing.T) {
	t.Run("multiple relearning steps clips W17 W18", func(t *testing.T) {
		p := DefaultParam()
		p.RelearningSteps = []float64{10, 20}
		p.W[17] = 2.5
		p.W[18] = 2.5
		f := NewFSRS(p)

		// w11=1.4835, w13=0.2629, w14=1.6483
		// value = -(ln(1.4835) + ln(2^0.2629-1) + 1.6483*0.3) / 2 ≈ 0.3601
		// ceiling = sqrt(0.3601) ≈ 0.6001
		expectedCeiling := 0.6001
		tolerance := 1e-2
		if math.Abs(f.W[17]-expectedCeiling) > tolerance {
			t.Errorf("W[17]: expected ceiling ≈ %v (with sqrt), got=%v", expectedCeiling, f.W[17])
		}
		if math.Abs(f.W[18]-expectedCeiling) > tolerance {
			t.Errorf("W[18]: expected ceiling ≈ %v (with sqrt), got=%v", expectedCeiling, f.W[18])
		}
	})

	t.Run("NaN weights produce finite ceiling", func(t *testing.T) {
		p := DefaultParam()
		p.RelearningSteps = []float64{10, 20}
		p.W[11] = -1
		p.W[17] = 1.5
		p.W[18] = 1.5
		f := NewFSRS(p)
		if math.IsNaN(f.W[17]) || math.IsInf(f.W[17], 0) {
			t.Errorf("W[17]: expected finite after NewFSRS with negative W[11], got=%v", f.W[17])
		}
		if math.IsNaN(f.W[18]) || math.IsInf(f.W[18], 0) {
			t.Errorf("W[18]: expected finite after NewFSRS with negative W[11], got=%v", f.W[18])
		}
		// W[11]=-1 clamped to 0.001 -> value ≈ 4.011 -> sqrt ≈ 2.003 -> clamped to 2.0
		if f.W[17] > 2.0 {
			t.Errorf("W[17]: expected <= 2.0, got=%v", f.W[17])
		}
	})

	t.Run("single relearning step uses default ceiling of 2.0", func(t *testing.T) {
		p := DefaultParam()
		p.RelearningSteps = []float64{10}
		p.W[17] = 2.5
		p.W[18] = 2.5
		f := NewFSRS(p)

		if f.W[17] > 2.0 {
			t.Errorf("W[17]: expected <= 2.0 with single step, got=%v", f.W[17])
		}
		if f.W[18] > 2.0 {
			t.Errorf("W[18]: expected <= 2.0 with single step, got=%v", f.W[18])
		}
	})

	t.Run("negative value produces finite ceiling via sqrt guard", func(t *testing.T) {
		p := DefaultParam()
		p.RelearningSteps = []float64{10, 20}
		p.W[11] = 5.0
		p.W[13] = 0.9
		p.W[14] = 4.0
		p.W[17] = 1.5
		p.W[18] = 1.5
		f := NewFSRS(p)

		// value = -(ln(5) + ln(2^0.9-1) + 4*0.3) / 2 ≈ -1.333
		// math.Sqrt(math.Max(-1.333, 0)) = sqrt(0) = 0
		// clamp(0, 0.01, 2.0) = 0.01
		if math.IsNaN(f.W[17]) || math.IsInf(f.W[17], 0) {
			t.Errorf("W[17]: expected finite for negative value, got=%v", f.W[17])
		}
		if math.IsNaN(f.W[18]) || math.IsInf(f.W[18], 0) {
			t.Errorf("W[18]: expected finite for negative value, got=%v", f.W[18])
		}
		if f.W[17] < 0 || f.W[18] < 0 {
			t.Errorf("W[17]=%v, W[18]=%v: expected non-negative", f.W[17], f.W[18])
		}
	})

	t.Run("W17/W18 ceiling with multiple relearning steps applies square root", func(t *testing.T) {
		p := DefaultParam()
		p.RelearningSteps = []float64{10, 20, 30, 40} // len = 4
		p.W[11] = 0.1
		p.W[13] = 0.1
		p.W[14] = 0.5
		p.W[17] = 2.0
		p.W[18] = 2.0
		f := NewFSRS(p)

		// Manual calculation:
		// w11 = 0.1 -> ln(0.1) ≈ -2.30258509
		// w13 = 0.1 -> ln(2^0.1 - 1) = ln(1.07177346 - 1) = ln(0.07177346) ≈ -2.63422967
		// w14 = 0.5 -> w14 * 0.3 = 0.15
		// sum = -2.30258509 - 2.63422967 + 0.15 = -4.78681476
		// value = -sum / 4 = 1.19670369
		// expected ceiling = sqrt(value) ≈ 1.0939395
		// Without sqrt, the ceiling would be 1.19670369.
		expectedCeiling := 1.0939395
		tolerance := 1e-5
		if math.Abs(f.W[17]-expectedCeiling) > tolerance {
			t.Errorf("W[17]: expected ceiling ≈ %v (with sqrt), got=%v", expectedCeiling, f.W[17])
		}
		if math.Abs(f.W[18]-expectedCeiling) > tolerance {
			t.Errorf("W[18]: expected ceiling ≈ %v (with sqrt), got=%v", expectedCeiling, f.W[18])
		}

		// Verify mathematical constraint: w17 * w18 <= value
		rawValue := 1.19670369
		constraintTolerance := 1e-4
		if f.W[17]*f.W[18] > rawValue+constraintTolerance {
			t.Errorf("W[17]*W[18] = %v, expected <= %v (raw value)", f.W[17]*f.W[18], rawValue)
		}
	})
}

func TestNewFSRSW19EnableShortTerm(t *testing.T) {
	t.Run("EnableShortTerm true raises W19 to 0.01", func(t *testing.T) {
		p := DefaultParam()
		p.EnableShortTerm = true
		p.W[19] = 0.001
		f := NewFSRS(p)
		if f.W[19] < 0.01 {
			t.Errorf("W[19]: expected >= 0.01 with EnableShortTerm, got=%v", f.W[19])
		}
	})

	t.Run("EnableShortTerm false allows W19 below 0.01", func(t *testing.T) {
		p := DefaultParam()
		p.EnableShortTerm = false
		p.W[19] = 0.001
		f := NewFSRS(p)
		if f.W[19] != 0.001 {
			t.Errorf("W[19]: expected 0.001 without EnableShortTerm, got=%v", f.W[19])
		}
	})
}

func TestRetrievability(t *testing.T) {
	retrievabilityList := []float64{}
	fsrs := NewFSRS(DefaultParam())
	card := NewCard()
	now := time.Date(2022, 11, 29, 12, 30, 0, 0, time.UTC)
	r, err := fsrs.Retrievability(card, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	retrievabilityList = append(retrievabilityList, roundFloat(r, 4))
	for range 3 {
		{
			rec, err := fsrs.Next(card, now, Good)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			card = rec.Card
		}
		now = card.Due
		r, err := fsrs.Retrievability(card, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		retrievabilityList = append(retrievabilityList, roundFloat(r, 4))
	}
	wantRetrievabilityList := []float64{0, 1, 0.9095, 0.8998}
	if !reflect.DeepEqual(retrievabilityList, wantRetrievabilityList) {
		t.Errorf("expected:%v, got:%v", wantRetrievabilityList, retrievabilityList)
	}
}

func TestGetRetrievabilityBackwardCompat(t *testing.T) {
	f := NewFSRS(DefaultParam())
	card := Card{LastReview: time.Now().Add(-24 * time.Hour), Stability: 1, State: Review}
	now := card.LastReview.Add(24 * time.Hour)

	got, err := f.GetRetrievability(card, now)
	if err != nil {
		t.Fatalf("deprecated GetRetrievability should still work: %v", err)
	}
	expected, err := f.Retrievability(card, now)
	if err != nil {
		t.Fatalf("Retrievability error: %v", err)
	}
	if got != expected {
		t.Errorf("GetRetrievability = %v, want %v (should match Retrievability)", got, expected)
	}
}

func BenchmarkNextStates(b *testing.B) {
	p := DefaultParam()
	current := &MemoryState{Stability: 51.344814, Difficulty: 7.005062}
	for b.Loop() {
		p.NextStates(current, 0.9, 21)
	}
}

func BenchmarkNextStates_NewCard(b *testing.B) {
	p := DefaultParam()
	for b.Loop() {
		p.NextStates(nil, 0.9, 0)
	}
}

func BenchmarkNextState(b *testing.B) {
	p := DefaultParam()
	current := &MemoryState{Stability: 51.344814, Difficulty: 7.005062}
	for b.Loop() {
		p.NextState(current, 0.9, 21, Good)
	}
}

func BenchmarkForgettingCurve(b *testing.B) {
	p := DefaultParam()
	for b.Loop() {
		p.ForgettingCurve(21, 51.344814)
	}
}
