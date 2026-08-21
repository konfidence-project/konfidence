package ui_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing/fstest"

	"github.com/konfidence-project/konfidence/internal/api/ui"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UI handler", func() {
	var handler http.Handler

	BeforeEach(func() {
		var err error
		handler, err = ui.New(fstest.MapFS{
			"index.html":                       {Data: []byte("<h1>Konfidence</h1>")},
			"favicon.ico":                      {Data: []byte("icon")},
			"_app/immutable/chunks/example.js": {Data: []byte("export {}")},
		})
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("serves the SPA index",
		func(target string) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(recorder.Body.String()).To(ContainSubstring("Konfidence"))
			Expect(recorder.Header().Get("Cache-Control")).To(Equal("no-cache, must-revalidate"))
			Expect(recorder.Header().Get("ETag")).NotTo(BeEmpty())
		},
		Entry("at the root", "/"),
		Entry("for a deep link", "/projects/example/landscape"),
	)

	It("serves static files", func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/favicon.ico", nil))
		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Body.String()).To(Equal("icon"))
		Expect(recorder.Header().Get("Cache-Control")).To(Equal("public, max-age=3600"))
	})

	It("caches immutable assets", func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/_app/immutable/chunks/example.js", nil))
		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Header().Get("Cache-Control")).To(Equal("public, max-age=31536000, immutable"))
	})

	It("returns not modified when the SPA index ETag matches", func() {
		initial := httptest.NewRecorder()
		handler.ServeHTTP(initial, httptest.NewRequest(http.MethodGet, "/projects/example", nil))

		request := httptest.NewRequest(http.MethodGet, "/projects/example", nil)
		request.Header.Set("If-None-Match", initial.Header().Get("ETag"))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		Expect(recorder.Code).To(Equal(http.StatusNotModified))
		Expect(recorder.Body.Len()).To(BeZero())
	})

	DescribeTable("does not serve files outside the UI asset path",
		func(target string, expectedStatus int) {
			baseDir := GinkgoT().TempDir()
			assetDir := filepath.Join(baseDir, "ui")
			Expect(os.Mkdir(assetDir, 0o700)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(assetDir, "index.html"), []byte("index"), 0o600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(baseDir, "secret.txt"), []byte("secret"), 0o600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(baseDir, "secret"), []byte("secret"), 0o600)).To(Succeed())

			assetHandler, err := ui.New(os.DirFS(assetDir))
			Expect(err).NotTo(HaveOccurred())
			recorder := httptest.NewRecorder()
			assetHandler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))

			Expect(recorder.Code).To(Equal(expectedStatus))
			Expect(recorder.Body.String()).NotTo(ContainSubstring("secret"))
		},
		Entry("for a parent segment", "/../secret.txt", http.StatusNotFound),
		Entry("for encoded parent segments", "/%2e%2e/secret.txt", http.StatusNotFound),
		Entry("for an encoded separator", "/..%2fsecret.txt", http.StatusNotFound),
		Entry("for an extensionless SPA route", "/../secret", http.StatusOK),
	)

	DescribeTable("does not use the SPA fallback",
		func(method, target string) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
			Expect(recorder.Code).To(Equal(http.StatusNotFound))
			Expect(recorder.Header().Get("Content-Type")).To(HavePrefix("text/plain"))
		},
		Entry("for missing files", http.MethodGet, "/missing.js"),
		Entry("for mutating requests", http.MethodPost, "/projects"),
	)

	DescribeTable("negotiates the not found body on Accept",
		func(accept, target, expectedContentType string) {
			request := httptest.NewRequest(http.MethodGet, target, nil)
			request.Header.Set("Accept", accept)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			Expect(recorder.Code).To(Equal(http.StatusNotFound))
			Expect(recorder.Header().Get("Content-Type")).To(HavePrefix(expectedContentType))
		},
		Entry("JSON for a JSON-only client", "application/json", "/missing.js", "application/json"),
		Entry("JSON for a JSON client with wildcard", "application/json, */*", "/missing.js", "application/json"),
		Entry("plain text for a browser", "text/html,application/xhtml+xml,*/*;q=0.8", "/missing.js", "text/plain"),
		Entry("plain text for a wildcard client", "*/*", "/missing.js", "text/plain"),
		Entry("plain text without Accept", "", "/missing.js", "text/plain"),
	)

	It("returns a JSON not found body in the API error shape", func() {
		request := httptest.NewRequest(http.MethodGet, "/missing.js", nil)
		request.Header.Set("Accept", "application/json")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		var body map[string]map[string]string
		Expect(json.NewDecoder(recorder.Body).Decode(&body)).To(Succeed())
		Expect(body["error"]).To(HaveKeyWithValue("code", "not_found"))
		Expect(body["error"]).To(HaveKeyWithValue("message", ContainSubstring("/missing.js")))
	})

	It("still serves the SPA index to JSON clients on extensionless routes", func() {
		request := httptest.NewRequest(http.MethodGet, "/projects/example", nil)
		request.Header.Set("Accept", "application/json")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Header().Get("Content-Type")).To(HavePrefix("text/html"))
	})
})
