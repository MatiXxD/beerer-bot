package telegram

import (
	"context"
	"sync"
	"time"

	"github.com/MatiXxD/beerer-bot/pkg/logger"
	"github.com/MatiXxD/beerer-bot/pkg/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Dispatcher represents dispatcher for telegram bot.
type Dispatcher struct {
	handler Handler

	mu   sync.Mutex
	wg   sync.WaitGroup
	done chan struct{}

	queues map[int64]chan tgbotapi.Update
}

// NewDispatcher creates a new dispatcher.
func NewDispatcher(h Handler) *Dispatcher {
	return &Dispatcher{
		handler: h,
		queues:  make(map[int64]chan tgbotapi.Update),
		done:    make(chan struct{}),
	}
}

// Dispatch dispatches the update to the handler.
func (d *Dispatcher) Dispatch(ctx context.Context, upd tgbotapi.Update) {
	log := utils.GetZeroLogger(ctx)

	from := upd.SentFrom()
	if from == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// create queue for the user if it doesn't exist'
	q, ok := d.queues[from.ID]
	if !ok {
		q = make(chan tgbotapi.Update, defaultQueueSize)
		d.queues[from.ID] = q
		d.wg.Add(1)

		// keep context values, but let queued updates finish during graceful
		// shutdown after the application context is canceled.
		go d.worker(context.WithoutCancel(ctx), from.ID, q)
	}

	// logging if queue is full
	select {
	case q <- upd:
	default:
		log.Warn().
			Str(logger.TagUnexpected, "queue for the user if full, dropping update").
			Msgf("queue for user %d is full", from.ID)
	}
}

// Stop stops the dispatcher.
func (d *Dispatcher) Stop() {
	close(d.done)
	d.wg.Wait()
}

// worker processes updates from the user.
func (d *Dispatcher) worker(
	ctx context.Context,
	userID int64,
	q chan tgbotapi.Update,
) {
	defer d.wg.Done()

	idle := time.NewTimer(defaultWorkerIdleTTL)
	defer idle.Stop()

	for {
		select {
		case upd := <-q:
			d.handle(ctx, upd)

			if !idle.Stop() {
				<-idle.C
			}

			idle.Reset(defaultWorkerIdleTTL)
		case <-d.done:
			// handle all user's updates before return
			for {
				select {
				case upd := <-q:
					d.handle(ctx, upd)
				default:
					d.mu.Lock()
					delete(d.queues, userID)
					d.mu.Unlock()
					return
				}
			}
		case <-idle.C:
			d.mu.Lock()

			// check if queue is empty; if so, remove the user queue
			if len(q) == 0 {
				delete(d.queues, userID)
				d.mu.Unlock()
				return
			}

			d.mu.Unlock()
			idle.Reset(defaultWorkerIdleTTL)
		}
	}
}

// handle handles the update.
func (d *Dispatcher) handle(ctx context.Context, upd tgbotapi.Update) {
	const op = utils.Operation("Dispatcher.handle")

	log := utils.GetZeroLogger(ctx)

	ctx, cancel := context.WithTimeout(ctx, defaultHandleTimeout)
	defer cancel()

	if err := d.handler(ctx, upd); err != nil {
		log.Error().
			Err(op.WithErrAndMsg(err, "failed to handle update")).
			Msg("failed to handle update")
	}
}
