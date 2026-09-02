package handler

import (
	"context"
	"net/http"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
)

type fakeAuthHandler struct {
	sessions session.Writer
	config   config.Parsed
}

func (f *fakeAuthHandler) LoginV1(ctx context.Context, request openapi.LoginV1RequestObject) (openapi.LoginV1ResponseObject, error) {
	name := staticAdminName
	email := staticAdminEmail
	givenName := staticAdminGivenName
	familyName := staticAdminFamilyName

	sess := &session.Session{}
	sess.Name = &name
	sess.Email = &email
	sess.GivenName = &givenName
	sess.FamilyName = &familyName
	sess.Groups = staticAdminGroups

	sessionID, err := f.sessions.Save(ctx, sess)
	if err != nil {
		return nil, err
	}

	returnURL := request.Params.ReturnUrl
	return &loginV1ResponseWithCookie{
		cookie:    f.sessionCookie(sessionID),
		returnURL: returnURL,
	}, nil
}

type loginV1ResponseWithCookie struct {
	cookie    *http.Cookie
	returnURL string
}

func (r *loginV1ResponseWithCookie) VisitLoginV1Response(w http.ResponseWriter) error {
	http.SetCookie(w, r.cookie)
	w.Header().Set("Location", r.returnURL)
	w.WriteHeader(http.StatusFound)
	return nil
}

func (f *fakeAuthHandler) AuthCallbackV1(_ context.Context, _ openapi.AuthCallbackV1RequestObject) (openapi.AuthCallbackV1ResponseObject, error) {
	return openapi.AuthCallbackV1400JSONResponse{}, nil
}

func (f *fakeAuthHandler) LogoutV1(ctx context.Context, _ openapi.LogoutV1RequestObject) (openapi.LogoutV1ResponseObject, error) {
	storedSession, err := session.FromContext(ctx)
	if err == nil {
		_ = f.sessions.Delete(ctx, storedSession.ID)
	}

	cookie := &http.Cookie{
		Name:     f.config.Session.Cookie.Name,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: f.config.Session.Cookie.HTTPOnly,
		Secure:   f.config.Session.Cookie.Secure,
		SameSite: sameSiteMode(f.config.Session.Cookie.SameSite),
		Path:     "/",
	}
	cookieStr := cookie.String()
	return openapi.LogoutV1200Response{
		Headers: openapi.LogoutV1200ResponseHeaders{
			SetCookie: &cookieStr,
		},
	}, nil
}

func (f *fakeAuthHandler) sessionCookie(sessionID string) *http.Cookie {
	return (&authHandler{config: f.config}).sessionCookie(sessionID)
}
