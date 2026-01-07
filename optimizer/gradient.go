package optimizer

import (
	"math"
	"sync"
)

// NumericalGradient computes the gradient using central differences
// grad[i] = (f(w + eps*e_i) - f(w - eps*e_i)) / (2 * eps)
func NumericalGradient(f LossFunction, params []float64, epsilon float64) []float64 {
	if epsilon <= 0 {
		epsilon = 1e-7
	}

	n := len(params)
	grad := make([]float64, n)

	// Create a copy of params to avoid modifying the original
	paramsCopy := make([]float64, n)
	copy(paramsCopy, params)

	for i := 0; i < n; i++ {
		// Forward step
		paramsCopy[i] = params[i] + epsilon
		fPlus := f(paramsCopy)

		// Backward step
		paramsCopy[i] = params[i] - epsilon
		fMinus := f(paramsCopy)

		// Restore original value
		paramsCopy[i] = params[i]

		// Central difference
		grad[i] = (fPlus - fMinus) / (2 * epsilon)

		// Handle NaN or Inf
		if math.IsNaN(grad[i]) || math.IsInf(grad[i], 0) {
			grad[i] = 0
		}
	}

	return grad
}

// NumericalGradientParallel computes the gradient in parallel
func NumericalGradientParallel(f LossFunction, params []float64, epsilon float64, numWorkers int) []float64 {
	if epsilon <= 0 {
		epsilon = 1e-7
	}
	if numWorkers <= 0 {
		numWorkers = 4
	}

	n := len(params)
	grad := make([]float64, n)

	// Use a worker pool
	var wg sync.WaitGroup
	work := make(chan int, n)

	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Each worker has its own copy of params
			localParams := make([]float64, n)

			for i := range work {
				copy(localParams, params)

				// Forward step
				localParams[i] = params[i] + epsilon
				fPlus := f(localParams)

				// Backward step
				localParams[i] = params[i] - epsilon
				fMinus := f(localParams)

				// Central difference
				g := (fPlus - fMinus) / (2 * epsilon)

				// Handle NaN or Inf
				if math.IsNaN(g) || math.IsInf(g, 0) {
					g = 0
				}

				grad[i] = g
			}
		}()
	}

	// Send work
	for i := 0; i < n; i++ {
		work <- i
	}
	close(work)

	// Wait for completion
	wg.Wait()

	return grad
}

// GradientClipping clips gradients to prevent exploding gradients
func GradientClipping(grad []float64, maxNorm float64) []float64 {
	if maxNorm <= 0 {
		return grad
	}

	// Compute L2 norm
	var norm float64
	for _, g := range grad {
		norm += g * g
	}
	norm = math.Sqrt(norm)

	if norm > maxNorm {
		scale := maxNorm / norm
		result := make([]float64, len(grad))
		for i, g := range grad {
			result[i] = g * scale
		}
		return result
	}

	return grad
}

// FreezeGradients sets specified gradient indices to zero
func FreezeGradients(grad []float64, indices []int) []float64 {
	result := make([]float64, len(grad))
	copy(result, grad)

	for _, idx := range indices {
		if idx >= 0 && idx < len(result) {
			result[idx] = 0
		}
	}

	return result
}

// FreezeInitialStability zeros out gradients for W[0:4]
func FreezeInitialStability(grad []float64) []float64 {
	return FreezeGradients(grad, []int{0, 1, 2, 3})
}

// FreezeShortTermStability zeros out gradients for W[17:20]
func FreezeShortTermStability(grad []float64) []float64 {
	return FreezeGradients(grad, []int{17, 18, 19})
}

// GradientDescent performs a single gradient descent step
func GradientDescent(params, grad []float64, lr float64) []float64 {
	result := make([]float64, len(params))
	for i := range params {
		result[i] = params[i] - lr*grad[i]
	}
	return result
}
