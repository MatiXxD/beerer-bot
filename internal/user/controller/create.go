package controller

import (
	"context"

	"github.com/MatiXxD/beerer-bot/pkg/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Create creates a new user.
func (c *Controller) Create(ctx context.Context, upd tgbotapi.Update) error {
	const op = utils.Operation("Controller.Create")

	log := utils.GetZeroLogger(ctx)

	log.Info().Msg(op.WithMsg("creating a new user"))

	return nil
}
