package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "modernc.org/sqlite"
)

// NewSQLiteDB opens (or creates) a SQLite database with WAL mode enabled.
func NewSQLiteDB(_ context.Context, path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}

	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	sdb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Single writer avoids SQLITE_BUSY under concurrent requests.
	sdb.SetMaxOpenConns(1)

	if err := sdb.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return sdb, nil
}

// MigrateSQLite runs the three embedded SQL migration files in order.
// It is idempotent — all statements use IF NOT EXISTS / INSERT OR IGNORE.
func MigrateSQLite(sdb *sql.DB, _ string) error {
	_, filename, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(filename), "migrations", "sqlite")

	files := []string{
		"001_initial_schema.sql",
		"002_model_pricing.sql",
		"003_payment_tables.sql",
	}

	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err := sdb.Exec(string(data)); err != nil {
			return fmt.Errorf("apply migration %s: %w", f, err)
		}
	}
	return nil
}
