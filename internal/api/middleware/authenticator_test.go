package middleware_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/konfidence-project/konfidence/internal/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testSessionStore struct {
	sessions   map[string]*session.Session
	getCalls   int
	deletedIDs []string
	err        error
}

type testAuthRepository struct {
	projectRoles auth.ProjectRoles
	groups       []string
	calls        int
	err          error

	tokenIdentity *auth.TokenIdentity
	rawToken      string
	tokenCalls    int
	tokenErr      error
}

func (r *testAuthRepository) GetProjectRoles(_ context.Context, groups []string) (auth.ProjectRoles, error) {
	r.calls++
	r.groups = append([]string(nil), groups...)
	return r.projectRoles, r.err
}

func (r *testAuthRepository) AuthenticateToken(_ context.Context, rawToken string) (*auth.TokenIdentity, error) {
	r.tokenCalls++
	r.rawToken = rawToken
	return r.tokenIdentity, r.tokenErr
}

func (s *testSessionStore) Get(_ context.Context, id string) (*session.Session, error) {
	s.getCalls++

	if s.err != nil {
		return nil, s.err
	}

	return s.sessions[id], nil
}

func (s *testSessionStore) Save(
	_ context.Context,
	storedSession *session.Session,
) (string, error) {
	if s.sessions == nil {
		s.sessions = make(map[string]*session.Session)
	}

	s.sessions[storedSession.ID] = storedSession
	return storedSession.ID, nil
}

func (s *testSessionStore) Delete(_ context.Context, id string) error {
	s.deletedIDs = append(s.deletedIDs, id)
	delete(s.sessions, id)
	return nil
}

func testAuthenticator(
	store *testSessionStore,
	authRepo *testAuthRepository,
	parsed config.Parsed,
	next http.Handler,
) http.Handler {
	GinkgoHelper()

	handler, err := middleware.Authenticator(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		store,
		authRepo,
		parsed,
		next,
	)
	Expect(err).NotTo(HaveOccurred())
	return handler
}

func sessionConfig() config.Parsed {
	return config.Parsed{
		OIDC: config.ParsedOIDCConfig{Enabled: true},
		Session: config.ParsedSessionConfig{
			Cookie: config.SessionCookieConfig{Name: "session"},
		},
	}
}

var _ = Describe("Session authentication", func() {
	Context("OpenAPI security", func() {
		var store *testSessionStore
		var authRepo *testAuthRepository
		var handler http.Handler

		BeforeEach(func() {
			store = &testSessionStore{sessions: map[string]*session.Session{
				"valid-session": {Context: session.Context{ID: "valid-session"}},
			}}
			authRepo = &testAuthRepository{}
			handler = testAuthenticator(store, authRepo, sessionConfig(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/identity" {
					storedSession, err := session.FromContext(r.Context())
					Expect(err).NotTo(HaveOccurred())
					Expect(storedSession.ID).To(Equal("valid-session"))
				}
				w.WriteHeader(http.StatusNoContent)
			}))
		})

		It("bypasses authentication for a public operation", func() {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/login?return_url=https%3A%2F%2Fdashboard.example.com", nil))

			Expect(response.Code).To(Equal(http.StatusNoContent))
			Expect(store.getCalls).To(BeZero())
			Expect(authRepo.calls).To(BeZero())
		})

		It("preserves the original validation status", func() {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/not-found", nil))
			Expect(response.Code).To(Equal(http.StatusNotFound))
		})

		It("rejects a protected operation without a session", func() {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil))
			Expect(response.Code).To(Equal(http.StatusUnauthorized))
		})

		It("accepts a protected operation with a valid session", func() {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
			request.AddCookie(&http.Cookie{Name: "session", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			Expect(response.Code).To(Equal(http.StatusNoContent))
		})

		It("rejects a differently named cookie", func() {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
			request.AddCookie(&http.Cookie{Name: "unknown-session-name", Value: "valid-session"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			Expect(response.Code).To(Equal(http.StatusUnauthorized))
		})

		It("rejects an unknown session", func() {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
			request.AddCookie(&http.Cookie{Name: "session", Value: "unknown"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			Expect(response.Code).To(Equal(http.StatusUnauthorized))
		})

		It("deletes a rejected session", func() {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
			request.AddCookie(&http.Cookie{Name: "session", Value: "unknown"})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			Expect(response.Code).To(Equal(http.StatusUnauthorized))
			Expect(store.deletedIDs).To(Equal([]string{"unknown"}))
		})
	})

	It("maps project roles onto the session", func() {
		store := &testSessionStore{sessions: map[string]*session.Session{
			"valid-session": {Groups: []string{"all-users", "platform-engineers"}},
		}}
		authRepo := &testAuthRepository{projectRoles: auth.ProjectRoles{
			"accessible": {"admin", "viewer"},
		}}
		handler := testAuthenticator(store, authRepo, sessionConfig(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			storedSession, err := session.FromContext(r.Context())
			Expect(err).NotTo(HaveOccurred())
			Expect(storedSession.ProjectRoles["accessible"]).To(Equal([]string{"admin", "viewer"}))
			Expect(storedSession.ProjectRoles).NotTo(HaveKey("hidden"))
			w.WriteHeader(http.StatusNoContent)
		}))
		request := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
		request.AddCookie(&http.Cookie{Name: "session", Value: "valid-session"})
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		Expect(response.Code).To(Equal(http.StatusNoContent))
		Expect(authRepo.calls).To(Equal(1))
		Expect(authRepo.groups).To(Equal([]string{"all-users", "platform-engineers"}))
		Expect(store.sessions["valid-session"].ProjectRoles["accessible"]).To(Equal([]string{"admin", "viewer"}))
	})

	DescribeTable("rejects session mapping failures",
		func(storeFailure, roleFailure bool) {
			store := &testSessionStore{sessions: map[string]*session.Session{"valid-session": {}}}
			authRepo := &testAuthRepository{}
			if storeFailure {
				store.err = errors.New("session store unavailable")
			}
			if roleFailure {
				authRepo.err = errors.New("project cache unavailable")
			}
			nextCalled := false
			handler := testAuthenticator(store, authRepo, sessionConfig(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				nextCalled = true
			}))
			request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
			request.AddCookie(&http.Cookie{Name: "session", Value: "valid-session"})
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			Expect(response.Code).To(Equal(http.StatusUnauthorized))
			Expect(nextCalled).To(BeFalse())
		},
		Entry("when session lookup fails", true, false),
		Entry("when role lookup fails", false, true),
	)

	DescribeTable("handles token expiry",
		func(scopes []string, expiryOffset time.Duration, zeroExpiry bool, expectedCode int, expectDeleted bool) {
			tokenExpiry := time.Now().Add(expiryOffset).Unix()
			if zeroExpiry {
				tokenExpiry = 0
			}
			store := &testSessionStore{sessions: map[string]*session.Session{
				"session-id": {Context: session.Context{ID: "session-id"}, TokenExpiry: tokenExpiry},
			}}
			authRepo := &testAuthRepository{}
			nextCalled := false
			parsed := sessionConfig()
			parsed.OIDC.Scopes = scopes
			handler := testAuthenticator(store, authRepo, parsed, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
			request.AddCookie(&http.Cookie{Name: "session", Value: "session-id"})
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			Expect(response.Code).To(Equal(expectedCode))
			if expectDeleted {
				Expect(store.deletedIDs).To(Equal([]string{"session-id"}))
				Expect(nextCalled).To(BeFalse())
				Expect(authRepo.calls).To(BeZero())
			} else {
				Expect(store.deletedIDs).To(BeEmpty())
				Expect(nextCalled).To(BeTrue())
			}
		},
		Entry("an expired token with offline access", []string{"offline_access"}, -time.Minute, false, http.StatusUnauthorized, true),
		Entry("a future token with offline access", []string{"offline_access"}, time.Hour, false, http.StatusNoContent, false),
		Entry("zero expiry with offline access", []string{"offline_access"}, time.Duration(0), true, http.StatusNoContent, false),
		Entry("an expired token without offline access", []string{"openid", "profile"}, -time.Minute, false, http.StatusNoContent, false),
	)
})

var _ = Describe("Bearer authentication", func() {
	It("takes precedence over a valid session cookie", func() {
		store := &testSessionStore{sessions: map[string]*session.Session{
			"valid-session": {Context: session.Context{ID: "valid-session"}},
		}}
		authRepo := &testAuthRepository{tokenIdentity: &auth.TokenIdentity{
			Subject:      "workload-subject",
			ProjectRoles: auth.ProjectRoles{"project-a": {"admin"}},
		}}
		handler := testAuthenticator(store, authRepo, sessionConfig(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, err := session.FromContext(r.Context())
			Expect(err).NotTo(HaveOccurred())
			Expect(identity.Subject).To(Equal("workload-subject"))
			Expect(identity.ProjectRoles["project-a"]).To(Equal([]string{"admin"}))
			w.WriteHeader(http.StatusNoContent)
		}))
		request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
		request.Header.Set("Authorization", "bEaReR workload-token")
		request.AddCookie(&http.Cookie{Name: "session", Value: "valid-session"})
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		Expect(response.Code).To(Equal(http.StatusNoContent))
		Expect(authRepo.tokenCalls).To(Equal(1))
		Expect(authRepo.rawToken).To(Equal("workload-token"))
		Expect(store.getCalls).To(BeZero())
		Expect(authRepo.calls).To(BeZero())
	})

	DescribeTable("rejects invalid credentials",
		func(authorization string, tokenErr error, expectTokenCall, expectChallenge bool) {
			store := &testSessionStore{sessions: map[string]*session.Session{
				"valid-session": {Context: session.Context{ID: "valid-session"}},
			}}
			authRepo := &testAuthRepository{
				tokenIdentity: &auth.TokenIdentity{Subject: "unexpected"},
				tokenErr:      tokenErr,
			}
			nextCalled := false
			handler := testAuthenticator(store, authRepo, sessionConfig(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				nextCalled = true
			}))
			request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
			request.Header.Set("Authorization", authorization)
			request.AddCookie(&http.Cookie{Name: "session", Value: "valid-session"})
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			Expect(response.Code).To(Equal(http.StatusUnauthorized))
			Expect(nextCalled).To(BeFalse())
			Expect(store.getCalls).To(BeZero())
			if expectTokenCall {
				Expect(authRepo.tokenCalls).To(BeNumerically(">", 0))
			} else {
				Expect(authRepo.tokenCalls).To(BeZero())
			}
			if expectChallenge {
				Expect(response.Header().Get("WWW-Authenticate")).To(Equal("Bearer"))
			} else {
				Expect(response.Header().Get("WWW-Authenticate")).To(BeEmpty())
			}
		},
		Entry("with a missing token", "Bearer", nil, false, true),
		Entry("with an empty token", "Bearer ", nil, false, true),
		Entry("with whitespace in the token", "Bearer first second", nil, false, true),
		Entry("with an unsupported scheme", "Basic credentials", nil, false, false),
		Entry("when the repository rejects the token", "Bearer rejected-token", auth.ErrInvalidBearerToken, true, true),
	)

	It("does not authenticate logout requests", func() {
		authRepo := &testAuthRepository{tokenIdentity: &auth.TokenIdentity{Subject: "workload-subject"}}
		nextCalled := false
		handler := testAuthenticator(
			&testSessionStore{sessions: make(map[string]*session.Session)},
			authRepo,
			sessionConfig(),
			http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true }),
		)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
		request.Header.Set("Authorization", "Bearer workload-token")
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		Expect(response.Code).To(Equal(http.StatusUnauthorized))
		Expect(nextCalled).To(BeFalse())
		Expect(authRepo.tokenCalls).To(BeZero())
	})

	It("rejects a nil identity", func() {
		authRepo := &testAuthRepository{}
		nextCalled := false
		handler := testAuthenticator(
			&testSessionStore{sessions: make(map[string]*session.Session)},
			authRepo,
			sessionConfig(),
			http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true }),
		)
		request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
		request.Header.Set("Authorization", "Bearer workload-token")
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		Expect(response.Code).To(Equal(http.StatusUnauthorized))
		Expect(nextCalled).To(BeFalse())
	})
})
