package fsrs

import (
	"math"
	"time"
)

func dateDiffInDays(last, cur time.Time) uint64 {
	lr := last.UTC()
	utc1 := time.Date(lr.Year(), lr.Month(), lr.Day(), 0, 0, 0, 0, time.UTC)
	n := cur.UTC()
	utc2 := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	hours := utc2.Sub(utc1).Hours()
	if hours < 0 {
		return 0
	}
	return uint64(math.Floor(hours / 24))
}

func dateDiffRaw(last, cur time.Time) float64 {
	return math.Floor(cur.Sub(last).Hours() / 24)
}

func minutesToDuration(minutes float64) time.Duration {
	return time.Duration(minutes * float64(time.Minute))
}

func daysToDuration(days, maxDays float64) time.Duration {
	days = math.Min(days, maxDays)
	hours := days * 24
	return time.Duration(hours * float64(time.Hour))
}

func againDelayMinutes(steps []float64) float64 {
	if len(steps) == 0 {
		return 0
	}
	return steps[0]
}

func hardDelayMinutes(steps []float64) float64 {
	if len(steps) == 0 {
		return 0
	}
	first := steps[0]
	if len(steps) == 1 {
		return math.Round(first * 1.5)
	}
	return math.Round((first + steps[1]) / 2)
}

func goodDelayMinutes(steps []float64, remaining int) (float64, bool) {
	nextIdx := len(steps) - remaining + 1
	if nextIdx < 0 || nextIdx >= len(steps) {
		return 0, false
	}
	return steps[nextIdx], true
}
