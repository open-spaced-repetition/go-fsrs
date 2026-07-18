package optimizer

import (
	"math"
)

// Adam implements the Adam optimizer
// Reference: https://arxiv.org/abs/1412.6980
type Adam struct {
	// LearningRate is the step size (default: 0.04 for FSRS)
	LearningRate float64

	// Beta1 is the exponential decay rate for the first moment estimates (default: 0.9)
	Beta1 float64

	// Beta2 is the exponential decay rate for the second moment estimates (default: 0.999)
	Beta2 float64

	// Epsilon is a small constant for numerical stability (default: 1e-8)
	Epsilon float64

	// M is the first moment vector (mean of gradients)
	M []float64

	// V is the second moment vector (uncentered variance of gradients)
	V []float64

	// T is the timestep (number of updates performed)
	T int
}

// NewAdam creates a new Adam optimizer with default hyperparameters
func NewAdam(learningRate float64, paramCount int) *Adam {
	return &Adam{
		LearningRate: learningRate,
		Beta1:        0.9,
		Beta2:        0.999,
		Epsilon:      1e-8,
		M:            make([]float64, paramCount),
		V:            make([]float64, paramCount),
		T:            0,
	}
}

// NewAdamWithConfig creates a new Adam optimizer with custom hyperparameters
func NewAdamWithConfig(learningRate, beta1, beta2, epsilon float64, paramCount int) *Adam {
	return &Adam{
		LearningRate: learningRate,
		Beta1:        beta1,
		Beta2:        beta2,
		Epsilon:      epsilon,
		M:            make([]float64, paramCount),
		V:            make([]float64, paramCount),
		T:            0,
	}
}

// Step performs one optimization step
// Returns the updated parameters
func (a *Adam) Step(grad, params []float64) []float64 {
	if len(grad) != len(params) || len(grad) != len(a.M) {
		return params
	}

	a.T++
	result := make([]float64, len(params))

	// Bias correction factors
	bc1 := 1.0 - math.Pow(a.Beta1, float64(a.T))
	bc2 := 1.0 - math.Pow(a.Beta2, float64(a.T))

	for i := range params {
		// Update biased first moment estimate
		a.M[i] = a.Beta1*a.M[i] + (1-a.Beta1)*grad[i]

		// Update biased second raw moment estimate
		a.V[i] = a.Beta2*a.V[i] + (1-a.Beta2)*grad[i]*grad[i]

		// Compute bias-corrected first moment estimate
		mHat := a.M[i] / bc1

		// Compute bias-corrected second raw moment estimate
		vHat := a.V[i] / bc2

		// Update parameters
		result[i] = params[i] - a.LearningRate*mHat/(math.Sqrt(vHat)+a.Epsilon)
	}

	return result
}

// StepWithLR performs one optimization step with a custom learning rate
func (a *Adam) StepWithLR(grad, params []float64, lr float64) []float64 {
	originalLR := a.LearningRate
	a.LearningRate = lr
	result := a.Step(grad, params)
	a.LearningRate = originalLR
	return result
}

// Reset resets the optimizer state
func (a *Adam) Reset() {
	a.T = 0
	for i := range a.M {
		a.M[i] = 0
		a.V[i] = 0
	}
}

// CosineAnnealingLR implements cosine annealing learning rate scheduler
type CosineAnnealingLR struct {
	// InitialLR is the initial learning rate
	InitialLR float64

	// TotalSteps is the total number of steps
	TotalSteps int

	// MinLR is the minimum learning rate (default: 0)
	MinLR float64

	// CurrentStep is the current step
	CurrentStep int
}

// NewCosineAnnealingLR creates a new cosine annealing scheduler
func NewCosineAnnealingLR(totalSteps int, initialLR float64) *CosineAnnealingLR {
	return &CosineAnnealingLR{
		InitialLR:   initialLR,
		TotalSteps:  totalSteps,
		MinLR:       0,
		CurrentStep: 0,
	}
}

// Step advances the scheduler and returns the current learning rate
func (s *CosineAnnealingLR) Step() float64 {
	s.CurrentStep++

	if s.CurrentStep >= s.TotalSteps {
		return s.MinLR
	}

	// Cosine annealing formula
	// lr = min_lr + 0.5 * (max_lr - min_lr) * (1 + cos(pi * t / T))
	cosValue := math.Cos(math.Pi * float64(s.CurrentStep) / float64(s.TotalSteps))
	lr := s.MinLR + 0.5*(s.InitialLR-s.MinLR)*(1+cosValue)

	return lr
}

// CurrentLR returns the current learning rate without advancing
func (s *CosineAnnealingLR) CurrentLR() float64 {
	if s.CurrentStep >= s.TotalSteps {
		return s.MinLR
	}

	cosValue := math.Cos(math.Pi * float64(s.CurrentStep) / float64(s.TotalSteps))
	return s.MinLR + 0.5*(s.InitialLR-s.MinLR)*(1+cosValue)
}

// Reset resets the scheduler
func (s *CosineAnnealingLR) Reset() {
	s.CurrentStep = 0
}

// SGD implements Stochastic Gradient Descent optimizer
type SGD struct {
	LearningRate float64
	Momentum     float64
	Velocity     []float64
}

// NewSGD creates a new SGD optimizer
func NewSGD(learningRate float64, momentum float64, paramCount int) *SGD {
	return &SGD{
		LearningRate: learningRate,
		Momentum:     momentum,
		Velocity:     make([]float64, paramCount),
	}
}

// Step performs one SGD step with momentum
func (s *SGD) Step(grad, params []float64) []float64 {
	if len(grad) != len(params) || len(grad) != len(s.Velocity) {
		return params
	}

	result := make([]float64, len(params))

	for i := range params {
		// Update velocity with momentum
		s.Velocity[i] = s.Momentum*s.Velocity[i] + grad[i]

		// Update parameters
		result[i] = params[i] - s.LearningRate*s.Velocity[i]
	}

	return result
}

// Reset resets the optimizer state
func (s *SGD) Reset() {
	for i := range s.Velocity {
		s.Velocity[i] = 0
	}
}
