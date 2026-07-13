package router_test

import (
	"bytes"
	"encoding/json"
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
	var idp *httptest.Server

	BeforeEach(func() {
		idp = newTestIDP()
		DeferCleanup(idp.Close)
		// nil scheme is acceptable in unit tests - no handler exercises the k8s client yet.
		h = router.New(slog.Default(), nil, testAuthConfig(idp.URL))
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

	Describe("auth routes", func() {
		It("rejects stage listing without a session", func() {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/stages", nil))
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		})

		It("allows stage listing with a session", func() {
			sessionID := uiLogin(h)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/stages", nil)
			req.Header.Set("X-Session-ID", sessionID)
			h.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("creates a UI session through the IDP token-handler redirect flow", func() {
			sessionID := uiLogin(h)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/identity", nil)
			req.Header.Set("X-Session-ID", sessionID)
			h.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))

			var body map[string]any
			Expect(json.NewDecoder(rec.Body).Decode(&body)).To(Succeed())
			Expect(body["username"]).To(Equal("alice"))
			Expect(body["role"]).To(Equal("ADMIN"))
		})

		It("accepts a bearer session identifier", func() {
			sessionID := uiLogin(h)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/identity", nil)
			req.Header.Set("Authorization", "Bearer "+sessionID)
			h.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("removes the session on logout", func() {
			sessionID := uiLogin(h)

			logoutRec := httptest.NewRecorder()
			logoutReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
			logoutReq.Header.Set("X-Session-ID", sessionID)
			h.ServeHTTP(logoutRec, logoutReq)
			Expect(logoutRec.Code).To(Equal(http.StatusOK))

			identityRec := httptest.NewRecorder()
			identityReq := httptest.NewRequest(http.MethodGet, "/identity", nil)
			identityReq.Header.Set("X-Session-ID", sessionID)
			h.ServeHTTP(identityRec, identityReq)
			Expect(identityRec.Code).To(Equal(http.StatusUnauthorized))
		})

		It("supports CLI exchange after the browser redirects to the local callback", func() {
			exchangeCode := cliExchangeCode(h)

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/sessions/exchange", bytes.NewBufferString(`{"code":"`+exchangeCode+`"}`)))
			Expect(rec.Code).To(Equal(http.StatusOK))

			var body struct {
				SessionID string `json:"sessionId"`
			}
			Expect(json.NewDecoder(rec.Body).Decode(&body)).To(Succeed())
			Expect(body.SessionID).NotTo(BeEmpty())
		})

		It("requires authorization callback parameters", func() {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/callback", nil))
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})

		It("rejects invalid session exchange codes", func() {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/sessions/exchange", bytes.NewBufferString(`{"code":"unknown"}`)))
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		})

		It("requires a session for identity", func() {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/identity", nil))
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})

func uiLogin(h http.Handler) string {
	callbackRec := followLoginRedirects(h, "/login")
	Expect(callbackRec.Code).To(Equal(http.StatusOK))

	var body struct {
		SessionID string `json:"sessionId"`
	}
	Expect(json.NewDecoder(callbackRec.Body).Decode(&body)).To(Succeed())
	Expect(body.SessionID).NotTo(BeEmpty())
	return body.SessionID
}

func cliExchangeCode(h http.Handler) string {
	callbackRec := followLoginRedirects(h, "/login?returnTo=http://127.0.0.1:12345/session")
	Expect(callbackRec.Code).To(Equal(http.StatusFound))

	location, err := url.Parse(callbackRec.Header().Get("Location"))
	Expect(err).NotTo(HaveOccurred())
	Expect(location.Scheme).To(Equal("http"))
	Expect(location.Host).To(Equal("127.0.0.1:12345"))
	Expect(location.Path).To(Equal("/session"))
	exchangeCode := location.Query().Get("code")
	Expect(exchangeCode).NotTo(BeEmpty())
	return exchangeCode
}

func followLoginRedirects(h http.Handler, startPath string) *httptest.ResponseRecorder {
	loginRec := httptest.NewRecorder()
	h.ServeHTTP(loginRec, httptest.NewRequest(http.MethodGet, startPath, nil))
	Expect(loginRec.Code).To(Equal(http.StatusFound))

	idpLocation, err := url.Parse(loginRec.Header().Get("Location"))
	Expect(err).NotTo(HaveOccurred())
	Expect(idpLocation.Path).To(Equal("/authorize"))
	Expect(idpLocation.Query().Get("state")).NotTo(BeEmpty())
	Expect(idpLocation.Query().Get("code_challenge")).NotTo(BeEmpty())
	Expect(idpLocation.Query().Get("code_challenge_method")).To(Equal("S256"))

	client := http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	idpResp, err := client.Get(idpLocation.String()) //nolint:noctx
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = idpResp.Body.Close() }()
	Expect(idpResp.StatusCode).To(Equal(http.StatusFound))

	callbackLocation, err := url.Parse(idpResp.Header.Get("Location"))
	Expect(err).NotTo(HaveOccurred())
	Expect(callbackLocation.Path).To(Equal("/auth/callback"))
	Expect(callbackLocation.Query().Get("code")).NotTo(BeEmpty())
	Expect(callbackLocation.Query().Get("state")).NotTo(BeEmpty())

	callbackRec := httptest.NewRecorder()
	h.ServeHTTP(callbackRec, httptest.NewRequest(http.MethodGet, callbackLocation.String(), nil))
	return callbackRec
}

func testAuthConfig(baseURL string) config.AuthConfig {
	return config.AuthConfig{
		AuthorizeURL: baseURL + "/authorize",
		TokenURL:     baseURL + "/token",
		UserInfoURL:  baseURL + "/userinfo",
		ClientID:     "kden-local",
		RedirectURI:  "http://api.example.test/auth/callback",
		Scopes:       "openid profile email groups",
	}
}

func newTestIDP() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")
		callback, err := url.Parse(redirectURI)
		Expect(err).NotTo(HaveOccurred())
		query := callback.Query()
		query.Set("code", "mock-code")
		query.Set("state", state)
		callback.RawQuery = query.Encode()
		http.Redirect(w, r, callback.String(), http.StatusFound)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(`{"sub":"dex|alice","preferred_username":"alice","email":"alice@example.com","name":"Alice Admin","email_verified":true,"groups":["admins"]}`),
		)
	})
	return httptest.NewServer(mux)
}
