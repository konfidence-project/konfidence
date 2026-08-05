package handler

import (
	"net/http"
)

// Healthz handles GET /healthz — liveness probe.
// Always returns 200 {"status":"ok"} while the process is alive.
func Healthz(w http.ResponseWriter, _ *http.Request) error {
	writeJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: "ok"})
	return nil
}

// Readyz handles GET /readyz — readiness probe.
// Returns 200 {"status":"ok"} when the server is ready to accept traffic.
// Gate additional dependency checks here once they exist (e.g. cluster reachability).
func Readyz(w http.ResponseWriter, _ *http.Request) error {
	writeJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: "ok"})
	return nil
}
