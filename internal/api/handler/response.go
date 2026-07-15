package handler

import (
	"encoding/json"
	"net/http"
)

// writeJSON sets Content-Type, writes the status code, and encodes v as JSON.
// Encoding errors are silently dropped - headers are already sent at this point.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteAPIError writes an *APIError as a structured JSON error response.
// Called by the ErrorHandler middleware - exported so middleware can use it
// without importing the full handler package circularly.
func WriteAPIError(w http.ResponseWriter, err *APIError) {
	writeJSON(w, err.Status, errorResponse{Error: newErrorBody(err.Code, err.Message)})
}

// WriteInternalError writes a generic 500 JSON error response.
// Used by both ErrorHandler and Recovery so the response shape is consistent.
func WriteInternalError(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError,
		errorResponse{Error: newErrorBody("internal_server_error", "an unexpected error occurred")},
	)
}
