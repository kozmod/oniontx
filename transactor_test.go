package oniontx

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func Test_CtxOperator(t *testing.T) {
	t.Run("success_extract_committer", func(t *testing.T) {
		t.Run("extract_pointer", func(t *testing.T) {
			var (
				ctx = context.Background()
				c   = committerMock{}
				b   = beginnerMock[*committerMock, any]{}
				o   = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
			)
			ctx = o.Inject(ctx, &c)
			extracted, ok := o.Extract(ctx)
			assertTrue(t, ok)
			assertTrue(t, extracted == &c)
		})
		t.Run("extract_value", func(t *testing.T) {
			var (
				ctx = context.Background()
				c   = committerValueMock{
					committer: &committerMock{},
				}
				b = beginnerValueMock[committerValueMock, any]{
					beginner: &beginnerMock[committerValueMock, any]{},
				}
				o = NewContextOperator[beginnerValueMock[committerValueMock, any], committerValueMock](b)
			)
			ctx = o.Inject(ctx, c)
			extracted, ok := o.Extract(ctx)
			assertTrue(t, ok)
			assertTrue(t, extracted == c)
		})
		t.Run("extract_nil_value", func(t *testing.T) {
			var (
				ctx = context.Background()
				c   = committerValueMock{
					committer: nil,
				}
				b = beginnerValueMock[committerValueMock, any]{
					beginner: nil,
				}
				o = NewContextOperator[beginnerValueMock[committerValueMock, any], committerValueMock](b)
			)
			ctx = o.Inject(ctx, c)
			extracted, ok := o.Extract(ctx)
			assertTrue(t, ok)
			assertTrue(t, extracted == c)
		})

	})
}

// nolint: dupl
func Test_Transactor(t *testing.T) {
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
			b = beginnerMock[*committerMock, any]{
				beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
					beginnerCalled = true
					assertTrue(t, opts == nil)
					return &c, nil
				},
			}
			o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
			tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](&b, o)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			tx, ok := tr.TryGetTx(ctx)
			assertTrue(t, ok)
			assertTrue(t, &c == tx)
			return nil
		})
		assertTrue(t, err == nil)
		assertTrue(t, beginnerCalled)
		assertTrue(t, commitCalled)
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
			b = beginnerMock[*committerMock, any]{
				beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
					beginnerCalled = true
					assertTrue(t, opts == nil)
					return &c, nil
				},
			}
			o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
			tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](&b, o)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			beginner := tr.TxBeginner()
			assertTrue(t, beginner != nil)
			assertTrue(t, &b == beginner)
			return nil
		})
		assertTrue(t, err == nil)
		assertTrue(t, beginnerCalled)
		assertTrue(t, commitCalled)
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
				b = beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						beginnerCalled = true
						assertTrue(t, opts == nil)
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](&b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return nil
			})
			assertTrue(t, err == nil)
			assertTrue(t, beginnerCalled)
			assertTrue(t, commitCalled)
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
				b  = beginnerMock[*committerMock, any]{}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](&b, o)
			)
			ctx = o.Inject(ctx, &c)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return nil
			})
			assertTrue(t, err == nil)
			assertTrue(t, !commitCalled)
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
				b = beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						assertTrue(t, opts == nil)
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](&b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return nil
			})
			assertTrue(t, errors.Is(err, ErrCommitFailed))
			assertTrue(t, errors.Is(err, expError))
			assertTrue(t, commitCalled)
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
				b = beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						beginCalled = true
						assertTrue(t, opts == nil)
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](&b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return expError
			})
			assertTrue(t, errors.Is(err, ErrRollbackSuccess))
			assertTrue(t, errors.Is(err, expError))
			assertTrue(t, rollbackCalled)
			assertTrue(t, beginCalled)
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
				b = beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						beginCalled = true
						assertTrue(t, opts == nil)
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](&b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return transactorError
			})
			assertTrue(t, errors.Is(err, ErrRollbackFailed))
			assertTrue(t, errors.Is(err, transactorError))
			assertTrue(t, errors.Is(err, rollbackErr))
			assertTrue(t, rollbackCalled)
			assertTrue(t, beginCalled)
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
				b = beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						beginCalled = true
						assertTrue(t, opts == nil)
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](&b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				panic(expPanic)
			})
			assertTrue(t, errors.Is(err, ErrRollbackSuccess))
			assertTrue(t, strings.Contains(err.Error(), expPanic))
			assertTrue(t, rollbackCalled)
			assertTrue(t, beginCalled)
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
				b = &beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						beginCalled = true
						assertTrue(t, opts == nil)
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				panic(expPanicMsg)
			})
			assertTrue(t, errors.Is(err, ErrRollbackFailed))
			assertTrue(t, errors.Is(err, rollbackErr))
			assertTrue(t, strings.Contains(err.Error(), expPanicMsg))
			assertTrue(t, rollbackCalled)
			assertTrue(t, beginCalled)
		})
		t.Run("failed_begin_tx", func(t *testing.T) {
			var (
				expError = fmt.Errorf("some_begin_error")

				ctx         = context.Background()
				beginCalled bool
				b           = &beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						beginCalled = true
						return nil, expError
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				_, ok := o.Extract(ctx)
				assertFalse(t, ok)
				return nil
			})
			assertTrue(t, errors.Is(err, ErrBeginTx))
			assertTrue(t, errors.Is(err, expError))
			assertTrue(t, beginCalled)
		})
		t.Run("error_when_beginner_is_nil", func(t *testing.T) {
			var (
				ctx = context.Background()
				o   = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](nil)
				tr  = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](nil, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				return nil
			})
			assertTrue(t, errors.Is(err, ErrNilTxBeginner))
		})
		t.Run("error_when_operator_is_nil", func(t *testing.T) {
			var (
				ctx = context.Background()
				b   = &beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						return nil, nil
					},
				}
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, nil)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				return nil
			})
			assertTrue(t, errors.Is(err, ErrNilTxOperator))
		})
	})
}

// nolint: dupl
func Test_Transactor_recursive_call(t *testing.T) {
	const (
		ctxValTopLvl    = "top_lvl"
		ctxValSecondLvl = "second_lvl"
		ctxValThirdLvl  = "third_lvl"
	)
	/*
		functions to inject and check recursion level
	*/
	var (
		ctxKey    struct{}
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
			// assert that rollback was called on the recursion "top" level.
			assertTrue(t, isLvlEqual(ctx, ctxValTopLvl))
			// assert that rollback call wasn't called on the "second" recursion level.
			assertFalse(t, isLvlEqual(ctx, ctxValSecondLvl))
			// assert that rollback call wasn't called on the "third" recursion level.
			assertFalse(t, isLvlEqual(ctx, ctxValThirdLvl))
		}
	)

	var (
		commitCalled   int
		rollbackCalled int
		beginCalled    int

		cleanup = func() {
			commitCalled = 0
			rollbackCalled = 0
			beginCalled = 0
		}
	)

	t.Run("success_rollback", func(t *testing.T) {
		defer t.Cleanup(cleanup)
		var (
			expError = fmt.Errorf("some_error")

			ctx = context.Background()
			c   = committerMock{
				rollbackFn: func(ctx context.Context) error {
					rollbackCalled++
					return nil
				},
			}
			b = &beginnerMock[*committerMock, any]{
				beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
					beginCalled++
					assertTrue(t, opts == nil)
					return &c, nil
				},
			}
			o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](b)
			tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			tx, ok := o.Extract(ctx)
			assertTrue(t, ok)
			assertTrue(t, &c == tx)
			return tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return expError
			})
		})
		assertTrue(t, errors.Is(err, ErrRollbackSuccess))
		assertTrue(t, errors.Is(err, expError))
		assertTrue(t, rollbackCalled == 1)
		assertTrue(t, beginCalled == 1)
	})
	t.Run("success_and_commit_on_top_lvl_func", func(t *testing.T) {
		defer t.Cleanup(cleanup)
		var (
			ctx = context.Background()
			c   = committerMock{
				commitFn: func(ctx context.Context) error {
					commitCalled++
					assertTopLvl(ctx)
					return nil
				},
			}
			b = &beginnerMock[*committerMock, any]{
				beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
					beginCalled++
					assertTrue(t, opts == nil)
					return &c, nil
				},
			}
			o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](b)
			tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
		)

		{
			// inject "top" level variable in context.Context
			ctx = injectLvl(ctx, ctxValTopLvl)
		}

		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			tx, ok := o.Extract(ctx)
			assertTrue(t, ok)
			assertTrue(t, &c == tx)

			// inject "second" level variable in context.Context.
			ctx = injectLvl(ctx, ctxValSecondLvl)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return nil
			})
			assertTrue(t, err == nil)

			err = tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return nil
			})
			assertTrue(t, err == nil)

			return err
		})
		assertTrue(t, beginCalled == 1)
		assertTrue(t, err == nil)
		assertTrue(t, commitCalled == 1)
	})
	t.Run("error_and_rollback_on_high_lvl_when_panic_on_low_lvl_func", func(t *testing.T) {
		defer t.Cleanup(cleanup)
		const (
			somePanicMsg = "some_panic"
		)
		var (
			ctx = context.Background()
			c   = committerMock{
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
			b = &beginnerMock[*committerMock, any]{
				beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
					beginCalled++
					assertTrue(t, opts == nil)
					return &c, nil
				},
			}
			o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](b)
			tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
		)

		{
			// inject "top" level variable in context.Context
			ctx = injectLvl(ctx, ctxValTopLvl)
		}

		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			tx, ok := o.Extract(ctx)
			assertTrue(t, ok)
			assertTrue(t, &c == tx)

			// inject "second" level variable in context.Context.
			ctx = injectLvl(ctx, ctxValSecondLvl)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)

				// inject "second" level variable in context.Context.
				ctx = injectLvl(ctx, ctxValThirdLvl)
				err := tr.WithinTx(ctx, func(ctx context.Context) error {
					tx, ok := o.Extract(ctx)
					assertTrue(t, ok)
					assertTrue(t, &c == tx)
					panic(somePanicMsg)
				})
				return err
			})
			assertTrue(t, err != nil)

			return err
		})
		assertTrue(t, errors.Is(err, ErrRollbackSuccess))
		assertTrue(t, strings.Contains(err.Error(), somePanicMsg))
		assertTrue(t, beginCalled == 1)
		assertTrue(t, commitCalled == 0)
		assertTrue(t, rollbackCalled == 1)
	})
	t.Run("error_and_rollback_on_high_lvl_when_error_on_low_lvl_func", func(t *testing.T) {
		defer t.Cleanup(cleanup)
		var (
			expError = fmt.Errorf("some_error")

			ctx = context.Background()
			c   = committerMock{
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
			b = &beginnerMock[*committerMock, any]{
				beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
					beginCalled++
					assertTrue(t, opts == nil)
					return &c, nil
				},
			}
			o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](b)
			tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
		)

		{
			// inject "top" level variable in context.Context
			ctx = injectLvl(ctx, ctxValTopLvl)
		}

		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			tx, ok := o.Extract(ctx)
			assertTrue(t, ok)
			assertTrue(t, &c == tx)

			// inject "second" level variable in context.Context.
			ctx = injectLvl(ctx, ctxValSecondLvl)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)

				// inject "third" level variable in context.Context.
				ctx = injectLvl(ctx, ctxValThirdLvl)
				err := tr.WithinTx(ctx, func(ctx context.Context) error {
					tx, ok := o.Extract(ctx)
					assertTrue(t, ok)
					assertTrue(t, &c == tx)
					return expError
				})
				return err
			})
			assertTrue(t, err != nil)

			return err
		})
		assertTrue(t, errors.Is(err, ErrRollbackSuccess))
		assertTrue(t, errors.Is(err, expError))
		assertTrue(t, beginCalled == 1)
		assertTrue(t, commitCalled == 0)
		assertTrue(t, rollbackCalled == 1)
	})
	t.Run("error_and_rollback_on_high_lvl_when_panic_on_middle_lvl_override_low_lvl", func(t *testing.T) {
		defer t.Cleanup(cleanup)
		const (
			lowLvlPanicMsg    = "some_low_panic"
			middleLvlPanicMsg = "some_middle_panic"
		)
		var (
			ctx = context.Background()
			c   = committerMock{
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
			b = &beginnerMock[*committerMock, any]{
				beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
					beginCalled++
					assertTrue(t, opts == nil)
					return &c, nil
				},
			}
			o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](b)
			tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
		)

		{
			// inject "top" level variable in context.Context
			ctx = injectLvl(ctx, ctxValTopLvl)
		}

		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			tx, ok := o.Extract(ctx)
			assertTrue(t, ok)
			assertTrue(t, &c == tx)

			// inject "second" level variable in context.Context.
			ctx = injectLvl(ctx, ctxValSecondLvl)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)

				// inject "second" level variable in context.Context.
				ctx = injectLvl(ctx, ctxValThirdLvl)
				err := tr.WithinTx(ctx, func(ctx context.Context) error {
					tx, ok := o.Extract(ctx)
					assertTrue(t, ok)
					assertTrue(t, &c == tx)
					panic(lowLvlPanicMsg)
				})
				assertTrue(t, err != nil)
				panic(middleLvlPanicMsg)
			})
			assertTrue(t, err != nil)

			return err
		})
		assertTrue(t, errors.Is(err, ErrRollbackSuccess))
		assertFalse(t, strings.Contains(err.Error(), lowLvlPanicMsg))
		assertTrue(t, strings.Contains(err.Error(), middleLvlPanicMsg))
		assertTrue(t, beginCalled == 1)
		assertTrue(t, commitCalled == 0)
		assertTrue(t, rollbackCalled == 1)
	})
}

// beginnerMock was added to avoid to use external dependencies for mocking (pointer receiver).
type beginnerMock[T Tx, O any] struct {
	beginFn func(ctx context.Context, opts ...Option[O]) (T, error)
}

func (b *beginnerMock[T, O]) BeginTx(ctx context.Context, opts ...Option[O]) (T, error) {
	return b.beginFn(ctx, opts...)
}

// committerMock was added to avoid to use external dependencies for mocking (pointer receiver).
type committerMock struct {
	commitFn   func(ctx context.Context) error
	rollbackFn func(ctx context.Context) error
}

func (c *committerMock) Commit(ctx context.Context) error {
	return c.commitFn(ctx)
}

func (c *committerMock) Rollback(ctx context.Context) error {
	return c.rollbackFn(ctx)
}

// beginnerValueMock was added to avoid to use external dependencies for mocking (value receiver).
type beginnerValueMock[T Tx, O any] struct {
	beginner *beginnerMock[T, O]
}

func (b beginnerValueMock[T, O]) BeginTx(ctx context.Context, opts ...Option[O]) (T, error) {
	return b.beginner.beginFn(ctx, opts...)
}

// committerValueMock was added to avoid to use external dependencies for mocking (value receiver).
type committerValueMock struct {
	committer *committerMock
}

func (c committerValueMock) Commit(ctx context.Context) error {
	return c.committer.commitFn(ctx)
}

func (c committerValueMock) Rollback(ctx context.Context) error {
	return c.committer.commitFn(ctx)
}

// assertTrue was added to avoid to use external dependencies for mocking
func assertTrue(t *testing.T, val bool) {
	t.Helper()
	if !val {
		t.Fatal()
	}
}

// assertFalse was added to avoid to use external dependencies for mocking
func assertFalse(t *testing.T, val bool) {
	t.Helper()
	if val {
		t.Fatal()
	}
}
