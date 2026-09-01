package auth_test

import (
	"context"
	"errors"
	"net/http"

	authcmd "github.com/konfidence-project/konfidence/cmd/kden/cmd/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type retryAuthenticator struct {
	usesAccessToken bool
	invalidateErr   error
	loginErr        error
	invalidateCalls int
	loginCalls      int
}

func (a *retryAuthenticator) Invalidate() error {
	a.invalidateCalls++
	return a.invalidateErr
}

func (a *retryAuthenticator) Login(context.Context) error {
	a.loginCalls++
	return a.loginErr
}

func (a *retryAuthenticator) UsesAccessToken() bool {
	return a.usesAccessToken
}

type retryResponse struct {
	status int
}

func (r *retryResponse) StatusCode() int {
	return r.status
}

var _ = Describe("RequestWithAuthRetry", func() {
	It("does not retry a rejected access token", func() {
		authenticator := &retryAuthenticator{
			usesAccessToken: true,
		}
		unauthorized := &retryResponse{
			status: http.StatusUnauthorized,
		}
		requestCalls := 0

		response, err := authcmd.RequestWithAuthRetry(
			context.Background(),
			authenticator,
			func(context.Context) (*retryResponse, error) {
				requestCalls++
				return unauthorized, nil
			},
		)

		Expect(response).To(BeIdenticalTo(unauthorized))
		Expect(err).To(MatchError(
			"authenticating with access token failed. token was rejected",
		))
		Expect(requestCalls).To(Equal(1))
		Expect(authenticator.invalidateCalls).To(BeZero())
		Expect(authenticator.loginCalls).To(BeZero())
	})

	It("retains the session login and retry behavior", func() {
		authenticator := &retryAuthenticator{}
		requestCalls := 0

		response, err := authcmd.RequestWithAuthRetry(
			context.Background(),
			authenticator,
			func(context.Context) (*retryResponse, error) {
				requestCalls++
				if requestCalls == 1 {
					return &retryResponse{
						status: http.StatusUnauthorized,
					}, nil
				}

				return &retryResponse{
					status: http.StatusOK,
				}, nil
			},
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(response.StatusCode()).To(Equal(http.StatusOK))
		Expect(requestCalls).To(Equal(2))
		Expect(authenticator.invalidateCalls).To(Equal(1))
		Expect(authenticator.loginCalls).To(Equal(1))
	})

	It("returns transport errors without authentication attempts", func() {
		requestErr := errors.New("connection failed")
		authenticator := &retryAuthenticator{}

		response, err := authcmd.RequestWithAuthRetry(
			context.Background(),
			authenticator,
			func(context.Context) (*retryResponse, error) {
				return nil, requestErr
			},
		)

		Expect(response).To(BeNil())
		Expect(errors.Is(err, requestErr)).To(BeTrue())
		Expect(authenticator.invalidateCalls).To(BeZero())
		Expect(authenticator.loginCalls).To(BeZero())
	})
})
