package fsrs

import (
	"math"
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
