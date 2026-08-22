package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// SQLite is a wrapper around database/sql.
type SQLite struct {
	DB *sql.DB
}

// New opens SQLite, applies connection settings and verifies the connection.
func New(ctx context.Context, cfg Config) (*SQLite, error) {
	cfg.Fix()

	dsn, err := buildDSN(cfg)
	if err != nil {
		return nil, fmt.Errorf("build sqlite DSN: %w", err)
	}

	db, err := sql.Open("sqlite", dsn)
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

// buildDSN constructs a SQLite DSN from the given configuration.
func buildDSN(cfg Config) (string, error) {
	path := cfg.Path
	if path != ":memory:" {
		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolve database path: %w", err)
		}

		if err = os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}

		path = filepath.ToSlash(absolutePath)
	}

	query := make(url.Values)
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", cfg.BusyTimeout.Microseconds()))
	query.Add("_pragma", fmt.Sprintf("journal_mode(%s)", cfg.JournalMode))

	return fmt.Sprintf("file:%s?%s", path, query.Encode()), nil
}

// Close closes the underlying connection pool.
func (s *SQLite) Close() error {
	return s.DB.Close()
}
