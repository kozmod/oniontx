package gorm

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/kozmod/oniontx/test/integration/internal/entity"
)

type (
	repoTransactor interface {
		GetExecutor(ctx context.Context) *gorm.DB
	}
)

type Text struct {
	Val string `gorm:"column:val"`
}

func (t *Text) TableName() string {
	return "gorm"
}

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

func (r *TextRepository) RawInsert(ctx context.Context, val string) error {
	if r.errorExpected {
		return entity.ErrExpected
	}
	ex := r.transactor.GetExecutor(ctx)
	ex = ex.Exec(`INSERT INTO gorm (val) VALUES ($1)`, val)
	if ex.Error != nil {
		return fmt.Errorf("gorm repository - raw insert: %w", ex.Error)
	}
	return nil
}

func (r *TextRepository) Insert(ctx context.Context, text Text) error {
	if r.errorExpected {
		return entity.ErrExpected
	}
	ex := r.transactor.GetExecutor(ctx)
	ex = ex.Create(text)
	if ex.Error != nil {
		return fmt.Errorf("gorm repository - raw insert: %w", ex.Error)
	}
	return nil
}
