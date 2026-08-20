package app

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/MatiXxD/beerer-bot/config"
	"github.com/MatiXxD/beerer-bot/pkg/logger"
	"github.com/MatiXxD/beerer-bot/pkg/utils"
)

// DI represents the dependency injection container.
type DI struct {
	// Common
	cfg *config.Config
	log *zerolog.Logger
}

// Run runs the application.
func Run(ctx context.Context, cfg *config.Config) error {
	var di DI

	// common
	di.cfg = cfg
	di.log = logger.NewZerolog(&cfg.Logger)

	// prepare global context
	ctx = utils.SetZeroLogger(ctx, di.log)

	_, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// just log for now
	di.log.Info().Msgf("app %s is to be done", cfg.AppCfg.Name)

	return nil
}
