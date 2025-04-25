package gorm

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type (
	repository interface {
		RawInsert(ctx context.Context, val string) error
		Insert(ctx context.Context, text Text) error
	}

	useCase interface {
		CreateTextRecords(ctx context.Context, text string) error
		CreateText(ctx context.Context, text Text) error
	}

	transactor interface {
		WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error)
		GetExecutor(ctx context.Context) *gorm.DB
	}
)

type UseCases struct {
	useCaseA useCase
	useCaseB useCase

	transactor transactor
}

func NewUseCases(useCaseA useCase, useCaseB useCase, transactor transactor) *UseCases {
	return &UseCases{
		useCaseA:   useCaseA,
		useCaseB:   useCaseB,
		transactor: transactor,
	}
}

func (u *UseCases) CreateTextRecords(ctx context.Context, text string) error {
	return u.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := u.useCaseA.CreateTextRecords(ctx, text)
		if err != nil {
			return fmt.Errorf("text usecase A: %w", err)
		}

		err = u.useCaseA.CreateTextRecords(ctx, text)
		if err != nil {
			return fmt.Errorf("text usecase B: %w", err)
		}
		return nil
	})
}

type UseCase struct {
	textRepoA repository
	textRepoB repository

	transactor transactor
}

func NewUseCase(textRepoA repository, textRepoB repository, transactor transactor) *UseCase {
	return &UseCase{
		textRepoA:  textRepoA,
		textRepoB:  textRepoB,
		transactor: transactor,
	}
}

func (u *UseCase) CreateTextRecords(ctx context.Context, text string) error {
	return u.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := u.textRepoA.RawInsert(ctx, text)
		if err != nil {
			return fmt.Errorf("text repo A: %w", err)
		}

		err = u.textRepoB.RawInsert(ctx, text)
		if err != nil {
			return fmt.Errorf("text repo B: %w", err)
		}
		return nil
	})
}

func (u *UseCase) CreateText(ctx context.Context, text Text) error {
	return u.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := u.textRepoA.Insert(ctx, text)
		if err != nil {
			return fmt.Errorf("text repo A: %w", err)
		}

		err = u.textRepoB.Insert(ctx, text)
		if err != nil {
			return fmt.Errorf("text repo B: %w", err)
		}
		return nil
	})
}
