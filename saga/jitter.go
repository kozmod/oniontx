package saga

import (
	"crypto/rand"
	"math/big"
	"time"
)

// FullJitter provides full jitter strategy for retry delays.
type FullJitter struct{}

// NewFullJitter creates a new full jitter strategy instance.
func NewFullJitter() FullJitter {
	return FullJitter{}
}

// Jitter applies full jitter to the base delay.
func (o FullJitter) Jitter(base time.Duration) time.Duration {
	jitter := randomDuration(base)
	return jitter
}

// EqualJitter provides equal jitter strategy for retry delays.
type EqualJitter struct{}

// NewEqualJitter creates a new equal jitter strategy instance.
func NewEqualJitter() EqualJitter {
	return EqualJitter{}
}

// Jitter applies equal jitter to the base delay.
func (o EqualJitter) Jitter(base time.Duration) time.Duration {
	var (
		temp   = base / 2
		jitter = temp + randomDuration(temp)
	)
	return jitter
}

func randomDuration(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return max
	}

	return time.Duration(n.Int64())
}
