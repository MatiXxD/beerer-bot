package telegram

import "time"

const (
	defaultQueueSize     = 64
	defaultWorkerIdleTTL = 300 * time.Second
	defaultHandleTimeout = 30 * time.Second

	defaultLongPollingTimeout = 10
)
