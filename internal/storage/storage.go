package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cdguru/micro-health-checker/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type Record struct {
	CheckID   string    `json:"check_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Success   bool      `json:"success"`
	StartedAt time.Time `json:"started_at"`
	Duration  float64   `json:"duration_ms"`
	Error     string    `json:"error,omitempty"`
}

type Store interface {
	Record(context.Context, Record) error
	History(context.Context, string, int) ([]Record, error)
	Prune(context.Context, time.Time) (int64, error)
	Ping(context.Context) error
	Close() error
}

type sqlStore struct {
	db      *sql.DB
	dialect string
}

func Open(ctx context.Context, cfg config.StorageConfig) (Store, error) {
	var driver, dsn string
	switch cfg.Type {
	case "sqlite":
		driver = "sqlite"
		if cfg.SQLite.Path == ":memory:" {
			dsn = "file:micro-health-checker?mode=memory&cache=shared&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
		} else {
			dsn = "file:" + filepath.ToSlash(cfg.SQLite.Path) + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)"
		}
	case "postgres":
		driver, dsn = "pgx", cfg.Postgres.DSN
	default:
		return nil, fmt.Errorf("unsupported storage type %q", cfg.Type)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s storage: %w", cfg.Type, err)
	}
	if cfg.Type == "sqlite" {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(2)
	}
	store := &sqlStore{db: db, dialect: cfg.Type}
	if err := store.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to %s storage: %w", cfg.Type, err)
	}
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *sqlStore) migrate(ctx context.Context) error {
	var statements []string
	if s.dialect == "postgres" {
		statements = []string{
			`CREATE TABLE IF NOT EXISTS check_results (
				id BIGSERIAL PRIMARY KEY,
				check_id VARCHAR(64) NOT NULL,
				check_name TEXT NOT NULL,
				check_type VARCHAR(32) NOT NULL,
				success SMALLINT NOT NULL,
				started_at_unix_ms BIGINT NOT NULL,
				duration_ms DOUBLE PRECISION NOT NULL,
				error_text TEXT NOT NULL DEFAULT ''
			)`,
			`CREATE INDEX IF NOT EXISTS idx_check_results_lookup ON check_results (check_id, started_at_unix_ms DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_check_results_retention ON check_results (started_at_unix_ms)`,
		}
	} else {
		statements = []string{
			`CREATE TABLE IF NOT EXISTS check_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				check_id TEXT NOT NULL,
				check_name TEXT NOT NULL,
				check_type TEXT NOT NULL,
				success INTEGER NOT NULL,
				started_at_unix_ms INTEGER NOT NULL,
				duration_ms REAL NOT NULL,
				error_text TEXT NOT NULL DEFAULT ''
			)`,
			`CREATE INDEX IF NOT EXISTS idx_check_results_lookup ON check_results (check_id, started_at_unix_ms DESC)`,
			`CREATE INDEX IF NOT EXISTS idx_check_results_retention ON check_results (started_at_unix_ms)`,
		}
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate %s storage: %w", s.dialect, err)
		}
	}
	return nil
}

func (s *sqlStore) Record(ctx context.Context, record Record) error {
	success := 0
	if record.Success {
		success = 1
	}
	query := `INSERT INTO check_results
		(check_id, check_name, check_type, success, started_at_unix_ms, duration_ms, error_text)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	if s.dialect == "postgres" {
		query = `INSERT INTO check_results
			(check_id, check_name, check_type, success, started_at_unix_ms, duration_ms, error_text)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`
	}
	_, err := s.db.ExecContext(ctx, query, record.CheckID, record.Name, record.Type, success, record.StartedAt.UnixMilli(), record.Duration, record.Error)
	if err != nil {
		return fmt.Errorf("record check result: %w", err)
	}
	return nil
}

func (s *sqlStore) History(ctx context.Context, checkID string, limit int) ([]Record, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	query := `SELECT check_id, check_name, check_type, success, started_at_unix_ms, duration_ms, error_text
		FROM check_results WHERE check_id = ? ORDER BY started_at_unix_ms DESC LIMIT ?`
	args := []any{checkID, limit}
	if s.dialect == "postgres" {
		query = `SELECT check_id, check_name, check_type, success, started_at_unix_ms, duration_ms, error_text
			FROM check_results WHERE check_id = $1 ORDER BY started_at_unix_ms DESC LIMIT $2`
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query check history: %w", err)
	}
	defer rows.Close()

	results := make([]Record, 0, limit)
	for rows.Next() {
		var record Record
		var success int
		var startedAt int64
		if err := rows.Scan(&record.CheckID, &record.Name, &record.Type, &success, &startedAt, &record.Duration, &record.Error); err != nil {
			return nil, fmt.Errorf("scan check history: %w", err)
		}
		record.Success = success == 1
		record.StartedAt = time.UnixMilli(startedAt).UTC()
		results = append(results, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate check history: %w", err)
	}
	return results, nil
}

func (s *sqlStore) Prune(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM check_results WHERE started_at_unix_ms < ?`
	if s.dialect == "postgres" {
		query = `DELETE FROM check_results WHERE started_at_unix_ms < $1`
	}
	result, err := s.db.ExecContext(ctx, query, before.UnixMilli())
	if err != nil {
		return 0, fmt.Errorf("prune check history: %w", err)
	}
	return result.RowsAffected()
}

func (s *sqlStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func (s *sqlStore) Close() error                   { return s.db.Close() }
