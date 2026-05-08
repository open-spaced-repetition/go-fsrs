package fsrs

import "math"

// FuzzRange defines a bucket for interval fuzz randomization.
type FuzzRange struct {
	Start  float64
	End    float64
	Factor float64
}

var FUZZ_RANGES = []FuzzRange{
	{Start: 2.5, End: 7.0, Factor: 0.15},
	{Start: 7.0, End: 20.0, Factor: 0.1},
	{Start: 20.0, End: math.Inf(1), Factor: 0.05},
}

func (p *Parameters) ApplyFuzz(ivl float64, elapsedDays float64, enableFuzz bool) float64 {
	if !enableFuzz || ivl < 2.5 {
		return ivl
	}

	generator := Alea(p.seed)
	fuzzFactor := generator.Double()

	minIvl, maxIvl := getFuzzRange(ivl, elapsedDays, p.MaximumInterval)

	return math.Floor(fuzzFactor*float64(maxIvl-minIvl+1)) + float64(minIvl)
}

func applyFuzz(ivl float64, elapsedDays float64, maximumInterval float64, seed string) float64 {
	generator := Alea(seed)
	fuzzFactor := generator.Double()

	minIvl, maxIvl := getFuzzRange(ivl, elapsedDays, maximumInterval)

	return math.Floor(fuzzFactor*float64(maxIvl-minIvl+1)) + float64(minIvl)
}

func getFuzzRange(interval, elapsedDays, maximumInterval float64) (minIvl, maxIvl int) {
	delta := 1.0
	for _, r := range FUZZ_RANGES {
		delta += r.Factor * max(min(interval, r.End)-r.Start, 0.0)
	}

	interval = min(interval, maximumInterval)
	minIvlFloat := max(2.0, math.Round(interval-delta))
	maxIvlFloat := min(math.Round(interval+delta), maximumInterval)

	if interval > elapsedDays {
		minIvlFloat = max(minIvlFloat, elapsedDays+1)
	}
	minIvlFloat = min(minIvlFloat, maxIvlFloat)

	minIvl = int(minIvlFloat)
	maxIvl = int(maxIvlFloat)

	return minIvl, maxIvl
}
