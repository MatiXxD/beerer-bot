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
		commands:  make(map[string]Handler),
		callbacks: []callback{},
		text:      defaultTextHandler,
		fallback:  defaultFallbackHandler,
	}
}

// WithFallback sets the fallback handler.
func (r *Router) WithFallback(h Handler) *Router {
	r.fallback = h
	return r
}

// WithText sets the text handler.
func (r *Router) WithText(h Handler) *Router {
	r.text = h
	return r
}

// Use adds middleware to the router.
func (r *Router) Use(m Middleware) {
	r.middlewares = append(r.middlewares, m)
}

// RegisterCommand registers a command handler for the given command name.
func (r *Router) RegisterCommand(name string, h Handler) {
	if h == nil {
		panic("telegram: nil handler for command " + name)
	}

	r.commands[name] = h
}

// RegisterCallback registers a callback handler for the given callback prefix
// more specific callbacks should be registered first.
func (r *Router) RegisterCallback(prefix string, h Handler) {
	if h == nil {
		panic("telegram: nil handler for callback prefix " + prefix)
	}

	r.callbacks = append(r.callbacks, callback{prefix: prefix, handler: h})
}

// RegisterText registers a text handler for the given text.
func (r *Router) RegisterText(h Handler) {
	if h == nil {
		panic("telegram: nil text handler")
	}

	r.text = h
}

// RegisterFallback registers the fallback handler.
func (r *Router) RegisterFallback(h Handler) {
	if h == nil {
		panic("telegram: nil fallback handler")
	}

	r.fallback = h
}

// Build builds the main handler for Telegram updates with all middlewares applied.
// Middlewares added after Build are not applied, but handlers can still be registered.
func (r *Router) Build() Handler {
	h := Handler(func(ctx context.Context, upd tgbotapi.Update) error {
		return r.resolveHandler(upd)(ctx, upd)
	})

	for _, m := range slices.Backward(r.middlewares) {
		h = m(h)
	}

	return h
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

	return r.fallback
}

// defaultFallbackHandler represents a default fallback handler, which will be used by default in NewRouter.
func defaultFallbackHandler(ctx context.Context, _ tgbotapi.Update) error {
	log := utils.GetZeroLogger(ctx)
	log.Warn().
		Str(logger.TagUnexpected, "handler was not found -> using fallback").
		Msg("fallback handler was called")

	return nil
}

// defaultTextHandler represents a default text handler, which will be used by default in NewRouter.
func defaultTextHandler(ctx context.Context, upd tgbotapi.Update) error {
	log := utils.GetZeroLogger(ctx)
	log.Warn().
		Str(logger.TagUnexpected, "text handler was not found").
		Msgf("using the default text handler: msg=%s", upd.Message.Text)

	return nil
}
