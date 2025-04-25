package gorm

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"github.com/kozmod/oniontx"
)

// Wrapper wraps [gorm.DB] and implements [oniontx.TxBeginner].
type Wrapper struct {
	*gorm.DB

	// txOptions is options for gorm transactions begin.
	txOptions sql.TxOptions
}

// NewDB returns [gorm.DB] wrapper with transaction's options.
func NewDB(db *gorm.DB, options sql.TxOptions) *Wrapper {
	return &Wrapper{
		DB:        db,
		txOptions: options,
	}
}

// BeginTx starts a transaction.
func (w *Wrapper) BeginTx(_ context.Context) (*Wrapper, error) {
	tx := w.Begin(&w.txOptions)
	return &Wrapper{DB: tx}, nil
}

// Rollback aborts the transaction.
func (w *Wrapper) Rollback(_ context.Context) error {
	tx := w.DB
	tx.Rollback()
	return nil
}

// Commit commits the transaction.
func (w *Wrapper) Commit(_ context.Context) error {
	tx := w.DB
	tx.Commit()
	return nil
}

// Transactor manage a transaction for single [gorm.DB] instance.
type Transactor struct {
	*oniontx.Transactor[*Wrapper, *Wrapper]
}

// NewTransactor returns new [Transactor] ([gorm] implementation).
func NewTransactor(db *Wrapper) *Transactor {
	var (
		operator   = oniontx.NewContextOperator[*Wrapper, *Wrapper](db)
		transactor = oniontx.NewTransactor[*Wrapper, *Wrapper](db, operator)
	)
	return &Transactor{
		Transactor: transactor,
	}
}

// TryGetTx returns pointer of [gorm.DB]([gorm.TxBeginner] or [gorm.ConnPoolBeginner]) and "true" from [context.Context] or return `false`.
func (t *Transactor) TryGetTx(ctx context.Context) (*gorm.DB, bool) {
	wrapper, ok := t.Transactor.TryGetTx(ctx)
	if !ok || wrapper == nil || wrapper.DB == nil {
		return nil, false
	}
	return wrapper.DB, true
}

// TxBeginner returns pointer of [gorm.DB].
func (t *Transactor) TxBeginner() *gorm.DB {
	return t.Transactor.TxBeginner().DB
}

// GetExecutor returns pointer of [*gorm.DB] with transaction state.
func (t *Transactor) GetExecutor(ctx context.Context) *gorm.DB {
	exec, ok := t.TryGetTx(ctx)
	if !ok {
		exec = t.TxBeginner()
	}
	return exec
}
