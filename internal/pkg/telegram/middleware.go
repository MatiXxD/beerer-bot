package telegram

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
			var (
				log  = utils.GetZeroLogger(ctx)
				from = upd.SentFrom()
			)

			if from != nil {
				log.Info().Msgf("received message %d from user %+v", upd.Message.MessageID, from)
			} else {
				log.Info().Msgf("received update %d from unknown user", upd.UpdateID)
			}

			return next(ctx, upd)
		}
	}
}
