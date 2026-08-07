package saga

import (
	"math"
	"time"
)

const maxDuration = math.MaxInt64

// ExponentialBackoff provides exponential backoff strategy without jitter.
type ExponentialBackoff struct{}

// NewExponentialBackoff creates a new exponential backoff strategy instance.
func NewExponentialBackoff() ExponentialBackoff {
	return ExponentialBackoff{}
}

// Backoff calculates exponential backoff delay.
//
// The result saturates at the largest representable [time.Duration] instead of
// overflowing into a negative duration.
func (ExponentialBackoff) Backoff(attempt uint32, delay time.Duration) time.Duration {
	if delay <= 0 {
		return delay
	}

	for range attempt {
		if delay > maxDuration/2 {
			return maxDuration
		}
		delay *= 2
	}

	return delay
}
