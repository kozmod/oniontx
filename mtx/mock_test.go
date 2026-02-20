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
	return c.committer.commitFn(ctx)
}
