package main

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/MatiXxD/beerer-bot/config"
	"github.com/MatiXxD/beerer-bot/internal/app"
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
