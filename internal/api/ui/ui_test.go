package ui_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/ui"
)

var _ = Describe("UI handler", func() {
	It("serves the SPA entry point for application routes", func() {
		request := httptest.NewRequest(http.MethodGet, "/projects/example/landscape", nil)
		response := httptest.NewRecorder()

		ui.Handler().ServeHTTP(response, request)

		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(response.Header().Get("Content-Type")).To(HavePrefix("text/html"))
	})
})
