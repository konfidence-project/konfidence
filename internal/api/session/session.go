package session

import (
	"context"
	"fmt"
)

type contextKey string

const contextSession contextKey = "session"

// Session represents a client session for an authenticated user.
type Session struct {
	ID string `json:"-"`
	// TODO add/refine session fields
	Subject           string   `json:"subject"`
	Name              *string  `json:"name,omitempty"`
	Email             *string  `json:"email,omitempty"`
	GivenName         *string  `json:"given_name,omitempty"`
	FamilyName        *string  `json:"family_name,omitempty"`
	PreferredUsername *string  `json:"preferred_username,omitempty"`
	Roles             []string `json:"roles,omitempty"`
	Groups            []string `json:"groups,omitempty"`
	AccessToken       string   `json:"access_token"`
	RefreshToken      *string  `json:"refresh_token"`
	Expiry            int64    `json:"expiry"`
}

func NewContext(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, contextSession, session)
}

func FromContext(ctx context.Context) (*Session, error) {
	session, ok := ctx.Value(contextSession).(*Session)
	if !ok || session == nil {
		return nil, fmt.Errorf("session not found in context")
	}

	return session, nil
}
