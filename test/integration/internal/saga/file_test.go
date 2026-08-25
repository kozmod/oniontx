package saga

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool"
	"github.com/kozmod/oniontx/saga"
	"github.com/stretchr/testify/require"
)

func Test_file_creation(t *testing.T) {
	var (
		name = "saga.txt"
		data = []byte("A")
	)
	t.Run("success", func(t *testing.T) {
		var (
			dir  = t.TempDir()
			path = filepath.Join(dir, name)
		)
		res, err := saga.NewSaga([]saga.Step{
			saga.NewStep("step_create_file").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					err := os.WriteFile(path, data, 0o644)
					require.NoError(t, err)
					return err
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, track saga.Track) error {
					require.Fail(t, "should not be called")
					return nil
				})),
			saga.NewStep("check_file").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					require.FileExists(t, path)

					info, err := os.Stat(path)
					require.NoError(t, err)
					require.Equal(t, int64(len(data)), info.Size())
					require.Equal(t, name, info.Name())
					return err
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, track saga.Track) error {
					require.Fail(t, "should not be called")
					return nil
				})),
		}).Execute(context.Background())

		require.NoError(t, err)
		require.Equal(t, saga.StageResultSuccess, res.Status)

		testtool.TestFn(t, func() {
			printResult(t, res, err)
		})
	})
	t.Run("compensate", func(t *testing.T) {
		var (
			dir  = t.TempDir()
			path = filepath.Join(dir, name)

			expError = fmt.Errorf("some_error")
		)
		res, err := saga.NewSaga([]saga.Step{
			saga.NewStep("step_create_file").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					err := os.WriteFile(path, data, 0o644)
					require.NoError(t, err)
					return err
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, track saga.Track) error {
					err := os.Remove(path)
					require.NoError(t, err)
					return nil
				})),
			saga.NewStep("check_file_compensation_delete").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					require.FileExists(t, path)
					return expError
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, track saga.Track) error {
					require.Fail(t, "should not be called")
					return nil
				})),
		}).Execute(context.Background())

		require.Error(t, err)
		require.Equal(t, saga.StageResultCompensated, res.Status)
		require.NoFileExists(t, path)

		testtool.TestFn(t, func() {
			printResult(t, res, err)
		})
	})
}
