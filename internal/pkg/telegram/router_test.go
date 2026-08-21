package telegram

import (
	"context"
	"reflect"
	"testing"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
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

			if err := router.Handle(context.Background(), tt.upd); err != nil {
				t.Fatalf("Handle() returned an unexpected error: %v", err)
			}
			if called != tt.want {
				t.Fatalf("called handler = %q, want %q", called, tt.want)
			}
		})
	}
}

func TestRouterUsesFallback(t *testing.T) {
	logger := zerolog.Nop()
	ctx := utils.SetZeroLogger(context.Background(), &logger)
	router := NewRouter()

	updates := []tgbotapi.Update{
		{Message: &tgbotapi.Message{Text: "/unknown", Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 8}}}},
		{Message: &tgbotapi.Message{}},
		{CallbackQuery: &tgbotapi.CallbackQuery{Data: "unknown"}},
		{},
	}
	for _, upd := range updates {
		if err := router.Handle(ctx, upd); err != nil {
			t.Fatalf("Handle() returned an unexpected error: %v", err)
		}
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

	if err := router.Handle(context.Background(), tgbotapi.Update{Message: &tgbotapi.Message{Text: "hello"}}); err != nil {
		t.Fatalf("Handle() returned an unexpected error: %v", err)
	}
	want := []string{"first:before", "second:before", "handler", "second:after", "first:after"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("call order = %v, want %v", calls, want)
	}
}
