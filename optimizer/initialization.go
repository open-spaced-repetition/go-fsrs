package optimizer

import (
	"errors"
	"math"
)

// ErrNotEnoughData is returned when there's insufficient data for parameter initialization
var ErrNotEnoughData = errors.New("not enough data for parameter initialization")

// InitializeStabilityParameters finds optimal initial stability (S0) for each rating
// using ternary search to minimize log loss
func InitializeStabilityParameters(items []FSRSItem, avgRecall float64) ([4]float64, map[uint32]uint32, error) {
	// Group items by first rating
	groupedData := GroupByFirstRating(items)
	ratingCount := TotalRatingCount(groupedData)

	// Search for optimal stability for each rating
	ratingStability := searchParameters(groupedData, avgRecall)

	// Smooth and fill missing values
	result, err := SmoothAndFill(ratingStability, ratingCount)
	if err != nil {
		return [4]float64{}, nil, err
	}

	return result, ratingCount, nil
}

// powerForgettingCurveInit calculates retrievability for initialization
// Uses default decay parameter
func powerForgettingCurveInit(t, s float64) float64 {
	decay := -float64(DefaultParameters[20])
	factor := math.Pow(0.9, 1.0/decay) - 1.0
	return math.Pow(1.0+factor*t/s, decay)
}

// initLoss calculates the loss for parameter initialization
// Includes log loss and L1 regularization
func initLoss(deltaT, recall, count []float64, initS0, defaultS0 float64) float64 {
	var logLoss float64

	for i := range deltaT {
		predicted := powerForgettingCurveInit(deltaT[i], initS0)

		// Clamp to avoid log(0)
		predicted = Clamp(predicted, 1e-10, 1.0-1e-10)

		// Binary cross entropy
		loss := -(recall[i]*math.Log(predicted) + (1-recall[i])*math.Log(1-predicted))
		logLoss += loss * count[i]
	}

	// L1 regularization
	l1 := math.Abs(initS0-defaultS0) / 16.0

	return logLoss + l1
}

// searchParameters uses ternary search to find optimal S0 for each rating
func searchParameters(groupedData map[uint32][]AverageRecall, avgRecall float64) map[uint32]float64 {
	result := make(map[uint32]float64)
	epsilon := 1e-10

	for rating, data := range groupedData {
		if len(data) == 0 {
			continue
		}

		// Get default S0 for this rating
		defaultS0 := DefaultParameters[rating-1]

		// Prepare arrays
		deltaT := make([]float64, len(data))
		count := make([]float64, len(data))
		recall := make([]float64, len(data))

		for i, d := range data {
			deltaT[i] = d.DeltaT
			count[i] = d.Count

			// Apply Laplace smoothing
			// (real_recall * n + average_recall) / (n + 1)
			recall[i] = (d.Recall*d.Count + avgRecall) / (d.Count + 1.0)
		}

		// Ternary search for optimal S0
		low := SMin
		high := InitSMax
		optimalS := defaultS0

		for iter := 0; high-low > epsilon && iter < 1000; iter++ {
			mid1 := low + (high-low)/3.0
			mid2 := high - (high-low)/3.0

			loss1 := initLoss(deltaT, recall, count, mid1, defaultS0)
			loss2 := initLoss(deltaT, recall, count, mid2, defaultS0)

			if loss1 < loss2 {
				high = mid2
			} else if loss1 > loss2 {
				low = mid1
			} else {
				// When losses are equal, narrow from both sides for faster convergence
				low = mid1
				high = mid2
			}

			optimalS = (high + low) / 2.0
		}

		result[rating] = optimalS
	}

	return result
}

// SmoothAndFill fills missing stability values using interpolation
// and ensures S0[1] <= S0[2] <= S0[3] <= S0[4]
func SmoothAndFill(ratingStability map[uint32]float64, ratingCount map[uint32]uint32) ([4]float64, error) {
	// Remove ratings that don't have counts
	for rating := range ratingStability {
		if _, ok := ratingCount[rating]; !ok {
			delete(ratingStability, rating)
		}
	}

	if len(ratingStability) == 0 {
		return [4]float64{}, ErrNotEnoughData
	}

	// Ensure ordering: smaller rating should have smaller or equal stability
	pairs := [][2]uint32{{1, 2}, {2, 3}, {3, 4}, {1, 3}, {2, 4}, {1, 4}}
	for _, pair := range pairs {
		smallRating, bigRating := pair[0], pair[1]
		smallVal, smallOK := ratingStability[smallRating]
		bigVal, bigOK := ratingStability[bigRating]

		if smallOK && bigOK && smallVal > bigVal {
			// Fix ordering based on which has more data
			if ratingCount[smallRating] > ratingCount[bigRating] {
				ratingStability[bigRating] = smallVal
			} else {
				ratingStability[smallRating] = bigVal
			}
		}
	}

	// Interpolation weights
	const w1 = 0.41
	const w2 = 0.54

	var initS0 [4]float64

	// Convert map to array for easier access (1-indexed to 0-indexed)
	ratingArr := [5]*float64{nil, nil, nil, nil, nil}
	for i := uint32(1); i <= 4; i++ {
		if v, ok := ratingStability[i]; ok {
			val := v
			ratingArr[i] = &val
		}
	}

	switch len(ratingStability) {
	case 1:
		// Single rating: scale all defaults by the same factor
		var rating uint32
		for r := range ratingStability {
			rating = r
			break
		}
		factor := ratingStability[rating] / DefaultParameters[rating-1]
		for i := 0; i < 4; i++ {
			initS0[i] = DefaultParameters[i] * factor
		}

	case 2:
		// Two ratings: interpolate missing values
		initS0 = interpolateTwoRatings(ratingArr, w1, w2)

	case 3:
		// Three ratings: fill the missing one
		initS0 = interpolateThreeRatings(ratingArr, w1, w2)

	case 4:
		// All four ratings present
		for i := uint32(1); i <= 4; i++ {
			initS0[i-1] = *ratingArr[i]
		}
	}

	// Clamp to valid range
	for i := range initS0 {
		initS0[i] = Clamp(initS0[i], SMin, InitSMax)
	}

	return initS0, nil
}

// interpolateTwoRatings fills missing values when only 2 ratings are present
func interpolateTwoRatings(arr [5]*float64, w1, w2 float64) [4]float64 {
	var result [4]float64

	switch {
	case arr[1] == nil && arr[2] == nil && arr[3] != nil && arr[4] != nil:
		r3, r4 := *arr[3], *arr[4]
		r2 := math.Pow(r3, 1.0/(1.0-w2)) * math.Pow(r4, 1.0-1.0/(1.0-w2))
		r1 := math.Pow(r2, 1.0/w1) * math.Pow(r3, 1.0-1.0/w1)
		result = [4]float64{r1, r2, r3, r4}

	case arr[1] == nil && arr[2] != nil && arr[3] == nil && arr[4] != nil:
		r2, r4 := *arr[2], *arr[4]
		r3 := math.Pow(r2, 1.0-w2) * math.Pow(r4, w2)
		r1 := math.Pow(r2, 1.0/w1) * math.Pow(r3, 1.0-1.0/w1)
		result = [4]float64{r1, r2, r3, r4}

	case arr[1] == nil && arr[2] != nil && arr[3] != nil && arr[4] == nil:
		r2, r3 := *arr[2], *arr[3]
		r4 := math.Pow(r2, 1.0-1.0/w2) * math.Pow(r3, 1.0/w2)
		r1 := math.Pow(r2, 1.0/w1) * math.Pow(r3, 1.0-1.0/w1)
		result = [4]float64{r1, r2, r3, r4}

	case arr[1] != nil && arr[2] == nil && arr[3] == nil && arr[4] != nil:
		r1, r4 := *arr[1], *arr[4]
		denom := w1 + w2 - w1*w2
		r2 := math.Pow(r1, w1/denom) * math.Pow(r4, 1.0-w1/denom)
		r3 := math.Pow(r1, 1.0-w2/denom) * math.Pow(r4, w2/denom)
		result = [4]float64{r1, r2, r3, r4}

	case arr[1] != nil && arr[2] == nil && arr[3] != nil && arr[4] == nil:
		r1, r3 := *arr[1], *arr[3]
		r2 := math.Pow(r1, w1) * math.Pow(r3, 1.0-w1)
		r4 := math.Pow(r2, 1.0-1.0/w2) * math.Pow(r3, 1.0/w2)
		result = [4]float64{r1, r2, r3, r4}

	case arr[1] != nil && arr[2] != nil && arr[3] == nil && arr[4] == nil:
		r1, r2 := *arr[1], *arr[2]
		r3 := math.Pow(r1, 1.0-1.0/(1.0-w1)) * math.Pow(r2, 1.0/(1.0-w1))
		r4 := math.Pow(r2, 1.0-1.0/w2) * math.Pow(r3, 1.0/w2)
		result = [4]float64{r1, r2, r3, r4}

	default:
		// Fallback: use defaults
		for i := 0; i < 4; i++ {
			result[i] = DefaultParameters[i]
		}
	}

	return result
}

// interpolateThreeRatings fills the missing value when 3 ratings are present
func interpolateThreeRatings(arr [5]*float64, w1, w2 float64) [4]float64 {
	var result [4]float64

	switch {
	case arr[1] == nil && arr[2] != nil && arr[3] != nil:
		r2, r3 := *arr[2], *arr[3]
		result[0] = math.Pow(r2, 1.0/w1) * math.Pow(r3, 1.0-1.0/w1)
		result[1] = r2
		result[2] = r3
		if arr[4] != nil {
			result[3] = *arr[4]
		} else {
			result[3] = math.Pow(r2, 1.0-1.0/w2) * math.Pow(r3, 1.0/w2)
		}

	case arr[2] == nil && arr[1] != nil && arr[3] != nil:
		r1, r3 := *arr[1], *arr[3]
		result[0] = r1
		result[1] = math.Pow(r1, w1) * math.Pow(r3, 1.0-w1)
		result[2] = r3
		if arr[4] != nil {
			result[3] = *arr[4]
		} else {
			result[3] = math.Pow(result[1], 1.0-1.0/w2) * math.Pow(r3, 1.0/w2)
		}

	case arr[3] == nil && arr[2] != nil && arr[4] != nil:
		r2, r4 := *arr[2], *arr[4]
		result[2] = math.Pow(r2, 1.0-w2) * math.Pow(r4, w2)
		result[1] = r2
		result[3] = r4
		if arr[1] != nil {
			result[0] = *arr[1]
		} else {
			result[0] = math.Pow(r2, 1.0/w1) * math.Pow(result[2], 1.0-1.0/w1)
		}

	case arr[4] == nil && arr[2] != nil && arr[3] != nil:
		r2, r3 := *arr[2], *arr[3]
		result[3] = math.Pow(r2, 1.0-1.0/w2) * math.Pow(r3, 1.0/w2)
		result[1] = r2
		result[2] = r3
		if arr[1] != nil {
			result[0] = *arr[1]
		} else {
			result[0] = math.Pow(r2, 1.0/w1) * math.Pow(r3, 1.0-1.0/w1)
		}

	default:
		// Copy available values
		for i := uint32(1); i <= 4; i++ {
			if arr[i] != nil {
				result[i-1] = *arr[i]
			} else {
				result[i-1] = DefaultParameters[i-1]
			}
		}
	}

	return result
}
