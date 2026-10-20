package telegram

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
)

func testContext() context.Context {
	logger := zerolog.Nop()
	return utils.SetZeroLogger(context.Background(), &logger)
}

func TestDispatcherIgnoresUpdatesWithoutSender(t *testing.T) {
	cfg := testConfig()

	called := make(chan struct{}, 1)
	dispatcher := NewDispatcher(func(context.Context, tgbotapi.Update) error {
		called <- struct{}{}
		return nil
	}, cfg.Dispatcher)
	dispatcher.Dispatch(testContext(), tgbotapi.Update{UpdateID: 1})
	dispatcher.Stop()

	select {
	case <-called:
		require.FailNow(t, "handler was called for an update without a sender")
	default:
	}
}

func TestDispatcherProcessesSameUserSequentially(t *testing.T) {
	var (
		mu    sync.Mutex
		order []int
		cfg   = testConfig()
	)
	cfg.Dispatcher.QueueSize = 10

	dispatcher := NewDispatcher(func(_ context.Context, upd tgbotapi.Update) error {
		mu.Lock()
		order = append(order, upd.UpdateID)
		mu.Unlock()
		return nil
	}, cfg.Dispatcher)

	for i := 1; i <= 10; i++ {
		dispatcher.Dispatch(testContext(), tgbotapi.Update{
			UpdateID: i,
			Message:  &tgbotapi.Message{From: &tgbotapi.User{ID: 42}},
		})
	}
	dispatcher.Stop()

	want := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	require.Equal(t, want, order)
}

func TestDispatcherProcessesDifferentUsersConcurrently(t *testing.T) {
	var (
		started = make(chan int64, 2)
		release = make(chan struct{})
		cfg     = testConfig()
	)

	dispatcher := NewDispatcher(func(_ context.Context, upd tgbotapi.Update) error {
		started <- upd.SentFrom().ID
		<-release
		return nil
	}, cfg.Dispatcher)

	for _, userID := range []int64{1, 2} {
		dispatcher.Dispatch(testContext(), tgbotapi.Update{
			Message: &tgbotapi.Message{From: &tgbotapi.User{ID: userID}},
		})
	}

	seen := make(map[int64]bool)
	for range 2 {
		select {
		case userID := <-started:
			seen[userID] = true
		case <-time.After(time.Second):
			require.FailNow(t, "handlers for different users did not run concurrently")
		}
	}
	close(release)
	dispatcher.Stop()

	require.Equal(t, map[int64]bool{1: true, 2: true}, seen)
}

func TestDispatcherStopDrainsQueue(t *testing.T) {
	const updateCount = 20
	var (
		handled atomic.Int32
		started = make(chan struct{})
		release = make(chan struct{})
		once    sync.Once
		cfg     = testConfig()
	)
	cfg.Dispatcher.QueueSize = updateCount

	dispatcher := NewDispatcher(func(context.Context, tgbotapi.Update) error {
		once.Do(func() {
			close(started)
			<-release
		})
		handled.Add(1)
		return nil
	}, cfg.Dispatcher)

	for i := range updateCount {
		dispatcher.Dispatch(testContext(), tgbotapi.Update{
			UpdateID: i,
			Message:  &tgbotapi.Message{From: &tgbotapi.User{ID: 52}},
		})
	}
	<-started

	stopped := make(chan struct{})
	go func() {
		dispatcher.Stop()
		close(stopped)
	}()

	<-dispatcher.done
	close(release)
	<-stopped

	require.Equal(t, int32(updateCount), handled.Load())
}

func TestDispatcherDropsUpdateWhenUserQueueIsFull(t *testing.T) {
	var (
		output  bytes.Buffer
		handled int
		cfg     = testConfig()

		logger  = zerolog.New(&output)
		ctx     = utils.SetZeroLogger(context.Background(), &logger)
		started = make(chan struct{})
		release = make(chan struct{})
	)

	dispatcher := NewDispatcher(func(context.Context, tgbotapi.Update) error {
		handled++
		if handled == 1 {
			close(started)
			<-release
		}
		return nil
	}, cfg.Dispatcher)

	update := tgbotapi.Update{Message: &tgbotapi.Message{From: &tgbotapi.User{ID: 42}}}
	dispatcher.Dispatch(ctx, update) // handled == 1

	// handled should be 1 + cfg.Dispatcher.QueueSize after this loop, 10 will be dropped
	<-started
	for range cfg.Dispatcher.QueueSize + 10 {
		dispatcher.Dispatch(ctx, update)
	}

	close(release)
	dispatcher.Stop()

	require.Equal(t, cfg.Dispatcher.QueueSize+1, handled)
	require.Contains(t, output.String(), "queue for user 42 is full")
}

func TestDispatcherLogsHandlerError(t *testing.T) {
	var (
		output bytes.Buffer
		cfg    = testConfig()

		logger  = zerolog.New(&output)
		ctx     = utils.SetZeroLogger(context.Background(), &logger)
		wantErr = errors.New("handler failed")
	)

	dispatcher := NewDispatcher(func(context.Context, tgbotapi.Update) error { return wantErr }, cfg.Dispatcher)
	dispatcher.Dispatch(ctx, tgbotapi.Update{Message: &tgbotapi.Message{From: &tgbotapi.User{ID: 42}}})
	dispatcher.Stop()

	require.Contains(t, output.String(), wantErr.Error())
}

func TestDispatcherHandlerIgnoresParentCancel(t *testing.T) {
	type handlerCtx struct {
		logger *zerolog.Logger
		err    error
	}

	var (
		cfg = testConfig()

		logger          = zerolog.Nop()
		ctx             = utils.SetZeroLogger(context.Background(), &logger)
		handlerStarted  = make(chan struct{})
		continueHandler = make(chan struct{})
		result          = make(chan handlerCtx, 1)
	)

	ctx, cancel := context.WithCancel(ctx)

	dispatcher := NewDispatcher(func(hctx context.Context, _ tgbotapi.Update) error {
		close(handlerStarted)
		<-continueHandler

		result <- handlerCtx{logger: utils.GetZeroLogger(hctx), err: hctx.Err()}
		return nil
	}, cfg.Dispatcher)

	dispatcher.Dispatch(ctx, tgbotapi.Update{
		Message: &tgbotapi.Message{From: &tgbotapi.User{ID: 42}},
	})

	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		require.FailNow(t, "handler was not started")
	}

	cancel()
	close(continueHandler)
	dispatcher.Stop()

	got := <-result
	require.Same(t, &logger, got.logger, "handler context lost its logger")
	require.NoError(t, got.err, "handler context was canceled")
}
