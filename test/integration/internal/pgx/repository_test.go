package pgx

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
	_, err := ex.Exec(ctx, `INSERT INTO pgx (val) VALUES ($1)`, val)
	if err != nil {
		return fmt.Errorf("pgx repository - raw insert: %w", err)
	}
	return nil
}
