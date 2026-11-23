package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection configured for multi-tenant persistence.
type DB struct {
	sql *sql.DB
}

// Open opens a SQLite database at dsn, enables foreign keys and busy timeout,
// and configures WAL journaling for file-backed databases.
func Open(dsn string) (*DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("sqlite: dsn must not be empty")
	}

	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", dsn, err)
	}

	isMemory := dsn == ":memory:" || strings.Contains(dsn, "mode=memory")
	if isMemory {
		raw.SetMaxOpenConns(1)
	}

	if err := raw.Ping(); err != nil {
		_ = raw.Close()
		return nil, fmt.Errorf("sqlite: ping: %w", err)
	}

	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	}
	if !isMemory {
		pragmas = append(pragmas, "PRAGMA journal_mode = WAL;")
	}

	for _, p := range pragmas {
		if _, err := raw.Exec(p); err != nil {
			_ = raw.Close()
			return nil, fmt.Errorf("sqlite: exec %q: %w", p, err)
		}
	}

	return &DB{sql: raw}, nil
}

// Close closes the underlying SQLite connection.
func (d *DB) Close() error {
	if d == nil || d.sql == nil {
		return nil
	}
	return d.sql.Close()
}

// SQL exposes the underlying *sql.DB for adapter-internal queries and tests.
func (d *DB) SQL() *sql.DB {
	return d.sql
}

// Migrate applies all idempotent schema migrations.
func (d *DB) Migrate(ctx context.Context) error {
	for _, stmt := range schemaMigrations {
		if _, err := d.sql.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("sqlite: migrate: %w", err)
		}
	}
	return nil
}
