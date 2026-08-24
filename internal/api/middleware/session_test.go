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
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/konfidence-project/konfidence/internal/auth"
	"golang.org/x/oauth2"
)

type testSessionStore struct {
	sessions    map[string]*session.Session
	getResults  []*session.Session
	getCalls    int
	updateCalls int
	deletedIDs  []string
	err         error
}

type testAuthRepository struct {
	projectRoles auth.ProjectRoles
	groups       []string
	calls        int
	err          error
}

type testOIDCRefresher struct {
	result *oidc.RefreshResult
	err    error
	calls  int
	token  *oauth2.Token
}

func (r *testOIDCRefresher) Refresh(
	_ context.Context,
	token *oauth2.Token,
) (*oidc.RefreshResult, error) {
	r.calls++
	r.token = token
	return r.result, r.err
}

func (r *testAuthRepository) GetProjectRoles(_ context.Context, groups []string) (auth.ProjectRoles, error) {
	r.calls++
	r.groups = append([]string(nil), groups...)
	return r.projectRoles, r.err
}

func (s *testSessionStore) Get(_ context.Context, id string) (*session.Session, error) {
	s.getCalls++

	if len(s.getResults) > 0 {
		result := s.getResults[0]
		s.getResults = s.getResults[1:]
		return result, s.err
	}

	return s.sessions[id], s.err
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

func (s *testSessionStore) Update(
	_ context.Context,
	storedSession *session.Session,
) error {
	s.updateCalls++

	if s.sessions == nil {
		s.sessions = make(map[string]*session.Session)
	}

	s.sessions[storedSession.ID] = storedSession
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

	refresher := &testOIDCRefresher{}
	h, err := middleware.SessionAuthentication(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		store,
		refresher,
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
	authRepo := &testAuthRepository{projectRoles: auth.ProjectRoles{
		"accessible": {"admin", "viewer"},
	}}
	refresher := &testOIDCRefresher{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		storedSession, err := session.FromContext(r.Context())
		if err != nil {
			t.Errorf("expected mapped session context: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if roles := storedSession.ProjectRoles["accessible"]; len(roles) != 2 || roles[0] != "admin" || roles[1] != "viewer" {
			t.Errorf("unexpected roles: %v", roles)
		}
		if _, ok := storedSession.ProjectRoles["hidden"]; ok {
			t.Error("unexpected access to hidden project")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h, err := middleware.SessionAuthentication(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		store,
		refresher,
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
	if len(roles) != 2 || roles[0] != "admin" || roles[1] != "viewer" {
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

	refresher := &testOIDCRefresher{}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			nextCalled := false
			h, err := middleware.SessionAuthentication(
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				test.store,
				refresher,
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

func TestSessionAuthenticationStoresZeroRefreshExpiry(t *testing.T) {
	refreshToken := "old-refresh-token"
	store := &testSessionStore{sessions: map[string]*session.Session{
		"expired-session": {
			Context:      session.Context{ID: "expired-session"},
			Subject:      "subject",
			AccessToken:  "old-access-token",
			RefreshToken: &refreshToken,
			TokenExpiry:  time.Now().Add(-time.Minute).Unix(),
		},
	}}
	refresher := &testOIDCRefresher{
		result: &oidc.RefreshResult{
			Token: &oauth2.Token{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				Expiry:       time.Time{},
			},
			Subject: "subject",
		},
	}

	h, err := middleware.SessionAuthentication(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		store,
		refresher,
		&testAuthRepository{},
		config.Parsed{
			OIDC: config.ParsedOIDCConfig{
				Scopes: []string{"offline_access"},
			},
			Session: config.ParsedSessionConfig{
				Cookie: config.SessionCookieConfig{Name: "session"},
			},
		},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
	request.AddCookie(&http.Cookie{
		Name:  "session",
		Value: "expired-session",
	})
	response := httptest.NewRecorder()

	h.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if refresher.calls != 1 {
		t.Fatalf("expected one token refresh, got %d", refresher.calls)
	}
	if expiry := store.sessions["expired-session"].TokenExpiry; expiry != 0 {
		t.Fatalf("expected stored expiry 0, got %d", expiry)
	}

	if store.updateCalls != 1 {
		t.Fatalf("expected one session update, got %d", store.updateCalls)
	}

	updated := store.sessions["expired-session"]
	if updated.AccessToken != "new-access-token" {
		t.Fatalf("expected refreshed access token, got %q", updated.AccessToken)
	}
	if updated.RefreshToken == nil || *updated.RefreshToken != "new-refresh-token" {
		t.Fatalf("expected refreshed refresh token, got %v", updated.RefreshToken)
	}
	if updated.TokenExpiry != 0 {
		t.Fatalf("expected stored expiry 0, got %d", updated.TokenExpiry)
	}
}

func TestSessionAuthenticationSkipsRefreshWhenSessionWasAlreadyRefreshed(t *testing.T) {
	tests := map[string]int64{
		"zero expiry":   0,
		"future expiry": time.Now().Add(time.Hour).Unix(),
	}

	for name, refreshedExpiry := range tests {
		t.Run(name, func(t *testing.T) {
			expiredSession := &session.Session{
				Context:     session.Context{ID: "session-id"},
				Subject:     "subject",
				Groups:      []string{"old-group"},
				TokenExpiry: time.Now().Add(-time.Minute).Unix(),
			}
			refreshedSession := &session.Session{
				Context:     session.Context{ID: "session-id"},
				Subject:     "subject",
				Groups:      []string{"new-group"},
				TokenExpiry: refreshedExpiry,
			}

			store := &testSessionStore{
				getResults: []*session.Session{
					expiredSession,
					refreshedSession,
				},
			}
			refresher := &testOIDCRefresher{}
			authRepo := &testAuthRepository{}

			h, err := middleware.SessionAuthentication(
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				store,
				refresher,
				authRepo,
				config.Parsed{
					OIDC: config.ParsedOIDCConfig{
						Scopes: []string{"offline_access"},
					},
					Session: config.ParsedSessionConfig{
						Cookie: config.SessionCookieConfig{Name: "session"},
					},
				},
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

			h.ServeHTTP(response, request)

			if response.Code != http.StatusNoContent {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusNoContent,
					response.Code,
				)
			}
			if store.getCalls != 2 {
				t.Fatalf(
					"expected initial and pre-refresh session reads, got %d",
					store.getCalls,
				)
			}
			if refresher.calls != 0 {
				t.Fatalf(
					"expected token refresh to be skipped, got %d calls",
					refresher.calls,
				)
			}
			if len(authRepo.groups) != 1 ||
				authRepo.groups[0] != "new-group" {
				t.Fatalf(
					"expected already-refreshed session to be used, got groups %v",
					authRepo.groups,
				)
			}
		})
	}
}
