package telegram

import (
	"context"
	"errors"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type stubUpdateSource struct {
	updates    chan tgbotapi.Update
	updatesErr error
	stopErr    error
	stopped    chan struct{}
}

func (s *stubUpdateSource) Updates() (<-chan tgbotapi.Update, error) {
	return s.updates, s.updatesErr
}

func TestBotRunReturnsUpdatesError(t *testing.T) {
	updatesErr := errors.New("updates failed")
	source := &stubUpdateSource{updatesErr: updatesErr, stopped: make(chan struct{})}
	bot := NewBot(source, func(context.Context, tgbotapi.Update) error { return nil })

	err := bot.Run(context.Background())
	if !errors.Is(err, updatesErr) {
		t.Fatalf("Run() error = %v, want it to wrap %v", err, updatesErr)
	}
}

func (s *stubUpdateSource) Stop() error {
	select {
	case <-s.stopped:
	default:
		close(s.stopped)
	}

	return s.stopErr
}

func TestBotRunReturnsOnClosedUpdates(t *testing.T) {
	source := &stubUpdateSource{
		updates: make(chan tgbotapi.Update),
		stopped: make(chan struct{}),
	}
	close(source.updates)

	bot := NewBot(source, func(context.Context, tgbotapi.Update) error { return nil })
	done := make(chan error, 1)
	go func() { done <- bot.Run(context.Background()) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() returned an unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after the updates channel was closed")
	}
}

func TestBotRunTreatsCancellationAsGracefulShutdown(t *testing.T) {
	source := &stubUpdateSource{
		updates: make(chan tgbotapi.Update),
		stopped: make(chan struct{}),
	}
	bot := NewBot(source, func(context.Context, tgbotapi.Update) error { return nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := bot.Run(ctx); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}

	select {
	case <-source.stopped:
	default:
		t.Fatal("Run() did not stop the update source")
	}
}

func TestBotRunReturnsStopError(t *testing.T) {
	stopErr := errors.New("stop failed")
	source := &stubUpdateSource{
		updates: make(chan tgbotapi.Update),
		stopErr: stopErr,
		stopped: make(chan struct{}),
	}
	bot := NewBot(source, func(context.Context, tgbotapi.Update) error { return nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := bot.Run(ctx)
	if !errors.Is(err, stopErr) {
		t.Fatalf("Run() error = %v, want it to wrap %v", err, stopErr)
	}
}

func TestBotRunReturnsDeadlineExceeded(t *testing.T) {
	source := &stubUpdateSource{
		updates: make(chan tgbotapi.Update),
		stopped: make(chan struct{}),
	}
	bot := NewBot(source, func(context.Context, tgbotapi.Update) error { return nil })
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := bot.Run(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want it to wrap deadline exceeded", err)
	}
}

func TestBotRunDispatchesUpdates(t *testing.T) {
	updates := make(chan tgbotapi.Update, 1)
	updates <- tgbotapi.Update{UpdateID: 7, Message: &tgbotapi.Message{From: &tgbotapi.User{ID: 42}}}
	close(updates)
	source := &stubUpdateSource{updates: updates, stopped: make(chan struct{})}
	handled := make(chan int, 1)
	bot := NewBot(source, func(_ context.Context, upd tgbotapi.Update) error {
		handled <- upd.UpdateID
		return nil
	})

	if err := bot.Run(context.Background()); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
	if got := <-handled; got != 7 {
		t.Fatalf("handled update ID = %d, want 7", got)
	}
}
