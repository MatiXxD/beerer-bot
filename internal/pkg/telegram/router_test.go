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

			require.NoError(t, router.Build()(context.Background(), tt.upd))
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
			require.NoError(t, router.Build()(ctx, tt.upd))
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

	require.NoError(t, router.Build()(context.Background(), tgbotapi.Update{Message: &tgbotapi.Message{Text: "hello"}}))
	want := []string{"first:before", "second:before", "handler", "second:after", "first:after"}
	require.Equal(t, want, calls)
}

func TestRouterPanicsOnNilHandler(t *testing.T) {
	tests := []struct {
		name     string
		register func(r *Router)
		want     string
	}{
		{name: "command", register: func(r *Router) { r.RegisterCommand("start", nil) }, want: "telegram: nil handler for command start"},
		{name: "callback", register: func(r *Router) { r.RegisterCallback("beer", nil) }, want: "telegram: nil handler for callback prefix beer"},
		{name: "text", register: func(r *Router) { r.RegisterText(nil) }, want: "telegram: nil text handler"},
		{name: "fallback", register: func(r *Router) { r.RegisterFallback(nil) }, want: "telegram: nil fallback handler"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.PanicsWithValue(t, tt.want, func() { tt.register(NewRouter()) })
		})
	}
}

func TestRouterBuildAppliesMiddlewaresOnce(t *testing.T) {
	var (
		router   = NewRouter()
		wrapped  int
		lateUsed bool
		handled  int
	)

	router.Use(func(next Handler) Handler {
		wrapped++
		return next
	})

	h := router.Build()

	// handlers registered after Build are still routed, middlewares are not applied
	router.RegisterText(func(context.Context, tgbotapi.Update) error {
		handled++
		return nil
	})
	router.Use(func(next Handler) Handler {
		lateUsed = true
		return next
	})

	upd := tgbotapi.Update{Message: &tgbotapi.Message{Text: "hello"}}
	for range 3 {
		require.NoError(t, h(context.Background(), upd))
	}

	require.Equal(t, 1, wrapped, "middleware chain was rebuilt")
	require.Equal(t, 3, handled)
	require.False(t, lateUsed, "middleware added after Build was applied")
}
