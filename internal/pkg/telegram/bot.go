package telegram

import (
	"context"
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
)

var (
	// ErrWebhookNotImplemented is returned when long polling is disabled, since webhook is not supported yet.
	ErrWebhookNotImplemented = errors.New("webhook is not implemented, enable long polling")
)

// UpdateSource represents a source of updates.
// Stop must eventually close the channel returned by Updates.
type UpdateSource interface {
	Updates() (<-chan tgbotapi.Update, error)
	Stop() error
}

// Bot represents a Telegram bot.
type Bot struct {
	source     UpdateSource
	dispatcher *Dispatcher

	cfg Config
}

// NewBot creates a new Bot instance.
func NewBot(h Handler, api *tgbotapi.BotAPI, cfg Config) (*Bot, error) {
	const op = utils.Operation("NewBot")

	// TODO: тут надо будет сделать поддержку webhook-а
	if !cfg.LongPolling.Enabled {
		return nil, op.WithErr(ErrWebhookNotImplemented)
	}

	var bot Bot

	cfg.Fix()
	bot.cfg = cfg

	// set dispatcher
	bot.dispatcher = NewDispatcher(h, cfg.Dispatcher)
	bot.source = NewLongPolling(api, cfg.LongPolling)

	return &bot, nil
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

			// read all remaining updates before return
			for upd := range updates {
				b.dispatcher.Dispatch(ctx, upd)
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
