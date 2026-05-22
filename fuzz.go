package fsrs

import "math"

// fuzzRange defines a bucket for interval fuzz randomization.
type fuzzRange struct {
	Start  float64
	End    float64
	Factor float64
}

var fuzzRanges = []fuzzRange{
	{Start: 2.5, End: 7.0, Factor: 0.15},
	{Start: 7.0, End: 20.0, Factor: 0.1},
	{Start: 20.0, End: math.Inf(1), Factor: 0.05},
}

// FuzzRanges returns a defensive copy of the internal fuzz range table used for
// interval randomization. The returned slice elements have exported fields
// (Start, End, Factor) that callers can read via range. Mutating the returned
// slice or its elements does not affect the library's internal state.
func FuzzRanges() []fuzzRange {
	out := make([]fuzzRange, len(fuzzRanges))
	copy(out, fuzzRanges)
	return out
}

func (p *Parameters) ApplyFuzz(interval float64, elapsedDays float64, enableFuzz bool) float64 {
	if !enableFuzz || interval < 2.5 {
		return interval
	}

	generator := Alea(p.seed)
	fuzzFactor := generator.Double()

	minInterval, maxInterval := getFuzzRange(interval, elapsedDays, p.MaximumInterval)

	return math.Floor(fuzzFactor*float64(maxInterval-minInterval+1)) + float64(minInterval)
}

func applyFuzz(interval float64, elapsedDays float64, maximumInterval float64, seed string) float64 {
	generator := Alea(seed)
	fuzzFactor := generator.Double()

	minInterval, maxInterval := getFuzzRange(interval, elapsedDays, maximumInterval)

	return math.Floor(fuzzFactor*float64(maxInterval-minInterval+1)) + float64(minInterval)
}

func getFuzzRange(interval, elapsedDays, maximumInterval float64) (minInterval, maxInterval int) {
	delta := 1.0
	for _, r := range fuzzRanges {
		delta += r.Factor * max(min(interval, r.End)-r.Start, 0.0)
	}

	interval = min(interval, maximumInterval)
	minIntervalFloat := max(2.0, math.Round(interval-delta))
	maxIntervalFloat := min(math.Round(interval+delta), maximumInterval)

	if interval > elapsedDays {
		minIntervalFloat = max(minIntervalFloat, elapsedDays+1)
	}
	minIntervalFloat = min(minIntervalFloat, maxIntervalFloat)

	minInterval = int(minIntervalFloat)
	maxInterval = int(maxIntervalFloat)

	return minInterval, maxInterval
}
