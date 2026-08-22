package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
)

type stubUpdateSource struct {
	updates    chan tgbotapi.Update
	updatesErr error
	stopErr    error
	stopped    chan struct{}
}

func newTestBot(source UpdateSource, handler Handler, cfg Config) *Bot {
	return &Bot{
		source:     source,
		dispatcher: NewDispatcher(handler, cfg.Dispatcher),
		cfg:        cfg,
	}
}

func (s *stubUpdateSource) Updates() (<-chan tgbotapi.Update, error) {
	return s.updates, s.updatesErr
}

func (s *stubUpdateSource) Stop() error {
	select {
	case <-s.stopped:
	default:
		close(s.stopped)
	}

	return s.stopErr
}

func TestNewBotAppliesConfig(t *testing.T) {
	apiClient := &telegramAPIClient{}
	api := apiClient.bot(t)
	bot := NewBot(func(context.Context, tgbotapi.Update) error { return nil }, api, Config{
		LongPolling: LongPollingConfig{Enabled: true},
	})

	require.Equal(t, defaultLongPollingTimeout, bot.cfg.LongPolling.Timeout)
	require.Equal(t, defaultQueueSize, bot.cfg.Dispatcher.QueueSize)
	require.Equal(t, defaultWorkerIdleTTL, bot.cfg.Dispatcher.WorkerIdleTTL)
	require.Equal(t, defaultHandleTimeout, bot.cfg.Dispatcher.HandleTimeout)
	require.Equal(t, bot.cfg.Dispatcher, bot.dispatcher.cfg)

	source, ok := bot.source.(*LongPolling)
	require.True(t, ok)
	require.Equal(t, bot.cfg.LongPolling, source.cfg)
}

func TestBotRunReturnsUpdatesError(t *testing.T) {
	updatesErr := errors.New("updates failed")
	source := &stubUpdateSource{updatesErr: updatesErr, stopped: make(chan struct{})}
	bot := newTestBot(source, func(context.Context, tgbotapi.Update) error { return nil }, testConfig())

	err := bot.Run(context.Background())
	require.ErrorIs(t, err, updatesErr)
}

func TestBotRunReturnsOnClosedUpdates(t *testing.T) {
	source := &stubUpdateSource{
		updates: make(chan tgbotapi.Update),
		stopped: make(chan struct{}),
	}
	close(source.updates)

	bot := newTestBot(source, func(context.Context, tgbotapi.Update) error { return nil }, testConfig())
	done := make(chan error, 1)
	go func() { done <- bot.Run(context.Background()) }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		require.FailNow(t, "Run() did not return after the updates channel was closed")
	}
}

func TestBotRunTreatsCancellationAsGracefulShutdown(t *testing.T) {
	source := &stubUpdateSource{
		updates: make(chan tgbotapi.Update),
		stopped: make(chan struct{}),
	}
	bot := newTestBot(source, func(context.Context, tgbotapi.Update) error { return nil }, testConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, bot.Run(ctx))

	select {
	case <-source.stopped:
	default:
		require.FailNow(t, "Run() did not stop the update source")
	}
}

func TestBotRunReturnsStopError(t *testing.T) {
	stopErr := errors.New("stop failed")
	source := &stubUpdateSource{
		updates: make(chan tgbotapi.Update),
		stopErr: stopErr,
		stopped: make(chan struct{}),
	}

	bot := newTestBot(source, func(context.Context, tgbotapi.Update) error { return nil }, testConfig())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := bot.Run(ctx)
	require.ErrorIs(t, err, stopErr)
}

func TestBotRunReturnsDeadlineExceeded(t *testing.T) {
	source := &stubUpdateSource{
		updates: make(chan tgbotapi.Update),
		stopped: make(chan struct{}),
	}

	bot := newTestBot(source, func(context.Context, tgbotapi.Update) error { return nil }, testConfig())
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := bot.Run(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestBotRunDispatchesUpdates(t *testing.T) {
	updates := make(chan tgbotapi.Update, 1)
	updates <- tgbotapi.Update{UpdateID: 7, Message: &tgbotapi.Message{From: &tgbotapi.User{ID: 42}}}
	close(updates)

	source := &stubUpdateSource{updates: updates, stopped: make(chan struct{})}
	handled := make(chan int, 1)

	bot := newTestBot(source, func(_ context.Context, upd tgbotapi.Update) error {
		handled <- upd.UpdateID
		return nil
	}, testConfig())

	require.NoError(t, bot.Run(context.Background()))
	require.Equal(t, 7, <-handled)
}
