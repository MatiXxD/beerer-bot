package utils

import "fmt"

// Operation represents the operation.
type Operation string

// WithMsg returns a message with the operation.
func (o Operation) WithMsg(msg string) string {
	return fmt.Sprintf("%s: %s", o, msg)
}

// WithErr return a new error, which contains the operation.
func (o Operation) WithErr(err error) error {
	return fmt.Errorf("%s: %w", o, err)
}

// WithErrAndMsg returns a new error, which contains the operation and message.
func (o Operation) WithErrAndMsg(err error, msg string) error {
	return fmt.Errorf("%s: %s: %w", o, msg, err)
}
