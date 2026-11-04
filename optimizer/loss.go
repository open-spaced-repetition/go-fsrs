package optimizer

import (
	"math"
)

// ReductionType specifies how to reduce the loss
type ReductionType int

const (
	// ReductionMean computes the mean of the losses
	ReductionMean ReductionType = iota
	// ReductionSum computes the sum of the losses
	ReductionSum
	// ReductionAuto computes weighted average (sum / sum of weights)
	ReductionAuto
)

// BCELoss computes Binary Cross Entropy loss
// loss = -(y * log(p) + (1-y) * log(1-p)) * weight
func BCELoss(predicted, actual, weights []float64, reduction ReductionType) float64 {
	if len(predicted) == 0 || len(predicted) != len(actual) {
		return 0
	}

	// Ensure weights exist
	if len(weights) != len(predicted) {
		weights = make([]float64, len(predicted))
		for i := range weights {
			weights[i] = 1.0
		}
	}

	var totalLoss float64
	var totalWeight float64

	for i := range predicted {
		p := predicted[i]
		y := actual[i]
		w := weights[i]

		// Clamp predicted value to avoid log(0)
		p = Clamp(p, 1e-10, 1.0-1e-10)

		// Binary cross entropy for single sample
		loss := -(y*math.Log(p) + (1-y)*math.Log(1-p))

		totalLoss += loss * w
		totalWeight += w
	}

	switch reduction {
	case ReductionMean:
		if len(predicted) > 0 {
			return totalLoss / float64(len(predicted))
		}
		return 0
	case ReductionSum:
		return totalLoss
	case ReductionAuto:
		if totalWeight > 0 {
			return totalLoss / totalWeight
		}
		return 0
	default:
		return totalLoss
	}
}

// L2Regularization computes L2 regularization penalty
// L2 = sum((w - w_init)^2 / stddev^2) * gamma * batchSize / totalSize
func L2Regularization(w, initW, stddev []float64, gamma float64, batchSize, totalSize int) float64 {
	if len(w) == 0 || len(w) != len(initW) || len(w) != len(stddev) {
		return 0
	}

	var penalty float64

	for i := range w {
		if stddev[i] > 0 {
			diff := w[i] - initW[i]
			penalty += (diff * diff) / (stddev[i] * stddev[i])
		}
	}

	return penalty * gamma * float64(batchSize) / float64(totalSize)
}

// ComputeBatchLoss computes the loss for a batch of items
func ComputeBatchLoss(model *Model, items []WeightedFSRSItem) float64 {
	if len(items) == 0 {
		return 0
	}

	predicted := make([]float64, len(items))
	actual := make([]float64, len(items))
	weights := make([]float64, len(items))

	for i, item := range items {
		// Predict retrievability
		predicted[i] = model.PredictRetrievability(item.Item)

		// Get actual label (1 if recalled, 0 if forgot)
		current := item.Item.Current()
		if current != nil && current.Rating > 1 {
			actual[i] = 1.0
		} else {
			actual[i] = 0.0
		}

		weights[i] = item.Weight
	}

	return BCELoss(predicted, actual, weights, ReductionSum)
}

// ComputeTotalLoss computes loss + L2 regularization
func ComputeTotalLoss(model *Model, items []WeightedFSRSItem, initW []float64, gamma float64, totalSize int) float64 {
	batchLoss := ComputeBatchLoss(model, items)

	l2Penalty := L2Regularization(
		model.W[:],
		initW,
		ParamsStdDev[:],
		gamma,
		len(items),
		totalSize,
	)

	return batchLoss + l2Penalty
}

// LossFunction is a function type that computes loss for given parameters
type LossFunction func(params []float64) float64

// CreateLossFunction creates a loss function for the given data
func CreateLossFunction(items []WeightedFSRSItem, initW []float64, gamma float64, totalSize int) LossFunction {
	return func(params []float64) float64 {
		model := NewModelFromSlice(params)
		return ComputeTotalLoss(model, items, initW, gamma, totalSize)
	}
}

// RMSE computes Root Mean Square Error
func RMSE(predicted, actual []float64) float64 {
	if len(predicted) == 0 || len(predicted) != len(actual) {
		return 0
	}

	var sumSq float64
	for i := range predicted {
		diff := predicted[i] - actual[i]
		sumSq += diff * diff
	}

	return math.Sqrt(sumSq / float64(len(predicted)))
}

// LogLoss computes average log loss (cross entropy)
func LogLoss(predicted, actual []float64) float64 {
	if len(predicted) == 0 || len(predicted) != len(actual) {
		return 0
	}

	var totalLoss float64
	for i := range predicted {
		p := Clamp(predicted[i], 1e-10, 1.0-1e-10)
		y := actual[i]
		loss := -(y*math.Log(p) + (1-y)*math.Log(1-p))
		totalLoss += loss
	}

	return totalLoss / float64(len(predicted))
}

// ModelEvaluation holds evaluation metrics
type ModelEvaluation struct {
	LogLoss  float64
	RMSE     float64
	RMSEBins float64
}

// Evaluate computes evaluation metrics for the model
func Evaluate(model *Model, items []FSRSItem) ModelEvaluation {
	if len(items) == 0 {
		return ModelEvaluation{}
	}

	predicted := make([]float64, len(items))
	actual := make([]float64, len(items))

	for i, item := range items {
		predicted[i] = model.PredictRetrievability(item)

		current := item.Current()
		if current != nil && current.Rating > 1 {
			actual[i] = 1.0
		} else {
			actual[i] = 0.0
		}
	}

	return ModelEvaluation{
		LogLoss:  LogLoss(predicted, actual),
		RMSE:     RMSE(predicted, actual),
		RMSEBins: computeRMSEBins(items, model),
	}
}

// computeRMSEBins computes RMSE based on R-matrix binning
func computeRMSEBins(items []FSRSItem, model *Model) float64 {
	// Group by R-matrix bins
	type binKey struct {
		deltaTBin, lengthBin, lapseBin uint32
	}
	type binValue struct {
		predictedSum float64
		actualSum    float64
		count        int
	}

	bins := make(map[binKey]*binValue)

	for _, item := range items {
		key := binKey{}
		key.deltaTBin, key.lengthBin, key.lapseBin = item.RMatrixIndex()

		predicted := model.PredictRetrievability(item)
		actual := 0.0
		if current := item.Current(); current != nil && current.Rating > 1 {
			actual = 1.0
		}

		if _, ok := bins[key]; !ok {
			bins[key] = &binValue{}
		}
		bins[key].predictedSum += predicted
		bins[key].actualSum += actual
		bins[key].count++
	}

	// Compute RMSE over bins
	var sumSq float64
	var count int

	for _, bin := range bins {
		if bin.count > 0 {
			avgPredicted := bin.predictedSum / float64(bin.count)
			avgActual := bin.actualSum / float64(bin.count)
			diff := avgPredicted - avgActual
			sumSq += diff * diff
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return math.Sqrt(sumSq / float64(count))
}
