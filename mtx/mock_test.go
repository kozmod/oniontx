package mtx

import (
	"context"
)

// beginnerMock was added to avoid to use external dependencies for mocking (pointer receiver).
type beginnerMock[T Tx] struct {
	beginFn func(ctx context.Context) (T, error)
}

func (b *beginnerMock[T]) BeginTx(ctx context.Context) (T, error) {
	return b.beginFn(ctx)
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
type beginnerValueMock[T Tx] struct {
	beginner *beginnerMock[T]
}

func (b beginnerValueMock[T]) BeginTx(ctx context.Context) (T, error) {
	return b.beginner.beginFn(ctx)
}

// committerValueMock was added to avoid to use external dependencies for mocking (value receiver).
type committerValueMock struct {
	committer *committerMock
}

func (c committerValueMock) Commit(ctx context.Context) error {
	return c.committer.commitFn(ctx)
}

func (c committerValueMock) Rollback(ctx context.Context) error {
	return c.committer.rollbackFn(ctx)
}

// refreshOperatorMock delegates context storage to the standard operator while
// allowing tests to replace or hide the transaction after the callback starts.
type refreshOperatorMock struct {
	*ContextOperator[*beginnerMock[*committerMock], *committerMock]
	replacement *committerMock
	missing     bool
	extracts    int
}

func (o *refreshOperatorMock) Extract(ctx context.Context) (*committerMock, bool) {
	o.extracts++
	if o.missing {
		return nil, false
	}
	if o.replacement != nil {
		return o.replacement, true
	}
	return o.ContextOperator.Extract(ctx)
}
