package telegram

import (
	"context"
	"slices"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/MatiXxD/beerer-bot/pkg/logger"
	"github.com/MatiXxD/beerer-bot/pkg/utils"
)

// Handler represents a function that handles a Telegram update.
type Handler func(ctx context.Context, upd tgbotapi.Update) error

// Middleware represents middleware for handling Telegram updates.
type Middleware func(next Handler) Handler

// callback represents a callback.
type callback struct {
	prefix  string
	handler Handler
}

// Router represents a Telegram router.
type Router struct {
	middlewares []Middleware

	text      Handler
	commands  map[string]Handler
	callbacks []callback

	fallback Handler
}

// NewRouter creates a new Telegram router.
func NewRouter() *Router {
	return &Router{
		commands: make(map[string]Handler),
		fallback: fallbackHandler,
	}
}

// Use adds middleware to the router.
func (r *Router) Use(m Middleware) {
	r.middlewares = append(r.middlewares, m)
}

// RegisterCommand registers a command handler for the given command name.
func (r *Router) RegisterCommand(name string, h Handler) {
	r.commands[name] = h
}

// RegisterCallback registers a callback handler for the given callback prefix
// more specific callbacks should be registered first.
func (r *Router) RegisterCallback(prefix string, h Handler) {
	r.callbacks = append(r.callbacks, callback{prefix: prefix, handler: h})
}

// RegisterText registers a text handler for the given text.
func (r *Router) RegisterText(h Handler) {
	r.text = h
}

// Handle represents the main handler for Telegram updates.
func (r *Router) Handle(ctx context.Context, upd tgbotapi.Update) error {
	// select right handler, fallback handler if not found
	h := r.resolveHandler(upd)

	// apply middlewares
	for i := range slices.Backward(r.middlewares) {
		h = r.middlewares[i](h)
	}

	return h(ctx, upd)
}

// resolveHandler resolves the handler for the given update.
func (r *Router) resolveHandler(upd tgbotapi.Update) Handler {
	switch {
	case upd.Message != nil && upd.Message.IsCommand():
		if h, ok := r.commands[upd.Message.Command()]; ok {
			return h
		}
	case upd.Message != nil && upd.Message.Text != "":
		if r.text != nil {
			return r.text
		}
	case upd.CallbackQuery != nil:
		// return the first callback handler that matches the callback prefix
		for _, cb := range r.callbacks {
			if strings.HasPrefix(upd.CallbackQuery.Data, cb.prefix) {
				return cb.handler
			}
		}
	}

	return fallbackHandler
}

// fallbackHandler represents a fallback handler, which will be used when no other handler is found.
func fallbackHandler(ctx context.Context, _ tgbotapi.Update) error {
	log := utils.GetZeroLogger(ctx)
	log.Warn().
		Str(logger.TagUnexpected, "handler was not found -> using fallback").
		Msg("fallback handler was called")

	return nil
}
