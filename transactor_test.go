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
		var (
			ctx = context.Background()
			c   = committerMock{}
			b   = &beginnerMock[*committerMock, any]{}
			o   = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
		)
		ctx = o.Inject(ctx, &c)
		extracted, ok := o.Extract(ctx)
		assertTrue(t, ok)
		assertTrue(t, extracted == &c)
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
			b = &beginnerMock[*committerMock, any]{
				beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
					beginnerCalled = true
					assertTrue(t, opts == nil)
					return &c, nil
				},
			}
			o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
			tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
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
			b = &beginnerMock[*committerMock, any]{
				beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
					beginnerCalled = true
					assertTrue(t, opts == nil)
					return &c, nil
				},
			}
			o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
			tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
		)
		err := tr.WithinTx(ctx, func(ctx context.Context) error {
			beginner := tr.TxBeginner()
			assertTrue(t, beginner != nil)
			assertTrue(t, b == beginner)
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
				b = &beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						beginnerCalled = true
						assertTrue(t, opts == nil)
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
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
		t.Run("success_commit_with_exists_tx", func(t *testing.T) {
			var (
				ctx          = context.Background()
				commitCalled bool
				c            = committerMock{
					commitFn: func(ctx context.Context) error {
						commitCalled = true
						return nil
					},
				}
				b  = &beginnerMock[*committerMock, any]{}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
			)
			ctx = o.Inject(ctx, &c)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return nil
			})
			assertTrue(t, err == nil)
			assertTrue(t, commitCalled)
		})
		t.Run("failed_commit", func(t *testing.T) {
			var (
				ctx          = context.Background()
				commitCalled bool
				c            = committerMock{
					commitFn: func(ctx context.Context) error {
						commitCalled = true
						return fmt.Errorf("some_commit_error")
					},
				}
				b = &beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						assertTrue(t, opts == nil)
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return nil
			})
			assertTrue(t, errors.Is(err, ErrCommitFailed))
			assertTrue(t, commitCalled)
		})
		t.Run("success_rollback", func(t *testing.T) {
			var (
				ctx            = context.Background()
				rollbackCalled bool
				beginCalled    bool
				c              = committerMock{
					rollbackFn: func(ctx context.Context) error {
						rollbackCalled = true
						return nil
					},
				}
				b = &beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						beginCalled = true
						assertTrue(t, opts == nil)
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return fmt.Errorf("some error")
			})
			assertTrue(t, errors.Is(err, ErrRollbackSuccess))
			assertTrue(t, rollbackCalled)
			assertTrue(t, beginCalled)
		})
		t.Run("failed_rollback", func(t *testing.T) {
			var (
				ctx            = context.Background()
				rollbackCalled bool
				beginCalled    bool
				execError      = fmt.Errorf("some exec error")
				rollbackErr    = fmt.Errorf("some rollbakc error")
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
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				return execError
			})
			assertTrue(t, errors.Is(err, ErrRollbackFailed))
			assertTrue(t, errors.Is(err, execError))
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
				b = &beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						beginCalled = true
						assertTrue(t, opts == nil)
						return &c, nil
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
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
			var (
				ctx            = context.Background()
				rollbackCalled bool
				beginCalled    bool
				rollbackErr    = fmt.Errorf("some rollbakc error")
				expPanic       = "some_problem"
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
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				tx, ok := o.Extract(ctx)
				assertTrue(t, ok)
				assertTrue(t, &c == tx)
				panic(expPanic)
			})
			assertTrue(t, errors.Is(err, ErrRollbackFailed))
			assertTrue(t, errors.Is(err, rollbackErr))
			assertTrue(t, strings.Contains(err.Error(), expPanic))
			assertTrue(t, rollbackCalled)
			assertTrue(t, beginCalled)
		})
		t.Run("failed_begin_tx", func(t *testing.T) {
			var (
				ctx         = context.Background()
				beginCalled bool
				beginErr    = fmt.Errorf("some_begin_error")
				b           = &beginnerMock[*committerMock, any]{
					beginFn: func(ctx context.Context, opts ...Option[any]) (*committerMock, error) {
						beginCalled = true
						return nil, beginErr
					},
				}
				o  = NewContextOperator[*beginnerMock[*committerMock, any], *committerMock](&b)
				tr = NewTransactor[*beginnerMock[*committerMock, any], *committerMock, any](b, o)
			)
			err := tr.WithinTx(ctx, func(ctx context.Context) error {
				_, ok := o.Extract(ctx)
				assertTrue(t, !ok)
				return nil
			})
			assertTrue(t, errors.Is(err, ErrBeginTx))
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
			assertTrue(t, errors.Is(err, ErrNilBeginner))
		})
	})
}

// beginnerMock was added to avoid to use external dependencies for mocking
type beginnerMock[C TxCommitter, O any] struct {
	beginFn func(ctx context.Context, opts ...Option[O]) (C, error)
}

func (b *beginnerMock[C, O]) BeginTx(ctx context.Context, opts ...Option[O]) (C, error) {
	return b.beginFn(ctx, opts...)
}

// committerMock was added to avoid to use external dependencies for mocking
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

// assertTrue was added to avoid to use external dependencies for mocking
func assertTrue(t *testing.T, val bool) {
	t.Helper()
	if !val {
		t.Fatal()
	}
}
