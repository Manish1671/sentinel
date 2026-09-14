package apierr

import (
	"errors"
	"net/http"
)

type Error struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Code + ": " + e.Message
}

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message, Details: map[string]any{}}
}

func WithDetails(status int, code, message string, details map[string]any) *Error {
	if details == nil {
		details = map[string]any{}
	}
	return &Error{Status: status, Code: code, Message: message, Details: details}
}

func Validation(fields []map[string]string) *Error {
	return WithDetails(http.StatusBadRequest, "validation_error", "Request validation failed.", map[string]any{
		"fields": fields,
	})
}

func Field(path, code, message string) map[string]string {
	return map[string]string{"path": path, "code": code, "message": message}
}

func Unauthenticated() *Error {
	return New(http.StatusUnauthorized, "unauthenticated", "Authentication required.")
}

func InvalidCredentials() *Error {
	return New(http.StatusUnauthorized, "unauthenticated", "Invalid email or password.")
}

func UserDisabled() *Error {
	return New(http.StatusUnauthorized, "user_disabled", "This account is disabled.")
}

func Forbidden() *Error {
	return New(http.StatusForbidden, "forbidden", "You are not allowed to perform this action.")
}

func NotFound(message string) *Error {
	return New(http.StatusNotFound, "not_found", message)
}

func IdempotencyConflict() *Error {
	return New(http.StatusConflict, "idempotency_key_conflict", "Idempotency-Key was reused with a different request body.")
}

func Internal() *Error {
	return New(http.StatusInternalServerError, "internal_error", "An internal error occurred.")
}

func As(err error) (*Error, bool) {
	var api *Error
	if errors.As(err, &api) {
		return api, true
	}
	return nil, false
}
