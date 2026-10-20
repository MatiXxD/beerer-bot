package controller

import "github.com/MatiXxD/beerer-bot/internal/pkg/telegram"

// Controller represents a user controller.
type Controller struct {
	// TODO: добавить тут usecase слой
}

// New creates a new user controller.
func New() *Controller {
	return &Controller{}
}

// RegisterRoutes initializes the routes for the user controller.
func (c *Controller) RegisterRoutes(r *telegram.Router) {
	r.RegisterCommand("start", c.Create)
}
