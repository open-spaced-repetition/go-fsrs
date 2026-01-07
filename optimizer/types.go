package optimizer

import "math"

// FSRSReview represents a single review event
type FSRSReview struct {
	// Rating is the user's rating (1=Again, 2=Hard, 3=Good, 4=Easy)
	Rating uint32 `json:"rating"`
	// DeltaT is the number of days since the last review (0 for first review)
	DeltaT uint32 `json:"delta_t"`
}

// FSRSItem represents the complete review history for a single card
// Each FSRSItem corresponds to a single review, but contains the previous
// reviews of the card as well
type FSRSItem struct {
	Reviews []FSRSReview `json:"reviews"`
}

// WeightedFSRSItem adds recency weight for training
type WeightedFSRSItem struct {
	Item   FSRSItem
	Weight float64
}

// MemoryState represents the memory state of a card
type MemoryState struct {
	Stability  float64
	Difficulty float64
}

// History returns all reviews except the last one (the current review)
func (item *FSRSItem) History() []FSRSReview {
	if len(item.Reviews) <= 1 {
		return nil
	}
	return item.Reviews[:len(item.Reviews)-1]
}

// Current returns the last review (the one being evaluated)
func (item *FSRSItem) Current() *FSRSReview {
	if len(item.Reviews) == 0 {
		return nil
	}
	return &item.Reviews[len(item.Reviews)-1]
}

// LongTermReviewCount returns the number of reviews with delta_t > 0
func (item *FSRSItem) LongTermReviewCount() int {
	count := 0
	for _, review := range item.Reviews {
		if review.DeltaT > 0 {
			count++
		}
	}
	return count
}

// FirstLongTermReview returns the first review with delta_t > 0
func (item *FSRSItem) FirstLongTermReview() *FSRSReview {
	for i := range item.Reviews {
		if item.Reviews[i].DeltaT > 0 {
			return &item.Reviews[i]
		}
	}
	return nil
}

// RMatrixIndex returns binned indices for the R-matrix
// Used for evaluation and outlier detection
func (item *FSRSItem) RMatrixIndex() (deltaTBin, lengthBin, lapseBin uint32) {
	current := item.Current()
	if current == nil {
		return 0, 0, 0
	}

	deltaT := float64(current.DeltaT)
	if deltaT <= 0 {
		deltaT = 1 // Avoid log(0)
	}
	deltaTBin = uint32(math.Round(2.48 * math.Pow(3.62, math.Floor(math.Log(deltaT)/math.Log(3.62))) * 100.0))

	length := float64(item.LongTermReviewCount() + 1)
	if length <= 0 {
		length = 1
	}
	lengthBin = uint32(math.Round(1.99 * math.Pow(1.89, math.Floor(math.Log(length)/math.Log(1.89)))))

	// Count lapses (Again ratings with delta_t > 0)
	lapse := 0
	for _, review := range item.History() {
		if review.Rating == 1 && review.DeltaT > 0 {
			lapse++
		}
	}

	if lapse == 0 {
		return deltaTBin, lengthBin, 0
	}

	lapseBin = uint32(math.Round(1.65 * math.Pow(1.73, math.Floor(math.Log(float64(lapse))/math.Log(1.73)))))
	return deltaTBin, lengthBin, lapseBin
}

// Constants for parameter bounds
const (
	SMin      = 0.01
	SMax      = 36500.0
	DMin      = 1.0
	DMax      = 10.0
	InitSMax  = 100.0
	NumParams = 21
)

// FSRS5DefaultDecay is the default decay for FSRS v5
const FSRS5DefaultDecay = 0.5

// FSRS6DefaultDecay is the default decay for FSRS v6
const FSRS6DefaultDecay = 0.1542

// DefaultParameters are the FSRS v6 default weights
var DefaultParameters = [NumParams]float64{
	0.212, 1.2931, 2.3065, 8.2956, // W[0-3]: Initial stability for ratings 1-4
	6.4133, 0.8334, 3.0194, 0.001, // W[4-7]: Difficulty parameters
	1.8722, 0.1666, 0.796, 1.4835, // W[8-11]: Stability after success
	0.0614, 0.2629, 1.6483, 0.6014, // W[12-15]: Stability after failure
	1.8729, 0.5425, 0.0912, 0.0658, // W[16-19]: Short-term stability
	FSRS6DefaultDecay, // W[20]: Decay
}

// ParamsStdDev are the standard deviations for L2 regularization
var ParamsStdDev = [NumParams]float64{
	6.43, 9.66, 17.58, 27.85, 0.57, 0.28, 0.6, 0.12, 0.39, 0.18,
	0.33, 0.3, 0.09, 0.16, 0.57, 0.25, 1.03, 0.31, 0.32, 0.14, 0.27,
}

// TrainingConfig holds configuration for the training process
type TrainingConfig struct {
	NumEpochs              int
	BatchSize              int
	LearningRate           float64
	Gamma                  float64 // L2 regularization strength
	MaxSeqLen              int
	Seed                   int64
	FreezeInitialStability bool
	FreezeShortTerm        bool
	NumRelearningSteps     int
}

// DefaultTrainingConfig returns the default training configuration
func DefaultTrainingConfig() TrainingConfig {
	return TrainingConfig{
		NumEpochs:              5,
		BatchSize:              512,
		LearningRate:           0.04,
		Gamma:                  1.0,
		MaxSeqLen:              64,
		Seed:                   2023,
		FreezeInitialStability: false,
		FreezeShortTerm:        false,
		NumRelearningSteps:     1,
	}
}

// ComputeParametersInput holds input for ComputeParameters
type ComputeParametersInput struct {
	TrainSet           []FSRSItem
	EnableShortTerm    bool
	NumRelearningSteps int
	ProgressFunc       func(current, total int) // Optional progress callback
}

// ProgressState represents training progress
type ProgressState struct {
	Epoch          int
	EpochTotal     int
	ItemsProcessed int
	ItemsTotal     int
}

// Current returns the current progress as a single number
func (p *ProgressState) Current() int {
	if p.Epoch == 0 {
		return 0
	}
	return (p.Epoch-1)*p.ItemsTotal + p.ItemsProcessed
}

// Total returns the total work units
func (p *ProgressState) Total() int {
	return p.EpochTotal * p.ItemsTotal
}
