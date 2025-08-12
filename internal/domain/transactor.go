package domain

import (
	"context"

	orm "github.com/GalaDe/payments-service/internal/sqlc"
)

// Transactor defines a Transaction Port interface
type Transactor interface {
	// Manage a transactions
	WithinTransaction(context.Context, func(ctx context.Context) error) error
	WithQtx(ctx context.Context) orm.Querier
}
