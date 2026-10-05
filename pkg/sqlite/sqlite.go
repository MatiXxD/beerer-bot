package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// SQLite is a wrapper around database/sql for SQLite database.
type SQLite struct {
	DB *sql.DB
}

// New opens SQLite, applies connection settings, and verifies the connection.
func New(ctx context.Context, cfg Config) (*SQLite, error) {
	cfg.Fix()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid sqlite config: %w", err)
	}

	path, err := preparePath(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("prepare sqlite path: %w", err)
	}

	db, err := sql.Open("sqlite", buildDSN(path, cfg))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	if err = db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return &SQLite{DB: db}, nil
}

// preparePath resolves the database path and creates its parent directory.
func preparePath(path string) (string, error) {
	if path == inMemoryPath {
		return path, nil
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve database path: %w", err)
	}

	if err = os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
		return "", fmt.Errorf("failed to create database directory: %w", err)
	}

	return filepath.ToSlash(absolutePath), nil // replace '\' with '/' to support windows
}

// buildDSN constructs a SQLite DSN from the prepared path and configuration.
func buildDSN(path string, cfg Config) string {
	query := make(url.Values)
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", cfg.BusyTimeout.Milliseconds()))
	query.Add("_pragma", fmt.Sprintf("journal_mode(%s)", cfg.JournalMode))

	// take the write lock at BEGIN of transactions
	query.Add("_txlock", "immediate")

	return fmt.Sprintf("file:%s?%s", path, query.Encode())
}

// Close closes the underlying connection pool.
func (s *SQLite) Close() error {
	return s.DB.Close()
}
