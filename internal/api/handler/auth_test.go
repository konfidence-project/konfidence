package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"
)

type recordingStateStore struct {
	saved      *oidc.StateData
	consumed   *oidc.StateData
	consumeErr error
}

func (s *recordingStateStore) Save(state *oidc.StateData) error {
	s.saved = state
	return nil
}

func (s *recordingStateStore) Consume(string) (*oidc.StateData, error) {
	return s.consumed, s.consumeErr
}

var _ = Describe("Allowed return URL", func() {
	allowReturnURLs := []string{
		"https://dashboard.example.com/callback",
		"http://localhost:3000/auth?source=kden",
	}

	DescribeTable("checks exact allowlist membership",
		func(returnURL string, expected bool) {
			Expect(allowedReturnURL(returnURL, allowReturnURLs)).
				To(Equal(expected))
		},
		Entry(
			"allows the configured dashboard callback",
			"https://dashboard.example.com/callback",
			true,
		),
		Entry(
			"allows the configured local callback including its query",
			"http://localhost:3000/auth?source=kden",
			true,
		),
		Entry(
			"rejects a different path",
			"https://dashboard.example.com/other",
			false,
		),
		Entry(
			"rejects a different host",
			"https://other.example.com/callback",
			false,
		),
		Entry(
			"rejects a different query",
			"http://localhost:3000/auth?source=attacker",
			false,
		),
		Entry("rejects a relative URL", "/projects", false),
		Entry("rejects an empty URL", "", false),
	)
})

var _ = Describe("LoginV1", func() {
	It("rejects an unlisted UI return URL without storing state", func() {
		stateStore := &recordingStateStore{}
		parsed := config.Parsed{
			Session: config.ParsedSessionConfig{
				Expiry: 2 * time.Minute,
			},
		}

		handler := newAuthHandler(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			*oidc.NewOIDCClient(oidc.Config{}),
			stateStore,
			oidc.NewExchangeCacheStore(5*time.Minute),
			session.NewInMemoryStore(parsed),
			config.Parsed{
				OIDC: config.ParsedOIDCConfig{
					AllowReturnURLs: []string{
						"https://dashboard.example.com/callback",
					},
				},
			},
		)

		response, err := handler.LoginV1(
			context.Background(),
			openapi.LoginV1RequestObject{
				Params: openapi.LoginV1Params{
					ReturnUrl: "https://attacker.example.com/callback",
				},
			},
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(response).To(BeAssignableToTypeOf(
			openapi.LoginV1400JSONResponse{},
		))
		Expect(stateStore.saved).To(BeNil())
	})

	var _ = Describe("CLI return URL validation", func() {
		DescribeTable("validates loopback callback URLs",
			func(rawURL string, allowed bool) {
				Expect(allowedCLIReturnURL(rawURL)).To(Equal(allowed))
			},
			Entry("IPv4 loopback", "http://127.0.0.1:12345/callback", true),
			Entry("IPv6 loopback", "http://[::1]:12345/callback", true),
			Entry("HTTPS", "https://127.0.0.1:12345/callback", false),
			Entry("localhost hostname", "http://localhost:12345/callback", false),
			Entry("external IP", "http://192.0.2.1:12345/callback", false),
			Entry("unspecified IP", "http://0.0.0.0:12345/callback", false),
			Entry("missing port", "http://127.0.0.1/callback", false),
			Entry("wrong path", "http://127.0.0.1:12345/other", false),
			Entry("malformed URL", "://invalid", false),
		)
	})

	var _ = Describe("CLI authentication callback", func() {
		It("forwards an OIDC error to the CLI callback", func() {
			authError := "access_denied"
			description := "the user denied access"

			stateStore := &recordingStateStore{
				consumed: &oidc.StateData{
					State:               "api-state",
					ReturnURL:           "http://127.0.0.1:12345/callback?state=local-state",
					ClientCodeChallenge: "challenge",
				},
			}
			handler := newTestAuthHandler(
				stateStore,
				oidc.NewExchangeCacheStore(time.Minute),
			)

			response, err := handler.AuthCallbackV1(
				context.Background(),
				openapi.AuthCallbackV1RequestObject{
					Params: openapi.AuthCallbackV1Params{
						State:            "api-state",
						Error:            &authError,
						ErrorDescription: &description,
					},
				},
			)

			Expect(err).NotTo(HaveOccurred())

			redirect := response.(openapi.AuthCallbackV1302Response)
			Expect(redirect.Headers.Location).NotTo(BeNil())

			location, err := url.Parse(*redirect.Headers.Location)
			Expect(err).NotTo(HaveOccurred())
			Expect(location.Query()).To(HaveKeyWithValue("state", []string{"local-state"}))
			Expect(location.Query()).To(HaveKeyWithValue("error", []string{"access_denied"}))
			Expect(location.Query()).To(HaveKeyWithValue(
				"error_description",
				[]string{"the user denied access"},
			))
		})

		It("rejects a callback with an unknown state", func() {
			handler := newTestAuthHandler(
				&recordingStateStore{},
				oidc.NewExchangeCacheStore(time.Minute),
			)

			response, err := handler.AuthCallbackV1(
				context.Background(),
				openapi.AuthCallbackV1RequestObject{
					Params: openapi.AuthCallbackV1Params{State: "unknown"},
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(BeAssignableToTypeOf(
				openapi.AuthCallbackV1400JSONResponse{},
			))
		})
	})

	var _ = Describe("CLI code exchange", func() {
		var (
			exchangeStore *oidc.ExchangeCacheStore
			handler       *authHandler
		)

		BeforeEach(func() {
			exchangeStore = oidc.NewExchangeCacheStore(time.Minute)
			handler = newTestAuthHandler(
				&recordingStateStore{},
				exchangeStore,
			)
		})

		It("returns a session cookie for a valid verifier", func() {
			verifier := oauth2.GenerateVerifier()
			challenge := oauth2.S256ChallengeFromVerifier(verifier)

			Expect(exchangeStore.Save("exchange-code", oidc.Exchange{
				SessionID:     "session-id",
				CodeChallenge: challenge,
			})).To(Succeed())

			response, err := handler.PostExchangeCodeV1(
				context.Background(),
				openapi.PostExchangeCodeV1RequestObject{
					Body: &openapi.PostExchangeCodeV1JSONRequestBody{
						Code:     "exchange-code",
						Verifier: verifier,
					},
				},
			)

			Expect(err).NotTo(HaveOccurred())

			success := response.(openapi.PostExchangeCodeV1200Response)
			Expect(success.Headers.SetCookie).NotTo(BeNil())

			cookie, err := http.ParseSetCookie(*success.Headers.SetCookie)
			Expect(err).NotTo(HaveOccurred())
			Expect(cookie.Name).To(Equal("kden-session"))
			Expect(cookie.Value).To(Equal("session-id"))
			Expect(cookie.HttpOnly).To(BeTrue())
			Expect(cookie.Path).To(Equal("/"))
		})

		It("rejects replay of a redeemed code", func() {
			verifier := oauth2.GenerateVerifier()
			Expect(exchangeStore.Save("exchange-code", oidc.Exchange{
				SessionID:     "session-id",
				CodeChallenge: oauth2.S256ChallengeFromVerifier(verifier),
			})).To(Succeed())

			request := openapi.PostExchangeCodeV1RequestObject{
				Body: &openapi.PostExchangeCodeV1JSONRequestBody{
					Code:     "exchange-code",
					Verifier: verifier,
				},
			}

			response, err := handler.PostExchangeCodeV1(
				context.Background(),
				request,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(BeAssignableToTypeOf(
				openapi.PostExchangeCodeV1200Response{},
			))

			response, err = handler.PostExchangeCodeV1(
				context.Background(),
				request,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(BeAssignableToTypeOf(
				openapi.PostExchangeCodeV1401JSONResponse{},
			))
		})

		It("consumes the code after an invalid verifier attempt", func() {
			validVerifier := oauth2.GenerateVerifier()
			Expect(exchangeStore.Save("exchange-code", oidc.Exchange{
				SessionID:     "session-id",
				CodeChallenge: oauth2.S256ChallengeFromVerifier(validVerifier),
			})).To(Succeed())

			response, err := handler.PostExchangeCodeV1(
				context.Background(),
				openapi.PostExchangeCodeV1RequestObject{
					Body: &openapi.PostExchangeCodeV1JSONRequestBody{
						Code:     "exchange-code",
						Verifier: oauth2.GenerateVerifier(),
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(BeAssignableToTypeOf(
				openapi.PostExchangeCodeV1401JSONResponse{},
			))

			response, err = handler.PostExchangeCodeV1(
				context.Background(),
				openapi.PostExchangeCodeV1RequestObject{
					Body: &openapi.PostExchangeCodeV1JSONRequestBody{
						Code:     "exchange-code",
						Verifier: validVerifier,
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(BeAssignableToTypeOf(
				openapi.PostExchangeCodeV1401JSONResponse{},
			))
		})

		DescribeTable("rejects incomplete requests",
			func(request openapi.PostExchangeCodeV1RequestObject) {
				response, err := handler.PostExchangeCodeV1(
					context.Background(),
					request,
				)

				Expect(err).NotTo(HaveOccurred())
				Expect(response).To(BeAssignableToTypeOf(
					openapi.PostExchangeCodeV1401JSONResponse{},
				))
			},
			Entry("missing body", openapi.PostExchangeCodeV1RequestObject{}),
			Entry("missing code", openapi.PostExchangeCodeV1RequestObject{
				Body: &openapi.PostExchangeCodeV1JSONRequestBody{
					Verifier: "verifier",
				},
			}),
			Entry("missing verifier", openapi.PostExchangeCodeV1RequestObject{
				Body: &openapi.PostExchangeCodeV1JSONRequestBody{
					Code: "exchange-code",
				},
			}),
		)
	})
})

func newTestAuthHandler(
	stateStore oidc.StateStore,
	exchangeStore oidc.ExchangeStore,
) *authHandler {
	parsed := config.Parsed{
		Session: config.ParsedSessionConfig{
			Expiry: time.Hour,
			Cookie: config.SessionCookieConfig{
				Name:     "kden-session",
				HTTPOnly: true,
				SameSite: "Strict",
			},
		},
	}

	return newAuthHandler(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		*oidc.NewOIDCClient(oidc.Config{}),
		stateStore,
		exchangeStore,
		session.NewInMemoryStore(parsed),
		parsed,
	)
}
