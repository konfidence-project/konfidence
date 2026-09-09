package middleware_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/konfidence-project/konfidence/internal/auth"
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

func TestSessionAuthenticationFollowsOpenAPISecurity(t *testing.T) {
	store := &testSessionStore{sessions: map[string]*session.Session{
		"valid-session": {Context: session.Context{ID: "valid-session"}},
	}}
	authRepo := &testAuthRepository{}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/identity" {
			storedSession, err := session.FromContext(r.Context())
			if err != nil || storedSession.ID != "valid-session" {
				t.Errorf("unexpected session context: session=%+v err=%v", storedSession, err)
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})

	h, err := middleware.Authenticator(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		store,
		authRepo,
		config.Parsed{Session: config.ParsedSessionConfig{Cookie: config.SessionCookieConfig{Name: "session"}}},
		next,
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("public operation bypasses authentication", func(t *testing.T) {
		before := store.getCalls
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/login?return_url=https%3A%2F%2Fdashboard.example.com", nil))

		if response.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
		}
		if store.getCalls != before {
			t.Fatalf("expected no session lookup, got %d", store.getCalls-before)
		}
		if authRepo.calls != 0 {
			t.Fatalf("expected no role lookup, got %d", authRepo.calls)
		}
	})

	t.Run("operations return original validation status", func(t *testing.T) {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/not-found", nil))

		if response.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
		}
	})

	t.Run("protected operation rejects missing session", func(t *testing.T) {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
		}
	})

	t.Run("protected operation accepts valid session", func(t *testing.T) {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
		request.AddCookie(&http.Cookie{Name: "session", Value: "valid-session"})
		h.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
		}
	})

	t.Run("protected operation rejects a differently named cookie", func(t *testing.T) {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
		request.AddCookie(&http.Cookie{Name: "unknown-session-name", Value: "valid-session"})
		h.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
		}
	})

	t.Run("protected operation rejects unknown session", func(t *testing.T) {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
		request.AddCookie(&http.Cookie{Name: "session", Value: "unknown"})
		h.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
		}
	})

	t.Run("protected operation deletes rejected session", func(t *testing.T) {
		deletionsBefore := len(store.deletedIDs)

		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
		request.AddCookie(&http.Cookie{Name: "session", Value: "unknown"})
		h.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Fatalf(
				"expected status %d, got %d",
				http.StatusUnauthorized,
				response.Code,
			)
		}

		if len(store.deletedIDs) != deletionsBefore+1 {
			t.Fatalf("expected one session deletion, got %d", len(store.deletedIDs)-deletionsBefore)
		}

		if deletedID := store.deletedIDs[len(store.deletedIDs)-1]; deletedID != "unknown" {
			t.Fatalf("expected session %q to be deleted, got %q", "unknown", deletedID)
		}
	})
}

func TestSessionAuthenticationMapsProjectRoles(t *testing.T) {
	store := &testSessionStore{sessions: map[string]*session.Session{
		"valid-session": {Groups: []string{"all-users", "platform-engineers"}},
	}}
	const adminRole = "admin"
	authRepo := &testAuthRepository{projectRoles: auth.ProjectRoles{
		"accessible": {adminRole, "viewer"},
	}}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		storedSession, err := session.FromContext(r.Context())
		if err != nil {
			t.Errorf("expected mapped session context: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if roles := storedSession.ProjectRoles["accessible"]; len(roles) != 2 || roles[0] != adminRole || roles[1] != "viewer" {
			t.Errorf("unexpected roles: %v", roles)
		}
		if _, ok := storedSession.ProjectRoles["hidden"]; ok {
			t.Error("unexpected access to hidden project")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h, err := middleware.Authenticator(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		store,
		authRepo,
		config.Parsed{Session: config.ParsedSessionConfig{Cookie: config.SessionCookieConfig{Name: "session"}}},
		next,
	)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "valid-session"})
	response := httptest.NewRecorder()

	h.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if authRepo.calls != 1 {
		t.Fatalf("expected one role lookup, got %d", authRepo.calls)
	}
	if len(authRepo.groups) != 2 || authRepo.groups[0] != "all-users" || authRepo.groups[1] != "platform-engineers" {
		t.Fatalf("unexpected groups passed to auth repository: %v", authRepo.groups)
	}
	roles := store.sessions["valid-session"].ProjectRoles["accessible"]
	if len(roles) != 2 || roles[0] != adminRole || roles[1] != "viewer" {
		t.Fatalf("expected stored session to be mapped, got %v", roles)
	}
}

func TestSessionAuthenticationRejectsSessionMappingFailures(t *testing.T) {
	tests := map[string]struct {
		store    *testSessionStore
		authRepo *testAuthRepository
	}{
		"session lookup failure": {
			store:    &testSessionStore{err: errors.New("session store unavailable")},
			authRepo: &testAuthRepository{},
		},
		"role lookup failure": {
			store: &testSessionStore{sessions: map[string]*session.Session{
				"valid-session": {},
			}},
			authRepo: &testAuthRepository{err: errors.New("project cache unavailable")},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			nextCalled := false
			h, err := middleware.Authenticator(
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				test.store,
				test.authRepo,
				config.Parsed{Session: config.ParsedSessionConfig{Cookie: config.SessionCookieConfig{Name: "session"}}},
				http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true }),
			)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
			request.AddCookie(&http.Cookie{Name: "session", Value: "valid-session"})
			response := httptest.NewRecorder()

			h.ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
			}
			if nextCalled {
				t.Fatal("expected request not to reach next handler")
			}
		})
	}
}

func TestSessionAuthenticationTokenExpiry(t *testing.T) {
	tests := map[string]struct {
		scopes        []string
		tokenExpiry   int64
		expectedCode  int
		expectDeleted bool
	}{
		"expired token with offline access": {
			scopes:        []string{"offline_access"},
			tokenExpiry:   time.Now().Add(-time.Minute).Unix(),
			expectedCode:  http.StatusUnauthorized,
			expectDeleted: true,
		},
		"future token with offline access": {
			scopes:       []string{"offline_access"},
			tokenExpiry:  time.Now().Add(time.Hour).Unix(),
			expectedCode: http.StatusNoContent,
		},
		"zero expiry with offline access": {
			scopes:       []string{"offline_access"},
			tokenExpiry:  0,
			expectedCode: http.StatusNoContent,
		},
		"expired token without offline access": {
			scopes:       []string{"openid", "profile"},
			tokenExpiry:  time.Now().Add(-time.Minute).Unix(),
			expectedCode: http.StatusNoContent,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store := &testSessionStore{
				sessions: map[string]*session.Session{
					"session-id": {
						Context: session.Context{
							ID: "session-id",
						},
						TokenExpiry: test.tokenExpiry,
					},
				},
			}
			authRepo := &testAuthRepository{}
			nextCalled := false

			handler, err := middleware.Authenticator(
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				store,
				authRepo,
				config.Parsed{
					OIDC: config.ParsedOIDCConfig{
						Scopes: test.scopes,
					},
					Session: config.ParsedSessionConfig{
						Cookie: config.SessionCookieConfig{
							Name: "session",
						},
					},
				},
				http.HandlerFunc(func(
					w http.ResponseWriter,
					_ *http.Request,
				) {
					nextCalled = true
					w.WriteHeader(http.StatusNoContent)
				}),
			)
			if err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/identity",
				nil,
			)
			request.AddCookie(&http.Cookie{
				Name:  "session",
				Value: "session-id",
			})

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != test.expectedCode {
				t.Fatalf(
					"expected status %d, got %d",
					test.expectedCode,
					response.Code,
				)
			}

			deleted := len(store.deletedIDs) > 0
			if deleted != test.expectDeleted {
				t.Fatalf(
					"expected deleted=%t, got %t",
					test.expectDeleted,
					deleted,
				)
			}

			if test.expectDeleted {
				if deletedID := store.deletedIDs[0]; deletedID != "session-id" {
					t.Fatalf(
						"expected session %q to be deleted, got %q",
						"session-id",
						deletedID,
					)
				}
				if nextCalled {
					t.Fatal("expected expired session not to reach next handler")
				}
				if authRepo.calls != 0 {
					t.Fatalf(
						"expected no role lookup, got %d calls",
						authRepo.calls,
					)
				}
			} else if !nextCalled {
				t.Fatal("expected valid session to reach next handler")
			}
		})
	}
}

func TestBearerAuthentication(t *testing.T) {
	store := &testSessionStore{
		sessions: map[string]*session.Session{
			"valid-session": {
				Context: session.Context{
					ID: "valid-session",
				},
			},
		},
	}
	authRepo := &testAuthRepository{
		tokenIdentity: &auth.TokenIdentity{
			Subject: "workload-subject",
			ProjectRoles: auth.ProjectRoles{
				"project-a": {"admin"},
			},
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, err := session.FromContext(r.Context())
		if err != nil {
			t.Errorf("expected bearer identity in context: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if identity.Subject != "workload-subject" {
			t.Errorf("unexpected subject: %q", identity.Subject)
		}
		if roles := identity.ProjectRoles["project-a"]; len(roles) != 1 ||
			roles[0] != "admin" {
			t.Errorf("unexpected project roles: %v", identity.ProjectRoles)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	handler, err := middleware.Authenticator(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		store,
		authRepo,
		config.Parsed{
			Session: config.ParsedSessionConfig{
				Cookie: config.SessionCookieConfig{
					Name: "session",
				},
			},
		},
		next,
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/identity",
		nil,
	)
	request.Header.Set("Authorization", "bEaReR workload-token")

	// Bearer authentication must take precedence over a valid cookie.
	request.AddCookie(&http.Cookie{
		Name:  "session",
		Value: "valid-session",
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			response.Code,
		)
	}
	if authRepo.tokenCalls != 1 {
		t.Fatalf(
			"expected one token authentication, got %d",
			authRepo.tokenCalls,
		)
	}
	if authRepo.rawToken != "workload-token" {
		t.Fatalf("unexpected token passed to repository")
	}
	if store.getCalls != 0 {
		t.Fatalf(
			"expected no session lookup, got %d",
			store.getCalls,
		)
	}
	if authRepo.calls != 0 {
		t.Fatalf(
			"expected no session role lookup, got %d",
			authRepo.calls,
		)
	}
}

func TestBearerAuthenticationRejectsInvalidCredentials(t *testing.T) {
	tests := map[string]struct {
		authorization   string
		tokenErr        error
		expectTokenCall bool
		expectChallenge bool
	}{
		"missing token": {
			authorization:   "Bearer",
			expectChallenge: true,
		},
		"empty token": {
			authorization:   "Bearer ",
			expectChallenge: true,
		},
		"token containing whitespace": {
			authorization:   "Bearer first second",
			expectChallenge: true,
		},
		"unsupported scheme": {
			authorization: "Basic credentials",
		},
		"repository rejection": {
			authorization:   "Bearer rejected-token",
			tokenErr:        auth.ErrInvalidBearerToken,
			expectTokenCall: true,
			expectChallenge: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store := &testSessionStore{
				sessions: map[string]*session.Session{
					"valid-session": {
						Context: session.Context{
							ID: "valid-session",
						},
					},
				},
			}
			authRepo := &testAuthRepository{
				tokenIdentity: &auth.TokenIdentity{
					Subject: "unexpected",
				},
				tokenErr: test.tokenErr,
			}
			nextCalled := false

			handler, err := middleware.Authenticator(
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				store,
				authRepo,
				config.Parsed{
					Session: config.ParsedSessionConfig{
						Cookie: config.SessionCookieConfig{
							Name: "session",
						},
					},
				},
				http.HandlerFunc(func(
					http.ResponseWriter,
					*http.Request,
				) {
					nextCalled = true
				}),
			)
			if err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/identity",
				nil,
			)
			request.Header.Set(
				"Authorization",
				test.authorization,
			)
			request.AddCookie(&http.Cookie{
				Name:  "session",
				Value: "valid-session",
			})

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusUnauthorized,
					response.Code,
				)
			}
			if nextCalled {
				t.Fatal("unexpected call to next handler")
			}
			if store.getCalls != 0 {
				t.Fatalf(
					"unexpected cookie fallback: %d lookups",
					store.getCalls,
				)
			}

			gotTokenCall := authRepo.tokenCalls > 0
			if gotTokenCall != test.expectTokenCall {
				t.Fatalf(
					"expected token call=%t, got %t",
					test.expectTokenCall,
					gotTokenCall,
				)
			}

			challenge := response.Header().
				Get("WWW-Authenticate")
			if test.expectChallenge && challenge != "Bearer" {
				t.Fatalf(
					"expected Bearer challenge, got %q",
					challenge,
				)
			}
			if !test.expectChallenge && challenge != "" {
				t.Fatalf(
					"expected no challenge, got %q",
					challenge,
				)
			}
		})
	}
}

func TestLogoutDoesNotAcceptBearerAuthentication(t *testing.T) {
	store := &testSessionStore{
		sessions: make(map[string]*session.Session),
	}
	authRepo := &testAuthRepository{
		tokenIdentity: &auth.TokenIdentity{
			Subject: "workload-subject",
		},
	}
	nextCalled := false

	handler, err := middleware.Authenticator(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		store,
		authRepo,
		config.Parsed{
			Session: config.ParsedSessionConfig{
				Cookie: config.SessionCookieConfig{
					Name: "session",
				},
			},
		},
		http.HandlerFunc(func(
			http.ResponseWriter,
			*http.Request,
		) {
			nextCalled = true
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/logout",
		nil,
	)
	request.Header.Set(
		"Authorization",
		"Bearer workload-token",
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			response.Code,
		)
	}
	if nextCalled {
		t.Fatal("bearer-authenticated logout reached next handler")
	}
	if authRepo.tokenCalls != 0 {
		t.Fatalf(
			"logout attempted bearer authentication %d times",
			authRepo.tokenCalls,
		)
	}
}

func TestBearerAuthenticationRejectsNilIdentity(t *testing.T) {
	authRepo := &testAuthRepository{}
	nextCalled := false

	handler, err := middleware.Authenticator(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		&testSessionStore{
			sessions: make(map[string]*session.Session),
		},
		authRepo,
		config.Parsed{
			Session: config.ParsedSessionConfig{
				Cookie: config.SessionCookieConfig{
					Name: "session",
				},
			},
		},
		http.HandlerFunc(func(
			http.ResponseWriter,
			*http.Request,
		) {
			nextCalled = true
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/identity",
		nil,
	)
	request.Header.Set("Authorization", "Bearer workload-token")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
	if nextCalled {
		t.Fatal("request with nil identity reached next handler")
	}
}
