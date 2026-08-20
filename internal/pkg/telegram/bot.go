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

	log := utils.GetZeroLogger(ctx)

	updates, err := b.source.Updates()
	if err != nil {
		return op.WithErr(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()

		stopErr := b.source.Stop()
		if err != nil {
			log.Error().
				Err(op.WithErrAndMsg(stopErr, "failed to stop update source")).
				Msg("failed to stop update source")
		}
	}()

	for upd := range updates {
		b.dispatcher.Dispatch(upd)
	}

	// blocks until the done channel is closed
	<-done
	b.dispatcher.Stop()

	if !errors.Is(err, context.Canceled) {
		return op.WithErr(err)
	}

	return nil
}
