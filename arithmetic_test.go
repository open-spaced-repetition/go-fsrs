package fsrs

import (
	"math"
	"reflect"
	"testing"
)

func TestStabilityToInterval(t *testing.T) {
	p := DefaultParam()
	decay, factor := p.decayAndFactor()

	t.Run("typical stability produces finite positive interval", func(t *testing.T) {
		for _, s := range []float64{0.212, 1.0, 10.0, 100.0} {
			got := stabilityToInterval(s, decay, factor, 0.9)
			if got <= 0 || math.IsNaN(got) || math.IsInf(got, 0) {
				t.Errorf("stabilityToInterval(%v) = %v, want finite positive", s, got)
			}
		}
	})

	t.Run("boundary stability sMin and sMax produce finite positive", func(t *testing.T) {
		for _, s := range []float64{sMin, sMax} {
			got := stabilityToInterval(s, decay, factor, 0.9)
			if got <= 0 || math.IsNaN(got) || math.IsInf(got, 0) {
				t.Errorf("stabilityToInterval(%v) = %v, want finite positive", s, got)
			}
		}
	})

	t.Run("retention=1.0 yields zero interval", func(t *testing.T) {
		got := stabilityToInterval(10.0, decay, factor, 1.0)
		if got != 0.0 {
			t.Errorf("stabilityToInterval at retention=1.0 = %v, want 0", got)
		}
	})

	t.Run("retention approaching zero yields +Inf", func(t *testing.T) {
		got := stabilityToInterval(10.0, decay, factor, 1e-300)
		if !math.IsInf(got, 1) {
			t.Errorf("stabilityToInterval at retention→0+ = %v, want +Inf", got)
		}
	})

	t.Run("monotonic in stability", func(t *testing.T) {
		prev := stabilityToInterval(0.001, decay, factor, 0.9)
		for s := 0.01; s <= 100.0; s *= 1.5 {
			cur := stabilityToInterval(s, decay, factor, 0.9)
			if cur <= prev {
				t.Errorf("not monotonic at s=%v: prev=%v cur=%v", s, prev, cur)
			}
			prev = cur
		}
	})

	t.Run("inverse of SM2 stability computation", func(t *testing.T) {
		s := 10.0
		retention := 0.9
		interval := stabilityToInterval(s, decay, factor, retention)
		recovered := interval * factor / (math.Pow(retention, 1/decay) - 1)
		if math.Abs(recovered-s) > 1e-9 {
			t.Errorf("round-trip failed: started s=%v, recovered=%v", s, recovered)
		}
	})

	t.Run("matches nextIntervalRaw for same retention", func(t *testing.T) {
		s := 5.0
		raw := stabilityToInterval(s, decay, factor, p.RequestRetention)
		fromMethod := p.nextIntervalRaw(s)
		if math.Abs(raw-fromMethod) > 1e-15 {
			t.Errorf("stabilityToInterval=%v != nextIntervalRaw=%v", raw, fromMethod)
		}
	})
}


func TestNextInterval(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)
	var intervalList []float64
	for i := 1; i <= 10; i++ {
		fsrs.RequestRetention = float64(i) / 10
		intervalList = append(intervalList, fsrs.nextInterval(1, 0))
	}
	wantIntervalList := []float64{36500, 34793, 2508, 387, 90, 27, 9, 3, 1, 1}
	if !reflect.DeepEqual(intervalList, wantIntervalList) {
		t.Errorf("expected:%v, got:%v", wantIntervalList, intervalList)
	}
}

func TestNextIntervalRaw(t *testing.T) {
	p := DefaultParam()
	fsrs := NewFSRS(p)

	interval := fsrs.nextIntervalRaw(0.212)
	if interval >= 1.0 {
		t.Errorf("nextIntervalRaw(0.212) should return < 1 day, got=%v", interval)
	}
	if interval <= 0 {
		t.Errorf("nextIntervalRaw(0.212) should return > 0, got=%v", interval)
	}

	interval = fsrs.nextIntervalRaw(1.0)
	if interval < 0.99 {
		t.Errorf("nextIntervalRaw(1.0) should return ~1 day, got=%v", interval)
	}

	interval = fsrs.nextIntervalRaw(0.001)
	if interval >= 0.5 {
		t.Errorf("nextIntervalRaw(0.001) should return < 0.5 days, got=%v", interval)
	}

	interval = fsrs.nextIntervalRaw(sMin)
	if interval <= 0 || math.IsNaN(interval) || math.IsInf(interval, 0) {
		t.Errorf("nextIntervalRaw(sMin) should return finite positive, got=%v", interval)
	}

	interval = fsrs.nextIntervalRaw(sMax)
	if interval <= 0 || math.IsNaN(interval) || math.IsInf(interval, 0) {
		t.Errorf("nextIntervalRaw(sMax) should return finite positive, got=%v", interval)
	}

	decay, factor := p.decayAndFactor()
	boundaryStab := 0.5 * factor / (math.Pow(p.RequestRetention, 1/decay) - 1)
	interval = fsrs.nextIntervalRaw(boundaryStab)
	if math.Abs(interval-0.5) > 1e-9 {
		t.Errorf("nextIntervalRaw at boundary stability should ≈ 0.5, got=%v", interval)
	}
}

func TestDecayAndFactorDerivedFromW20(t *testing.T) {
	p := DefaultParam()
	p.W[20] = 0.2
	p.Decay = -0.5
	p.Factor = math.Pow(0.9, 1.0/p.Decay) - 1.0

	got := p.ForgettingCurve(1, 1)
	wantDecay := -p.W[20]
	wantFactor := math.Pow(0.9, 1.0/wantDecay) - 1.0
	want := math.Pow(1+wantFactor, wantDecay)

	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("ForgettingCurve should use W[20], got=%v want=%v", got, want)
	}
}

func TestInvalidW20ValidationAndFallback(t *testing.T) {
	p := DefaultParam()
	p.W[20] = 0
	if err := p.Validate(); err == nil {
		t.Fatal("expected validation error for invalid W[20]")
	}

	gotDecay, gotFactor := p.decayAndFactor()
	wantDecay, wantFactor := defaultDecayAndFactor()
	if gotDecay != wantDecay || gotFactor != wantFactor {
		t.Fatalf("decayAndFactor should fallback to defaults, got=(%v,%v) want=(%v,%v)", gotDecay, gotFactor, wantDecay, wantFactor)
	}

	pDefault := DefaultParam()
	if got, want := p.nextInterval(1, 0), pDefault.nextInterval(1, 0); got != want {
		t.Fatalf("nextInterval should fallback safely, got=%v want=%v", got, want)
	}
}

func TestStabilityIsClamped(t *testing.T) {
	p := DefaultParam()
	p.W[0] = 1e9

	if got := p.initStability(Again); got != sMax {
		t.Fatalf("initStability should clamp to sMax, got=%v", got)
	}

	if got := p.shortTermStability(1e9, Good); got != sMax {
		t.Fatalf("shortTermStability should clamp to sMax, got=%v", got)
	}
}
