package router_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/router"
)

var _ = Describe("Router", func() {
	var h http.Handler

	BeforeEach(func() {
		// nil client is acceptable in unit tests - no handler exercises the k8s client yet.
		h = router.New(slog.Default(), nil, config.AuthConfig{})
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

	DescribeTable("protected auth routes require a session",
		func(method, path string) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		},
		Entry("identity", http.MethodGet, "/api/identity"),
		Entry("logout", http.MethodPost, "/api/logout"),
	)

	It("keeps login under the api base path", func() {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/login", nil))
		Expect(rec.Code).To(Equal(http.StatusInternalServerError))
	})

	It("accepts relative returnTo values", func() {
		h = router.New(slog.Default(), nil, validAuthConfig())

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/login?returnTo=/app/callback%3Ftab%3Dstages", nil))
		Expect(rec.Code).To(Equal(http.StatusFound))

		location, err := url.Parse(rec.Header().Get("Location"))
		Expect(err).NotTo(HaveOccurred())
		Expect(location.Query().Get("state")).NotTo(BeEmpty())
	})

	DescribeTable("rejects unsafe returnTo values",
		func(returnTo string) {
			h = router.New(slog.Default(), nil, validAuthConfig())

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/login?returnTo="+url.QueryEscape(returnTo), nil))
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		},
		Entry("absolute URL", "https://evil.example/callback"),
		Entry("protocol-relative URL", "//evil.example/callback"),
		Entry("relative without leading slash", "app/callback"),
	)
})

func validAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		AuthorizeURL: "http://idp.example.test/authorize",
		TokenURL:     "http://idp.example.test/token",
		UserInfoURL:  "http://idp.example.test/userinfo",
		ClientID:     "kden-local",
		RedirectURI:  "http://localhost:8090/api/auth/callback",
		Scopes:       "openid profile email groups",
	}
}
