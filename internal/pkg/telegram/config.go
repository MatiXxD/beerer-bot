package telegram

import "time"

const (
	defaultQueueSize     = 64
	defaultWorkerIdleTTL = 300 * time.Second
	defaultHandleTimeout = 30 * time.Second

	defaultLongPollingTimeout = 10
)

// LongPollingConfig config for long polling source.
type LongPollingConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Timeout int  `mapstructure:"timeout"`
}

// DispatcherConfig config for the dispatcher.
type DispatcherConfig struct {
	QueueSize     int           `mapstructure:"queue_size"`
	WorkerIdleTTL time.Duration `mapstructure:"worker_idle_ttl"`
	HandleTimeout time.Duration `mapstructure:"handle_timeout"`
}

// Config configuration for the telegram bot.
type Config struct {
	LongPolling LongPollingConfig `mapstructure:"long_polling"`
	Dispatcher  DispatcherConfig  `mapstructure:"dispatcher"`
}

// Fix fixes the config.
func (cfg *Config) Fix() {
	if cfg.LongPolling.Timeout <= 0 {
		cfg.LongPolling.Timeout = defaultLongPollingTimeout
	}

	if cfg.Dispatcher.QueueSize <= 0 {
		cfg.Dispatcher.QueueSize = defaultQueueSize
	}

	if cfg.Dispatcher.WorkerIdleTTL <= 0 {
		cfg.Dispatcher.WorkerIdleTTL = defaultWorkerIdleTTL
	}

	if cfg.Dispatcher.HandleTimeout <= 0 {
		cfg.Dispatcher.HandleTimeout = defaultHandleTimeout
	}
}
