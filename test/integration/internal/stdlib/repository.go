package stdlib

import (
	"context"
	"fmt"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

type (
	repoTransactor interface {
		GetExecutor(ctx context.Context) Executor
	}
)

type TextRepository struct {
	transactor repoTransactor

	// errorExpected - need to emulate error
	errorExpected bool
}

func NewTextRepository(transactor repoTransactor, errorExpected bool) *TextRepository {
	return &TextRepository{
		transactor:    transactor,
		errorExpected: errorExpected,
	}
}

func (r *TextRepository) Insert(ctx context.Context, val string) error {
	if r.errorExpected {
		return entity.ErrExpected
	}
	ex := r.transactor.GetExecutor(ctx)
	_, err := ex.ExecContext(ctx, `INSERT INTO stdlib (val) VALUES ($1)`, val)
	if err != nil {
		return fmt.Errorf("stdlib repository insert: %w", err)
	}
	return nil
}

func (r *TextRepository) Delete(ctx context.Context, val string) error {
	ex := r.transactor.GetExecutor(ctx)
	_, err := ex.ExecContext(ctx, `DELETE FROM stdlib WHERE val = $1`, val)
	if err != nil {
		return fmt.Errorf("stdlib repository delete: %w", err)
	}
	return nil
}
