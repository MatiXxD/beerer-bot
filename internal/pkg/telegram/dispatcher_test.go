package telegram

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
)

func testContext() context.Context {
	logger := zerolog.Nop()
	return utils.SetZeroLogger(context.Background(), &logger)
}

func TestDispatcherIgnoresUpdatesWithoutSender(t *testing.T) {
	called := make(chan struct{}, 1)
	dispatcher := NewDispatcher(func(context.Context, tgbotapi.Update) error {
		called <- struct{}{}
		return nil
	})
	dispatcher.Dispatch(testContext(), tgbotapi.Update{UpdateID: 1})
	dispatcher.Stop()

	select {
	case <-called:
		t.Fatal("handler was called for an update without a sender")
	default:
	}
}

func TestDispatcherProcessesSameUserSequentially(t *testing.T) {
	var (
		mu    sync.Mutex
		order []int
	)
	dispatcher := NewDispatcher(func(_ context.Context, upd tgbotapi.Update) error {
		mu.Lock()
		order = append(order, upd.UpdateID)
		mu.Unlock()
		return nil
	})

	for i := 1; i <= 10; i++ {
		dispatcher.Dispatch(testContext(), tgbotapi.Update{
			UpdateID: i,
			Message:  &tgbotapi.Message{From: &tgbotapi.User{ID: 42}},
		})
	}
	dispatcher.Stop()

	want := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("processing order = %v, want %v", order, want)
	}
}

func TestDispatcherProcessesDifferentUsersConcurrently(t *testing.T) {
	started := make(chan int64, 2)
	release := make(chan struct{})
	dispatcher := NewDispatcher(func(_ context.Context, upd tgbotapi.Update) error {
		started <- upd.SentFrom().ID
		<-release
		return nil
	})

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
			t.Fatal("handlers for different users did not run concurrently")
		}
	}
	close(release)
	dispatcher.Stop()

	if !seen[1] || !seen[2] {
		t.Fatalf("handled users = %v, want users 1 and 2", seen)
	}
}

func TestDispatcherStopDrainsQueue(t *testing.T) {
	const updateCount = 20
	var (
		mu      sync.Mutex
		handled int
	)
	dispatcher := NewDispatcher(func(context.Context, tgbotapi.Update) error {
		mu.Lock()
		handled++
		mu.Unlock()
		return nil
	})
	for i := range updateCount {
		dispatcher.Dispatch(testContext(), tgbotapi.Update{
			UpdateID: i,
			Message:  &tgbotapi.Message{From: &tgbotapi.User{ID: 42}},
		})
	}
	dispatcher.Stop()

	if handled != updateCount {
		t.Fatalf("handled updates = %d, want %d", handled, updateCount)
	}
}

func TestDispatcherDropsUpdateWhenUserQueueIsFull(t *testing.T) {
	var output bytes.Buffer
	logger := zerolog.New(&output)
	ctx := utils.SetZeroLogger(context.Background(), &logger)
	started := make(chan struct{})
	release := make(chan struct{})
	var handled int
	dispatcher := NewDispatcher(func(context.Context, tgbotapi.Update) error {
		handled++
		if handled == 1 {
			close(started)
			<-release
		}
		return nil
	})
	update := tgbotapi.Update{Message: &tgbotapi.Message{From: &tgbotapi.User{ID: 42}}}
	dispatcher.Dispatch(ctx, update)
	<-started
	for range defaultQueueSize + 1 {
		dispatcher.Dispatch(ctx, update)
	}
	close(release)
	dispatcher.Stop()

	if handled != defaultQueueSize+1 {
		t.Fatalf("handled updates = %d, want %d", handled, defaultQueueSize+1)
	}
	if !strings.Contains(output.String(), "queue for user 42 is full") {
		t.Fatalf("log = %q, want queue-full warning", output.String())
	}
}

func TestDispatcherLogsHandlerError(t *testing.T) {
	var output bytes.Buffer
	logger := zerolog.New(&output)
	ctx := utils.SetZeroLogger(context.Background(), &logger)
	wantErr := errors.New("handler failed")
	dispatcher := NewDispatcher(func(context.Context, tgbotapi.Update) error { return wantErr })
	dispatcher.Dispatch(ctx, tgbotapi.Update{Message: &tgbotapi.Message{From: &tgbotapi.User{ID: 42}}})
	dispatcher.Stop()

	if !strings.Contains(output.String(), wantErr.Error()) {
		t.Fatalf("log = %q, want handler error", output.String())
	}
}

func TestDispatcherPreservesContextValuesDuringShutdown(t *testing.T) {
	logger := zerolog.Nop()
	ctx := utils.SetZeroLogger(context.Background(), &logger)
	ctx, cancel := context.WithCancel(ctx)
	handlerStarted := make(chan struct{})
	continueHandler := make(chan struct{})
	result := make(chan error, 1)

	dispatcher := NewDispatcher(func(ctx context.Context, _ tgbotapi.Update) error {
		close(handlerStarted)
		<-continueHandler

		if got := utils.GetZeroLogger(ctx); got != &logger {
			result <- context.Canceled
			return nil
		}

		result <- ctx.Err()
		return nil
	})
	dispatcher.Dispatch(ctx, tgbotapi.Update{
		Message: &tgbotapi.Message{From: &tgbotapi.User{ID: 42}},
	})

	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		t.Fatal("handler was not started")
	}

	cancel()
	close(continueHandler)
	dispatcher.Stop()

	if err := <-result; err != nil {
		t.Fatalf("handler context was canceled or lost its logger: %v", err)
	}
}
