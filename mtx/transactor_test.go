package mtx

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool/assert"
)

func Test_CtxOperator(t *testing.T) {
	t.Run("success_extract_committer", func(t *testing.T) {
		t.Run("extract_pointer", func(t *testing.T) {
			var (
				ctx = context.Background()
				c   = committerMock{}
				b   = beginnerMock[*committerMock]{}
				o   = NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b)
			)
			ctx = o.Inject(ctx, &c)
			extracted, ok := o.Extract(ctx)
			assert.True(t, ok)
			assert.True(t, extracted == &c)
		})
		t.Run("extract_value", func(t *testing.T) {
			var (
				ctx = context.Background()
				c   = committerValueMock{
					committer: &committerMock{},
				}
				b = beginnerValueMock[committerValueMock]{
					beginner: &beginnerMock[committerValueMock]{},
				}
				o = NewContextOperator[beginnerValueMock[committerValueMock], committerValueMock](b)
			)
			ctx = o.Inject(ctx, c)
			extracted, ok := o.Extract(ctx)
			assert.True(t, ok)
			assert.Equal(t, c, extracted)
		})
		t.Run("extract_nil_value", func(t *testing.T) {
			var (
				ctx = context.Background()
				c   = committerValueMock{
					committer: nil,
				}
				b = beginnerValueMock[committerValueMock]{
					beginner: nil,
				}
				o = NewContextOperator[beginnerValueMock[committerValueMock], committerValueMock](b)
			)
			ctx = o.Inject(ctx, c)
			extracted, ok := o.Extract(ctx)
			assert.True(t, ok)
			assert.Equal(t, c, extracted)
		})

	})
}

// nolint: dupl
func Test_Transactor(t *testing.T) { //nolint: dupl
	t.Run("TryGetTx", func(t *testing.T) {
		var (
			ctx            = context.Background()
			commitCalled   bool
			beginnerCalled bool
			c              = committerMock{
				commitFn: func(ctx context.Context) error {
					commitCalled = true
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginnerCalled = true
					return &c, nil
				},
			}
			o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b)
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, o)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			tx, ok := tr.TryGetTx(ctx)
			assert.True(t, ok)
			assert.Equal(t, &c, tx)
			return nil
		})
		assert.NoError(t, err)
		assert.True(t, beginnerCalled)
		assert.True(t, commitCalled)
	})
	t.Run("TxBeginner", func(t *testing.T) {
		var (
			ctx            = context.Background()
			commitCalled   bool
			beginnerCalled bool
			c              = committerMock{
				commitFn: func(ctx context.Context) error {
					commitCalled = true
					return nil
				},
			}
			b = beginnerMock[*committerMock]{
				beginFn: func(ctx context.Context) (*committerMock, error) {
					beginnerCalled = true
					return &c, nil
				},
			}
			o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b)
			tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, o)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			beginner := tr.TxBeginner()
			assert.NotNil(t, beginner)
			assert.Equal(t, &b, beginner)
			return nil
		})
		assert.NoError(t, err)
		assert.True(t, beginnerCalled)
		assert.True(t, commitCalled)
	})
	t.Run("WithinTx", func(t *testing.T) {
		t.Run("success_commit", func(t *testing.T) {
			var (
				ctx            = context.Background()
				commitCalled   bool
				beginnerCalled bool
				c              = committerMock{
					commitFn: func(ctx context.Context) error {
						commitCalled = true
						return nil
					},
				}
				b = beginnerMock[*committerMock]{
					beginFn: func(ctx context.Context) (*committerMock, error) {
						beginnerCalled = true
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.Equal(t, &c, tx)
				return nil
			})
			assert.NoError(t, err)
			assert.True(t, beginnerCalled)
			assert.True(t, commitCalled)
		})
		t.Run("success_and_not_commit_with_exists_tx", func(t *testing.T) {
			var (
				ctx          = context.Background()
				commitCalled bool
				c            = committerMock{
					commitFn: func(ctx context.Context) error {
						commitCalled = true
						return nil
					},
				}
				b  = beginnerMock[*committerMock]{}
				o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, o)
			)
			ctx = o.Inject(ctx, &c)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.Equal(t, &c, tx)
				return nil
			})
			assert.NoError(t, err)
			assert.True(t, !commitCalled)
		})
		t.Run("failed_commit", func(t *testing.T) {
			var (
				expError = fmt.Errorf("some_commit_error")

				ctx          = context.Background()
				commitCalled bool
				c            = committerMock{
					commitFn: func(ctx context.Context) error {
						commitCalled = true
						return expError
					},
				}
				b = beginnerMock[*committerMock]{
					beginFn: func(ctx context.Context) (*committerMock, error) {
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.Equal(t, &c, tx)
				return nil
			})
			assert.ErrorIs(t, err, ErrCommitFailed)
			assert.ErrorIs(t, err, expError)
			assert.True(t, commitCalled)
		})
		t.Run("success_rollback", func(t *testing.T) {
			var (
				expError = fmt.Errorf("some_transactor_error")

				ctx            = context.Background()
				rollbackCalled bool
				beginCalled    bool
				c              = committerMock{
					rollbackFn: func(ctx context.Context) error {
						rollbackCalled = true
						return nil
					},
				}
				b = beginnerMock[*committerMock]{
					beginFn: func(ctx context.Context) (*committerMock, error) {
						beginCalled = true
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.Equal(t, &c, tx)
				return expError
			})
			assert.ErrorIs(t, err, ErrTxRolledBack)
			assert.ErrorIs(t, err, expError)
			assert.True(t, rollbackCalled)
			assert.True(t, beginCalled)
		})
		t.Run("failed_rollback", func(t *testing.T) {
			var (
				transactorError = fmt.Errorf("some_exec_error")
				rollbackErr     = fmt.Errorf("some_rollbakc_error")

				ctx            = context.Background()
				rollbackCalled bool
				beginCalled    bool
				c              = committerMock{
					rollbackFn: func(ctx context.Context) error {
						rollbackCalled = true
						return rollbackErr
					},
				}
				b = beginnerMock[*committerMock]{
					beginFn: func(ctx context.Context) (*committerMock, error) {
						beginCalled = true
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.Equal(t, &c, tx)
				return transactorError
			})
			assert.ErrorIs(t, err, ErrRollbackFailed)
			assert.ErrorIs(t, err, transactorError)
			assert.ErrorIs(t, err, rollbackErr)
			assert.True(t, rollbackCalled)
			assert.True(t, beginCalled)
		})
		t.Run("success_panic_rollback", func(t *testing.T) {
			var (
				ctx            = context.Background()
				rollbackCalled bool
				beginCalled    bool
				expPanic       = "some_problem"
				c              = committerMock{
					rollbackFn: func(ctx context.Context) error {
						rollbackCalled = true
						return nil
					},
				}
				b = beginnerMock[*committerMock]{
					beginFn: func(ctx context.Context) (*committerMock, error) {
						beginCalled = true
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, o).
					WithPanicRecovery(true)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.Equal(t, &c, tx)
				panic(expPanic)
			})
			assert.ErrorIs(t, err, ErrTxRolledBack)
			assert.ErrorIs(t, err, ErrPanicRecovered)
			assert.True(t, strings.Contains(err.Error(), expPanic))
			assert.True(t, rollbackCalled)
			assert.True(t, beginCalled)
		})
		t.Run("failed_panic_rollback", func(t *testing.T) {
			const (
				expPanicMsg = "some_problem"
			)
			var (
				rollbackErr = fmt.Errorf("some_rollbakc_error")

				ctx            = context.Background()
				rollbackCalled bool
				beginCalled    bool
				c              = committerMock{
					rollbackFn: func(ctx context.Context) error {
						rollbackCalled = true
						return rollbackErr
					},
				}
				b = &beginnerMock[*committerMock]{
					beginFn: func(ctx context.Context) (*committerMock, error) {
						beginCalled = true
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](b)
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](b, o).
					WithPanicRecovery(true)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.Equal(t, &c, tx)
				panic(expPanicMsg)
			})
			assert.ErrorIs(t, err, ErrRollbackFailed)
			assert.ErrorIs(t, err, ErrPanicRecovered)
			assert.ErrorIs(t, err, rollbackErr)
			assert.True(t, strings.Contains(err.Error(), expPanicMsg))
			assert.True(t, rollbackCalled)
			assert.True(t, beginCalled)
		})
		t.Run("failed_begin_tx", func(t *testing.T) {
			var (
				expError = fmt.Errorf("some_begin_error")

				ctx         = context.Background()
				beginCalled bool
				b           = &beginnerMock[*committerMock]{
					beginFn: func(ctx context.Context) (*committerMock, error) {
						beginCalled = true
						return nil, expError
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](b)
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				_, ok := o.Extract(ctx)
				assert.False(t, ok)
				return nil
			})
			assert.ErrorIs(t, err, ErrBeginTx)
			assert.ErrorIs(t, err, expError)
			assert.True(t, beginCalled)
		})
		t.Run("error_when_beginner_is_nil", func(t *testing.T) {
			var (
				ctx = context.Background()
				o   = NewContextOperator[*beginnerMock[*committerMock], *committerMock](nil)
				tr  = NewTransactor[*beginnerMock[*committerMock], *committerMock](nil, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				return nil
			})
			assert.ErrorIs(t, err, ErrNilTxBeginner)
		})
		t.Run("error_when_operator_is_nil", func(t *testing.T) {
			var (
				ctx = context.Background()
				b   = &beginnerMock[*committerMock]{
					beginFn: func(ctx context.Context) (*committerMock, error) {
						return nil, nil
					},
				}
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](b, nil)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				return nil
			})
			assert.ErrorIs(t, err, ErrNilTxOperator)
		})
		t.Run("error_when_transaction_function_is_nil", func(t *testing.T) {
			var (
				beginCalled    bool
				commitCalled   bool
				rollbackCalled bool
				ctx            = context.Background()
				c              = &committerMock{
					commitFn: func(context.Context) error {
						commitCalled = true
						return nil
					},
					rollbackFn: func(context.Context) error {
						rollbackCalled = true
						return nil
					},
				}
				b = &beginnerMock[*committerMock]{
					beginFn: func(context.Context) (*committerMock, error) {
						beginCalled = true
						return c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](b)
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](b, o)
			)

			err := tr.WithinTx(ctx, nil)

			assert.ErrorIs(t, err, ErrNilTxFunc)
			assert.False(t, beginCalled)
			assert.False(t, commitCalled)
			assert.False(t, rollbackCalled)
		})
	})
}

func Test_Transactor_RollbackCtxFactory(t *testing.T) {
	newTransactor := func(c *committerMock) *Transactor[*beginnerMock[*committerMock], *committerMock] {
		b := &beginnerMock[*committerMock]{
			beginFn: func(context.Context) (*committerMock, error) {
				return c, nil
			},
		}
		o := NewContextOperator[*beginnerMock[*committerMock], *committerMock](b)
		return NewTransactor[*beginnerMock[*committerMock], *committerMock](b, o)
	}

	t.Run("uses_context_without_cancel_for_rollback", func(t *testing.T) {
		var (
			factoryCalled  = false
			rollbackCalled = false
			c              = &committerMock{
				rollbackFn: func(rollbackCtx context.Context) error {
					rollbackCalled = true
					assert.NoError(t, rollbackCtx.Err())
					return nil
				},
			}

			expErr = fmt.Errorf("action failed")

			ctx, cancel = context.WithCancel(context.Background())
		)
		cancel()

		tr := newTransactor(c).WithRollbackCtxFactory(func(factoryCtx context.Context) context.Context {
			factoryCalled = true
			assert.ErrorIs(t, factoryCtx.Err(), context.Canceled)
			return context.WithoutCancel(factoryCtx)
		})

		err := tr.WithinTx(ctx, func(context.Context) error {
			return expErr
		})
		assert.ErrorIs(t, err, ErrTxRolledBack)
		assert.ErrorIs(t, err, expErr)
		assert.True(t, factoryCalled)
		assert.True(t, rollbackCalled)
	})

	t.Run("is_not_called_for_commit_or_nested_transaction", func(t *testing.T) {
		var (
			factoryCalls = 0
			commitCalls  = 0
			c            = &committerMock{
				commitFn: func(context.Context) error {
					commitCalls++
					return nil
				},
			}
			ctx = context.Background()
		)
		tr := newTransactor(c).WithRollbackCtxFactory(func(ctx context.Context) context.Context {
			factoryCalls++
			return context.WithoutCancel(ctx)
		})

		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			return tr.WithinTx(ctx, func(context.Context) error {
				return nil
			})
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, commitCalls)
		assert.Equal(t, 0, factoryCalls)
	})

	t.Run("is_called_only_for_top_level_rollback", func(t *testing.T) {
		var (
			factoryCalls = 0
			c            = &committerMock{
				rollbackFn: func(context.Context) error {
					return nil
				},
			}
			ctx    = context.Background()
			expErr = fmt.Errorf("nested action failed")
		)

		tr := newTransactor(c).WithRollbackCtxFactory(func(ctx context.Context) context.Context {
			factoryCalls++
			return context.WithoutCancel(ctx)
		})

		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			return tr.WithinTx(ctx, func(context.Context) error {
				return expErr
			})
		})
		assert.ErrorIs(t, err, ErrTxRolledBack)
		assert.ErrorIs(t, err, expErr)
		assert.Equal(t, 1, factoryCalls)
	})

	t.Run("uses_original_context_for_nil_factory_or_result", func(t *testing.T) {
		var (
			expErr    = fmt.Errorf("action failed")
			testCases = []struct {
				name    string
				factory func(context.Context) context.Context
			}{
				{
					name:    "nil_factory",
					factory: nil,
				},
				{
					name: "nil_result",
					factory: func(context.Context) context.Context {
						return nil
					},
				},
			}
		)

		for i, testCase := range testCases {
			t.Run(fmt.Sprintf("%d_%s", i, testCase.name), func(t *testing.T) {
				var (
					operationCtx context.Context
					c            = &committerMock{
						rollbackFn: func(rollbackCtx context.Context) error {
							assert.Equal(t, rollbackCtx, operationCtx)
							return nil
						},
					}

					ctx = context.Background()
				)
				tr := newTransactor(c).WithRollbackCtxFactory(testCase.factory)

				err := tr.WithinTx(ctx, func(currentCtx context.Context) error {
					operationCtx = currentCtx
					return expErr
				})
				assert.ErrorIs(t, err, ErrTxRolledBack)
				assert.ErrorIs(t, err, expErr)
			})
		}
	})
}

func Test_Transactor_PanicRecoveryDisabled(t *testing.T) {
	const panicMessage = "boom"

	type panicStruct struct {
		message string
	}

	var (
		panicStructValue = &panicStruct{message: panicMessage}
	)

	newTransactor := func(rollbackCalls *int, rollbackErr error) *Transactor[*beginnerMock[*committerMock], *committerMock] {
		c := &committerMock{
			rollbackFn: func(context.Context) error {
				*rollbackCalls = *rollbackCalls + 1
				return rollbackErr
			},
		}
		b := &beginnerMock[*committerMock]{
			beginFn: func(context.Context) (*committerMock, error) {
				return c, nil
			},
		}
		o := NewContextOperator[*beginnerMock[*committerMock], *committerMock](b)
		return NewTransactor[*beginnerMock[*committerMock], *committerMock](b, o)
	}

	callAndRecover := func(
		t *testing.T,
		tr *Transactor[*beginnerMock[*committerMock], *committerMock],
		value any,
		nested bool,
	) (recovered any) {
		t.Helper()

		defer func() {
			recovered = recover()
			assert.NotNil(t, recovered)
			v, ok := recovered.(*panicStruct)
			assert.True(t, ok)
			assert.True(t, v == value)
		}()

		err := tr.WithinTx(context.Background(), func(ctx context.Context) error {
			if nested {
				return tr.WithinTx(ctx, func(context.Context) error {
					panic(value)
				})
			}
			panic(value)
		})
		t.Fatalf("WithinTx returned instead of resuming panic: %v", err)
		return nil
	}

	t.Run("default_repanics_after_successful_rollback", func(t *testing.T) {
		rollbackCalls := 0
		tr := newTransactor(&rollbackCalls, nil)

		recovered := callAndRecover(t, tr, panicStructValue, false)

		assert.True(t, recovered == panicStructValue)
		assert.Equal(t, 1, rollbackCalls)
	})

	t.Run("repanics_even_when_rollback_fails", func(t *testing.T) {
		var (
			rollbackCalls = 0
			value         = &panicStruct{message: panicMessage}
			tr            = newTransactor(&rollbackCalls, fmt.Errorf("rollback failed"))
		)

		recovered := callAndRecover(t, tr, value, false)

		assert.True(t, recovered == value)
		assert.Equal(t, 1, rollbackCalls)
	})

	t.Run("nested_call_leaves_rollback_to_transaction_owner", func(t *testing.T) {
		var (
			rollbackCalls = 0
			value         = &panicStruct{message: panicMessage}
			tr            = newTransactor(&rollbackCalls, nil)
		)

		recovered := callAndRecover(t, tr, value, true)

		assert.True(t, recovered == value)
		assert.Equal(t, 1, rollbackCalls)
	})

	t.Run("can_disable_recovery_on_derived_transactor", func(t *testing.T) {
		var (
			rollbackCalls = 0
			value         = &panicStruct{message: panicMessage}
		)

		tr := newTransactor(&rollbackCalls, nil).
			WithPanicRecovery(true).
			WithPanicRecovery(false)

		recovered := callAndRecover(t, tr, value, false)

		assert.True(t, recovered == value)
		assert.Equal(t, 1, rollbackCalls)
	})
}

// Test_Transactor_recursive_call - testing recursive [mtx.Transactor] calls.
func Test_Transactor_recursive_call(t *testing.T) { //nolint: dupl
	const (
		ctxValTopLvl    = "top_lvl"
		ctxValSecondLvl = "second_lvl"
		ctxValThirdLvl  = "third_lvl"
	)
	type (
		Key struct{}
	)

	/*
		functions to inject and check recursion level
	*/
	var (
		ctxKey    Key
		injectLvl = func(ctx context.Context, lvl string) context.Context {
			t.Helper()
			return context.WithValue(ctx, ctxKey, lvl)
		}

		isLvlEqual = func(ctx context.Context, required string) bool {
			t.Helper()
			lvl, ok := ctx.Value(ctxKey).(string)
			if !ok {
				return false
			}
			return strings.EqualFold(lvl, required)
		}

		assertTopLvl = func(ctx context.Context) {
			// tool.Assert that rollback was called on the recursion "top" level.
			assert.True(t, isLvlEqual(ctx, ctxValTopLvl))
			// tool.Assert that rollback call wasn't called on the "second" recursion level.
			assert.False(t, isLvlEqual(ctx, ctxValSecondLvl))
			// tool.Assert that rollback call wasn't called on the "third" recursion level.
			assert.False(t, isLvlEqual(ctx, ctxValThirdLvl))
		}
	)

	var (
		commitCalled, rollbackCalled, beginCalled int

		cleanup = func() {
			commitCalled, rollbackCalled, beginCalled = 0, 0, 0
		}
	)

	var (
		newInstance = func(ctx context.Context) (
			*committerMock,
			*ContextOperator[*beginnerMock[*committerMock], *committerMock],
			*Transactor[*beginnerMock[*committerMock], *committerMock]) {
			var (
				c = committerMock{
					commitFn: func(ctx context.Context) error {
						commitCalled++
						return nil
					},
					rollbackFn: func(ctx context.Context) error {
						rollbackCalled++
						assertTopLvl(ctx)
						return nil
					},
				}
				b = &beginnerMock[*committerMock]{
					beginFn: func(ctx context.Context) (*committerMock, error) {
						beginCalled++
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock], *committerMock](b)
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](b, o)
			)
			return &c, o, tr
		}
	)

	t.Run("success_rollback", func(t *testing.T) {
		defer t.Cleanup(cleanup)
		var (
			expError = fmt.Errorf("some_error")

			ctx      = context.Background()
			c, o, tr = newInstance(ctx)
		)

		{
			// inject "top" level variable in context.Context
			ctx = injectLvl(ctx, ctxValTopLvl)
		}

		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			tx, ok := o.Extract(ctx)
			assert.True(t, ok)
			assert.True(t, c == tx)

			// inject "second" level variable in context.Context.
			ctx = injectLvl(ctx, ctxValSecondLvl)
			return tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.True(t, c == tx)
				return expError
			})
		})
		assert.ErrorIs(t, err, ErrTxRolledBack)
		assert.ErrorIs(t, err, expError)
		assert.Equal(t, 1, rollbackCalled)
		assert.Equal(t, 1, beginCalled)
	})

	t.Run("success_and_commit_on_top_lvl_func", func(t *testing.T) {
		defer t.Cleanup(cleanup)
		var (
			ctx      = context.Background()
			c, o, tr = newInstance(ctx)
		)

		{
			// inject "top" level variable in context.Context
			ctx = injectLvl(ctx, ctxValTopLvl)
		}

		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			tx, ok := o.Extract(ctx)
			assert.True(t, ok)
			assert.True(t, c == tx)

			// inject "second" level variable in context.Context.
			ctx = injectLvl(ctx, ctxValSecondLvl)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.True(t, c == tx)
				return nil
			})
			assert.NoError(t, err)

			err = tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.True(t, c == tx)
				return nil
			})
			assert.True(t, err == nil)

			return err
		})
		assert.Equal(t, 1, beginCalled)
		assert.NoError(t, err)
		assert.Equal(t, 1, commitCalled)
	})
	t.Run("error_and_rollback_on_high_lvl_when_error_on_low_lvl_func", func(t *testing.T) {
		defer t.Cleanup(cleanup)
		var (
			expError = fmt.Errorf("some_error")

			ctx      = context.Background()
			c, o, tr = newInstance(ctx)
		)

		{
			// inject "top" level variable in context.Context
			ctx = injectLvl(ctx, ctxValTopLvl)
		}

		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			tx, ok := o.Extract(ctx)
			assert.True(t, ok)
			assert.True(t, c == tx)

			// inject "second" level variable in context.Context.
			ctx = injectLvl(ctx, ctxValSecondLvl)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.True(t, c == tx)

				// inject "third" level variable in context.Context.
				ctx = injectLvl(ctx, ctxValThirdLvl)
				err := tr.WithinTx(ctx, func(ctx context.Context) error {
					tx, ok := o.Extract(ctx)
					assert.True(t, ok)
					assert.True(t, c == tx)
					return expError
				})
				return err
			})
			assert.Error(t, err)

			return err
		})
		assert.ErrorIs(t, err, ErrTxRolledBack)
		assert.ErrorIs(t, err, expError)
		assert.Equal(t, 1, beginCalled)
		assert.Equal(t, 0, commitCalled)
		assert.Equal(t, 1, rollbackCalled)
	})
	t.Run("panic", func(t *testing.T) {
		const (
			lowLvlPanicMsg    = "some_low_panic"
			middleLvlPanicMsg = "some_middle_panic"
		)

		t.Run("error_and_rollback_on_high_lvl_when_panic_on_low_lvl_func", func(t *testing.T) {
			defer t.Cleanup(cleanup)
			var (
				ctx      = context.Background()
				c, o, tr = newInstance(ctx)
			)

			{
				// inject "top" level variable in context.Context
				ctx = injectLvl(ctx, ctxValTopLvl)
			}

			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.True(t, c == tx)

				// inject "second" level variable in context.Context.
				ctx = injectLvl(ctx, ctxValSecondLvl)
				err := tr.WithPanicRecovery(true).
					WithinTx(ctx, func(ctx context.Context) error {
						tx, ok := o.Extract(ctx)
						assert.True(t, ok)
						assert.True(t, c == tx)

						// inject "second" level variable in context.Context.
						ctx = injectLvl(ctx, ctxValThirdLvl)
						err := tr.WithPanicRecovery(true).
							WithinTx(ctx, func(ctx context.Context) error {
								tx, ok := o.Extract(ctx)
								assert.True(t, ok)
								assert.True(t, c == tx)
								panic(lowLvlPanicMsg)
							})
						assert.Error(t, err)
						return err
					})
				assert.Error(t, err)

				return err
			})
			assert.ErrorIs(t, err, ErrTxRolledBack)
			assert.ErrorIs(t, err, ErrPanicRecovered)
			assert.True(t, strings.Contains(err.Error(), lowLvlPanicMsg))
			assert.Equal(t, 1, beginCalled)
			assert.Equal(t, 0, commitCalled)
			assert.Equal(t, 1, rollbackCalled)
		})

		t.Run("error_and_rollback_on_high_lvl_when_panic_on_middle_lvl_override_low_lvl", func(t *testing.T) {
			defer t.Cleanup(cleanup)
			var (
				ctx      = context.Background()
				c, o, tr = newInstance(ctx)
			)

			{
				// inject "top" level variable in context.Context
				ctx = injectLvl(ctx, ctxValTopLvl)
			}

			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assert.True(t, ok)
				assert.True(t, c == tx)

				// inject "second" level variable in context.Context.
				ctx = injectLvl(ctx, ctxValSecondLvl)
				err := tr.WithPanicRecovery(true).
					WithinTx(ctx, func(ctx context.Context) error {
						tx, ok := o.Extract(ctx)
						assert.True(t, ok)
						assert.True(t, c == tx)

						// inject "second" level variable in context.Context.
						ctx = injectLvl(ctx, ctxValThirdLvl)
						err := tr.WithPanicRecovery(true).
							WithinTx(ctx, func(ctx context.Context) error {
								tx, ok := o.Extract(ctx)
								assert.True(t, ok)
								assert.True(t, c == tx)
								panic(lowLvlPanicMsg)
							})
						assert.Error(t, err)
						panic(middleLvlPanicMsg)
					})
				assert.True(t, err != nil)

				return err
			})
			assert.ErrorIs(t, err, ErrTxRolledBack)
			assert.ErrorIs(t, err, ErrPanicRecovered)
			assert.False(t, strings.Contains(err.Error(), lowLvlPanicMsg))
			assert.True(t, strings.Contains(err.Error(), middleLvlPanicMsg))
			assert.Equal(t, 1, beginCalled)
			assert.Equal(t, 0, commitCalled)
			assert.Equal(t, 1, rollbackCalled)
		})
	})
}
