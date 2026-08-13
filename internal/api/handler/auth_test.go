package handler

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
)

type recordingStateStore struct {
	saved bool
}

func (s *recordingStateStore) Save(*oidc.StateData) error {
	s.saved = true
	return nil
}

func (*recordingStateStore) Get(string) (*oidc.StateData, error) { return nil, nil }
func (*recordingStateStore) Delete(*oidc.StateData) error        { return nil }

func TestAllowedReturnURL(t *testing.T) {
	allowReturnURLs := []string{
		"https://dashboard.example.com/callback",
		"http://localhost:3000/auth?source=kden",
	}
	tests := map[string]bool{
		"https://dashboard.example.com/callback":     true,
		"http://localhost:3000/auth?source=kden":     true,
		"https://dashboard.example.com/other":        false,
		"https://other.example.com/callback":         false,
		"http://localhost:3000/auth?source=attacker": false,
		"/projects": false,
		"":          false,
	}

	for returnURL, expected := range tests {
		t.Run(returnURL, func(t *testing.T) {
			if actual := allowedReturnURL(returnURL, allowReturnURLs); actual != expected {
				t.Fatalf("allowedReturnURL(%q) = %t, want %t", returnURL, actual, expected)
			}
		})
	}
}

func TestLoginRejectsUnlistedReturnURL(t *testing.T) {
	stateStore := &recordingStateStore{}
	handler := newAuthHandler(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		*oidc.NewOIDCClient(oidc.Config{}),
		stateStore,
		nil,
		config.Parsed{OIDC: config.ParsedOIDCConfig{AllowReturnURLs: []string{"https://dashboard.example.com/callback"}}},
	)

	response, err := handler.LoginV1(context.Background(), openapi.LoginV1RequestObject{
		Params: openapi.LoginV1Params{ReturnUrl: "https://attacker.example.com/callback"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(openapi.LoginV1400JSONResponse); !ok {
		t.Fatalf("expected bad request response, got %T", response)
	}
	if stateStore.saved {
		t.Fatal("OIDC state must not be saved for an unlisted return URL")
	}
}
