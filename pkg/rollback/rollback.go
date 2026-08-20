package rollback

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
)

// rollbackKey type for the rollback key.
type rollbackKey struct{}

// rbKey is the key for the rollback in the context.
var rbKey rollbackKey

// Func represents a rollback function.
type Func func() error

// Rollback represents a rollback.
type Rollback struct {
	rollbacks []Func
	mu        sync.RWMutex
}

// Add adds a rollback function.
func (rb *Rollback) Add(f Func) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.rollbacks = append(rb.rollbacks, f)
}

// Run runs the rollbacks.
func (rb *Rollback) Run() error {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	var errs []error

	for _, f := range slices.Backward(rb.rollbacks) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					errs = append(errs, fmt.Errorf("panic: %v", r))
				}
			}()

			if err := f(); err != nil {
				errs = append(errs, err)
			}
		}()
	}

	return errors.Join(errs...)
}

// RunAsync runs the rollbacks asynchronously.
func (rb *Rollback) RunAsync() error {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	var (
		errs  []error
		errMu sync.Mutex
		wg    sync.WaitGroup
	)

	for _, f := range slices.Backward(rb.rollbacks) {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					errMu.Lock()
					defer errMu.Unlock()
					errs = append(errs, fmt.Errorf("panic: %v", r))
				}
			}()

			if err := f(); err != nil {
				errMu.Lock()
				defer errMu.Unlock()
				errs = append(errs, err)
			}
		})
	}

	wg.Wait()

	return errors.Join(errs...)
}

// New creates a new rollback.
func New(ctx context.Context) (context.Context, *Rollback) {
	rb := Rollback{rollbacks: []Func{}}
	return Set(ctx, &rb), &rb
}

// Set sets the rollback in the context.
func Set(ctx context.Context, rb *Rollback) context.Context {
	return context.WithValue(ctx, rbKey, rb)
}

// Get gets the rollback from the context.
func Get(ctx context.Context) (*Rollback, bool) {
	rb, ok := ctx.Value(rbKey).(*Rollback)
	if !ok {
		return nil, false
	}

	return rb, true
}
