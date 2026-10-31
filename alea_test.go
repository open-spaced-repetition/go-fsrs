package fsrs

import (
	"math"
	"strings"
	"testing"
	"time"
)

const (
	aleaSequenceLength = 1000
	aleaSampleCount    = 10000
)

func collectSequence(prng PRNG, n int) []float64 {
	seq := make([]float64, n)
	for i := range seq {
		seq[i] = prng()
	}
	return seq
}

func sequencesDifferAt(a, b []float64) bool {
	if len(a) != len(b) {
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return true
		}
	}
	return false
}

func assertUnitIntervalSequence(t *testing.T, seq []float64) {
	t.Helper()
	for i, v := range seq {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("value at index %d is not finite: %v", i, v)
		}
		if v < 0 || v >= 1 {
			t.Fatalf("value at index %d is outside [0, 1): %v", i, v)
		}
	}
}

func TestAleaDeterminism(t *testing.T) {
	tests := []struct {
		name string
		seed any
	}{
		{name: "string seed", seed: "deterministic-seed"},
		{name: "int seed", seed: 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := collectSequence(Alea(tt.seed), aleaSequenceLength)
			second := collectSequence(Alea(tt.seed), aleaSequenceLength)

			for i := range first {
				if first[i] != second[i] {
					t.Fatalf("sequence diverged at index %d: %v != %v", i, first[i], second[i])
				}
			}
		})
	}
}

func TestAleaDifferentSeeds(t *testing.T) {
	tests := []struct {
		name string
		a    any
		b    any
	}{
		{name: "string seeds", a: "seed-a", b: "seed-b"},
		{name: "int seeds", a: 1, b: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := collectSequence(Alea(tt.a), 8)
			b := collectSequence(Alea(tt.b), 8)

			if !sequencesDifferAt(a, b) {
				t.Fatalf("sequences for seeds %v and %v are identical", tt.a, tt.b)
			}
		})
	}
}

func TestNilSeed(t *testing.T) {
	t.Run("includes the timestamp", func(t *testing.T) {
		first := nilSeed(1000)
		second := nilSeed(2000)

		if !strings.HasPrefix(first, "1000-") {
			t.Fatalf("seed %q does not start with the timestamp", first)
		}
		if !strings.HasPrefix(second, "2000-") {
			t.Fatalf("seed %q does not start with the timestamp", second)
		}
	})

	t.Run("disambiguates the same timestamp", func(t *testing.T) {
		first := nilSeed(1000)
		second := nilSeed(1000)

		if first == second {
			t.Fatalf("same timestamp produced identical seeds: %q", first)
		}

		firstSeq := collectSequence(Alea(first), 3)
		secondSeq := collectSequence(Alea(second), 3)
		if !sequencesDifferAt(firstSeq, secondSeq) {
			t.Fatal("same-timestamp seeds produced identical sequences")
		}
	})
}

func TestAleaNilSeedSameClockTick(t *testing.T) {
	fixed := func() time.Time { return time.Unix(0, 1000) }

	first := newAleaWithClock(nil, fixed)
	second := newAleaWithClock(nil, fixed)

	firstSeq := collectSequence(PRNG(first.Next), 3)
	secondSeq := collectSequence(PRNG(second.Next), 3)

	if !sequencesDifferAt(firstSeq, secondSeq) {
		t.Fatal("nil seeds from the same clock tick produced identical sequences")
	}
}

func TestAleaNilSeed(t *testing.T) {
	t.Run("does not panic and stays in range", func(t *testing.T) {
		seq := collectSequence(Alea(nil), aleaSequenceLength)
		assertUnitIntervalSequence(t, seq)
	})

	t.Run("does not reuse the empty string seed", func(t *testing.T) {
		nilSeq := collectSequence(Alea(nil), 3)
		emptySeq := collectSequence(Alea(""), 3)

		if !sequencesDifferAt(nilSeq, emptySeq) {
			t.Fatal("Alea(nil) produced the same sequence as Alea(\"\")")
		}
	})

	t.Run("successive calls produce distinct sequences", func(t *testing.T) {
		const calls = 100
		sequences := make([][]float64, calls)
		for i := range sequences {
			sequences[i] = collectSequence(Alea(nil), 3)
		}

		for i := range sequences {
			for j := i + 1; j < len(sequences); j++ {
				if !sequencesDifferAt(sequences[i], sequences[j]) {
					t.Fatalf("calls %d and %d produced the same sequence", i, j)
				}
			}
		}
	})
}

func TestPRNGNext(t *testing.T) {
	tests := []struct {
		name string
		seed any
	}{
		{name: "string seed", seed: "range-string"},
		{name: "zero int seed", seed: 0},
		{name: "one int seed", seed: 1},
		{name: "int seed", seed: 42},
		{name: "other string seed", seed: "another-seed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := collectSequence(Alea(tt.seed), aleaSampleCount)
			assertUnitIntervalSequence(t, seq)
		})
	}
}

func TestPRNGDouble(t *testing.T) {
	t.Run("range and finiteness", func(t *testing.T) {
		prng := Alea("double-range")
		seq := make([]float64, aleaSampleCount)
		for i := range seq {
			seq[i] = prng.Double()
		}

		assertUnitIntervalSequence(t, seq)
	})

	t.Run("extends precision beyond Next", func(t *testing.T) {
		prng := Alea("double-precision")

		for i := 0; i < aleaSequenceLength; i++ {
			scaled := prng.Double() * (1 << 32)
			if scaled != math.Trunc(scaled) {
				return
			}
		}

		t.Fatal("Double never produced precision below 2^-32")
	})

	t.Run("is exactly representable at 53 bits", func(t *testing.T) {
		prng := Alea("double-exact")

		for i := 0; i < aleaSampleCount; i++ {
			scaled := prng.Double() * (1 << 53)
			if scaled != math.Trunc(scaled) {
				t.Fatalf("value at index %d is not a multiple of 2^-53: %v", i, scaled)
			}
		}
	})
}
