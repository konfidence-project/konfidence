package auth

import (
	"errors"
	"net/http"

	kdenapi "github.com/konfidence-project/konfidence/internal/kden/apiclient"
)

var (
	ErrUnauthorized        = errors.New("API request was unauthorized")
	ErrAccessTokenRejected = errors.New("API access token was rejected")
)

type unauthorizedDoer struct {
	doer            kdenapi.HttpRequestDoer
	usesAccessToken bool
}

func (d unauthorizedDoer) Do(request *http.Request) (*http.Response, error) {
	response, err := d.doer.Do(request)
	if err != nil {
		return response, err
	}

	if response.StatusCode != http.StatusUnauthorized {
		return response, nil
	}

	_ = response.Body.Close()

	if d.usesAccessToken {
		return nil, ErrAccessTokenRejected
	}

	return nil, ErrUnauthorized
}
