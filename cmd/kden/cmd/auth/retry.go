package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type Authenticator interface {
	Invalidate() error
	Login(context.Context) error
	UsesAccessToken() bool
}

type statusResponse interface {
	StatusCode() int
}

func RequestWithAuthRetry[T statusResponse](
	ctx context.Context,
	authenticator Authenticator,
	request func(context.Context) (T, error),
) (T, error) {
	response, err := request(ctx)
	if err != nil {
		return response, err
	}

	if response.StatusCode() != http.StatusUnauthorized {
		return response, nil
	}

	if authenticator.UsesAccessToken() {
		return response, errors.New("authenticating with access token failed. token was rejected")
	}

	if err := authenticator.Invalidate(); err != nil {
		return response, fmt.Errorf("removing invalid session: %w", err)
	}

	if err := authenticator.Login(ctx); err != nil {
		return response, fmt.Errorf(
			"authenticating with Konfidence API failed: %w",
			err,
		)
	}

	response, err = request(ctx)
	if err != nil {
		return response, fmt.Errorf("retrying request: %w", err)
	}

	return response, nil
}
