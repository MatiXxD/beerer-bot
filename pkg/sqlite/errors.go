package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	modernsqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var (
	ErrAlreadyExists = errors.New("already exists")
	ErrNotFound      = errors.New("not found")
)

// ParseError converts driver-specific errors into errors shared by repositories.
func ParseError(err error) (error, bool) {
	if err == nil {
		return nil, false
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound, true
	}

	var sqliteErr *modernsqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() {
		case sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, sqlite3.SQLITE_CONSTRAINT_UNIQUE:
			return ErrAlreadyExists, true
		}
	}

	return fmt.Errorf("sqlite error: %w", err), false
}
