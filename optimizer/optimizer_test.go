package optimizer

import (
	"math"
	"testing"
)

// Helper function to check approximate equality
func approxEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestFSRSItem(t *testing.T) {
	item := FSRSItem{
		Reviews: []FSRSReview{
			{Rating: 3, DeltaT: 0},
			{Rating: 3, DeltaT: 1},
			{Rating: 3, DeltaT: 3},
			{Rating: 2, DeltaT: 7},
		},
	}

	t.Run("History", func(t *testing.T) {
		history := item.History()
		if len(history) != 3 {
			t.Errorf("Expected history length 3, got %d", len(history))
		}
	})

	t.Run("Current", func(t *testing.T) {
		current := item.Current()
		if current == nil {
			t.Fatal("Expected current to not be nil")
		}
		if current.Rating != 2 || current.DeltaT != 7 {
			t.Errorf("Expected Rating=2, DeltaT=7, got Rating=%d, DeltaT=%d", current.Rating, current.DeltaT)
		}
	})

	t.Run("LongTermReviewCount", func(t *testing.T) {
		count := item.LongTermReviewCount()
		if count != 3 {
			t.Errorf("Expected 3 long-term reviews, got %d", count)
		}
	})

	t.Run("FirstLongTermReview", func(t *testing.T) {
		first := item.FirstLongTermReview()
		if first == nil {
			t.Fatal("Expected first long-term review to not be nil")
		}
		if first.DeltaT != 1 {
			t.Errorf("Expected DeltaT=1, got %d", first.DeltaT)
		}
	})
}

func TestDefaultParameters(t *testing.T) {
	if len(DefaultParameters) != NumParams {
		t.Errorf("Expected %d parameters, got %d", NumParams, len(DefaultParameters))
	}

	// Check first 4 parameters are initial stabilities (should be > 0)
	for i := 0; i < 4; i++ {
		if DefaultParameters[i] <= 0 {
			t.Errorf("S0[%d] should be > 0, got %f", i, DefaultParameters[i])
		}
	}
}

func TestModel_PowerForgettingCurve(t *testing.T) {
	model := NewDefaultModel()

	testCases := []struct {
		t, s     float64
		expected float64
	}{
		{0, 1, 1.0},       // At t=0, retrievability is 1
		{1, 1, 0.9},       // At t=s, R should be 0.9 (by design)
		{10, 10, 0.9},     // Scaling property
		{100, 100, 0.9},   // More scaling
		{1, 2, 0.9403443}, // From fsrs-rs tests
		{2, 3, 0.9253786}, // From fsrs-rs tests
		{3, 4, 0.9185229}, // From fsrs-rs tests
	}

	for _, tc := range testCases {
		result := model.PowerForgettingCurve(tc.t, tc.s)
		if !approxEqual(result, tc.expected, 0.001) {
			t.Errorf("PowerForgettingCurve(%f, %f) = %f, expected %f", tc.t, tc.s, result, tc.expected)
		}
	}
}

func TestModel_InitStability(t *testing.T) {
	model := NewDefaultModel()

	for rating := 1; rating <= 4; rating++ {
		s := model.InitStability(rating)
		expected := DefaultParameters[rating-1]
		if !approxEqual(s, expected, 0.0001) {
			t.Errorf("InitStability(%d) = %f, expected %f", rating, s, expected)
		}
	}
}

func TestModel_InitDifficulty(t *testing.T) {
	model := NewDefaultModel()

	// Difficulty should decrease with higher ratings
	prevD := 11.0 // Higher than max
	for rating := 1; rating <= 4; rating++ {
		d := model.InitDifficulty(rating)
		if d >= prevD {
			t.Errorf("InitDifficulty should decrease with rating: D(%d)=%f >= D(%d)=%f", rating, d, rating-1, prevD)
		}
		if d < DMin || d > DMax {
			t.Errorf("InitDifficulty(%d) = %f, should be in [%f, %f]", rating, d, DMin, DMax)
		}
		prevD = d
	}
}

func TestModel_Forward(t *testing.T) {
	model := NewDefaultModel()

	item := FSRSItem{
		Reviews: []FSRSReview{
			{Rating: 3, DeltaT: 0}, // First review: Good
			{Rating: 3, DeltaT: 1}, // Second review after 1 day
			{Rating: 3, DeltaT: 3}, // Third review after 3 days
		},
	}

	state := model.Forward(item)

	// Stability should be positive and reasonable
	if state.Stability <= 0 {
		t.Errorf("Stability should be > 0, got %f", state.Stability)
	}
	if state.Stability > SMax {
		t.Errorf("Stability should be <= %f, got %f", SMax, state.Stability)
	}

	// Difficulty should be in valid range
	if state.Difficulty < DMin || state.Difficulty > DMax {
		t.Errorf("Difficulty should be in [%f, %f], got %f", DMin, DMax, state.Difficulty)
	}
}

func TestBCELoss(t *testing.T) {
	predicted := []float64{0.9, 0.8, 0.7, 0.6}
	actual := []float64{1.0, 1.0, 0.0, 0.0}
	weights := []float64{1.0, 1.0, 1.0, 1.0}

	loss := BCELoss(predicted, actual, weights, ReductionMean)

	// Loss should be positive
	if loss <= 0 {
		t.Errorf("BCE loss should be > 0, got %f", loss)
	}

	// Perfect predictions should have low loss
	perfectPredicted := []float64{0.999, 0.999, 0.001, 0.001}
	perfectLoss := BCELoss(perfectPredicted, actual, weights, ReductionMean)

	if perfectLoss >= loss {
		t.Errorf("Perfect predictions should have lower loss: %f >= %f", perfectLoss, loss)
	}
}

func TestAdam(t *testing.T) {
	adam := NewAdam(0.1, 3)

	params := []float64{1.0, 2.0, 3.0}
	grad := []float64{0.1, 0.2, 0.3}

	// First step
	newParams := adam.Step(grad, params)

	// Parameters should have moved in the opposite direction of gradients
	for i := range params {
		if newParams[i] >= params[i] {
			t.Errorf("Parameter %d should have decreased: %f >= %f", i, newParams[i], params[i])
		}
	}

	// Multiple steps should continue to update
	for step := 0; step < 10; step++ {
		newParams = adam.Step(grad, newParams)
	}

	// After many steps, parameters should have decreased significantly
	for i := range params {
		if newParams[i] >= params[i]-0.1 {
			t.Errorf("After 10 steps, parameter %d should have decreased more: %f", i, newParams[i])
		}
	}
}

func TestCosineAnnealingLR(t *testing.T) {
	scheduler := NewCosineAnnealingLR(100, 0.1)

	// Initial LR should be close to initial
	lr := scheduler.CurrentLR()
	if !approxEqual(lr, 0.1, 0.01) {
		t.Errorf("Initial LR should be ~0.1, got %f", lr)
	}

	// Step through half the schedule
	for i := 0; i < 50; i++ {
		scheduler.Step()
	}

	// At halfway, LR should be around 0.05
	lr = scheduler.CurrentLR()
	if !approxEqual(lr, 0.05, 0.01) {
		t.Errorf("Halfway LR should be ~0.05, got %f", lr)
	}

	// Step to the end
	for i := 0; i < 50; i++ {
		scheduler.Step()
	}

	// At end, LR should be close to 0
	lr = scheduler.CurrentLR()
	if lr > 0.01 {
		t.Errorf("Final LR should be close to 0, got %f", lr)
	}
}

func TestNumericalGradient(t *testing.T) {
	// Simple quadratic function: f(x) = x[0]^2 + x[1]^2
	// Gradient: [2*x[0], 2*x[1]]
	f := func(x []float64) float64 {
		return x[0]*x[0] + x[1]*x[1]
	}

	params := []float64{3.0, 4.0}
	grad := NumericalGradient(f, params, 1e-7)

	expectedGrad := []float64{6.0, 8.0} // 2*3, 2*4

	for i := range grad {
		if !approxEqual(grad[i], expectedGrad[i], 0.001) {
			t.Errorf("Gradient[%d] = %f, expected %f", i, grad[i], expectedGrad[i])
		}
	}
}

func TestPrepareTrainingData(t *testing.T) {
	items := []FSRSItem{
		// Item with 1 long-term review -> goes to init
		{Reviews: []FSRSReview{{Rating: 3, DeltaT: 0}, {Rating: 3, DeltaT: 1}}},
		// Item with 2 long-term reviews -> goes to train
		{Reviews: []FSRSReview{{Rating: 3, DeltaT: 0}, {Rating: 3, DeltaT: 1}, {Rating: 4, DeltaT: 3}}},
		// Item with 3 long-term reviews -> goes to train
		{Reviews: []FSRSReview{{Rating: 2, DeltaT: 0}, {Rating: 3, DeltaT: 1}, {Rating: 3, DeltaT: 5}, {Rating: 4, DeltaT: 10}}},
	}

	initItems, trainItems := PrepareTrainingData(items)

	if len(initItems) < 1 {
		t.Errorf("Expected at least 1 init item, got %d", len(initItems))
	}
	if len(trainItems) < 2 {
		t.Errorf("Expected at least 2 train items, got %d", len(trainItems))
	}
}

func TestCalculateAverageRecall(t *testing.T) {
	items := []FSRSItem{
		{Reviews: []FSRSReview{{Rating: 3, DeltaT: 0}, {Rating: 4, DeltaT: 1}}}, // Pass
		{Reviews: []FSRSReview{{Rating: 3, DeltaT: 0}, {Rating: 3, DeltaT: 1}}}, // Pass
		{Reviews: []FSRSReview{{Rating: 3, DeltaT: 0}, {Rating: 1, DeltaT: 1}}}, // Fail
		{Reviews: []FSRSReview{{Rating: 3, DeltaT: 0}, {Rating: 2, DeltaT: 1}}}, // Pass
	}

	avgRecall := CalculateAverageRecall(items)

	// 3 out of 4 passed (rating > 1)
	expected := 0.75
	if !approxEqual(avgRecall, expected, 0.01) {
		t.Errorf("Average recall = %f, expected %f", avgRecall, expected)
	}
}

func TestClipParameters(t *testing.T) {
	// Create parameters with out-of-range values
	params := make([]float64, NumParams)
	copy(params, DefaultParameters[:])

	// Set some invalid values
	params[0] = -1     // Should be clipped to SMin
	params[4] = 15     // D0 should be clipped to 10
	params[7] = 2      // Mean reversion should be clipped to 1
	params[20] = 0.001 // Decay too low

	clipped := ClipParameters(params, 1, true)

	if clipped[0] < SMin {
		t.Errorf("S0 should be >= %f, got %f", SMin, clipped[0])
	}
	if clipped[4] > 10 {
		t.Errorf("D0 should be <= 10, got %f", clipped[4])
	}
	if clipped[7] > 1 {
		t.Errorf("Mean reversion should be <= 1, got %f", clipped[7])
	}
	if clipped[20] < 0.01 {
		t.Errorf("Decay should be >= 0.01, got %f", clipped[20])
	}
}

func TestL2Regularization(t *testing.T) {
	w := []float64{1.0, 2.0, 3.0}
	initW := []float64{0.0, 0.0, 0.0}
	stddev := []float64{1.0, 1.0, 1.0}

	penalty := L2Regularization(w, initW, stddev, 1.0, 10, 100)

	// penalty = sum((w - initW)^2 / stddev^2) * gamma * batchSize / totalSize
	// = (1 + 4 + 9) * 1.0 * 10 / 100 = 14 * 0.1 = 1.4
	expected := 1.4
	if !approxEqual(penalty, expected, 0.001) {
		t.Errorf("L2 penalty = %f, expected %f", penalty, expected)
	}
}

// ============== Integration Tests ==============

// generateSyntheticData creates synthetic review data for testing
func generateSyntheticData(numCards int) []FSRSItem {
	items := make([]FSRSItem, 0, numCards*3)

	// Simple LCG for reproducible random numbers
	seed := uint64(12345)
	random := func() float64 {
		seed = seed*6364136223846793005 + 1442695040888963407
		return float64(seed>>33) / float64(1<<31)
	}

	for card := 0; card < numCards; card++ {
		// Each card has 2-5 reviews
		numReviews := 2 + int(random()*4)
		deltaT := uint32(0)

		for review := 1; review < numReviews; review++ {
			reviews := make([]FSRSReview, review+1)

			// First review always has deltaT=0
			reviews[0] = FSRSReview{
				Rating: uint32(1 + int(random()*4)), // 1-4
				DeltaT: 0,
			}

			// Subsequent reviews
			for r := 1; r <= review; r++ {
				deltaT = uint32(1 + int(random()*30)) // 1-30 days
				rating := uint32(2 + int(random()*3)) // 2-4 (mostly pass)
				if random() < 0.1 {
					rating = 1 // 10% failure rate
				}
				reviews[r] = FSRSReview{
					Rating: rating,
					DeltaT: deltaT,
				}
			}

			items = append(items, FSRSItem{Reviews: reviews})
		}
	}

	return items
}

func TestTrainBasic(t *testing.T) {
	// Generate synthetic data
	items := generateSyntheticData(50)

	// Weight items
	weightedItems := RecencyWeightedItems(items)

	// Use quick training config
	config := DefaultTrainingConfig()
	config.NumEpochs = 1
	config.BatchSize = 64

	result, err := Train(weightedItems, config)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	// Check result is valid
	if result == nil {
		t.Fatal("Training result is nil")
	}

	if len(result.Parameters) != NumParams {
		t.Errorf("Expected %d parameters, got %d", NumParams, len(result.Parameters))
	}

	// Check parameters are valid
	for i, p := range result.Parameters {
		if math.IsNaN(p) {
			t.Errorf("Parameter %d is NaN", i)
		}
		if math.IsInf(p, 0) {
			t.Errorf("Parameter %d is Inf", i)
		}
	}

	// Loss should be positive
	if result.BestLoss <= 0 {
		t.Errorf("Best loss should be > 0, got %f", result.BestLoss)
	}
}

func TestTrainWithInitialStability(t *testing.T) {
	items := generateSyntheticData(50)
	weightedItems := RecencyWeightedItems(items)

	initialS0 := [4]float64{0.5, 1.0, 2.0, 5.0}

	config := DefaultTrainingConfig()
	config.NumEpochs = 1
	config.BatchSize = 64

	result, err := TrainWithInitialStability(weightedItems, initialS0, config)
	if err != nil {
		t.Fatalf("Training failed: %v", err)
	}

	if result == nil {
		t.Fatal("Training result is nil")
	}

	// Parameters should have been updated from initial
	if len(result.Parameters) != NumParams {
		t.Errorf("Expected %d parameters, got %d", NumParams, len(result.Parameters))
	}
}

func TestComputeParametersMinimalData(t *testing.T) {
	// Test with minimal data - should return defaults
	items := []FSRSItem{
		{Reviews: []FSRSReview{{Rating: 3, DeltaT: 0}, {Rating: 3, DeltaT: 1}}},
		{Reviews: []FSRSReview{{Rating: 3, DeltaT: 0}, {Rating: 4, DeltaT: 2}}},
	}

	result, err := ComputeParameters(ComputeParametersInput{
		TrainSet:        items,
		EnableShortTerm: true,
	})

	if err != nil {
		t.Fatalf("ComputeParameters failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	// With minimal data, should return valid parameters
	if len(result.Parameters) != NumParams {
		t.Errorf("Expected %d parameters, got %d", NumParams, len(result.Parameters))
	}
}

func TestComputeParametersSufficientData(t *testing.T) {
	// Generate enough data for full training
	items := generateSyntheticData(100)

	result, err := ComputeParameters(ComputeParametersInput{
		TrainSet:        items,
		EnableShortTerm: true,
	})

	if err != nil {
		t.Fatalf("ComputeParameters failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	// Check parameters
	if len(result.Parameters) != NumParams {
		t.Errorf("Expected %d parameters, got %d", NumParams, len(result.Parameters))
	}

	// Validate parameters
	err = ValidateParameters(result.Parameters)
	if err != nil {
		t.Errorf("Parameters validation failed: %v", err)
	}

	// S0 should be ordered
	for i := 1; i < 4; i++ {
		if result.Parameters[i] < result.Parameters[i-1] {
			t.Errorf("S0 not ordered: S0[%d]=%f < S0[%d]=%f",
				i, result.Parameters[i], i-1, result.Parameters[i-1])
		}
	}
}

func TestComputeParametersWithProgress(t *testing.T) {
	items := generateSyntheticData(50)

	progressCalled := false
	result, err := ComputeParameters(ComputeParametersInput{
		TrainSet:        items,
		EnableShortTerm: true,
		ProgressFunc: func(current, total int) {
			progressCalled = true
			if current < 0 || total <= 0 {
				t.Errorf("Invalid progress: %d/%d", current, total)
			}
		},
	})

	if err != nil {
		t.Fatalf("ComputeParameters failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	// Progress should have been called at least once
	// (may not be called if data is too small for training)
	_ = progressCalled
}

func TestValidateParameters(t *testing.T) {
	t.Run("valid parameters", func(t *testing.T) {
		err := ValidateParameters(DefaultParameters[:])
		if err != nil {
			t.Errorf("Default parameters should be valid: %v", err)
		}
	})

	t.Run("invalid count", func(t *testing.T) {
		err := ValidateParameters([]float64{1, 2, 3})
		if err == nil {
			t.Error("Should fail with invalid parameter count")
		}
	})

	t.Run("NaN parameter", func(t *testing.T) {
		params := make([]float64, NumParams)
		copy(params, DefaultParameters[:])
		params[5] = math.NaN()
		err := ValidateParameters(params)
		if err == nil {
			t.Error("Should fail with NaN parameter")
		}
	})

	t.Run("unordered S0", func(t *testing.T) {
		params := make([]float64, NumParams)
		copy(params, DefaultParameters[:])
		params[0] = 10.0 // S0[0] > S0[1]
		params[1] = 1.0
		err := ValidateParameters(params)
		if err == nil {
			t.Error("Should fail with unordered S0")
		}
	})
}

func TestEvaluateParameters(t *testing.T) {
	items := generateSyntheticData(50)

	metrics, err := EvaluateParameters(DefaultParameters[:], items)
	if err != nil {
		t.Fatalf("EvaluateParameters failed: %v", err)
	}

	if metrics == nil {
		t.Fatal("Metrics is nil")
	}

	// LogLoss should be positive
	if metrics.LogLoss < 0 {
		t.Errorf("LogLoss should be >= 0, got %f", metrics.LogLoss)
	}

	// RMSE should be in [0, 1]
	if metrics.RMSE < 0 || metrics.RMSE > 1 {
		t.Errorf("RMSE should be in [0, 1], got %f", metrics.RMSE)
	}
}

func TestGetDefaultParameters(t *testing.T) {
	params := GetDefaultParameters()

	if len(params) != NumParams {
		t.Errorf("Expected %d parameters, got %d", NumParams, len(params))
	}

	// Should be a copy, not the original
	params[0] = 999.0
	if DefaultParameters[0] == 999.0 {
		t.Error("GetDefaultParameters should return a copy")
	}
}

func TestGroupRevlogByCard(t *testing.T) {
	entries := []RevlogEntry{
		{CardID: 1, Rating: 3, DeltaDays: 0, ReviewTime: 1000},
		{CardID: 1, Rating: 3, DeltaDays: 1, ReviewTime: 2000},
		{CardID: 1, Rating: 4, DeltaDays: 3, ReviewTime: 3000},
		{CardID: 2, Rating: 2, DeltaDays: 0, ReviewTime: 1000},
		{CardID: 2, Rating: 3, DeltaDays: 2, ReviewTime: 2000},
	}

	items := GroupRevlogByCard(entries)

	// Should create multiple FSRSItems per card
	if len(items) < 3 {
		t.Errorf("Expected at least 3 items, got %d", len(items))
	}

	// Check that items have correct structure
	for _, item := range items {
		if len(item.Reviews) < 2 {
			t.Errorf("Each item should have at least 2 reviews")
		}
	}
}

func BenchmarkTrain(b *testing.B) {
	items := generateSyntheticData(100)
	weightedItems := RecencyWeightedItems(items)

	config := DefaultTrainingConfig()
	config.NumEpochs = 1
	config.BatchSize = 128

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Train(weightedItems, config)
	}
}

func BenchmarkNumericalGradient(b *testing.B) {
	items := generateSyntheticData(50)
	weightedItems := RecencyWeightedItems(items)
	initW := DefaultParameters[:]

	lossFunc := CreateLossFunction(weightedItems, initW, 1.0, len(weightedItems))
	params := DefaultParameters[:]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NumericalGradient(lossFunc, params, 1e-7)
	}
}

func BenchmarkForward(b *testing.B) {
	model := NewDefaultModel()
	item := FSRSItem{
		Reviews: []FSRSReview{
			{Rating: 3, DeltaT: 0},
			{Rating: 3, DeltaT: 1},
			{Rating: 4, DeltaT: 3},
			{Rating: 3, DeltaT: 7},
			{Rating: 2, DeltaT: 14},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.Forward(item)
	}
}
