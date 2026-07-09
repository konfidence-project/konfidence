package router_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/router"
)

var _ = Describe("Router", func() {
	var h http.Handler

	BeforeEach(func() {
		// nil scheme is acceptable in unit tests - no handler exercises the k8s client yet.
		h = router.New(slog.Default(), nil)
	})

	DescribeTable("probe routes return 200",
		func(path string) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Header().Get("Content-Type")).To(Equal("application/json"))
		},
		Entry("/healthz", "/healthz"),
		Entry("/readyz", "/readyz"),
	)

	It("returns 404 for unknown paths", func() {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/unknown", nil))
		Expect(rec.Code).To(Equal(http.StatusNotFound))
	})
})
