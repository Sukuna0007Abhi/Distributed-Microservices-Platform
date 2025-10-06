package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateHalfOpen
	StateOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	settings        CircuitBreakerSettings
}

// CircuitBreakerSettings defines circuit breaker configuration
type CircuitBreakerSettings struct {
	MaxFailures     int           // Maximum failures before opening circuit
	ResetTimeout    time.Duration // Time to wait before attempting reset
	SuccessThreshold int          // Successful calls needed to close circuit
	Timeout         time.Duration // Request timeout
}

// DefaultSettings returns default circuit breaker settings
func DefaultSettings() CircuitBreakerSettings {
	return CircuitBreakerSettings{
		MaxFailures:      5,
		ResetTimeout:     60 * time.Second,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(settings CircuitBreakerSettings) *CircuitBreaker {
	return &CircuitBreaker{
		state:    StateClosed,
		settings: settings,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	state := cb.getState()

	switch state {
	case StateOpen:
		return errors.New("circuit breaker is open")
	case StateHalfOpen:
		return cb.executeHalfOpen(ctx, fn)
	default:
		return cb.executeClosed(ctx, fn)
	}
}

// getState returns the current state of the circuit breaker
func (cb *CircuitBreaker) getState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.settings.ResetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			if cb.state == StateOpen && time.Since(cb.lastFailureTime) > cb.settings.ResetTimeout {
				cb.state = StateHalfOpen
				cb.successCount = 0
			}
			cb.mu.Unlock()
			cb.mu.RLock()
		}
	}

	return cb.state
}

// executeClosed executes function when circuit is closed
func (cb *CircuitBreaker) executeClosed(ctx context.Context, fn func() error) error {
	err := cb.executeWithTimeout(ctx, fn)
	
	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// executeHalfOpen executes function when circuit is half-open
func (cb *CircuitBreaker) executeHalfOpen(ctx context.Context, fn func() error) error {
	err := cb.executeWithTimeout(ctx, fn)
	
	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// executeWithTimeout executes function with timeout
func (cb *CircuitBreaker) executeWithTimeout(ctx context.Context, fn func() error) error {
	ctx, cancel := context.WithTimeout(ctx, cb.settings.Timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// onSuccess handles successful execution
func (cb *CircuitBreaker) onSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successCount++

	if cb.state == StateHalfOpen && cb.successCount >= cb.settings.SuccessThreshold {
		cb.state = StateClosed
		cb.failureCount = 0
		cb.successCount = 0
	} else if cb.state == StateClosed {
		cb.failureCount = 0
	}
}

// onFailure handles failed execution
func (cb *CircuitBreaker) onFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.settings.MaxFailures {
		cb.state = StateOpen
	}
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"state":         cb.state.String(),
		"failure_count": cb.failureCount,
		"success_count": cb.successCount,
		"last_failure":  cb.lastFailureTime,
	}
}

// String returns string representation of circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateHalfOpen:
		return "HALF_OPEN"
	case StateOpen:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

// Retry implements exponential backoff retry logic
type Retry struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
	Jitter      bool
}

// DefaultRetry returns default retry configuration
func DefaultRetry() Retry {
	return Retry{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
	}
}

// Execute executes a function with retry logic
func (r *Retry) Execute(ctx context.Context, fn func() error) error {
	var lastErr error
	
	for attempt := 0; attempt <= r.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := r.calculateDelay(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := fn(); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return lastErr
}

// calculateDelay calculates the delay for the given attempt
func (r *Retry) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(r.BaseDelay) * pow(r.Multiplier, float64(attempt-1)))
	
	if delay > r.MaxDelay {
		delay = r.MaxDelay
	}

	if r.Jitter {
		delay = time.Duration(float64(delay) * (0.5 + rand.Float64()*0.5))
	}

	return delay
}

// pow calculates base^exp for float64
func pow(base, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}