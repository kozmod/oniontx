package mockery

//go:generate mockery --dir=. --name=repository --outpkg=mockery --output=.  --filename=mock_repository_test.go --structname=repositoryMock
//go:generate mockery --dir=. --name=transactor --outpkg=mockery --output=.  --filename=mock_transactor_test.go --structname=transactorMock
//go:generate git add .

import (
	"context"
	"fmt"
)

type (
	repository interface {
		Insert(ctx context.Context, val string) error
	}

	transactor interface {
		WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error)
	}
)

type UseCase struct {
	textRepoA repository
	textRepoB repository

	transactor transactor
}

func (u *UseCase) CreateTextRecords(ctx context.Context, text string) error {
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
