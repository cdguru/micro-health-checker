package checker

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/christiandente/micro-health-checker/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresChecker struct {
	db    *sql.DB
	query string
}

func newPostgres(cfg *config.PostgresConfig) (Checker, error) {
	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("configure postgres client: %w", err)
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	return &postgresChecker{db: db, query: strings.TrimSpace(cfg.Query)}, nil
}

func (c *postgresChecker) Check(ctx context.Context) error {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("postgres connection failed: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback is best effort for a read-only health transaction

	rows, err := tx.QueryContext(ctx, c.query)
	if err != nil {
		return fmt.Errorf("postgres health query failed: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return fmt.Errorf("postgres health query failed: %w", err)
		}
		return fmt.Errorf("postgres health query returned no rows")
	}
	return nil
}

func (c *postgresChecker) Close() error { return c.db.Close() }
