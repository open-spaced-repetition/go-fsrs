package fsrs

import (
	"math"
	"testing"
)

func TestApplyFuzz(t *testing.T) {
	p := DefaultParam()

	t.Run("returns input when fuzz disabled", func(t *testing.T) {
		p.EnableFuzz = false
		got := p.ApplyFuzz(5.0, 0, false)
		if got != 5.0 {
			t.Errorf("expected 5.0, got=%v", got)
		}
	})

	t.Run("returns input when interval below 2.5", func(t *testing.T) {
		got := p.ApplyFuzz(2.3, 0, true)
		if got != 2.3 {
			t.Errorf("expected 2.3, got=%v", got)
		}
	})

	t.Run("fuzz result within expected range at boundary 2.5", func(t *testing.T) {
		p.EnableFuzz = true
		p.seed = "test-seed-2.5"
		minInterval, maxInterval := getFuzzRange(3, 0, p.MaximumInterval)
		got := p.ApplyFuzz(2.5, 0, true)
		if got < float64(minInterval) || got > float64(maxInterval) {
			t.Errorf("expected result in [%d, %d], got=%v", minInterval, maxInterval, got)
		}
	})

	t.Run("fuzz result within expected range for interval 3", func(t *testing.T) {
		p.EnableFuzz = true
		p.seed = "test-seed-3"
		minInterval, maxInterval := getFuzzRange(3, 0, p.MaximumInterval)
		got := p.ApplyFuzz(3.0, 0, true)
		if got < float64(minInterval) || got > float64(maxInterval) {
			t.Errorf("expected result in [%d, %d], got=%v", minInterval, maxInterval, got)
		}
	})

	t.Run("fuzz result clamped by maximum interval", func(t *testing.T) {
		p.EnableFuzz = true
		p.MaximumInterval = 5
		p.seed = "test-seed-max"
		got := p.ApplyFuzz(100.0, 0, true)
		if got > 5 {
			t.Errorf("expected result <= 5 (MaximumInterval), got=%v", got)
		}
	})

	t.Run("fuzz result respects elapsed days floor", func(t *testing.T) {
		p.EnableFuzz = true
		p.MaximumInterval = 36500
		p.seed = "test-seed-elapsed"
		interval := 5.0
		elapsedDays := 4.0
		minInterval, maxInterval := getFuzzRange(interval, elapsedDays, p.MaximumInterval)
		got := p.ApplyFuzz(interval, elapsedDays, true)
		if got < float64(minInterval) || got > float64(maxInterval) {
			t.Errorf("expected result in [%d, %d], got=%v", minInterval, maxInterval, got)
		}
	})

	t.Run("fuzz is deterministic for same seed", func(t *testing.T) {
		p.EnableFuzz = true
		p.seed = "deterministic-seed"
		first := p.ApplyFuzz(10.0, 0, true)
		p.seed = "deterministic-seed"
		second := p.ApplyFuzz(10.0, 0, true)
		if first != second {
			t.Errorf("expected deterministic results: first=%v, second=%v", first, second)
		}
	})

	t.Run("fuzz range at large interval", func(t *testing.T) {
		p.EnableFuzz = true
		p.MaximumInterval = 36500
		p.seed = "test-seed-large"
		minInterval, maxInterval := getFuzzRange(100.0, 0, p.MaximumInterval)
		got := p.ApplyFuzz(100.0, 0, true)
		if got < float64(minInterval) || got > float64(maxInterval) {
			t.Errorf("expected result in [%d, %d], got=%v", minInterval, maxInterval, got)
		}
		if maxInterval > 36500 {
			t.Errorf("expected maxInterval <= 36500, got=%d", maxInterval)
		}
	})
}

func TestGetFuzzRange(t *testing.T) {
	t.Run("small interval uses minimal delta", func(t *testing.T) {
		minInterval, maxInterval := getFuzzRange(3, 0, 36500)
		if minInterval > maxInterval {
			t.Errorf("minInterval > maxInterval: %d > %d", minInterval, maxInterval)
		}
		if minInterval < 2 {
			t.Errorf("expected minInterval >= 2, got=%d", minInterval)
		}
	})

	t.Run("clamped by maximum interval", func(t *testing.T) {
		_, maxInterval := getFuzzRange(100000, 0, 10)
		if maxInterval > 10 {
			t.Errorf("expected maxInterval <= 10, got=%d", maxInterval)
		}
	})

	t.Run("elapsed days floor when interval exceeds elapsed", func(t *testing.T) {
		minInterval, _ := getFuzzRange(5, 4, 36500)
		if minInterval < 5 {
			t.Errorf("expected minInterval >= 5 (elapsedDays+1), got=%d", minInterval)
		}
	})

	t.Run("min never exceeds max", func(t *testing.T) {
		for _, tc := range []struct {
			interval    float64
			elapsed     float64
			maxInterval float64
		}{
			{2.5, 0, 36500},
			{7.0, 0, 36500},
			{20.0, 0, 36500},
			{5.0, 10, 36500},
			{3.0, 0, 2},
		} {
			minInterval, maxInterval := getFuzzRange(tc.interval, tc.elapsed, tc.maxInterval)
			if minInterval > maxInterval {
				t.Errorf("interval=%v elapsed=%v maxInterval=%v: minInterval=%d > maxInterval=%d", tc.interval, tc.elapsed, tc.maxInterval, minInterval, maxInterval)
			}
		}
	})
}

func TestFuzzRanges(t *testing.T) {
	ranges := FuzzRanges()

	if len(ranges) != 3 {
		t.Fatalf("expected 3 fuzz ranges, got %d", len(ranges))
	}

	expected := []fuzzRange{
		{Start: 2.5, End: 7.0, Factor: 0.15},
		{Start: 7.0, End: 20.0, Factor: 0.10},
		{Start: 20.0, End: math.Inf(1), Factor: 0.05},
	}

	for i, r := range ranges {
		if r.Start != expected[i].Start || r.End != expected[i].End || r.Factor != expected[i].Factor {
			t.Errorf("range[%d] got={%.2f,%.2f,%.2f} want={%.2f,%.2f,%.2f}",
				i, r.Start, r.End, r.Factor, expected[i].Start, expected[i].End, expected[i].Factor)
		}
	}

	t.Run("defensive copy", func(t *testing.T) {
		ranges2 := FuzzRanges()
		ranges[0].Start = 999
		if ranges2[0].Start == 999 {
			t.Error("mutating returned slice should not affect internal state")
		}
	})
}
