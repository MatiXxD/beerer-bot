package telegram

import (
	"context"
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
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

	cfg Config
}

// NewBot creates a new Bot instance.
func NewBot(h Handler, api *tgbotapi.BotAPI, cfg Config) *Bot {
	var bot Bot

	cfg.Fix()
	bot.cfg = cfg

	// set dispatcher
	bot.dispatcher = NewDispatcher(h, cfg.Dispatcher)

	// use long polling if specified in config, otherwise use webhook
	//nolint:gocritic
	if cfg.LongPolling.Enabled {
		bot.source = NewLongPolling(api, cfg.LongPolling)
	} else {
		// TODO: тут надо будет сделать поддержку webhook-а
		bot.source = NewLongPolling(api, cfg.LongPolling)
	}

	return &bot
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
