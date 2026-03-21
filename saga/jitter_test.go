package saga

import (
	"testing"
	"time"

	"github.com/kozmod/oniontx/internal/testtool/assert"
)

func Test_retry_jitter(t *testing.T) {
	t.Run("equal_jitter", func(t *testing.T) {
		var (
			jitter = NewEqualJitter()
			base   = 10 * time.Second
		)

		delay := jitter.Jitter(base)
		assert.True(t, delay < base)
	})
	t.Run("full_jitter", func(t *testing.T) {
		var (
			jitter = NewFullJitter()
			base   = 10 * time.Second
		)

		delay := jitter.Jitter(base)
		assert.True(t, delay < base)
	})
}

func Test_jitter(t *testing.T) {
	t.Run("full_jitter_v1", func(t *testing.T) {
		var (
			baseTime   = 10 * time.Nanosecond
			fullJitter = NewFullJitter()
		)
		jitter := fullJitter.Jitter(baseTime)
		if jitter > baseTime {
			t.Fatalf("jitter is greater than base time[jitter: %v, base_time: %v]", jitter, baseTime)
		}
		if jitter < 0 {
			t.Fatalf("jitter is less than zero[jitter: %v]", jitter)
		}
	})
}
