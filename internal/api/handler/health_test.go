package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/handler"
)

var _ = Describe("Health handlers", func() {
	Describe("Healthz", func() {
		It("returns 200 with status ok", func() {
			rec := httptest.NewRecorder()
			Expect(handler.Healthz(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))).To(Succeed())

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

			var body map[string]string
			Expect(json.NewDecoder(rec.Body).Decode(&body)).To(Succeed())
			Expect(body["status"]).To(Equal("ok"))
		})
	})

	Describe("Readyz", func() {
		It("returns 200 with status ok", func() {
			rec := httptest.NewRecorder()
			Expect(handler.Readyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))).To(Succeed())

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))

			var body map[string]string
			Expect(json.NewDecoder(rec.Body).Decode(&body)).To(Succeed())
			Expect(body["status"]).To(Equal("ok"))
		})
	})
})

var _ = Describe("APIError", func() {
	Describe("constructors", func() {
		It("NewNotFound produces a 404 with not_found code", func() {
			err := handler.NewNotFound("stage", "my-stage")
			Expect(err.Status).To(Equal(http.StatusNotFound))
			Expect(err.Code).To(Equal("not_found"))
			Expect(err.Message).To(ContainSubstring("my-stage"))
		})

		It("NewBadRequest produces a 400 with bad_request code", func() {
			cause := fmt.Errorf("missing field")
			err := handler.NewBadRequest("invalid input", cause)
			Expect(err.Status).To(Equal(http.StatusBadRequest))
			Expect(err.Code).To(Equal("bad_request"))
			Expect(err.Unwrap()).To(Equal(cause))
		})

		It("NewInternal produces a 500 with internal_server_error code", func() {
			cause := fmt.Errorf("db connection failed")
			err := handler.NewInternal(cause)
			Expect(err.Status).To(Equal(http.StatusInternalServerError))
			Expect(err.Code).To(Equal("internal_server_error"))
			Expect(err.Unwrap()).To(Equal(cause))
		})
	})

	Describe("AsAPIError", func() {
		It("returns the *APIError directly", func() {
			apiErr := handler.NewNotFound("vector", "v1")
			Expect(handler.AsAPIError(apiErr)).To(Equal(apiErr))
		})

		It("returns nil for a plain error", func() {
			Expect(handler.AsAPIError(fmt.Errorf("plain"))).To(BeZero())
		})

		It("unwraps a wrapped *APIError", func() {
			apiErr := handler.NewNotFound("stage", "s1")
			Expect(handler.AsAPIError(fmt.Errorf("outer: %w", apiErr))).To(Equal(apiErr))
		})
	})
})
