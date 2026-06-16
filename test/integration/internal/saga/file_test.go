package saga

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool"
	"github.com/kozmod/oniontx/saga"
	"github.com/stretchr/testify/assert"
)

func Test_file_creation(t *testing.T) {
	var (
		dir  = t.TempDir()
		name = "saga.txt"
		path = filepath.Join(dir, name)
		data = []byte("A")
	)
	t.Run("success", func(t *testing.T) {
		res, err := saga.NewSaga([]saga.Step{
			saga.NewStep("step_create_file").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					err := os.WriteFile(path, data, 0o644)
					assert.NoError(t, err)
					return err
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, track saga.Track) error {
					assert.Fail(t, "should not be called")
					return nil
				})),
			saga.NewStep("check_file").
				WithAction(saga.NewOperation(func(ctx context.Context, _ saga.Track) error {
					assert.FileExists(t, path)

					info, err := os.Stat(path)
					assert.NoError(t, err)
					assert.Equal(t, int64(len(data)), info.Size())
					assert.Equal(t, name, info.Name())
					return err
				})).
				WithCompensation(saga.NewOperation(func(ctx context.Context, track saga.Track) error {
					assert.Fail(t, "should not be called")
					return nil
				})),
		}).Execute(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, saga.StageResultSuccess, res.Status)

		testtool.TestFn(t, func() {
			printResult(t, res, err)
		})
	})
}
