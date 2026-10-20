package telegram

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testConfig() Config {
	return Config{
		LongPolling: LongPollingConfig{
			Enabled: true,
			Timeout: 17,
		},
		Dispatcher: DispatcherConfig{
			QueueSize:     4,
			WorkerIdleTTL: time.Second,
			HandleTimeout: time.Second,
		},
	}
}

func TestConfigFix(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want Config
	}{
		{
			name: "zero and negative values use defaults",
			cfg: Config{
				LongPolling: LongPollingConfig{Enabled: true, Timeout: -1},
				Dispatcher:  DispatcherConfig{QueueSize: -1, WorkerIdleTTL: -1, HandleTimeout: -1},
			},
			want: Config{
				LongPolling: LongPollingConfig{Enabled: true, Timeout: defaultLongPollingTimeout},
				Dispatcher: DispatcherConfig{
					QueueSize:     defaultQueueSize,
					WorkerIdleTTL: defaultWorkerIdleTTL,
					HandleTimeout: defaultHandleTimeout,
				},
			},
		},
		{
			name: "configured values are preserved",
			cfg:  testConfig(),
			want: testConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.Fix()
			require.Equal(t, tt.want, tt.cfg)
		})
	}
}
