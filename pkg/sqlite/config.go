package sqlite

import (
	"strings"
	"time"
)

const (
	defaultPath         = "./data/data.db"
	defaultBusyTimeout  = 5 * time.Second
	defaultJournalMode  = "wal"
	defaultMaxOpenConns = 1
	defaultMaxIdleConns = 1
)

// Config contains SQLite connection settings.
type Config struct {
	Path         string        `mapstructure:"path"`
	BusyTimeout  time.Duration `mapstructure:"busy_timeout"`
	JournalMode  string        `mapstructure:"journal_mode"`
	MaxOpenConns int           `mapstructure:"max_open_conns"`
	MaxIdleConns int           `mapstructure:"max_idle_conns"`
}

// Fix fills unset or invalid settings with safe defaults.
func (cfg *Config) Fix() {
	if cfg.Path == "" {
		cfg.Path = defaultPath
	}

	if cfg.BusyTimeout <= 0 {
		cfg.BusyTimeout = defaultBusyTimeout
	}

	cfg.JournalMode = strings.ToLower(cfg.JournalMode)
	if cfg.JournalMode == "" {
		cfg.JournalMode = defaultJournalMode
	}

	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = defaultMaxOpenConns
	}

	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = defaultMaxIdleConns
	}
}
