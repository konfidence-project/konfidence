package session

import (
	"context"
	"fmt"
)

type contextKey string

const (
	Id        string     = "session_id"
	ContextId contextKey = contextKey(Id)
)

// Session represents a client session for an authenticated user.
type Session struct {
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

func GetSessionIdFromContext(ctx context.Context) (string, error) {
	sessionId, ok := ctx.Value(ContextId).(string)
	if !ok {
		return "", fmt.Errorf("sessionId not found in context")
	}

	return sessionId, nil
}
