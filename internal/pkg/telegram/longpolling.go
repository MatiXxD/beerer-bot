package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
)

// LongPolling is a struct that represents the long polling mechanism for receiving
// updates from the Telegram Bot API.
type LongPolling struct {
	api *tgbotapi.BotAPI
	cfg LongPollingConfig
}

// NewLongPolling creates a new LongPolling instance.
func NewLongPolling(api *tgbotapi.BotAPI, cfg LongPollingConfig) *LongPolling {
	return &LongPolling{
		api: api,
		cfg: cfg,
	}
}

// Updates returns a channel of updates from the Telegram Bot API.
func (lp *LongPolling) Updates() (<-chan tgbotapi.Update, error) {
	const op = utils.Operation("LongPolling.Updates")

	// remove webhook
	if _, err := lp.api.Request(tgbotapi.DeleteWebhookConfig{}); err != nil {
		return nil, op.WithErr(err)
	}

	updateCfg := tgbotapi.NewUpdate(0)
	updateCfg.Timeout = lp.cfg.Timeout

	return lp.api.GetUpdatesChan(updateCfg), nil
}

// Stop stops the long polling mechanism.
func (lp *LongPolling) Stop() error {
	lp.api.StopReceivingUpdates()
	return nil
}
