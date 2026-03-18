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
