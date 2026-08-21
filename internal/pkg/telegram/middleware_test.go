package telegram

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
)

func TestRecovery(t *testing.T) {
	handler := Recovery()(func(context.Context, tgbotapi.Update) error {
		panic("boom")
	})

	err := handler(context.Background(), tgbotapi.Update{})
	if err == nil || !strings.Contains(err.Error(), "panic: boom") || !strings.Contains(err.Error(), "goroutine") {
		t.Fatalf("Recovery() error = %v, want panic value and stack", err)
	}
}

func TestRecoveryPassesThroughResult(t *testing.T) {
	wantErr := errors.New("handler failed")
	handler := Recovery()(func(context.Context, tgbotapi.Update) error { return wantErr })

	if err := handler(context.Background(), tgbotapi.Update{}); !errors.Is(err, wantErr) {
		t.Fatalf("Recovery() error = %v, want %v", err, wantErr)
	}
}

func TestLoggingHandlesCallbackQuery(t *testing.T) {
	logger := zerolog.Nop()
	ctx := utils.SetZeroLogger(context.Background(), &logger)
	called := false
	handler := Logging()(func(context.Context, tgbotapi.Update) error {
		called = true
		return nil
	})
	upd := tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "callback-id",
			From: &tgbotapi.User{ID: 42},
		},
	}

	if err := handler(ctx, upd); err != nil {
		t.Fatalf("handler returned an unexpected error: %v", err)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
}

func TestLoggingUpdateKinds(t *testing.T) {
	tests := []struct {
		name string
		upd  tgbotapi.Update
		want string
	}{
		{name: "message", upd: tgbotapi.Update{Message: &tgbotapi.Message{MessageID: 5, From: &tgbotapi.User{ID: 1}}}, want: "received message 5"},
		{name: "callback", upd: tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{ID: "cb", From: &tgbotapi.User{ID: 2}}}, want: "received callback query cb"},
		{name: "inline", upd: tgbotapi.Update{InlineQuery: &tgbotapi.InlineQuery{ID: "inline", From: &tgbotapi.User{ID: 3}}}, want: "received inline query inline"},
		{name: "unknown", upd: tgbotapi.Update{UpdateID: 9}, want: "received update 9 from unknown user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := zerolog.New(&output)
			ctx := utils.SetZeroLogger(context.Background(), &logger)
			wantErr := errors.New("next result")
			handler := Logging()(func(context.Context, tgbotapi.Update) error { return wantErr })

			if err := handler(ctx, tt.upd); !errors.Is(err, wantErr) {
				t.Fatalf("Logging() error = %v, want %v", err, wantErr)
			}
			if !strings.Contains(output.String(), tt.want) {
				t.Fatalf("log = %q, want it to contain %q", output.String(), tt.want)
			}
		})
	}
}
