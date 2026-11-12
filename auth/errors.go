package auth

import (
	"errors"
	"fmt"
)

// APIError describes a failed auth API operation and preserves its underlying cause.
type APIError struct {
	Operation  string
	StatusCode int
	Message    string
	Err        error
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}

	prefix := "auth"
	if e.Operation != "" {
		prefix += " " + e.Operation
	}
	if e.StatusCode != 0 {
		prefix += fmt.Sprintf(" (HTTP %d)", e.StatusCode)
	}

	switch {
	case e.Message != "" && e.Err != nil:
		return fmt.Sprintf("%s: %s: %v", prefix, e.Message, e.Err)
	case e.Message != "":
		return fmt.Sprintf("%s: %s", prefix, e.Message)
	case e.Err != nil:
		return fmt.Sprintf("%s: %v", prefix, e.Err)
	default:
		return prefix + " failed"
	}
}

// Unwrap exposes the original transport, encoding, or decoding error.
func (e *APIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ConfigError identifies invalid client configuration and preserves parse errors.
type ConfigError struct {
	Field string
	Err   error
}

func (e *ConfigError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err != nil {
		return fmt.Sprintf("invalid auth client config %s: %v", e.Field, e.Err)
	}
	return fmt.Sprintf("invalid auth client config %s", e.Field)
}

func (e *ConfigError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

var errBaseURLRequired = errors.New("base URL is required")
