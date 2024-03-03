package stdlib

import (
	"context"
	"fmt"

	ostdlib "github.com/kozmod/oniontx/stdlib"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

type (
	repoTransactor interface {
		GetExecutor(ctx context.Context) ostdlib.Executor
	}
)

type TextRepository struct {
	transactor    repoTransactor
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
		return fmt.Errorf("stdlib repository: %w", err)
	}
	return nil
}
