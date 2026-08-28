package mtx

import "context"

var dummyRollbackCtxFactory = func(ctx context.Context) context.Context {
	return ctx
}

type transactorOptions struct {
	rollbackCtxFactory func(ctx context.Context) context.Context
	recoverPanic       bool
}

func defaultTransactorOptions() transactorOptions {
	return transactorOptions{
		rollbackCtxFactory: dummyRollbackCtxFactory,
	}
}
