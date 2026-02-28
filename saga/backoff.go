package saga

import (
	"math"
	"time"
)

// ExponentialBackoff provides exponential backoff strategy without jitter.
type ExponentialBackoff struct{}

// NewExponentialBackoff creates a new exponential backoff strategy instance.
func NewExponentialBackoff() ExponentialBackoff {
	return ExponentialBackoff{}
}

// Backoff calculates exponential backoff delay.
func (o ExponentialBackoff) Backoff(attempt uint32, delay time.Duration) time.Duration {
	backoffTime := delay * time.Duration(math.Pow(2, float64(attempt)))
	return backoffTime
}
