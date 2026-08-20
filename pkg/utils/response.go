package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	BadRequestErrorMsgJSON     = `{"error": "BAD_REQUEST"}`
	UnauthorizedErrorMsgJSON   = `{"error": "UNAUTHORIZED"}`
	NotFoundErrorMsgJSON       = `{"error": "NOT_FOUND"}`
	ConflictErrorMsgJSON       = `{"error": "CONFLICT"}`
	UnprocessableEntityMsgJSON = `{"error": "UNPROCESSABLE_ENTITY"}`
	InternalServerErrorMsgJSON = `{"error": "INTERNAL"}`
)

// HttpError is a custom error type for HTTP responses
type HttpError struct {
	Error string `json:"error"`
}

// WriteJSON writes a JSON response to the writer.
func WriteJSON(ctx context.Context, w http.ResponseWriter, data any) {
	WriteStatusJSON(ctx, w, http.StatusOK, data)
}

// WriteStatusJSON writes a JSON response with a specific status code to the writer.
func WriteStatusJSON(ctx context.Context, w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log := GetZeroLogger(ctx)
		if log != nil {
			log.Error().Err(err).Msg("failed to encode json")
		}

		InternalServerErrorJSON(w)
	}
}

// BadRequestErrorJSON writes a JSON response with a 400 status code to the writer.
func BadRequestErrorJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = fmt.Fprint(w, BadRequestErrorMsgJSON)
}

// UnauthorizedErrorJSON writes a JSON response with a 401 status code to the writer.
func UnauthorizedErrorJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = fmt.Fprint(w, UnauthorizedErrorMsgJSON)
}

// NotFoundErrorJSON writes a JSON response with a 404 status code to the writer.
func NotFoundErrorJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprint(w, NotFoundErrorMsgJSON)
}

// ConflictErrorJSON writes a JSON response with a 409 status code to the writer.
func ConflictErrorJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_, _ = fmt.Fprint(w, ConflictErrorMsgJSON)
}

// UnprocessableEntityErrorJSON writes a JSON response with a 422 status code to the writer.
func UnprocessableEntityErrorJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_, _ = fmt.Fprint(w, UnprocessableEntityMsgJSON)
}

// InternalServerErrorJSON writes a JSON response with a 500 status code to the writer.
func InternalServerErrorJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = fmt.Fprint(w, InternalServerErrorMsgJSON)
}
