package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/session"
)

type testSessionStore struct {
	sessions map[string]*session.Session
	getCalls int
}

func (s *testSessionStore) Save(context.Context, *session.Session) (string, error) { return "", nil }
func (s *testSessionStore) Delete(context.Context, string) error                   { return nil }
func (s *testSessionStore) Get(_ context.Context, id string) (*session.Session, error) {
	s.getCalls++
	return s.sessions[id], nil
}

func TestSessionAuthenticationFollowsOpenAPISecurity(t *testing.T) {
	store := &testSessionStore{sessions: map[string]*session.Session{
		"valid-session": {ID: "valid-session"},
	}}
	sessions := &sessionMiddleware{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		store:  store,
		config: config.Parsed{SessionCookieName: "session"},
	}
	middleware, err := newSessionAuthMiddleware(sessions)
	if err != nil {
		t.Fatal(err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/identity" {
			storedSession, err := session.FromContext(r.Context())
			if err != nil || storedSession.ID != "valid-session" {
				t.Errorf("unexpected session context: session=%+v err=%v", storedSession, err)
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h := middleware(next)

	t.Run("public operation bypasses authentication", func(t *testing.T) {
		before := store.getCalls
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/login", nil))

		if response.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
		}
		if store.getCalls != before {
			t.Fatalf("expected no session lookup, got %d", store.getCalls-before)
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

	t.Run("protected operation rejects unknown session", func(t *testing.T) {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/identity", nil)
		request.AddCookie(&http.Cookie{Name: "session", Value: "unknown"})
		h.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
		}
	})
}
