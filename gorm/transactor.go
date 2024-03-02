package gorm

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"github.com/kozmod/oniontx"
)

// dbWrapper wraps [gorm.DB] and implements [oniontx.TxBeginner].
type dbWrapper struct {
	*gorm.DB
}

// BeginTx starts a transaction.
func (w *dbWrapper) BeginTx(_ context.Context, opts ...oniontx.Option[*sql.TxOptions]) (*dbWrapper, error) {
	var txOptions sql.TxOptions
	for _, opt := range opts {
		opt.Apply(&txOptions)
	}
	tx := w.Begin(&txOptions)
	return &dbWrapper{DB: tx}, nil
}

// Rollback aborts the transaction.
func (w *dbWrapper) Rollback(_ context.Context) error {
	tx := w.DB
	tx.Rollback()
	return nil
}

// Commit commits the transaction.
func (w *dbWrapper) Commit(_ context.Context) error {
	tx := w.DB
	tx.Commit()
	return nil
}

// Transactor manage a transaction for single [gorm.DB] instance.
type Transactor struct {
	*oniontx.Transactor[*dbWrapper, *dbWrapper, *sql.TxOptions]
}

// NewTransactor returns new [Transactor] ([gorm] implementation).
func NewTransactor(db *gorm.DB) *Transactor {
	var (
		base       = dbWrapper{DB: db}
		operator   = oniontx.NewContextOperator[*dbWrapper, *dbWrapper](&base)
		transactor = oniontx.NewTransactor[*dbWrapper, *dbWrapper, *sql.TxOptions](&base, operator)
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
