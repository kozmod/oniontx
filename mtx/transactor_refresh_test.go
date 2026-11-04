package mtx

import (
	"context"
	"fmt"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool/require"
)

//nolint:dupl
func Test_Transactor_WithTxRefreshFromContext(t *testing.T) {
	const panicMessage = "panic identity"
	t.Run("disabled_by_default", func(t *testing.T) {
		var (
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			o.replacement = &replacement
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 1, originalCommits)
		require.Equal(t, 0, originalRollbacks)
		require.Equal(t, 0, replacementCommits)
		require.Equal(t, 0, replacementRollbacks)
		require.Equal(t, 1, beginCalled)
		require.Equal(t, 1, o.extracts)
	})

	t.Run("explicitly_disabled", func(t *testing.T) {
		var (
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o).WithTxRefreshFromContext(false)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			o.replacement = &replacement
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 1, originalCommits)
		require.Equal(t, 0, originalRollbacks)
		require.Equal(t, 0, replacementCommits)
		require.Equal(t, 0, replacementRollbacks)
		require.Equal(t, 1, beginCalled)
		require.Equal(t, 1, o.extracts)
	})

	t.Run("replacement_commit", func(t *testing.T) {
		var (
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o).WithTxRefreshFromContext(true)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			o.replacement = &replacement
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 0, originalCommits)
		require.Equal(t, 0, originalRollbacks)
		require.Equal(t, 1, replacementCommits)
		require.Equal(t, 0, replacementRollbacks)
		require.Equal(t, 1, beginCalled)
		require.Equal(t, 2, o.extracts)
	})

	t.Run("replacement_rollback", func(t *testing.T) {
		var (
			expError                                 = fmt.Errorf("callback failed")
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o).WithTxRefreshFromContext(true)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			o.replacement = &replacement
			return expError
		})
		require.ErrorIs(t, err, expError)
		require.ErrorIs(t, err, ErrTxRolledBack)
		require.Equal(t, 0, originalCommits)
		require.Equal(t, 0, originalRollbacks)
		require.Equal(t, 0, replacementCommits)
		require.Equal(t, 1, replacementRollbacks)
		require.Equal(t, 1, beginCalled)
		require.Equal(t, 2, o.extracts)
	})

	t.Run("missing_replacement_commit", func(t *testing.T) {
		var (
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o).WithTxRefreshFromContext(true)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			o.replacement = &replacement
			o.missing = true
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 1, originalCommits)
		require.Equal(t, 0, originalRollbacks)
		require.Equal(t, 0, replacementCommits)
		require.Equal(t, 0, replacementRollbacks)
		require.Equal(t, 1, beginCalled)
		require.Equal(t, 2, o.extracts)
	})

	t.Run("missing_replacement_rollback", func(t *testing.T) {
		var (
			expError                                 = fmt.Errorf("callback failed")
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o).WithTxRefreshFromContext(true)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			o.replacement = &replacement
			o.missing = true
			return expError
		})
		require.ErrorIs(t, err, expError)
		require.ErrorIs(t, err, ErrTxRolledBack)
		require.Equal(t, 0, originalCommits)
		require.Equal(t, 1, originalRollbacks)
		require.Equal(t, 0, replacementCommits)
		require.Equal(t, 0, replacementRollbacks)
		require.Equal(t, 1, beginCalled)
		require.Equal(t, 2, o.extracts)
	})

	t.Run("nested_success", func(t *testing.T) {
		var (
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o).WithTxRefreshFromContext(true)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				o.replacement = &replacement
				return nil
			})
			require.Equal(t, 0, originalCommits+originalRollbacks)
			require.Equal(t, 0, replacementCommits+replacementRollbacks)
			require.Equal(t, 2, o.extracts)
			return err
		})
		require.NoError(t, err)
		require.Equal(t, 0, originalCommits)
		require.Equal(t, 0, originalRollbacks)
		require.Equal(t, 1, replacementCommits)
		require.Equal(t, 0, replacementRollbacks)
		require.Equal(t, 1, beginCalled)
		require.Equal(t, 3, o.extracts)
	})

	t.Run("nested_error", func(t *testing.T) {
		var (
			expError                                 = fmt.Errorf("callback failed")
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o).WithTxRefreshFromContext(true)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				o.replacement = &replacement
				return expError
			})
			require.Equal(t, 0, originalCommits+originalRollbacks)
			require.Equal(t, 0, replacementCommits+replacementRollbacks)
			require.Equal(t, 2, o.extracts)
			return err
		})
		require.ErrorIs(t, err, expError)
		require.ErrorIs(t, err, ErrTxRolledBack)
		require.Equal(t, 0, originalCommits)
		require.Equal(t, 0, originalRollbacks)
		require.Equal(t, 0, replacementCommits)
		require.Equal(t, 1, replacementRollbacks)
		require.Equal(t, 1, beginCalled)
		require.Equal(t, 3, o.extracts)
	})

	t.Run("panic_recovered", func(t *testing.T) {
		var (
			panicValue                               = &struct{ message string }{message: panicMessage}
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o).WithTxRefreshFromContext(true).WithPanicRecovery(true)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			o.replacement = &replacement
			panic(panicValue)
		})
		require.ErrorIs(t, err, ErrPanicRecovered)
		require.ErrorIs(t, err, ErrTxRolledBack)
		require.Equal(t, 0, originalCommits)
		require.Equal(t, 0, originalRollbacks)
		require.Equal(t, 0, replacementCommits)
		require.Equal(t, 1, replacementRollbacks)
		require.Equal(t, 1, beginCalled)
		require.Equal(t, 2, o.extracts)
	})

	t.Run("panic_propagated", func(t *testing.T) {
		var (
			panicValue                               = &struct{ message string }{message: panicMessage}
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o).WithTxRefreshFromContext(true)
		)
		defer func() {
			require.True(t, recover() == panicValue)
			require.Equal(t, 0, originalCommits+originalRollbacks)
			require.Equal(t, 0, replacementCommits)
			require.Equal(t, 1, replacementRollbacks)
			require.Equal(t, 1, beginCalled)
			require.Equal(t, 2, o.extracts)
		}()
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			o.replacement = &replacement
			panic(panicValue)
		})
		t.Fatalf("expected panic, got error: %v", err)
	})

	t.Run("replacement_commit_error", func(t *testing.T) {
		var (
			finalizeError                            = fmt.Errorf("finalization failed")
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return finalizeError
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o).WithTxRefreshFromContext(true)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			o.replacement = &replacement
			return nil
		})
		require.ErrorIs(t, err, finalizeError)
		require.ErrorIs(t, err, ErrCommitFailed)
		require.Equal(t, 0, originalCommits)
		require.Equal(t, 0, originalRollbacks)
		require.Equal(t, 1, replacementCommits)
		require.Equal(t, 0, replacementRollbacks)
		require.Equal(t, 1, beginCalled)
		require.Equal(t, 2, o.extracts)
	})

	t.Run("replacement_rollback_error", func(t *testing.T) {
		var (
			expError                                 = fmt.Errorf("callback failed")
			finalizeError                            = fmt.Errorf("finalization failed")
			ctx                                      = context.Background()
			originalCommits, originalRollbacks       int
			replacementCommits, replacementRollbacks int
			beginCalled                              int
			original                                 = committerMock{
				commitFn: func(ctx context.Context) error {
					originalCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					originalRollbacks++
					return nil
				},
			}
			replacement = committerMock{
				commitFn: func(ctx context.Context) error {
					replacementCommits++
					return nil
				},
				rollbackFn: func(ctx context.Context) error {
					replacementRollbacks++
					return finalizeError
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginCalled++
					return &original, nil
				},
			}
			o = refreshOperatorMock{
				ContextOperator: NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b),
			}
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, &o).WithTxRefreshFromContext(true)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			o.replacement = &replacement
			return expError
		})
		require.ErrorIs(t, err, expError)
		require.ErrorIs(t, err, finalizeError)
		require.ErrorIs(t, err, ErrRollbackFailed)
		require.Equal(t, 0, originalCommits)
		require.Equal(t, 0, originalRollbacks)
		require.Equal(t, 0, replacementCommits)
		require.Equal(t, 1, replacementRollbacks)
		require.Equal(t, 1, beginCalled)
		require.Equal(t, 2, o.extracts)
	})

	t.Run("original_transactor_is_not_modified", func(t *testing.T) {
		var (
			base = NewTransactor[*beginnerMock[*committerMock], *committerMock](nil, nil).
				WithPanicRecovery(true).
				WithRollbackCtxFactory(context.WithoutCancel)
			enabled     = base.WithTxRefreshFromContext(true)
			disabled    = enabled.WithTxRefreshFromContext(false)
			ctx, cancel = context.WithCancel(context.Background())
		)
		cancel()
		require.True(t, base != enabled && enabled != disabled && base != disabled)
		require.False(t, base.opts.refreshTxFromContext)
		require.True(t, enabled.opts.refreshTxFromContext)
		require.False(t, disabled.opts.refreshTxFromContext)
		require.True(t, base.opts.recoverPanic)
		require.True(t, enabled.opts.recoverPanic)
		require.True(t, disabled.opts.recoverPanic)
		require.NoError(t, base.opts.rollbackCtxFactory(ctx).Err())
		require.NoError(t, enabled.opts.rollbackCtxFactory(ctx).Err())
		require.NoError(t, disabled.opts.rollbackCtxFactory(ctx).Err())
	})
}
