package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigFix(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want Config
	}{
		{name: "defaults", cfg: Config{MaxIdleConns: -1}, want: Config{Path: defaultPath, BusyTimeout: defaultBusyTimeout, JournalMode: defaultJournalMode, MaxOpenConns: defaultMaxOpenConns, MaxIdleConns: defaultMaxIdleConns}},
		{name: "configured values", cfg: Config{Path: "custom.db", BusyTimeout: time.Second, JournalMode: "DELETE", MaxOpenConns: 2, MaxIdleConns: 1}, want: Config{Path: "custom.db", BusyTimeout: time.Second, JournalMode: "delete", MaxOpenConns: 2, MaxIdleConns: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.Fix()
			require.Equal(t, tt.want, tt.cfg)
		})
	}
}

func TestNew(t *testing.T) {
	storage, err := New(context.Background(), Config{Path: filepath.Join(t.TempDir(), "nested", "beerer.db")})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, storage.Close()) })

	_, err = storage.DB.Exec(`CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id));`)
	require.NoError(t, err)
	_, err = storage.DB.Exec("INSERT INTO child(id, parent_id) VALUES (?, ?)", 1, 404)
	require.Error(t, err, "foreign_keys pragma must be enabled")

	var journalMode string
	require.NoError(t, storage.DB.QueryRow("PRAGMA journal_mode").Scan(&journalMode))
	require.Equal(t, "wal", journalMode)
}

func TestNewInMemory(t *testing.T) {
	storage, err := New(context.Background(), Config{Path: ":memory:", JournalMode: "MEMORY"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, storage.Close()) })

	_, err = storage.DB.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
	_, err = storage.DB.Exec("INSERT INTO test(id) VALUES (?)", 1)
	require.NoError(t, err)

	var count int
	require.NoError(t, storage.DB.QueryRow("SELECT COUNT(*) FROM test").Scan(&count))
	require.Equal(t, 1, count)
}

func TestParseError(t *testing.T) {
	storage, err := New(context.Background(), Config{Path: filepath.Join(t.TempDir(), "errors.db")})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, storage.Close()) })
	_, err = storage.DB.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT UNIQUE NOT NULL)")
	require.NoError(t, err)
	_, err = storage.DB.Exec("INSERT INTO users(id, name) VALUES (?, ?)", 1, "alice")
	require.NoError(t, err)
	_, duplicateErr := storage.DB.Exec("INSERT INTO users(id, name) VALUES (?, ?)", 2, "alice")
	require.Error(t, duplicateErr)
	unknownErr := errors.New("boom")

	tests := []struct {
		name       string
		err        error
		want       error
		recognized bool
	}{
		{name: "nil", err: nil},
		{name: "not found", err: sql.ErrNoRows, want: ErrNotFound, recognized: true},
		{name: "unique constraint", err: duplicateErr, want: ErrAlreadyExists, recognized: true},
		{name: "unknown", err: unknownErr, want: unknownErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, recognized := ParseError(tt.err)
			require.Equal(t, tt.recognized, recognized)
			if tt.err == nil {
				require.NoError(t, got)
				return
			}
			require.ErrorIs(t, got, tt.want)
		})
	}
}
