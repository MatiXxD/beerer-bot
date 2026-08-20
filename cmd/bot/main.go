package main

import (
	"context"

	"github.com/MatiXxD/beerer-bot/config"
	"github.com/MatiXxD/beerer-bot/internal/app"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	if err = app.Run(ctx, cfg); err != nil {
		log.Fatal().Err(err).Msg("failed to run app")
	}
}
