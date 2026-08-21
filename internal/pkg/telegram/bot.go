package telegram

import (
	"context"
	"errors"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// UpdateSource represents a source of updates.
type UpdateSource interface {
	Updates() (<-chan tgbotapi.Update, error)
	Stop() error
}

// Bot represents a Telegram bot.
type Bot struct {
	source     UpdateSource
	dispatcher *Dispatcher
}

// NewBot creates a new Bot instance.
func NewBot(src UpdateSource, h Handler) *Bot {
	// TODO: добавить конфиг, вместо того, чтобы повсюду брать дефолтные значения
	return &Bot{
		source:     src,
		dispatcher: NewDispatcher(h),
	}
}

// Run runs the bot logic.
func (b *Bot) Run(ctx context.Context) error {
	const op = utils.Operation("Bot.Run")

	updates, err := b.source.Updates()
	if err != nil {
		return op.WithErr(err)
	}

	defer b.dispatcher.Stop()

	for {
		select {
		case <-ctx.Done():
			if err = b.source.Stop(); err != nil {
				return op.WithErrAndMsg(err, "failed to stop update source")
			}

			if err = ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
				return op.WithErr(err)
			}

			return nil
		case upd, ok := <-updates:
			if !ok {
				if err = b.source.Stop(); err != nil {
					return op.WithErrAndMsg(err, "failed to stop update source")
				}

				return nil
			}

			b.dispatcher.Dispatch(ctx, upd)
		}
	}
}
