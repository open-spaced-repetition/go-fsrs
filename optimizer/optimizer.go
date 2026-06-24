package optimizer

import (
	"fmt"
	"math"
)

// ComputeParametersResult holds the result of parameter optimization
type ComputeParametersResult struct {
	// Parameters are the optimized FSRS parameters (21 values)
	Parameters []float64

	// InitialStability are the S0 values found during initialization
	InitialStability [4]float64

	// Metrics contains evaluation metrics
	Metrics *ModelEvaluation

	// TrainingInfo contains training statistics
	TrainingInfo *TrainResult
}

// ComputeParameters computes optimized FSRS parameters from review history
// This is the main entry point for the optimizer
func ComputeParameters(input ComputeParametersInput) (*ComputeParametersResult, error) {
	if len(input.TrainSet) == 0 {
		return nil, fmt.Errorf("train set is empty")
	}

	// Set defaults
	if input.NumRelearningSteps == 0 {
		input.NumRelearningSteps = 1
	}

	// 1. Prepare training data
	initItems, trainItems := PrepareTrainingData(input.TrainSet)

	// 2. Calculate average recall for Laplace smoothing
	allItems := append(initItems, trainItems...)
	avgRecall := CalculateAverageRecall(allItems)

	// 3. Check minimum data requirement
	if len(allItems) < 8 {
		// Not enough data - return defaults
		return &ComputeParametersResult{
			Parameters:       DefaultParameters[:],
			InitialStability: [4]float64{DefaultParameters[0], DefaultParameters[1], DefaultParameters[2], DefaultParameters[3]},
		}, nil
	}

	// 4. Initialize S0 via ternary search
	initialS0, _, err := InitializeStabilityParameters(initItems, avgRecall)
	if err != nil {
		// If initialization fails, use defaults
		initialS0 = [4]float64{DefaultParameters[0], DefaultParameters[1], DefaultParameters[2], DefaultParameters[3]}
	}

	// 5. Build initialized parameters
	initializedParams := make([]float64, NumParams)
	copy(initializedParams[:4], initialS0[:])
	copy(initializedParams[4:], DefaultParameters[4:])

	// 6. If not enough data for full training, return initialized params
	if len(trainItems) < 64 || len(trainItems) == len(initItems) {
		return &ComputeParametersResult{
			Parameters:       initializedParams,
			InitialStability: initialS0,
		}, nil
	}

	// 7. Prepare training configuration
	config := DefaultTrainingConfig()
	config.FreezeInitialStability = input.FreezeInitialStability
	config.FreezeShortTerm = !input.EnableShortTerm
	config.NumRelearningSteps = input.NumRelearningSteps

	// 8. Apply recency weighting
	weightedItems := RecencyWeightedItems(trainItems)

	// 9. Filter by sequence length
	weightedItems = FilterBySeqLen(weightedItems, config.MaxSeqLen)

	if len(weightedItems) == 0 {
		return &ComputeParametersResult{
			Parameters:       initializedParams,
			InitialStability: initialS0,
		}, nil
	}

	// 10. Train the model
	var trainResult *TrainResult
	if input.ProgressFunc != nil {
		trainResult, err = TrainWithProgress(weightedItems, config, func(state ProgressState) {
			input.ProgressFunc(state.Current(), state.Total())
		})
	} else {
		trainResult, err = TrainWithInitialStability(weightedItems, initialS0, config)
	}

	if err != nil {
		return nil, fmt.Errorf("training failed: %w", err)
	}

	// 11. Get optimized parameters
	optimizedParams := trainResult.Parameters

	// 12. Check for invalid values
	for i, p := range optimizedParams {
		if math.IsInf(p, 0) || math.IsNaN(p) {
			return nil, fmt.Errorf("invalid parameter at index %d: %v", i, p)
		}
	}

	// 13. Smooth and validate initial stability
	optimizedParams = smoothOptimizedStability(optimizedParams, initialS0)

	// 14. Final parameter clipping
	optimizedParams = ClipParameters(optimizedParams, config.NumRelearningSteps, input.EnableShortTerm)

	// 15. Compute evaluation metrics
	model := NewModelFromSlice(optimizedParams)
	metrics := Evaluate(model, trainItems)

	return &ComputeParametersResult{
		Parameters:       optimizedParams,
		InitialStability: initialS0,
		Metrics:          &metrics,
		TrainingInfo:     trainResult,
	}, nil
}

// smoothOptimizedStability ensures initial stability values are properly ordered
func smoothOptimizedStability(params []float64, initialS0 [4]float64) []float64 {
	result := make([]float64, len(params))
	copy(result, params)

	// Ensure S0 values are valid and ordered
	for i := 0; i < 4; i++ {
		if result[i] < SMin || math.IsNaN(result[i]) {
			result[i] = initialS0[i]
		}
		result[i] = Clamp(result[i], SMin, InitSMax)
	}

	// Ensure ordering: S0[0] <= S0[1] <= S0[2] <= S0[3]
	for i := 1; i < 4; i++ {
		if result[i] < result[i-1] {
			result[i] = result[i-1]
		}
	}

	return result
}

// OptimizeParameters is an alias for ComputeParameters for API compatibility
func OptimizeParameters(items []FSRSItem, enableShortTerm bool) ([]float64, error) {
	result, err := ComputeParameters(ComputeParametersInput{
		TrainSet:        items,
		EnableShortTerm: enableShortTerm,
	})
	if err != nil {
		return nil, err
	}
	return result.Parameters, nil
}

// Benchmark runs parameter optimization and returns the parameters
// without smoothing (useful for testing/benchmarking)
func Benchmark(input ComputeParametersInput) ([]float64, error) {
	if len(input.TrainSet) == 0 {
		return nil, fmt.Errorf("train set is empty")
	}

	avgRecall := CalculateAverageRecall(input.TrainSet)

	// Separate initialization and training sets
	initItems, trainItems := PrepareTrainingData(input.TrainSet)

	// Initialize S0
	initialS0, _, err := InitializeStabilityParameters(initItems, avgRecall)
	if err != nil {
		initialS0 = [4]float64{DefaultParameters[0], DefaultParameters[1], DefaultParameters[2], DefaultParameters[3]}
	}

	// Configure training
	config := DefaultTrainingConfig()
	config.FreezeInitialStability = input.FreezeInitialStability
	config.FreezeShortTerm = !input.EnableShortTerm

	// Prepare weighted items
	weightedItems := RecencyWeightedItems(trainItems)
	weightedItems = FilterBySeqLen(weightedItems, config.MaxSeqLen)

	if len(weightedItems) < 64 {
		// Not enough data
		result := make([]float64, NumParams)
		copy(result[:4], initialS0[:])
		copy(result[4:], DefaultParameters[4:])
		return result, nil
	}

	// Train
	trainResult, err := TrainWithInitialStability(weightedItems, initialS0, config)
	if err != nil {
		return nil, err
	}

	return trainResult.Parameters, nil
}

// EvaluateParameters evaluates a set of parameters on a dataset
func EvaluateParameters(params []float64, items []FSRSItem) (*ModelEvaluation, error) {
	if len(params) < NumParams {
		return nil, fmt.Errorf("invalid parameter count: got %d, expected %d", len(params), NumParams)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("items is empty")
	}

	model := NewModelFromSlice(params)
	metrics := Evaluate(model, items)
	return &metrics, nil
}

// GetDefaultParameters returns the default FSRS v6 parameters
func GetDefaultParameters() []float64 {
	result := make([]float64, NumParams)
	copy(result, DefaultParameters[:])
	return result
}

// ValidateParameters checks if parameters are within valid ranges
func ValidateParameters(params []float64) error {
	if len(params) != NumParams {
		return fmt.Errorf("invalid parameter count: got %d, expected %d", len(params), NumParams)
	}

	// Check for NaN/Inf
	for i, p := range params {
		if math.IsNaN(p) {
			return fmt.Errorf("parameter %d is NaN", i)
		}
		if math.IsInf(p, 0) {
			return fmt.Errorf("parameter %d is infinite", i)
		}
	}

	// Check S0 ordering
	for i := 1; i < 4; i++ {
		if params[i] < params[i-1] {
			return fmt.Errorf("S0 parameters not ordered: S0[%d]=%f < S0[%d]=%f", i, params[i], i-1, params[i-1])
		}
	}

	// Check bounds
	for i := 0; i < 4; i++ {
		if params[i] < SMin || params[i] > InitSMax {
			return fmt.Errorf("S0[%d]=%f out of bounds [%f, %f]", i, params[i], SMin, InitSMax)
		}
	}

	if params[4] < DMin || params[4] > DMax {
		return fmt.Errorf("D0=%f out of bounds [%f, %f]", params[4], DMin, DMax)
	}

	return nil
}

// ConvertRevlogToFSRSItems converts review log entries to FSRSItems
// This is a helper for loading data from databases
type RevlogEntry struct {
	CardID     int64
	Rating     uint32 // 1-4
	DeltaDays  uint32 // Days since last review
	ReviewTime int64  // Unix timestamp in milliseconds
}

// GroupRevlogByCard groups revlog entries by card ID and creates FSRSItems
func GroupRevlogByCard(entries []RevlogEntry) []FSRSItem {
	// Group by card ID
	cardMap := make(map[int64][]RevlogEntry)
	for _, entry := range entries {
		cardMap[entry.CardID] = append(cardMap[entry.CardID], entry)
	}

	// Create FSRSItems
	items := make([]FSRSItem, 0, len(cardMap))

	for _, cardEntries := range cardMap {
		if len(cardEntries) < 2 {
			continue // Need at least 2 reviews
		}

		// Sort by review time (should already be sorted, but ensure)
		// Simple bubble sort for small arrays
		for i := 0; i < len(cardEntries)-1; i++ {
			for j := 0; j < len(cardEntries)-i-1; j++ {
				if cardEntries[j].ReviewTime > cardEntries[j+1].ReviewTime {
					cardEntries[j], cardEntries[j+1] = cardEntries[j+1], cardEntries[j]
				}
			}
		}

		// Build FSRSItem
		reviews := make([]FSRSReview, len(cardEntries))
		for i, entry := range cardEntries {
			reviews[i] = FSRSReview{
				Rating: entry.Rating,
				DeltaT: entry.DeltaDays,
			}
		}

		// Create one FSRSItem per review (except the first)
		for i := 1; i < len(reviews); i++ {
			item := FSRSItem{
				Reviews: make([]FSRSReview, i+1),
			}
			copy(item.Reviews, reviews[:i+1])
			items = append(items, item)
		}
	}

	return items
}
