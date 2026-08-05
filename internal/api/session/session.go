package session

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
)

type contextKey string

const (
	Id        string     = "session_id"
	ContextId contextKey = contextKey(Id)
)

// Session represents a client session for an authenticated user.
type Session struct {
	// TODO add/refine session fields
	Email             string       `json:"email,omitempty"`
	Name              string       `json:"name"`
	GivenName         string       `json:"given_name,omitempty"`
	MiddleName        string       `json:"middle_name,omitempty"`
	FamilyName        string       `json:"family_name,omitempty"`
	PreferredUsername string       `json:"preferred_username,omitempty"`
	Roles             []string     `json:"roles,omitempty"`
	Groups            []string     `json:"groups,omitempty"`
	IdToken           oidc.IDToken `json:"id_token"`
	AccessToken       string       `json:"access_token"`
	RefreshToken      string       `json:"refresh_token,omitempty"`
	Expiry            int64        `json:"expiry"`
}

func GetSessionIdFromContext(ctx context.Context) *string {
	sessionId, ok := ctx.Value(ContextId).(string)
	if !ok {
		return nil
	}

	return &sessionId
}
