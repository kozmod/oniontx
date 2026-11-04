package stdlib

import (
	"context"
	"fmt"
	"strings"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

type (
	repoTransactor interface {
		TryGetTx(ctx context.Context) (Tx, bool)
		GetExecutor(ctx context.Context) Executor
		// InjectSavepointTx supports transaction replacement in the savepoint tests.
		InjectSavepointTx(ctx context.Context, tx Tx, savepoint string)
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

// Savepoint creates a savepoint in an active transaction and installs its wrapper.
// It is used by the savepoint tests to demonstrate consumer-defined finalization:
// errors after this call preserve writes made before the savepoint.
func (r *TextRepository) Savepoint(ctx context.Context, savepoint string) error {
	tx, ok := r.transactor.TryGetTx(ctx)
	if !ok {
		return fmt.Errorf("stdlib create savepoint: no active transaction")
	}
	savepoint = `"` + strings.ReplaceAll(savepoint, `"`, `""`) + `"`
	_, err := tx.ExecContext(ctx, "SAVEPOINT "+savepoint)
	if err != nil {
		return fmt.Errorf("stdlib create savepoint: %w", err)
	}
	r.transactor.InjectSavepointTx(ctx, tx, savepoint)
	return nil
}
