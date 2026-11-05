# mtx

## Custom transaction finalization (savepoints, wrappers)

By default, `WithinTx` finalizes the transaction it started: it commits on success
and rolls back on error. Some scenarios require the callback to **replace** the
transaction before finalization — for example, to implement PostgreSQL savepoints
with partial rollback, or to install an observability wrapper around `Tx`.

For these cases, use `WithTxRefreshFromContext(true)`. When enabled, `WithinTx`
re-reads the transaction from the callback context before top-level
commit/rollback, so a replacement installed by the callback is used for
finalization. Nested calls still leave finalization to the outermost call.
If extraction fails, the original transaction is used. The option is disabled
by default and returns a new transactor without modifying the original.

This requires a custom `CtxOperator` that makes transaction replacements visible
through the context retained by `WithinTx`. The default context operator stores
values using `context.WithValue`; calling its `Inject` inside the callback creates
a new context and does not replace the transaction in the context retained by
`WithinTx`.

The following example uses consumer-defined `Wrapper`, `Tx`, `operator`, and
repositories. `repo.Savepoint` creates a SQL savepoint and installs a transaction
wrapper through the custom operator.

```go
transactor := mtx.NewTransactor[*Wrapper, Tx](db, operator).
    // Allow the callback to replace the transaction before finalization.
    WithTxRefreshFromContext(true)

err := transactor.WithinTx(ctx, func(ctx context.Context) error {
    if err := repo.Insert(ctx, "before"); err != nil {
        return err
    }

    // Create a savepoint and install a wrapper that performs partial rollback.
    if err := repo.Savepoint(ctx, "sp1"); err != nil {
        return err
    }

    if err := repo.Insert(ctx, "after"); err != nil {
        return err
    }

    // Any error below triggers the wrapper's Rollback, which rolls back
    // only the writes after the savepoint and commits the rest.
    return repoB.Insert(ctx, "fail")
})
```

The replacement must wrap and complete the original transaction when its
`Commit` or `Rollback` returns. In this example, `Rollback` rolls back to the
savepoint and commits earlier writes. `WithinTx` still returns the callback
error, even though those earlier writes have been committed. The same `Rollback`
method is invoked during panic cleanup, so the example also preserves earlier
writes on panic. This behavior is implemented by the consumer's wrapper;
the option itself does not implement savepoints or partial rollback.

See the [savepoint integration test](../test/integration/internal/stdlib/savepoint/savepoint_test.go)
for a working example. It's implementation
includes a [mutable context operator](../test/integration/internal/stdlib/operator.go),
a [transaction wrapper](../test/integration/internal/stdlib/transactor.go), and
the [repository's savepoint method](../test/integration/internal/stdlib/repository.go).
