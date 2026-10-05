package sqlite

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	modernsqlite "modernc.org/sqlite"
)

func TestConfigFix(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want Config
	}{
		{name: "defaults", cfg: Config{MaxIdleConns: -1}, want: Config{Path: defaultPath, BusyTimeout: defaultBusyTimeout, JournalMode: defaultJournalMode, MaxOpenConns: defaultMaxOpenConns, MaxIdleConns: defaultMaxOpenConns}},
		{name: "configured values", cfg: Config{Path: "custom.db", BusyTimeout: time.Second, JournalMode: "DELETE", MaxOpenConns: 2, MaxIdleConns: 1}, want: Config{Path: "custom.db", BusyTimeout: time.Second, JournalMode: "delete", MaxOpenConns: 2, MaxIdleConns: 1}},
		{name: "idle conns follow open conns", cfg: Config{MaxOpenConns: 4}, want: Config{Path: defaultPath, BusyTimeout: defaultBusyTimeout, JournalMode: defaultJournalMode, MaxOpenConns: 4, MaxIdleConns: 4}},
		{name: "in-memory keeps single connection", cfg: Config{Path: inMemoryPath, MaxOpenConns: 4, MaxIdleConns: 2}, want: Config{Path: inMemoryPath, BusyTimeout: defaultBusyTimeout, JournalMode: defaultJournalMode, MaxOpenConns: 1, MaxIdleConns: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.Fix()
			require.Equal(t, tt.want, tt.cfg)
		})
	}
}

func TestConfigValidate(t *testing.T) {
	for mode := range journalModes {
		t.Run(mode, func(t *testing.T) {
			cfg := Config{JournalMode: mode}
			require.NoError(t, cfg.Validate())
		})
	}

	t.Run("unsupported", func(t *testing.T) {
		cfg := Config{JournalMode: "wall"}
		require.ErrorContains(t, cfg.Validate(), `unsupported journal mode "wall"`)
	})
}

func TestNew(t *testing.T) {
	ctx := t.Context()
	dir := filepath.Join(t.TempDir(), "nested")
	storage, err := New(ctx, Config{Path: filepath.Join(dir, "beerer.db"), BusyTimeout: 3 * time.Second})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, storage.Close()) })

	info, err := os.Stat(dir)
	require.NoError(t, err)
	require.True(t, info.IsDir())

	_, err = storage.DB.ExecContext(ctx, `CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id));`)
	require.NoError(t, err)
	_, err = storage.DB.ExecContext(ctx, "INSERT INTO child(id, parent_id) VALUES (?, ?)", 1, 404)
	require.Error(t, err, "foreign_keys pragma must be enabled")

	var journalMode string
	require.NoError(t, storage.DB.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode))
	require.Equal(t, "wal", journalMode)

	var busyTimeout int
	require.NoError(t, storage.DB.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout))
	require.Equal(t, 3000, busyTimeout)
}

func TestNewInvalidJournalMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "invalid.db")
	_, err := New(t.Context(), Config{Path: path, JournalMode: "wall"})
	require.ErrorContains(t, err, "unsupported journal mode")

	_, err = os.Stat(filepath.Dir(path))
	require.ErrorIs(t, err, os.ErrNotExist, "directory must not be created for invalid config")
}

func TestNewInMemory(t *testing.T) {
	ctx := t.Context()
	storage, err := New(ctx, Config{Path: inMemoryPath, JournalMode: "MEMORY", MaxOpenConns: 4})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, storage.Close()) })

	require.Equal(t, 1, storage.DB.Stats().MaxOpenConnections)

	_, err = storage.DB.ExecContext(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
	_, err = storage.DB.ExecContext(ctx, "INSERT INTO test(id) VALUES (?)", 1)
	require.NoError(t, err)

	var count int
	require.NoError(t, storage.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM test").Scan(&count))
	require.Equal(t, 1, count)
}

func TestParseError(t *testing.T) {
	ctx := t.Context()
	storage, err := New(ctx, Config{Path: filepath.Join(t.TempDir(), "errors.db")})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, storage.Close()) })
	_, err = storage.DB.ExecContext(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT UNIQUE NOT NULL)")
	require.NoError(t, err)
	_, err = storage.DB.ExecContext(ctx, "INSERT INTO users(id, name) VALUES (?, ?)", 1, "alice")
	require.NoError(t, err)
	_, uniqueErr := storage.DB.ExecContext(ctx, "INSERT INTO users(id, name) VALUES (?, ?)", 2, "alice")
	require.Error(t, uniqueErr)
	_, primaryKeyErr := storage.DB.ExecContext(ctx, "INSERT INTO users(id, name) VALUES (?, ?)", 1, "bob")
	require.Error(t, primaryKeyErr)
	unknownErr := errors.New("boom")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "not found", err: sql.ErrNoRows, want: ErrNotFound},
		{name: "unique constraint", err: uniqueErr, want: ErrAlreadyExists},
		{name: "primary key constraint", err: primaryKeyErr, want: ErrAlreadyExists},
		{name: "unknown", err: unknownErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseError(tt.err)
			require.ErrorIs(t, got, tt.err, "original error must stay in the chain")
			if tt.want != nil {
				require.ErrorIs(t, got, tt.want)
			} else {
				require.NotErrorIs(t, got, ErrNotFound)
				require.NotErrorIs(t, got, ErrAlreadyExists)
			}
		})
	}

	t.Run("nil", func(t *testing.T) {
		require.NoError(t, ParseError(nil))
	})

	t.Run("driver error is reachable", func(t *testing.T) {
		var sqliteErr *modernsqlite.Error
		require.ErrorAs(t, ParseError(uniqueErr), &sqliteErr)
	})
}
