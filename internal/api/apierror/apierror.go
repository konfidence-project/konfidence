package apierror

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Error carries an HTTP status and a message safe to send to the client.
type Error struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func NewNotFound(resource, name string) *Error {
	return &Error{
		Status:  http.StatusNotFound,
		Code:    "not_found",
		Message: resource + " \"" + name + "\" not found",
	}
}

func NewBadRequest(message string, cause error) *Error {
	return &Error{
		Status:  http.StatusBadRequest,
		Code:    "bad_request",
		Message: message,
		Err:     cause,
	}
}

func NewInternal(cause error) *Error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Code:    "internal_server_error",
		Message: "an unexpected error occurred",
		Err:     cause,
	}
}

func As(err error) *Error {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

func Write(w http.ResponseWriter, err *Error) {
	writeJSON(w, err.Status, response{Error: body{Code: err.Code, Message: err.Message}})
}

func WriteInternal(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, response{Error: body{
		Code:    "internal_server_error",
		Message: "an unexpected error occurred",
	}})
}

type body struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type response struct {
	Error body `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
