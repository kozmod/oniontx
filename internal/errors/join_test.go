package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool/require"
)

func Test_Errors_Join(t *testing.T) {
	const (
		nilStr = "<nil>"
	)
	var (
		errA = fmt.Errorf("errA")
		errB = fmt.Errorf("errB")
		errC = fmt.Errorf("errC")
	)
	t.Run("compare_with_errors_package", func(t *testing.T) {
		t.Run("errors.Is", func(t *testing.T) {
			assertFn := func(err error) {
				require.ErrorIs(t, err, errA)
				require.ErrorIs(t, err, errB)
				require.ErrorIs(t, err, errC)
			}

			err := errors.Join(errA, errB, errC)
			assertFn(err)

			customErr := Join(errA, errB, errC)
			assertFn(customErr)
		})

		t.Run("single_error_equal", func(t *testing.T) {
			require.True(t, errors.Join(errA).Error() == Join(errA).Error())
			require.Equal(t,
				fmt.Errorf("%w", errors.Join(errA)).Error(),
				fmt.Errorf("%w", Join(errA)).Error(),
			)
		})

		t.Run("nil", func(t *testing.T) {
			assertFn := func(err error) {
				require.True(t, strings.Contains(err.Error(), errA.Error()))
				require.True(t, strings.Contains(err.Error(), errC.Error()))
				require.False(t, strings.Contains(err.Error(), errB.Error()))
				require.False(t, strings.Contains(err.Error(), nilStr))
			}
			assertFn(errors.Join(nil, errA, nil, errC))
			assertFn(Join(nil, errA, nil, errC))

			assertFn = func(err error) {
				require.True(t, strings.Contains(fmt.Sprintf("%v", err), nilStr))
			}
			assertFn(errors.Join(nil))
			assertFn(Join(nil))
			assertFn(errors.Join())
			assertFn(Join())
		})
	})
	t.Run("equal", func(t *testing.T) {
		var (
			err    = Join(errA, errB, errC)
			expErr = fmt.Errorf("errA: errB: errC")
		)

		require.Equal(t, expErr.Error(), err.Error())
		require.Equal(t,
			fmt.Sprintf("%v", expErr),
			fmt.Sprintf("%v", err),
		)
		require.Equal(t,
			fmt.Sprintf("%+v", expErr),
			fmt.Sprintf("%+v", err),
		)
	})
	t.Run("wrap", func(t *testing.T) {
		var (
			err = Join(errA, errB, errC)
		)
		require.Equal(t,
			fmt.Errorf("failed: %w", err).Error(),
			fmt.Sprintf("failed: %v", err),
		)
	})
}
