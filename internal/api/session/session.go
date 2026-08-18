package session

import (
	"context"
	"fmt"
)

type contextKey string

const contextSession contextKey = "session"

// Session represents a client session for an authenticated user.
type Session struct {
	Context
	Subject      string  `json:"subject"`
	AccessToken  string  `json:"access_token"`
	RefreshToken *string `json:"refresh_token"`
	Expiry       int64   `json:"expiry"`
}

// Context represents the session subset stored in the context.
type Context struct {
	ID                string   `json:"-"`
	Name              *string  `json:"name,omitempty"`
	Email             *string  `json:"email,omitempty"`
	GivenName         *string  `json:"given_name,omitempty"`
	FamilyName        *string  `json:"family_name,omitempty"`
	PreferredUsername *string  `json:"preferred_username,omitempty"`
	Roles             []string `json:"roles,omitempty"`
	Groups            []string `json:"groups,omitempty"`
}

func NewContext(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, contextSession, session.Context)
}

func FromContext(ctx context.Context) (*Context, error) {
	sess, ok := ctx.Value(contextSession).(Context)
	if !ok {
		return nil, fmt.Errorf("session not found in context")
	}

	return &sess, nil
}
