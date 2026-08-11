package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/konfidence-project/konfidence/internal/api/apierror"
)

type ErrHandler func(http.ResponseWriter, *http.Request) error

// Handle wraps an ErrHandler as an http.Handler, logging and responding to
// any returned error in one place.
//
//   - *apierror.Error    → write its Status + Code + Message. Internal cause
//     (Err != nil) is logged but never sent to the client.
//   - any other error    → log at error level, respond 500 with a generic body.
//   - nil                → handler already wrote its own response; do nothing.
func Handle(logger *slog.Logger, h ErrHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			handleError(w, r, err, logger)
		}
	})
}

// Logging returns middleware that logs each request: method, path, status, and duration.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration", time.Since(start).String(),
			)
		})
	}
}

// Recovery is a last-resort safety net that catches unexpected panics so the
// server process stays alive. It is NOT the primary error handling path —
// handlers must return errors via ErrHandler, not panic.
// Panics that reach here represent programming bugs and are always logged as errors.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					logger.Error("panic recovered - this is a bug",
						"panic", fmt.Sprint(v),
						"method", r.Method,
						"path", r.URL.Path,
					)
					apierror.WriteInternal(w)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func handleError(w http.ResponseWriter, r *http.Request, err error, logger *slog.Logger) {
	if apiErr := apierror.As(err); apiErr != nil {
		if apiErr.Err != nil {
			logger.Error("request failed",
				"method", r.Method,
				"path", r.URL.Path,
				"code", apiErr.Code,
				"error", apiErr.Err.Error(),
			)
		}
		apierror.Write(w, apiErr)
		return
	}

	logger.Error("unhandled error",
		"method", r.Method,
		"path", r.URL.Path,
		"error", err.Error(),
	)
	apierror.WriteInternal(w)
}

// responseWriter wraps http.ResponseWriter to capture the written status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}
