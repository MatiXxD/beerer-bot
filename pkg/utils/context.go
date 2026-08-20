package utils

import (
	"context"

	"github.com/rs/zerolog"
)

// ctxKey is a type for context keys to avoid collisions.
type ctxKey int

const (
	ctxKeyReqID ctxKey = iota
	ctxKeyLogger
)

// SetReqID sets the request ID in the context.
func SetReqID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, ctxKeyReqID, reqID)
}

// GetReqID gets the request ID from the context.
func GetReqID(ctx context.Context) string {
	reqid, ok := ctx.Value(ctxKeyReqID).(string)
	if !ok {
		return ""
	}

	return reqid
}

// SetZeroLogger sets the zerolog logger in the context.
func SetZeroLogger(ctx context.Context, l *zerolog.Logger) context.Context {
	if l == nil {
		return ctx
	}

	return context.WithValue(ctx, ctxKeyLogger, l)
}

// GetZeroLogger gets the zerolog logger from the context.
func GetZeroLogger(ctx context.Context) *zerolog.Logger {
	l, ok := ctx.Value(ctxKeyLogger).(*zerolog.Logger)
	if !ok {
		return nil
	}

	return l
}
