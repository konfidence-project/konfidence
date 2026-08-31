package middleware_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
	"github.com/konfidence-project/konfidence/internal/api/session"
)

var _ = Describe("SessionAuthentication routing", func() {
	var handler http.Handler

	BeforeEach(func() {
		var err error
		handler, err = middleware.SessionAuthentication(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			&testSessionStore{sessions: map[string]*session.Session{}},
			&testAuthRepository{},
			config.Parsed{Session: config.ParsedSessionConfig{Cookie: config.SessionCookieConfig{Name: "session"}}},
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		)
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("returns a JSON 404 for paths outside the OpenAPI spec",
		func(method, target string) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(method, target, nil))

			Expect(recorder.Code).To(Equal(http.StatusNotFound))
			Expect(recorder.Header().Get("Content-Type")).To(Equal("application/json"))

			var body map[string]map[string]string
			Expect(json.NewDecoder(recorder.Body).Decode(&body)).To(Succeed())
			Expect(body["error"]).To(HaveKeyWithValue("code", "not_found"))
			Expect(body["error"]).To(HaveKeyWithValue("message", ContainSubstring(target)))
		},
		Entry("for the API root", http.MethodGet, "/api"),
		Entry("for an unknown versioned path", http.MethodGet, "/api/v1/unknown"),
		Entry("for an unknown unversioned path", http.MethodGet, "/api/unknown"),
		Entry("for a mutating request to an unknown path", http.MethodPost, "/api/v1/unknown"),
	)

	It("still returns 401 for a known path without a session", func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil))

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		var body map[string]map[string]string
		Expect(json.NewDecoder(recorder.Body).Decode(&body)).To(Succeed())
		Expect(body["error"]).To(HaveKeyWithValue("code", "unauthorized"))
	})
})
