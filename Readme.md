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
## Main idea
# <img src=".github/assets/clean_arch.png" alt="drawing"  width="250" />
Move a maintaining of transactions to `Application` layer using owner defined contract.
___

## Example
1️⃣ Execution different repositories with the same `sql.DB` instance
```go
package repoA

import (
	"context"
	"fmt"

	"github.com/kozmod/oniontx"
)

type RepositoryA struct {
	Transactor *oniontx.Transactor
}

func (r RepositoryA) Do(ctx context.Context) error {
	executor := r.Transactor.ExtractExecutorOrDefault(ctx)
	_, err := executor.ExecContext(ctx, "UPDATE some_A SET value = 1")
	if err != nil {
		return fmt.Errorf("update 'some_A': %w", err)
	}
	return nil
}
```
```go
package repoB

import (
	"context"
	"fmt"
	
	"github.com/kozmod/oniontx"
)

type RepositoryB struct {
	Transactor *oniontx.Transactor
}

func (r RepositoryB) Do(ctx context.Context) error {
	executor := r.Transactor.ExtractExecutorOrDefault(ctx)
	_, err := executor.ExecContext(ctx, "UPDATE some_B SET value = 1")
	if err != nil {
		return fmt.Errorf("update 'some_B': %w", err)
	}
	return nil
}
```
```go
package usecase

import (
	"context"
	"fmt"
)

// transactor is the contract of  the oniontx.Transactor
type transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) (err error)
}

// Repo is the contract of repositories
type repo interface {
	Do(ctx context.Context) error
}

type Usecase struct {
	RepositoryA repo
	RepositoryB repo

	Transactor transactor
}

func  (s *Usecase)Do(ctx context.Context) error{
	err := s.Transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.RepositoryA.Do(ctx); err != nil {
			return fmt.Errorf("call repositoryA: %+v", err)
		}
		if err := s.RepositoryB.Do(ctx); err != nil {
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
```go
package main

import (
	"context"
	"os"
	
	"github.com/kozmod/oniontx"
	
	"github.com/user/some_project/internal/repoA"
	"github.com/user/some_project/internal/repoB"
	"github.com/user/some_project/internal/usecase"
)


func main() {
	var (
		//db *sql.DB = // ...

		transactor  = oniontx.NewTransactor(db)
		repositoryA = repoA.RepositoryA{
			Transactor: transactor,
		}
		repositoryB = repoB.RepositoryB{
			Transactor: transactor,
		}

		service = usecase.Usecase{
			RepositoryA: &repositoryA,
			RepositoryB: &repositoryB,
			Transactor:  transactor,
		}
	)

	err := service.Do(context.Background())
	if err != nil {
		os.Exit(1)
	}
}

```
2️⃣ Start transaction with `sql.TxOptions`
```go
func (s *Service) Do(ctx context.Context) error {
	err := s.Transactor.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.RepositoryA.Do(ctx); err != nil {
			return fmt.Errorf("call repositoryA: %+v", err)
		}
		if err := s.RepositoryB.Do(ctx); err != nil {
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