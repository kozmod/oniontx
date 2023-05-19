# OnionTx
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/kozmod/oniontx)
![GitHub MIT license](https://img.shields.io/github/license/kozmod/progen)

The utility for transferring transaction management of the stdlib to the service layer.

## Example
```go
type RepositoryA struct {
	db *sql.DB
}

func (r RepositoryA) Do(ctx context.Context) error {
	executor := oniontx.ExtractExecutorOrDefault(ctx, r.db)
	_, err := executor.ExecContext(ctx, "UPDATE some_A SET value = 1")
	if err != nil {
		return fmt.Errorf("update 'some_A': %w", err)
	}
	return nil
}

type RepositoryB struct {
	db *sql.DB
}

func (r RepositoryB) Do(ctx context.Context) error {
	executor := oniontx.ExtractExecutorOrDefault(ctx, r.db)
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