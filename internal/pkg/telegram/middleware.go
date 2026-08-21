package telegram

import (
	"context"
	"fmt"
	"runtime/debug"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
)

// Recovery is a middleware that recovers from panics.
func Recovery() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, upd tgbotapi.Update) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic: %v; stack: %s", r, debug.Stack())
				}
			}()

			return next(ctx, upd)
		}
	}
}

// Logging is a middleware that logs the request.
func Logging() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, upd tgbotapi.Update) error {
			log := utils.GetZeroLogger(ctx)

			switch {
			case upd.Message != nil:
				log.Info().Msgf("received message %d from user %+v", upd.Message.MessageID, upd.Message.From)
			case upd.CallbackQuery != nil:
				log.Info().Msgf("received callback query %s from user %+v", upd.CallbackQuery.ID, upd.CallbackQuery.From)
			case upd.InlineQuery != nil:
				log.Info().Msgf("received inline query %s from user %+v", upd.InlineQuery.ID, upd.InlineQuery.From)
			default:
				log.Info().Msgf("received update %d from unknown user", upd.UpdateID)
			}

			return next(ctx, upd)
		}
	}
}
