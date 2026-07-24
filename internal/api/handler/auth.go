package handler // nolint

import (
	"context"

	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AuthHandler struct{ k8s func() (client.Client, error) }

func (h *AuthHandler) Login(_ context.Context, _ openapi.LoginRequestObject) (openapi.LoginResponseObject, error) {
	return nil, nil
}

func (h *AuthHandler) Logout(_ context.Context, _ openapi.LogoutRequestObject) (openapi.LogoutResponseObject, error) {
	return nil, nil
}

func (h *AuthHandler) AuthCallback(_ context.Context, _ openapi.AuthCallbackRequestObject) (openapi.AuthCallbackResponseObject, error) {
	return nil, nil
}

func (h *AuthHandler) GetIdentity(_ context.Context, _ openapi.GetIdentityRequestObject) (openapi.GetIdentityResponseObject, error) {
	return nil, nil
}

func (h *AuthHandler) PostExchangeCode(_ context.Context, _ openapi.PostExchangeCodeRequestObject) (openapi.PostExchangeCodeResponseObject, error) {
	return nil, nil
}
