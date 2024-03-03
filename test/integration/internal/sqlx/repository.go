package sqlx

import (
	"context"
	"fmt"

	osqlx "github.com/kozmod/oniontx/sqlx"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

type (
	repoTransactor interface {
		GetExecutor(ctx context.Context) osqlx.Executor
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
	_, err := ex.ExecContext(ctx, `INSERT INTO sqlx (val) VALUES ($1)`, val)
	if err != nil {
		return fmt.Errorf("sqlx repository - raw insert: %w", err)
	}
	return nil
}
