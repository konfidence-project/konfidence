package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"
)

type recordingCookieStore struct {
	mu          sync.Mutex
	loaded      *http.Cookie
	saved       *http.Cookie
	loadErr     error
	saveErr     error
	deleteErr   error
	deleteCalls int
}

func (s *recordingCookieStore) Load(string) (*http.Cookie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loaded, s.loadErr
}

func (s *recordingCookieStore) Save(
	_ string,
	cookie *http.Cookie,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved = cookie
	return s.saveErr
}

func (s *recordingCookieStore) Delete(string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteCalls++
	s.loaded = nil
	return s.deleteErr
}

var _ = Describe("Client login", func() {
	It("completes the loopback PKCE flow and persists the session", func() {
		var (
			callbackURL       string
			observedBrowser   string
			observedCode      string
			observedVerifier  string
			observedChallenge string
		)

		server := httptest.NewServer(http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			switch request.URL.Path {
			case "/api/v1/identity":
				writer.WriteHeader(http.StatusUnauthorized)

			case "/api/v1/login":
				callbackURL = request.URL.Query().Get("return_url")
				observedChallenge = request.URL.Query().Get("code_challenge")
				writer.Header().Set(
					"Location",
					"https://idp.example.test/authorize",
				)
				writer.WriteHeader(http.StatusFound)

			case "/api/v1/exchange":
				var body struct {
					Code     string `json:"code"`
					Verifier string `json:"verifier"`
				}
				Expect(json.NewDecoder(request.Body).Decode(&body)).To(Succeed())

				observedCode = body.Code
				observedVerifier = body.Verifier
				writer.Header().Set(
					"Set-Cookie",
					"kden-session=session-id; Path=/; HttpOnly",
				)
				writer.WriteHeader(http.StatusOK)

			default:
				http.NotFound(writer, request)
			}
		}))
		DeferCleanup(server.Close)

		store := &recordingCookieStore{}
		client, err := NewClient(
			server.URL+"/api",
			"",
			store,
			time.Second,
			time.Second,
		)
		Expect(err).NotTo(HaveOccurred())

		client.openURL = func(location string) error {
			observedBrowser = location

			response, err := http.Get(callbackURL + "&code=exchange-code")
			if err != nil {
				return err
			}

			defer func() { _ = response.Body.Close() }()

			Expect(response.StatusCode).To(Equal(http.StatusOK))
			return nil
		}

		Expect(client.Login(context.Background())).To(Succeed())

		Expect(observedBrowser).To(Equal(
			"https://idp.example.test/authorize",
		))
		Expect(observedCode).To(Equal("exchange-code"))
		Expect(observedVerifier).NotTo(BeEmpty())
		Expect(observedChallenge).To(Equal(
			oauth2.S256ChallengeFromVerifier(observedVerifier),
		))

		store.mu.Lock()
		defer store.mu.Unlock()
		Expect(store.deleteCalls).To(Equal(1))
		Expect(store.saved).NotTo(BeNil())
		Expect(store.saved.Name).To(Equal("kden-session"))
		Expect(store.saved.Value).To(Equal("session-id"))
	})

	It("does not start login when the existing session is valid", func() {
		var loginCalls int

		server := httptest.NewServer(http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			switch request.URL.Path {
			case "/api/v1/identity":
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(`{
					"name":"Test User",
					"givenName":"Test",
					"familyName":"User",
					"email":"test@example.com",
					"projectRoles":{}
				}`))
			case "/api/v1/login":
				loginCalls++
				writer.WriteHeader(http.StatusFound)
			}
		}))
		DeferCleanup(server.Close)

		client, err := NewClient(
			server.URL+"/api",
			"",
			&recordingCookieStore{},
			time.Second,
			time.Second,
		)
		Expect(err).NotTo(HaveOccurred())

		client.openURL = func(string) error {
			return errors.New("browser must not be opened")
		}

		Expect(client.Login(context.Background())).To(Succeed())
		Expect(loginCalls).To(BeZero())
	})

	It("times out a stalled API request", func() {
		server := httptest.NewServer(http.HandlerFunc(func(
			_ http.ResponseWriter,
			request *http.Request,
		) {
			<-request.Context().Done()
		}))
		DeferCleanup(server.Close)

		client, err := NewClient(
			server.URL,
			"",
			&recordingCookieStore{},
			time.Second,
			50*time.Millisecond,
		)
		Expect(err).NotTo(HaveOccurred())

		_, err = client.hasValidSession(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue())
	})
})

var _ = Describe("Login callback handler", func() {
	var results chan loginResult

	BeforeEach(func() {
		results = make(chan loginResult, 1)
	})

	It("accepts a valid callback", func() {
		request := httptest.NewRequest(
			http.MethodGet,
			"/callback?state=local-state&code=exchange-code",
			nil,
		)
		response := httptest.NewRecorder()

		loginCallbackHandler("local-state", results).
			ServeHTTP(response, request)

		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(response.Header().Get("Cache-Control")).To(Equal("no-store"))
		Expect(response.Header().Get("X-Frame-Options")).To(Equal("DENY"))
		Expect(response.Header().Get("Content-Security-Policy")).
			To(ContainSubstring("style-src 'self' 'unsafe-inline'"))

		result := <-results
		Expect(result.err).NotTo(HaveOccurred())
		Expect(result.code).To(Equal("exchange-code"))
	})

	It("forwards an authentication error", func() {
		request := httptest.NewRequest(
			http.MethodGet,
			"/callback?state=local-state&error=access_denied"+
				"&error_description=user+denied+access",
			nil,
		)
		response := httptest.NewRecorder()

		loginCallbackHandler("local-state", results).
			ServeHTTP(response, request)

		Expect(response.Code).To(Equal(http.StatusUnauthorized))

		result := <-results
		Expect(result.err).To(MatchError(
			"authentication failed: access_denied: user denied access",
		))
	})

	DescribeTable("rejects invalid callbacks",
		func(method, target string, expectedStatus int) {
			request := httptest.NewRequest(method, target, nil)
			response := httptest.NewRecorder()

			loginCallbackHandler("local-state", results).
				ServeHTTP(response, request)

			Expect(response.Code).To(Equal(expectedStatus))
			Expect(results).To(BeEmpty())
		},
		Entry(
			"wrong method",
			http.MethodPost,
			"/callback?state=local-state&code=exchange-code",
			http.StatusMethodNotAllowed,
		),
		Entry(
			"wrong path",
			http.MethodGet,
			"/other?state=local-state&code=exchange-code",
			http.StatusNotFound,
		),
		Entry(
			"wrong state",
			http.MethodGet,
			"/callback?state=wrong&code=exchange-code",
			http.StatusBadRequest,
		),
		Entry(
			"missing code",
			http.MethodGet,
			"/callback?state=local-state",
			http.StatusBadRequest,
		),
	)

	It("rejects a duplicate callback", func() {
		results <- loginResult{code: "first-code"}

		request := httptest.NewRequest(
			http.MethodGet,
			"/callback?state=local-state&code=second-code",
			nil,
		)
		response := httptest.NewRecorder()

		loginCallbackHandler("local-state", results).
			ServeHTTP(response, request)

		Expect(response.Code).To(Equal(http.StatusConflict))
		Expect((<-results).code).To(Equal("first-code"))
	})
})

var _ = Describe("Client access-token authentication", func() {
	It("adds the bearer token to API requests", func() {
		authorization := make(chan string, 1)

		server := httptest.NewServer(http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			authorization <- request.Header.Get("Authorization")
			writer.WriteHeader(http.StatusUnauthorized)
		}))
		DeferCleanup(server.Close)

		store := &recordingCookieStore{
			loadErr: errors.New("cookie store must not be loaded"),
		}

		client, err := NewClient(
			server.URL+"/api",
			"access-token",
			store,
			time.Second,
			time.Second,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(client.UsesAccessToken()).To(BeTrue())

		response, err := client.KdenApiClient().
			GetIdentityV1WithResponse(context.Background())

		Expect(err).NotTo(HaveOccurred())
		Expect(response.StatusCode()).To(Equal(http.StatusUnauthorized))
		Expect(<-authorization).To(Equal("Bearer access-token"))
	})

	It("does not add an authorization header without a token", func() {
		authorization := make(chan string, 1)

		server := httptest.NewServer(http.HandlerFunc(func(
			writer http.ResponseWriter,
			request *http.Request,
		) {
			authorization <- request.Header.Get("Authorization")
			writer.WriteHeader(http.StatusUnauthorized)
		}))
		DeferCleanup(server.Close)

		client, err := NewClient(
			server.URL+"/api",
			"",
			&recordingCookieStore{},
			time.Second,
			time.Second,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(client.UsesAccessToken()).To(BeFalse())

		_, err = client.KdenApiClient().
			GetIdentityV1WithResponse(context.Background())

		Expect(err).NotTo(HaveOccurred())
		Expect(<-authorization).To(BeEmpty())
	})

	It("does not touch the cookie store in access-token mode", func() {
		store := &recordingCookieStore{
			loadErr:   errors.New("cookie store must not be loaded"),
			deleteErr: errors.New("cookie store must not be deleted"),
		}

		client, err := NewClient(
			"https://api.example.test/api",
			"access-token",
			store,
			time.Second,
			time.Second,
		)
		Expect(err).NotTo(HaveOccurred())

		Expect(client.Invalidate()).To(Succeed())
		Expect(store.deleteCalls).To(BeZero())
		Expect(store.saved).To(BeNil())
	})

	It("disables session operations in access-token mode", func() {
		client, err := NewClient(
			"https://api.example.test/api",
			"access-token",
			&recordingCookieStore{
				loadErr: errors.New("cookie store must not be loaded"),
			},
			time.Second,
			time.Second,
		)
		Expect(err).NotTo(HaveOccurred())

		Expect(client.Login(context.Background())).To(MatchError(
			"login is not available when access-token authentication is active",
		))
		Expect(client.Logout(context.Background())).To(MatchError(
			"logout is not available when access-token authentication is active",
		))

		valid, err := client.hasValidSession(context.Background())
		Expect(valid).To(BeFalse())
		Expect(err).To(MatchError(
			"session validation is not available when access-token authentication is active",
		))
	})
})
