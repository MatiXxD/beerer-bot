package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

const (
	TagUnexpected = "UNEXPECTED"
)

// ZerologConfig represents the configuration of the zerolog logger.
type ZerologConfig struct {
	AppName       string `mapstructure:"appname"`
	AppVersion    string `mapstructure:"version"`
	Level         string `mapstructure:"level"`
	PrettyConsole bool   `mapstructure:"pretty_console"`
}

// NewZerolog creates a new zerolog logger.
func NewZerolog(cfg *ZerologConfig) *zerolog.Logger {
	// set lvl
	lvl, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	// define writer
	var w io.Writer = os.Stdout
	if cfg.PrettyConsole {
		w = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	zl := zerolog.New(w).
		Level(lvl).
		With().
		Timestamp().
		Str("app_name", cfg.AppName).
		Str("app_version", cfg.AppVersion).
		Logger()

	return &zl
}
