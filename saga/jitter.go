package saga

import (
	"math/rand"
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
	var (
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	)
	return time.Duration(r.Int63n(int64(base)))
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
		r    = rand.New(rand.NewSource(time.Now().UnixNano()))
		temp = base / 2
	)
	if temp <= 0 {
		return 0
	}
	jitter := temp + time.Duration(r.Int63n(int64(temp)))
	return jitter
}
