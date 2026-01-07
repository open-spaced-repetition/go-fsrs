package optimizer

import (
	"math"
)

// Model represents the FSRS model with trainable parameters
type Model struct {
	W [NumParams]float64
}

// NewModel creates a new model with the given parameters
func NewModel(params [NumParams]float64) *Model {
	return &Model{W: params}
}

// NewModelFromSlice creates a new model from a slice of parameters
func NewModelFromSlice(params []float64) *Model {
	m := &Model{}
	copy(m.W[:], params)
	return m
}

// NewDefaultModel creates a model with default parameters
func NewDefaultModel() *Model {
	return NewModel(DefaultParameters)
}

// Parameters returns the model parameters as a slice
func (m *Model) Parameters() []float64 {
	result := make([]float64, NumParams)
	copy(result, m.W[:])
	return result
}

// SetParameters sets all parameters from a slice
func (m *Model) SetParameters(params []float64) {
	copy(m.W[:], params)
}

// PowerForgettingCurve calculates retrievability R given time t and stability s
// R = (1 + factor * t/s)^decay
func (m *Model) PowerForgettingCurve(t, s float64) float64 {
	if s <= 0 {
		return 0
	}
	decay := -m.W[20]
	factor := math.Pow(0.9, 1.0/decay) - 1.0
	return math.Pow(1.0+factor*t/s, decay)
}

// NextInterval calculates the optimal interval for desired retention
func (m *Model) NextInterval(stability, desiredRetention float64) float64 {
	if stability <= 0 || desiredRetention <= 0 || desiredRetention >= 1 {
		return 1
	}
	decay := -m.W[20]
	factor := math.Pow(0.9, 1.0/decay) - 1.0
	return stability / factor * (math.Pow(desiredRetention, 1.0/decay) - 1.0)
}

// InitStability returns initial stability for a rating (1-4)
func (m *Model) InitStability(rating int) float64 {
	if rating < 1 || rating > 4 {
		return m.W[2] // Default to Good
	}
	return math.Max(m.W[rating-1], SMin)
}

// InitDifficulty returns initial difficulty for a rating (1-4)
// D0 = W[4] - exp(W[5] * (rating-1)) + 1
func (m *Model) InitDifficulty(rating int) float64 {
	d := m.W[4] - math.Exp(m.W[5]*float64(rating-1)) + 1
	return Clamp(d, DMin, DMax)
}

// LinearDamping applies linear damping to difficulty change
func (m *Model) LinearDamping(deltaD, oldD float64) float64 {
	return (10.0 - oldD) * deltaD / 9.0
}

// MeanReversion applies mean reversion to difficulty
func (m *Model) MeanReversion(initD, currentD float64) float64 {
	return m.W[7]*initD + (1-m.W[7])*currentD
}

// NextDifficulty calculates difficulty after a review
func (m *Model) NextDifficulty(d float64, rating int) float64 {
	deltaD := -m.W[6] * float64(rating-3)
	nextD := d + m.LinearDamping(deltaD, d)
	nextD = m.MeanReversion(m.InitDifficulty(4), nextD) // Mean revert to Easy difficulty
	return Clamp(nextD, DMin, DMax)
}

// StabilityAfterSuccess calculates new stability after successful recall (rating >= 2)
// S' = S * (1 + exp(W[8]) * (11-D) * S^(-W[9]) * (exp((1-R)*W[10])-1) * hardPenalty * easyBonus)
func (m *Model) StabilityAfterSuccess(s, d, r float64, rating int) float64 {
	hardPenalty := 1.0
	easyBonus := 1.0

	if rating == 2 { // Hard
		hardPenalty = m.W[15]
	} else if rating == 4 { // Easy
		easyBonus = m.W[16]
	}

	newS := s * (1 + math.Exp(m.W[8])*
		(11-d)*
		math.Pow(s, -m.W[9])*
		(math.Exp((1-r)*m.W[10])-1)*
		hardPenalty*
		easyBonus)

	return Clamp(newS, SMin, SMax)
}

// StabilityAfterFailure calculates new stability after forgetting (rating = 1)
// S' = W[11] * D^(-W[12]) * ((S+1)^W[13] - 1) * exp((1-R)*W[14])
func (m *Model) StabilityAfterFailure(s, d, r float64) float64 {
	newS := m.W[11] *
		math.Pow(d, -m.W[12]) *
		(math.Pow(s+1, m.W[13]) - 1) *
		math.Exp((1-r)*m.W[14])

	// Apply minimum bound based on short-term parameters
	sMin := s / math.Exp(m.W[17]*m.W[18])
	if newS < sMin {
		newS = sMin
	}

	return Clamp(newS, SMin, SMax)
}

// StabilityShortTerm calculates stability for same-day reviews (delta_t = 0)
// S' = S * exp(W[17] * (rating - 3 + W[18])) * S^(-W[19])
func (m *Model) StabilityShortTerm(s float64, rating int) float64 {
	sinc := math.Exp(m.W[17]*(float64(rating)-3+m.W[18])) * math.Pow(s, -m.W[19])

	// For rating >= 2, ensure stability doesn't decrease
	if rating >= 2 && sinc < 1.0 {
		sinc = 1.0
	}

	return Clamp(s*sinc, SMin, SMax)
}

// Step performs one step of the memory state update
func (m *Model) Step(deltaT float64, rating int, state MemoryState, isFirst bool) MemoryState {
	// Clamp current state
	lastS := Clamp(state.Stability, SMin, SMax)
	lastD := Clamp(state.Difficulty, DMin, DMax)

	var newS, newD float64

	if isFirst && state.Stability == 0 {
		// First review - initialize
		newS = m.InitStability(rating)
		newD = m.InitDifficulty(rating)
	} else {
		// Calculate retrievability
		r := m.PowerForgettingCurve(deltaT, lastS)

		// Calculate new stability based on rating and delta_t
		if deltaT == 0 {
			// Same-day review
			newS = m.StabilityShortTerm(lastS, rating)
		} else if rating == 1 {
			// Forgot
			newS = m.StabilityAfterFailure(lastS, lastD, r)
		} else {
			// Recalled
			newS = m.StabilityAfterSuccess(lastS, lastD, r, rating)
		}

		// Calculate new difficulty
		newD = m.NextDifficulty(lastD, rating)
	}

	return MemoryState{
		Stability:  Clamp(newS, SMin, SMax),
		Difficulty: Clamp(newD, DMin, DMax),
	}
}

// Forward processes a complete review sequence and returns the final memory state
func (m *Model) Forward(item FSRSItem) MemoryState {
	state := MemoryState{Stability: 0, Difficulty: 0}

	for i, review := range item.Reviews {
		isFirst := i == 0
		state = m.Step(float64(review.DeltaT), int(review.Rating), state, isFirst)
	}

	return state
}

// ForwardBatch processes multiple items and returns their final states
func (m *Model) ForwardBatch(items []WeightedFSRSItem) []MemoryState {
	states := make([]MemoryState, len(items))
	for i, item := range items {
		states[i] = m.Forward(item.Item)
	}
	return states
}

// PredictRetrievability predicts the retrievability for each item
// Returns the predicted retrievability at the time of the last review
func (m *Model) PredictRetrievability(item FSRSItem) float64 {
	if len(item.Reviews) < 2 {
		return 0
	}

	// Get state before the last review
	history := item.History()
	state := MemoryState{Stability: 0, Difficulty: 0}

	for i, review := range history {
		isFirst := i == 0
		state = m.Step(float64(review.DeltaT), int(review.Rating), state, isFirst)
	}

	// Calculate retrievability at the time of the current review
	current := item.Current()
	if current == nil || state.Stability == 0 {
		return 0
	}

	return m.PowerForgettingCurve(float64(current.DeltaT), state.Stability)
}

// PredictRetrievabilityBatch predicts retrievability for multiple items
func (m *Model) PredictRetrievabilityBatch(items []WeightedFSRSItem) []float64 {
	predictions := make([]float64, len(items))
	for i, item := range items {
		predictions[i] = m.PredictRetrievability(item.Item)
	}
	return predictions
}

// ClipParameters clips all parameters to valid ranges
// Bounds follow the fsrs-rs reference implementation
func ClipParameters(w []float64, numRelearningSteps int, enableShortTerm bool) []float64 {
	result := make([]float64, len(w))
	copy(result, w)

	if len(result) < NumParams {
		return result
	}

	// S0 bounds (W[0-3])
	for i := 0; i < 4; i++ {
		result[i] = Clamp(result[i], SMin, InitSMax)
	}

	// Ensure S0 ordering: S0[1] <= S0[2] <= S0[3] <= S0[4]
	for i := 1; i < 4; i++ {
		if result[i] < result[i-1] {
			result[i] = result[i-1]
		}
	}

	// D0 (W[4])
	result[4] = Clamp(result[4], 1.0, 10.0)

	// W[5]: difficulty slope
	result[5] = Clamp(result[5], 0.001, 4.0)

	// W[6]: difficulty change
	result[6] = Clamp(result[6], 0.001, 4.0)

	// W[7]: mean reversion
	result[7] = Clamp(result[7], 0.001, 0.75)

	// W[8]: stability growth base
	result[8] = Clamp(result[8], 0.0, 4.5)

	// W[9]: stability growth exponent
	result[9] = Clamp(result[9], 0.0, 0.8)

	// W[10]: retrievability factor
	result[10] = Clamp(result[10], 0.001, 3.5)

	// W[11]: forget stability base
	result[11] = Clamp(result[11], 0.001, 5.0)

	// W[12]: forget difficulty factor
	result[12] = Clamp(result[12], 0.001, 0.25)

	// W[13]: forget stability exponent
	result[13] = Clamp(result[13], 0.001, 0.9)

	// W[14]: forget retrievability factor
	result[14] = Clamp(result[14], 0.0, 4.0)

	// W[15]: hard penalty
	result[15] = Clamp(result[15], 0.0, 1.0)

	// W[16]: easy bonus
	result[16] = Clamp(result[16], 1.0, 6.0)

	// Short-term parameters (W[17-19])
	if enableShortTerm {
		result[17] = Clamp(result[17], 0.0, 2.0)
		result[18] = Clamp(result[18], 0.0, 2.0)
		result[19] = Clamp(result[19], 0.0, 1.0)
	} else {
		result[17] = 0
		result[18] = 0
		result[19] = 0
	}

	// W[20]: decay
	result[20] = Clamp(result[20], 0.1, 0.8)

	return result
}
