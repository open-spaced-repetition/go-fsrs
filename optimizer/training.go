package optimizer

import (
	"fmt"
	"math"
)

// TrainResult holds the result of training
type TrainResult struct {
	Parameters []float64
	FinalLoss  float64
	BestLoss   float64
	Epochs     int
}

// Train trains the FSRS model on the given dataset
func Train(trainSet []WeightedFSRSItem, config TrainingConfig) (*TrainResult, error) {
	if len(trainSet) == 0 {
		return nil, fmt.Errorf("training set is empty")
	}

	// Filter by sequence length
	trainSet = FilterBySeqLen(trainSet, config.MaxSeqLen)
	if len(trainSet) == 0 {
		return nil, fmt.Errorf("no items remaining after filtering by sequence length")
	}

	totalSize := len(trainSet)

	// Initialize model with default parameters
	model := NewDefaultModel()

	// If initial stability is provided, use it
	// (This would be set by InitializeStabilityParameters)

	// Store initial weights for L2 regularization
	initW := model.Parameters()

	// Calculate total iterations for learning rate scheduler
	batchCount := (totalSize + config.BatchSize - 1) / config.BatchSize
	totalIterations := batchCount * config.NumEpochs

	// Initialize optimizer and scheduler
	adam := NewAdam(config.LearningRate, NumParams)
	scheduler := NewCosineAnnealingLR(totalIterations, config.LearningRate)

	// Training state
	bestLoss := math.Inf(1)
	bestParams := model.Parameters()
	var finalLoss float64

	// Training loop
	for epoch := 1; epoch <= config.NumEpochs; epoch++ {
		// Shuffle data at the start of each epoch
		ShuffleItems(trainSet, config.Seed+int64(epoch))

		// Process batches
		batches := BatchItems(trainSet, config.BatchSize)
		epochLoss := 0.0

		for _, batch := range batches {
			// Get current learning rate
			lr := scheduler.Step()

			// Create loss function for this batch
			lossFunc := CreateLossFunction(batch, initW, config.Gamma, totalSize)

			// Compute current loss
			currentParams := model.Parameters()
			batchLoss := lossFunc(currentParams)

			// Compute gradients
			grad := NumericalGradient(lossFunc, currentParams, 1e-7)

			// Freeze gradients if configured
			if config.FreezeInitialStability {
				grad = FreezeInitialStability(grad)
			}
			if config.FreezeShortTerm {
				grad = FreezeShortTermStability(grad)
			}

			// Clip gradients to prevent explosion
			grad = GradientClipping(grad, 10.0)

			// Adam step with current learning rate
			newParams := adam.StepWithLR(grad, currentParams, lr)

			// Clip parameters to valid ranges
			newParams = ClipParameters(newParams, config.NumRelearningSteps, !config.FreezeShortTerm)

			// Update model
			model.SetParameters(newParams)

			epochLoss += batchLoss
		}

		// Average epoch loss
		epochLoss /= float64(len(batches))
		finalLoss = epochLoss

		// Track best model
		if epochLoss < bestLoss {
			bestLoss = epochLoss
			bestParams = model.Parameters()
		}
	}

	return &TrainResult{
		Parameters: bestParams,
		FinalLoss:  finalLoss,
		BestLoss:   bestLoss,
		Epochs:     config.NumEpochs,
	}, nil
}

// TrainWithProgress trains with progress reporting
func TrainWithProgress(trainSet []WeightedFSRSItem, config TrainingConfig, progress func(state ProgressState)) (*TrainResult, error) {
	if len(trainSet) == 0 {
		return nil, fmt.Errorf("training set is empty")
	}

	// Filter by sequence length
	trainSet = FilterBySeqLen(trainSet, config.MaxSeqLen)
	if len(trainSet) == 0 {
		return nil, fmt.Errorf("no items remaining after filtering by sequence length")
	}

	totalSize := len(trainSet)

	// Initialize model
	model := NewDefaultModel()
	initW := model.Parameters()

	// Calculate total iterations
	batchCount := (totalSize + config.BatchSize - 1) / config.BatchSize
	totalIterations := batchCount * config.NumEpochs

	// Initialize optimizer and scheduler
	adam := NewAdam(config.LearningRate, NumParams)
	scheduler := NewCosineAnnealingLR(totalIterations, config.LearningRate)

	// Training state
	bestLoss := math.Inf(1)
	bestParams := model.Parameters()
	var finalLoss float64

	// Training loop
	for epoch := 1; epoch <= config.NumEpochs; epoch++ {
		ShuffleItems(trainSet, config.Seed+int64(epoch))
		batches := BatchItems(trainSet, config.BatchSize)
		epochLoss := 0.0
		itemsProcessed := 0

		for batchIdx, batch := range batches {
			lr := scheduler.Step()

			lossFunc := CreateLossFunction(batch, initW, config.Gamma, totalSize)
			currentParams := model.Parameters()
			batchLoss := lossFunc(currentParams)

			grad := NumericalGradient(lossFunc, currentParams, 1e-7)

			if config.FreezeInitialStability {
				grad = FreezeInitialStability(grad)
			}
			if config.FreezeShortTerm {
				grad = FreezeShortTermStability(grad)
			}

			grad = GradientClipping(grad, 10.0)
			newParams := adam.StepWithLR(grad, currentParams, lr)
			newParams = ClipParameters(newParams, config.NumRelearningSteps, !config.FreezeShortTerm)
			model.SetParameters(newParams)

			epochLoss += batchLoss
			itemsProcessed += len(batch)

			// Report progress
			if progress != nil {
				progress(ProgressState{
					Epoch:          epoch,
					EpochTotal:     config.NumEpochs,
					ItemsProcessed: itemsProcessed,
					ItemsTotal:     totalSize,
				})
			}

			_ = batchIdx // unused but helps with debugging
		}

		epochLoss /= float64(len(batches))
		finalLoss = epochLoss

		if epochLoss < bestLoss {
			bestLoss = epochLoss
			bestParams = model.Parameters()
		}
	}

	return &TrainResult{
		Parameters: bestParams,
		FinalLoss:  finalLoss,
		BestLoss:   bestLoss,
		Epochs:     config.NumEpochs,
	}, nil
}

// TrainWithInitialStability trains with pre-computed initial stability values
func TrainWithInitialStability(trainSet []WeightedFSRSItem, initialS0 [4]float64, config TrainingConfig) (*TrainResult, error) {
	if len(trainSet) == 0 {
		return nil, fmt.Errorf("training set is empty")
	}

	trainSet = FilterBySeqLen(trainSet, config.MaxSeqLen)
	if len(trainSet) == 0 {
		return nil, fmt.Errorf("no items remaining after filtering by sequence length")
	}

	totalSize := len(trainSet)

	// Initialize model with custom initial stability
	model := NewDefaultModel()
	params := model.Parameters()
	for i := 0; i < 4; i++ {
		params[i] = initialS0[i]
	}

	// If short-term is disabled, zero out those parameters
	if config.FreezeShortTerm {
		params[17] = 0
		params[18] = 0
		params[19] = 0
	}

	model.SetParameters(params)
	initW := model.Parameters()

	batchCount := (totalSize + config.BatchSize - 1) / config.BatchSize
	totalIterations := batchCount * config.NumEpochs

	adam := NewAdam(config.LearningRate, NumParams)
	scheduler := NewCosineAnnealingLR(totalIterations, config.LearningRate)

	bestLoss := math.Inf(1)
	bestParams := model.Parameters()
	var finalLoss float64

	for epoch := 1; epoch <= config.NumEpochs; epoch++ {
		ShuffleItems(trainSet, config.Seed+int64(epoch))
		batches := BatchItems(trainSet, config.BatchSize)
		epochLoss := 0.0

		for _, batch := range batches {
			lr := scheduler.Step()

			lossFunc := CreateLossFunction(batch, initW, config.Gamma, totalSize)
			currentParams := model.Parameters()
			batchLoss := lossFunc(currentParams)

			grad := NumericalGradient(lossFunc, currentParams, 1e-7)

			if config.FreezeInitialStability {
				grad = FreezeInitialStability(grad)
			}
			if config.FreezeShortTerm {
				grad = FreezeShortTermStability(grad)
			}

			grad = GradientClipping(grad, 10.0)
			newParams := adam.StepWithLR(grad, currentParams, lr)
			newParams = ClipParameters(newParams, config.NumRelearningSteps, !config.FreezeShortTerm)
			model.SetParameters(newParams)

			epochLoss += batchLoss
		}

		epochLoss /= float64(len(batches))
		finalLoss = epochLoss

		if epochLoss < bestLoss {
			bestLoss = epochLoss
			bestParams = model.Parameters()
		}
	}

	return &TrainResult{
		Parameters: bestParams,
		FinalLoss:  finalLoss,
		BestLoss:   bestLoss,
		Epochs:     config.NumEpochs,
	}, nil
}

// QuickTrain performs a quick training with reduced epochs for testing
func QuickTrain(trainSet []WeightedFSRSItem) (*TrainResult, error) {
	config := DefaultTrainingConfig()
	config.NumEpochs = 2
	config.BatchSize = 256
	return Train(trainSet, config)
}
