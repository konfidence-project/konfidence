package handler

import (
	"errors"
	"net/http"
)

// Handler is like http.HandlerFunc but returns an error.
// Returning a non-nil error signals to Handle() that the request could not
// be completed. Handlers must not write to w after returning an error —
// the caller owns the response at that point.
type Handler func(w http.ResponseWriter, r *http.Request) error

// APIError is a domain error that carries the HTTP status and a message safe
// to send to the client. The internal Err field is logged by Handle() but
// never included in the response body.
type APIError struct {
	Status  int    // HTTP status code to respond with
	Code    string // machine-readable error code, e.g. "not_found"
	Message string // human-readable message, safe to expose to the client
	Err     error  // underlying cause — logged, never sent to the client
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *APIError) Unwrap() error { return e.Err }

func NewNotFound(resource, name string) *APIError {
	return &APIError{
		Status:  http.StatusNotFound,
		Code:    "not_found",
		Message: resource + " \"" + name + "\" not found",
	}
}

func NewBadRequest(message string, cause error) *APIError {
	return &APIError{
		Status:  http.StatusBadRequest,
		Code:    "bad_request",
		Message: message,
		Err:     cause,
	}
}

func NewInternal(cause error) *APIError {
	return &APIError{
		Status:  http.StatusInternalServerError,
		Code:    "internal_server_error",
		Message: "an unexpected error occurred",
		Err:     cause,
	}
}

// AsAPIError unwraps err to find an *APIError. Returns nil if not found.
func AsAPIError(err error) *APIError {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

// errorBody is the JSON shape sent to the client on error.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newErrorBody(code, message string) errorBody {
	return errorBody{Code: code, Message: message}
}

// errorResponse wraps errorBody in an "error" envelope.
type errorResponse struct {
	Error errorBody `json:"error"`
}
