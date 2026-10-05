package telegram

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
)

func TestRouterRoutesUpdates(t *testing.T) {
	tests := []struct {
		name string
		upd  tgbotapi.Update
		want string
	}{
		{name: "command", upd: tgbotapi.Update{Message: &tgbotapi.Message{Text: "/start", Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 6}}}}, want: "command"},
		{name: "text", upd: tgbotapi.Update{Message: &tgbotapi.Message{Text: "hello"}}, want: "text"},
		{name: "specific callback first", upd: tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{Data: "beer:order:1"}}, want: "specific"},
		{name: "general callback", upd: tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{Data: "beer:list"}}, want: "general"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter()
			called := ""
			handler := func(name string) Handler {
				return func(context.Context, tgbotapi.Update) error { called = name; return nil }
			}
			router.RegisterCommand("start", handler("command"))
			router.RegisterText(handler("text"))
			router.RegisterCallback("beer:order", handler("specific"))
			router.RegisterCallback("beer", handler("general"))

			require.NoError(t, router.Handle(context.Background(), tt.upd))
			require.Equal(t, tt.want, called)
		})
	}
}

func TestRouterUsesFallback(t *testing.T) {
	logger := zerolog.Nop()
	ctx := utils.SetZeroLogger(context.Background(), &logger)
	router := NewRouter()

	tests := []struct {
		name string
		upd  tgbotapi.Update
	}{
		{name: "unknown command", upd: tgbotapi.Update{Message: &tgbotapi.Message{Text: "/unknown", Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 8}}}}},
		{name: "empty message", upd: tgbotapi.Update{Message: &tgbotapi.Message{}}},
		{name: "unknown callback", upd: tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{Data: "unknown"}}},
		{name: "empty update", upd: tgbotapi.Update{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, router.Handle(ctx, tt.upd))
		})
	}
}

func TestRouterAppliesMiddlewareInRegistrationOrder(t *testing.T) {
	router := NewRouter()
	var calls []string
	middleware := func(name string) Middleware {
		return func(next Handler) Handler {
			return func(ctx context.Context, upd tgbotapi.Update) error {
				calls = append(calls, name+":before")
				err := next(ctx, upd)
				calls = append(calls, name+":after")
				return err
			}
		}
	}
	router.Use(middleware("first"))
	router.Use(middleware("second"))
	router.RegisterText(func(context.Context, tgbotapi.Update) error {
		calls = append(calls, "handler")
		return nil
	})

	require.NoError(t, router.Handle(context.Background(), tgbotapi.Update{Message: &tgbotapi.Message{Text: "hello"}}))
	want := []string{"first:before", "second:before", "handler", "second:after", "first:after"}
	require.Equal(t, want, calls)
}
