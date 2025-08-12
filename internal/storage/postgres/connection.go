package postgres

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	orm "github.com/GalaDe/payments-service/internal/sqlc"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	_defaultMaxPoolSize  = 3
	_defaultConnAttempts = 5
	_defaultConnTimeout  = time.Second
)

type PostgresSecret struct {
	Name         string `json:"name"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Host         string `json:"host"`
	Port         uint16 `json:"port"`
	DBConnString string `json:"dbConnString"`
}

type Postgres struct {
	maxPoolSize  int
	connAttempts int
	connTimeout  time.Duration

	Builder squirrel.StatementBuilderType
	Pool    *pgxpool.Pool
	Orm     *orm.Queries
}

type PostgresInterface interface {
	Ping() error
	Close()
}

func NewPostgresDB(cfg *PostgresSecret, opts ...Option) (*Postgres, error) {
	pg := &Postgres{
		maxPoolSize:  _defaultMaxPoolSize,
		connAttempts: _defaultConnAttempts,
		connTimeout:  _defaultConnTimeout,
		Orm:          &orm.Queries{},
	}

	for _, opt := range opts {
		opt(pg)
	}

	pg.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Respect provided DSN; default sslmode to disable if not set.
	dsn := cfg.DBConnString
	if !strings.Contains(dsn, "sslmode=") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn = dsn + sep + "sslmode=disable"
	}

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres - NewPostgres - pgxpool.ParseConfig: %w", err)
	}
	poolConfig.MaxConns = int32(pg.maxPoolSize)

	for pg.connAttempts > 0 {
		pg.Pool, err = pgxpool.ConnectConfig(context.Background(), poolConfig)
		if err == nil {
			break
		}
		log.Printf("Postgres is trying to connect, attempts left: %d - %v", pg.connAttempts, err)
		time.Sleep(pg.connTimeout)
		pg.connAttempts--
	}
	if err != nil {
		return nil, fmt.Errorf("postgres - NewPostgres - connAttempts == 0: %w", err)
	}

	return pg, nil
}

func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}

// Ping is a method that pings the database
func (p *Postgres) Ping() error {
	conn, err := p.Pool.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()
	return conn.Conn().Ping(context.Background())
}
