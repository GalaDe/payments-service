package postgres

import (
	"context"

	"github.com/jackc/pgx/v4"
)

// Source: https://www.kaznacheev.me/posts/en/clean-transactions-in-hexagon/

// txKey is a context key for holding pgx.Tx
type txKey struct{}

// injectTx injects transactions into context
func InjectTx(ctx context.Context, tx pgx.Tx) context.Context {
	if tx == nil {
		return ctx
	}
	return context.WithValue(ctx, txKey{}, tx)
}

// extractTx extracts transaction from context
func ExtractTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}
