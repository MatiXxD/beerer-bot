package controller

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
)

// Create creates a new user.
func (c *Controller) Create(ctx context.Context, _ tgbotapi.Update) error {
	const op = utils.Operation("Controller.Create")

	log := utils.GetZeroLogger(ctx)

	log.Info().Msg(op.WithMsg("creating a new user"))

	return nil
}
