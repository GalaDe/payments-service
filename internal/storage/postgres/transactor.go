package postgres

import (
	"context"
	"fmt"
	"log"

	"github.com/GalaDe/payments-service/internal/domain"
	orm "github.com/GalaDe/payments-service/internal/sqlc"
)

type PostgresTransactor struct {
	conn *Postgres
	orm  *orm.Queries
}

type PostgresTransactorInterface interface {
	domain.Transactor
}

func NewPostgresTransactor(conn *Postgres, orm *orm.Queries) *PostgresTransactor {
	return &PostgresTransactor{conn, orm}
}

// WithinTx runs queries within a transaction
//
// The transaction commits when functions are finished without error
// and is rolledback otherwise.
// ref: https://www.kaznacheev.me/posts/en/clean-transactions-in-hexagon/
func (p *PostgresTransactor) WithinTransaction(ctx context.Context, txFunc func(ctx context.Context) error) error {
	tx, err := p.conn.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error begin tx: %w", err)
	}

	// run callback
	err = txFunc(InjectTx(ctx, tx))
	if err != nil {
		// if err, rollback
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			log.Printf("rollback tx: %v", errRollback)
		}
		return err
	}
	// if no err, commit
	if errCommit := tx.Commit(ctx); errCommit != nil {
		log.Printf("commit tx: %v", err)
	}

	return nil
}

func (p *PostgresTransactor) WithQtx(ctx context.Context) orm.Querier {
	return p.orm.WithTx(ExtractTx(ctx))
}
