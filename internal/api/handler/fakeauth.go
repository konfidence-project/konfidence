package handler

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"golang.org/x/oauth2"
)

type fakeAuthHandler struct {
	auth          *authHandler
	exchangeStore oidc.ExchangeStore
	serverBaseURL string
}

type fakeAuthState struct {
	ReturnURL     string `json:"return_url"`
	CodeChallenge string `json:"code_challenge,omitempty"`
}

// LoginV1 redirects to the callback endpoint, encoding the return URL and optional
// PKCE code challenge into the state parameter.
func (f *fakeAuthHandler) LoginV1(_ context.Context, request openapi.LoginV1RequestObject) (openapi.LoginV1ResponseObject, error) {
	state := fakeAuthState{ReturnURL: request.Params.ReturnUrl}
	if request.Params.CodeChallenge != nil {
		state.CodeChallenge = *request.Params.CodeChallenge
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("state", string(stateJSON))
	callbackURL := f.serverBaseURL + "/api/v1/auth/callback?" + q.Encode()
	return openapi.LoginV1302Response{
		Headers: openapi.LoginV1302ResponseHeaders{
			Location: &callbackURL,
		},
	}, nil
}

// AuthCallbackV1 creates a static admin session. For CLI flows (identified by a
// code_challenge in the encoded state) it saves an exchange code and redirects to
// the CLI callback URL. For browser flows it sets a session cookie directly.
func (f *fakeAuthHandler) AuthCallbackV1(ctx context.Context, request openapi.AuthCallbackV1RequestObject) (openapi.AuthCallbackV1ResponseObject, error) {
	var state fakeAuthState
	if err := json.Unmarshal([]byte(request.Params.State), &state); err != nil || state.ReturnURL == "" {
		state.ReturnURL = "/"
	}

	name := staticAdminName
	email := staticAdminEmail
	givenName := staticAdminGivenName
	familyName := staticAdminFamilyName

	sess := &session.Session{
		Groups: staticAdminGroups,
	}
	sess.Name = &name
	sess.Email = &email
	sess.GivenName = &givenName
	sess.FamilyName = &familyName

	sessionID, err := f.auth.sessions.Save(ctx, sess)
	if err != nil {
		return nil, err
	}

	// CLI flow: a code_challenge was included in the state by LoginV1
	if state.CodeChallenge != "" {
		exchangeCode := oauth2.GenerateVerifier()
		if err := f.exchangeStore.Save(ctx, exchangeCode, oidc.Exchange{
			SessionID:     sessionID,
			CodeChallenge: state.CodeChallenge,
		}); err != nil {
			return nil, err
		}

		callbackURL, err := url.Parse(state.ReturnURL)
		if err != nil {
			return nil, err
		}
		q := callbackURL.Query()
		q.Set("code", exchangeCode)
		callbackURL.RawQuery = q.Encode()
		location := callbackURL.String()

		return openapi.AuthCallbackV1302Response{
			Headers: openapi.AuthCallbackV1302ResponseHeaders{
				Location: &location,
			},
		}, nil
	}

	cookieValue := f.auth.sessionCookie(sessionID).String()
	return openapi.AuthCallbackV1302Response{
		Headers: openapi.AuthCallbackV1302ResponseHeaders{
			Location:  &state.ReturnURL,
			SetCookie: &cookieValue,
		},
	}, nil
}

func (f *fakeAuthHandler) LogoutV1(ctx context.Context, req openapi.LogoutV1RequestObject) (openapi.LogoutV1ResponseObject, error) {
	return f.auth.LogoutV1(ctx, req)
}

func (f *fakeAuthHandler) GetIdentityV1(ctx context.Context, req openapi.GetIdentityV1RequestObject) (openapi.GetIdentityV1ResponseObject, error) {
	return f.auth.GetIdentityV1(ctx, req)
}

func (f *fakeAuthHandler) PostExchangeCodeV1(
	ctx context.Context, req openapi.PostExchangeCodeV1RequestObject,
) (openapi.PostExchangeCodeV1ResponseObject, error) {
	return f.auth.PostExchangeCodeV1(ctx, req)
}
