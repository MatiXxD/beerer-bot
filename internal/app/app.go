package app

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/MatiXxD/beerer-bot/config"
	"github.com/MatiXxD/beerer-bot/internal/pkg/telegram"
	"github.com/MatiXxD/beerer-bot/pkg/logger"
	"github.com/MatiXxD/beerer-bot/pkg/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog"
)

// DI represents the dependency injection container.
type DI struct {
	// Telegram
	source telegram.UpdateSource
	router *telegram.Router
	bot    *telegram.Bot
	api    *tgbotapi.BotAPI

	// Common
	cfg *config.Config
	log *zerolog.Logger
}

// Run runs the application.
func Run(ctx context.Context, cfg *config.Config) error {
	var (
		di  DI
		err error
	)

	// common
	di.cfg = cfg
	di.log = logger.NewZerolog(&cfg.Logger)

	// prepare global context
	ctx = utils.SetZeroLogger(ctx, di.log)

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// prepare telegram bot
	err = initBot(&di)
	if err != nil {
		return fmt.Errorf("failed to run the application: %v", err)
	}

	// domains
	UserDomain(&di)

	// start telegram bot
	di.log.Info().Msg("starting bot...")

	err = di.bot.Run(ctx)
	if err != nil {
		return fmt.Errorf("bot failed with: %v", err)
	}

	return nil
}

// initBot initialize all that is needed for the bot to work.
func initBot(di *DI) error {
	// TODO: не забыть накинуть конфиг на пакет с telegram ботом
	var err error

	// create telegram api
	di.api, err = tgbotapi.NewBotAPI(di.cfg.TelegramBot.BotToken)
	if err != nil {
		return fmt.Errorf("failed to create telegram api: %w", err)
	}

	// create router
	di.router = telegram.NewRouter()

	mws := []telegram.Middleware{
		telegram.Recovery(),
		telegram.Logging(),
	}

	for _, mw := range mws {
		di.router.Use(mw)
	}

	// use long polling if specified in config, otherwise use webhook
	if di.cfg.TelegramBot.LongPolling {
		di.source = telegram.NewLongPolling(di.api)
	} else {
		return errors.New("webhook not supported yet")
	}

	// init bot
	di.bot = telegram.NewBot(di.source, di.router.Handle)

	return nil
}
