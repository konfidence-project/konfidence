package session

import (
	"context"
	"fmt"
	"time"

	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/auth"
)

type contextKey string

const contextSession contextKey = "session"

// Session represents a client session for an authenticated user.
type Session struct {
	Context
	Groups       []string `json:"groups,omitempty"`
	AccessToken  string   `json:"access_token"`
	RefreshToken *string  `json:"refresh_token"`
	TokenExpiry  int64    `json:"token_expiry"`
}

// Context represents the session subset stored in the context.
type Context struct {
	ID                string            `json:"-"`
	Subject           string            `json:"subject"`
	Name              *string           `json:"name,omitempty"`
	Email             *string           `json:"email,omitempty"`
	GivenName         *string           `json:"given_name,omitempty"`
	FamilyName        *string           `json:"family_name,omitempty"`
	PreferredUsername *string           `json:"preferred_username,omitempty"`
	ProjectRoles      auth.ProjectRoles `json:"projectroles,omitempty"`
}

func NewContext(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, contextSession, session.Context)
}

func NewRequestContext(ctx context.Context, identity Context) context.Context {
	return context.WithValue(ctx, contextSession, identity)
}

func FromContext(ctx context.Context) (*Context, error) {
	sess, ok := ctx.Value(contextSession).(Context)
	if !ok {
		return nil, fmt.Errorf("session not found in context")
	}

	return &sess, nil
}

func UnixExpiry(expiry time.Time) int64 {
	if expiry.IsZero() {
		return 0
	}
	return expiry.Unix()
}

func (s *Session) IsTokenExpiryZero() bool {
	return s.TokenExpiry == 0
}

func (s *Session) ApplyOIDCValues(subject string, claims oidc.IDTokenAdditionalClaims,
	accessToken string, refreshToken *string, tokenExpiry time.Time) {
	s.Subject = subject
	s.Groups = append([]string(nil), claims.Groups...)
	s.Name = claims.Name
	s.Email = claims.Email
	s.GivenName = claims.GivenName
	s.FamilyName = claims.FamilyName
	s.PreferredUsername = claims.PreferredUsername
	s.AccessToken = accessToken
	s.RefreshToken = refreshToken
	s.TokenExpiry = UnixExpiry(tokenExpiry)
}

func (c *Context) IsAuthenticatedForProject(projectId string) bool {
	return len(c.ProjectRoles[projectId]) > 0
}
