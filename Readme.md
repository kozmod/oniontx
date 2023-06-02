# OnionTx <img align="right" src=".github/assets/onion_1.png" alt="drawing"  width="90" />
[![test](https://github.com/kozmod/oniontx/actions/workflows/test.yml/badge.svg)](https://github.com/kozmod/oniontx/actions/workflows/test.yml)
[![Release](https://github.com/kozmod/oniontx/actions/workflows/release.yml/badge.svg)](https://github.com/kozmod/oniontx/actions/workflows/release.yml)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/kozmod/oniontx)
[![Go Report Card](https://goreportcard.com/badge/github.com/kozmod/oniontx)](https://goreportcard.com/report/github.com/kozmod/oniontx)
![GitHub release date](https://img.shields.io/github/release-date/kozmod/oniontx)
![GitHub last commit](https://img.shields.io/github/last-commit/kozmod/oniontx)
[![GitHub MIT license](https://img.shields.io/github/license/kozmod/oniontx)](https://github.com/kozmod/oniontx/blob/dev/LICENSE)

The utility for transferring transaction management of the stdlib to the service layer.

**NOTE**: Transactor was developed to work with only the same instance of tha `*sql.DB`
___

## Example
1️⃣Execution different repositories with the same `sql.DB` instance
```go
package some

type RepositoryA struct {
	transactor oniontx.Transactor
}

func (r RepositoryA) Do(ctx context.Context) error {
	executor := r.transactor.ExtractExecutorOrDefault(ctx)
	_, err := executor.ExecContext(ctx, "UPDATE some_A SET value = 1")
	if err != nil {
		return fmt.Errorf("update 'some_A': %w", err)
	}
	return nil
}

type RepositoryB struct {
	transactor oniontx.Transactor
}

func (r RepositoryB) Do(ctx context.Context) error {
	executor := r.transactor.ExtractExecutorOrDefault(ctx)
	_, err := executor.ExecContext(ctx, "UPDATE some_B SET value = 1")
	if err != nil {
		return fmt.Errorf("update 'some_B': %w", err)
	}
	return nil
}

type Service struct {
	repositoryA *RepositoryA
	repositoryB *RepositoryB

	transactor oniontx.Transactor
}

func  (s *Service)Do(ctx context.Context) error{
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.repositoryA.Do(ctx); err != nil {
			return fmt.Errorf("call repositoryA: %+v", err)
		}
		if err := s.repositoryB.Do(ctx); err != nil {
			return fmt.Errorf("call repositoryB: %+v", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("execute: %v", err)
	}
	return nil
}
```
2️⃣ Start transaction with `sql.TxOptions`
```go
package some

func (s *Service) Do(ctx context.Context) error {
	err := s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.repositoryA.Do(ctx); err != nil {
			return fmt.Errorf("call repositoryA: %+v", err)
		}
		if err := s.repositoryB.Do(ctx); err != nil {
			return fmt.Errorf("call repositoryB: %+v", err)
		}
		return nil
	},
		oniontx.WithReadOnly(true),
		oniontx.WithIsolationLevel(6))
	if err != nil {
		return fmt.Errorf("execute: %v", err)
	}
	return nil
}
```