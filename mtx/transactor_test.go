package mtx

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kozmod/oniontx/internal/testtool"
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
			testtool.AssertTrue(t, ok)
			testtool.AssertTrue(t, extracted == &c)
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
			testtool.AssertTrue(t, ok)
			testtool.AssertTrue(t, extracted == c)
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
			testtool.AssertTrue(t, ok)
			testtool.AssertTrue(t, extracted == c)
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
			testtool.AssertTrue(t, ok)
			testtool.AssertTrue(t, &c == tx)
			return nil
		})
		testtool.AssertNoError(t, err)
		testtool.AssertTrue(t, beginnerCalled)
		testtool.AssertTrue(t, commitCalled)
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
			testtool.AssertTrue(t, beginner != nil)
			testtool.AssertTrue(t, &b == beginner)
			return nil
		})
		testtool.AssertNoError(t, err)
		testtool.AssertTrue(t, beginnerCalled)
		testtool.AssertTrue(t, commitCalled)
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
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, &c == tx)
				return nil
			})
			testtool.AssertNoError(t, err)
			testtool.AssertTrue(t, beginnerCalled)
			testtool.AssertTrue(t, commitCalled)
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
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, &c == tx)
				return nil
			})
			testtool.AssertNoError(t, err)
			testtool.AssertTrue(t, !commitCalled)
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
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, &c == tx)
				return nil
			})
			testtool.AssertTrue(t, errors.Is(err, ErrCommitFailed))
			testtool.AssertTrue(t, errors.Is(err, expError))
			testtool.AssertTrue(t, commitCalled)
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
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, &c == tx)
				return expError
			})
			testtool.AssertTrue(t, errors.Is(err, ErrRollbackSuccess))
			testtool.AssertTrue(t, errors.Is(err, expError))
			testtool.AssertTrue(t, rollbackCalled)
			testtool.AssertTrue(t, beginCalled)
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
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, &c == tx)
				return transactorError
			})
			testtool.AssertTrue(t, errors.Is(err, ErrRollbackFailed))
			testtool.AssertTrue(t, errors.Is(err, transactorError))
			testtool.AssertTrue(t, errors.Is(err, rollbackErr))
			testtool.AssertTrue(t, rollbackCalled)
			testtool.AssertTrue(t, beginCalled)
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
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](&b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, &c == tx)
				panic(expPanic)
			})
			testtool.AssertTrue(t, errors.Is(err, ErrRollbackSuccess))
			testtool.AssertTrue(t, errors.Is(err, ErrPanicRecovered))
			testtool.AssertTrue(t, strings.Contains(err.Error(), expPanic))
			testtool.AssertTrue(t, rollbackCalled)
			testtool.AssertTrue(t, beginCalled)
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
				tr = NewTransactor[*beginnerMock[*committerMock], *committerMock](b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, &c == tx)
				panic(expPanicMsg)
			})
			testtool.AssertTrue(t, errors.Is(err, ErrRollbackFailed))
			testtool.AssertTrue(t, errors.Is(err, ErrPanicRecovered))
			testtool.AssertTrue(t, errors.Is(err, rollbackErr))
			testtool.AssertTrue(t, strings.Contains(err.Error(), expPanicMsg))
			testtool.AssertTrue(t, rollbackCalled)
			testtool.AssertTrue(t, beginCalled)
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
				testtool.AssertFalse(t, ok)
				return nil
			})
			testtool.AssertTrue(t, errors.Is(err, ErrBeginTx))
			testtool.AssertTrue(t, errors.Is(err, expError))
			testtool.AssertTrue(t, beginCalled)
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
			testtool.AssertTrue(t, errors.Is(err, ErrNilTxBeginner))
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
			testtool.AssertTrue(t, errors.Is(err, ErrNilTxOperator))
		})
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
			testtool.AssertTrue(t, isLvlEqual(ctx, ctxValTopLvl))
			// tool.Assert that rollback call wasn't called on the "second" recursion level.
			testtool.AssertFalse(t, isLvlEqual(ctx, ctxValSecondLvl))
			// tool.Assert that rollback call wasn't called on the "third" recursion level.
			testtool.AssertFalse(t, isLvlEqual(ctx, ctxValThirdLvl))
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
			testtool.AssertTrue(t, ok)
			testtool.AssertTrue(t, c == tx)

			// inject "second" level variable in context.Context.
			ctx = injectLvl(ctx, ctxValSecondLvl)
			return tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, c == tx)
				return expError
			})
		})
		testtool.AssertTrue(t, errors.Is(err, ErrRollbackSuccess))
		testtool.AssertTrue(t, errors.Is(err, expError))
		testtool.AssertTrue(t, rollbackCalled == 1)
		testtool.AssertTrue(t, beginCalled == 1)
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
			testtool.AssertTrue(t, ok)
			testtool.AssertTrue(t, c == tx)

			// inject "second" level variable in context.Context.
			ctx = injectLvl(ctx, ctxValSecondLvl)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, c == tx)
				return nil
			})
			testtool.AssertNoError(t, err)

			err = tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, c == tx)
				return nil
			})
			testtool.AssertTrue(t, err == nil)

			return err
		})
		testtool.AssertTrue(t, beginCalled == 1)
		testtool.AssertNoError(t, err)
		testtool.AssertTrue(t, commitCalled == 1)
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
			testtool.AssertTrue(t, ok)
			testtool.AssertTrue(t, c == tx)

			// inject "second" level variable in context.Context.
			ctx = injectLvl(ctx, ctxValSecondLvl)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, c == tx)

				// inject "third" level variable in context.Context.
				ctx = injectLvl(ctx, ctxValThirdLvl)
				err := tr.WithinTx(ctx, func(ctx context.Context) error {
					tx, ok := o.Extract(ctx)
					testtool.AssertTrue(t, ok)
					testtool.AssertTrue(t, c == tx)
					return expError
				})
				return err
			})
			testtool.AssertError(t, err)

			return err
		})
		testtool.AssertTrue(t, errors.Is(err, ErrRollbackSuccess))
		testtool.AssertTrue(t, errors.Is(err, expError))
		testtool.AssertTrue(t, beginCalled == 1)
		testtool.AssertTrue(t, commitCalled == 0)
		testtool.AssertTrue(t, rollbackCalled == 1)
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
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, c == tx)

				// inject "second" level variable in context.Context.
				ctx = injectLvl(ctx, ctxValSecondLvl)
				err := tr.WithinTx(ctx, func(ctx context.Context) error {
					tx, ok := o.Extract(ctx)
					testtool.AssertTrue(t, ok)
					testtool.AssertTrue(t, c == tx)

					// inject "second" level variable in context.Context.
					ctx = injectLvl(ctx, ctxValThirdLvl)
					err := tr.WithinTx(ctx, func(ctx context.Context) error {
						tx, ok := o.Extract(ctx)
						testtool.AssertTrue(t, ok)
						testtool.AssertTrue(t, c == tx)
						panic(lowLvlPanicMsg)
					})
					testtool.AssertError(t, err)
					return err
				})
				testtool.AssertError(t, err)

				return err
			})
			testtool.AssertTrue(t, errors.Is(err, ErrRollbackSuccess))
			testtool.AssertTrue(t, errors.Is(err, ErrPanicRecovered))
			testtool.AssertTrue(t, strings.Contains(err.Error(), lowLvlPanicMsg))
			testtool.AssertTrue(t, beginCalled == 1)
			testtool.AssertTrue(t, commitCalled == 0)
			testtool.AssertTrue(t, rollbackCalled == 1)
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
				testtool.AssertTrue(t, ok)
				testtool.AssertTrue(t, c == tx)

				// inject "second" level variable in context.Context.
				ctx = injectLvl(ctx, ctxValSecondLvl)
				err := tr.WithinTx(ctx, func(ctx context.Context) error {
					tx, ok := o.Extract(ctx)
					testtool.AssertTrue(t, ok)
					testtool.AssertTrue(t, c == tx)

					// inject "second" level variable in context.Context.
					ctx = injectLvl(ctx, ctxValThirdLvl)
					err := tr.WithinTx(ctx, func(ctx context.Context) error {
						tx, ok := o.Extract(ctx)
						testtool.AssertTrue(t, ok)
						testtool.AssertTrue(t, c == tx)
						panic(lowLvlPanicMsg)
					})
					testtool.AssertError(t, err)
					panic(middleLvlPanicMsg)
				})
				testtool.AssertTrue(t, err != nil)

				return err
			})
			testtool.AssertTrue(t, errors.Is(err, ErrRollbackSuccess))
			testtool.AssertTrue(t, errors.Is(err, ErrPanicRecovered))
			testtool.AssertFalse(t, strings.Contains(err.Error(), lowLvlPanicMsg))
			testtool.AssertTrue(t, strings.Contains(err.Error(), middleLvlPanicMsg))
			testtool.AssertTrue(t, beginCalled == 1)
			testtool.AssertTrue(t, commitCalled == 0)
			testtool.AssertTrue(t, rollbackCalled == 1)
		})
	})
}
