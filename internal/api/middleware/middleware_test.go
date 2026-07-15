package middleware_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
)

var _ = Describe("Logging middleware", func() {
	It("passes the request through and preserves the status code", func() {
		mw := middleware.Logging(slog.Default())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))

		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))
		Expect(rec.Code).To(Equal(http.StatusNoContent))
	})
})

var _ = Describe("Recovery middleware", func() {
	It("returns 500 JSON and keeps the server alive when the handler panics", func() {
		mw := middleware.Recovery(slog.Default())(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			panic("unexpected nil pointer")
		}))

		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))

		Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

		var body map[string]any
		Expect(json.NewDecoder(rec.Body).Decode(&body)).To(Succeed())
		Expect(body["error"]).NotTo(BeNil())
	})
})

var _ = Describe("ErrorHandler middleware", func() {
	var logger *slog.Logger

	BeforeEach(func() {
		logger = slog.Default()
	})

	wrap := func(h handler.Handler) http.Handler {
		return middleware.Handle(logger, h)
	}

	It("does nothing when handler returns nil", func() {
		h := handler.Handler(func(w http.ResponseWriter, _ *http.Request) error {
			w.WriteHeader(http.StatusOK)
			return nil
		})
		rec := httptest.NewRecorder()
		wrap(h).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
	})

	It("writes the APIError status and body when handler returns *APIError", func() {
		h := handler.Handler(func(_ http.ResponseWriter, _ *http.Request) error {
			return handler.NewNotFound("stage", "prod")
		})
		rec := httptest.NewRecorder()
		wrap(h).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		Expect(rec.Code).To(Equal(http.StatusNotFound))
		Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

		var body map[string]any
		Expect(json.NewDecoder(rec.Body).Decode(&body)).To(Succeed())
		errObj, ok := body["error"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(errObj["code"]).To(Equal("not_found"))
		Expect(errObj["message"]).To(ContainSubstring("prod"))
	})

	It("writes 500 and does not leak detail when handler returns a plain error", func() {
		h := handler.Handler(func(_ http.ResponseWriter, _ *http.Request) error {
			return fmt.Errorf("db password is hunter2")
		})
		rec := httptest.NewRecorder()
		wrap(h).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		Expect(rec.Code).To(Equal(http.StatusInternalServerError))

		var body map[string]any
		Expect(json.NewDecoder(rec.Body).Decode(&body)).To(Succeed())
		errObj := body["error"].(map[string]any)
		Expect(errObj["code"]).To(Equal("internal_server_error"))
		Expect(rec.Body.String()).NotTo(ContainSubstring("hunter2"))
	})

	It("does not leak the internal cause of an *APIError to the client", func() {
		h := handler.Handler(func(_ http.ResponseWriter, _ *http.Request) error {
			return handler.NewInternal(fmt.Errorf("secret connection string"))
		})
		rec := httptest.NewRecorder()
		wrap(h).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		Expect(rec.Body.String()).NotTo(ContainSubstring("secret connection string"))
	})
})
