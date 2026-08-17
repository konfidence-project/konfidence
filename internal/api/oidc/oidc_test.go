package oidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"
)

type providerFixture struct {
	server      *httptest.Server
	tokenForm   url.Values
	tokenHeader http.Header
	userHeader  http.Header
}

func newProviderFixture() *providerFixture {
	fixture := &providerFixture{}

	fixture.server = httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			writer.Header().Set("Content-Type", "application/json")
			Expect(json.NewEncoder(writer).Encode(map[string]any{
				"issuer":                 fixture.server.URL,
				"authorization_endpoint": fixture.server.URL + "/authorize",
				"token_endpoint":         fixture.server.URL + "/token",
				"userinfo_endpoint":      fixture.server.URL + "/userinfo",
				"jwks_uri":               fixture.server.URL + "/keys",
			})).To(Succeed())

		case "/token":
			Expect(request.ParseForm()).To(Succeed())
			fixture.tokenForm = request.Form
			fixture.tokenHeader = request.Header.Clone()

			writer.Header().Set("Content-Type", "application/json")
			Expect(json.NewEncoder(writer).Encode(map[string]any{
				"access_token":  "access-token",
				"token_type":    "Bearer",
				"refresh_token": "refresh-token",
				"expires_in":    3600,
			})).To(Succeed())

		case "/userinfo":
			fixture.userHeader = request.Header.Clone()

			writer.Header().Set("Content-Type", "application/json")
			Expect(json.NewEncoder(writer).Encode(map[string]any{
				"sub":                "user-id",
				"name":               "Test User",
				"given_name":         "Test",
				"family_name":        "User",
				"preferred_username": "test.user",
				"email":              "test@example.com",
				"groups":             []string{"developers", "admins"},
			})).To(Succeed())

		case "/keys":
			writer.Header().Set("Content-Type", "application/json")
			_, err := io.WriteString(writer, `{"keys":[]}`)
			Expect(err).NotTo(HaveOccurred())

		default:
			http.NotFound(writer, request)
		}
	}))

	return fixture
}

func signIDToken(
	privateKey *rsa.PrivateKey,
	claims map[string]any,
) string {
	signer, err := jose.NewSigner(
		jose.SigningKey{
			Algorithm: jose.RS256,
			Key:       privateKey,
		},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	Expect(err).NotTo(HaveOccurred())

	payload, err := json.Marshal(claims)
	Expect(err).NotTo(HaveOccurred())

	signed, err := signer.Sign(payload)
	Expect(err).NotTo(HaveOccurred())

	rawToken, err := signed.CompactSerialize()
	Expect(err).NotTo(HaveOccurred())

	return rawToken
}

var _ = Describe("OIDC Client", func() {
	Describe("Setup", func() {
		It("configures the client from provider discovery", func() {
			fixture := newProviderFixture()
			DeferCleanup(fixture.server.Close)

			client := NewOIDCClient(Config{
				IdentityProviderURI: fixture.server.URL,
				ClientID:            "client-id",
				ClientSecret:        "client-secret",
				RedirectURI:         "https://api.example.com/callback",
				Scopes:              []string{"openid", "profile", "email"},
			})

			Expect(client.Setup(context.Background())).To(Succeed())

			Expect(client.oidcProvider).NotTo(BeNil())
			Expect(client.oauth2Config.ClientID).To(Equal("client-id"))
			Expect(client.oauth2Config.ClientSecret).
				To(Equal("client-secret"))
			Expect(client.oauth2Config.RedirectURL).
				To(Equal("https://api.example.com/callback"))
			Expect(client.oauth2Config.Scopes).
				To(Equal([]string{"openid", "profile", "email"}))
			Expect(client.oauth2Config.Endpoint.AuthURL).
				To(Equal(fixture.server.URL + "/authorize"))
			Expect(client.oauth2Config.Endpoint.TokenURL).
				To(Equal(fixture.server.URL + "/token"))
			Expect(client.oauth2Config.Endpoint.AuthStyle).
				To(Equal(oauth2.AuthStyleInHeader))
		})

		It("uses configured OAuth endpoint overrides", func() {
			fixture := newProviderFixture()
			DeferCleanup(fixture.server.Close)

			client := NewOIDCClient(Config{
				IdentityProviderURI: fixture.server.URL,
				ClientID:            "client-id",
				TokenURL:            "https://override.example.com/token",
				AuthorizationURL:    "https://override.example.com/authorize",
				DeviceAuthURL:       "https://override.example.com/device",
			})

			Expect(client.Setup(context.Background())).To(Succeed())

			Expect(client.oauth2Config.Endpoint.TokenURL).
				To(Equal("https://override.example.com/token"))
			Expect(client.oauth2Config.Endpoint.AuthURL).
				To(Equal("https://override.example.com/authorize"))
			Expect(client.oauth2Config.Endpoint.DeviceAuthURL).
				To(Equal("https://override.example.com/device"))
		})

		It("returns an error when provider discovery fails", func() {
			server := httptest.NewServer(http.HandlerFunc(func(
				writer http.ResponseWriter,
				_ *http.Request,
			) {
				http.Error(
					writer,
					"provider unavailable",
					http.StatusInternalServerError,
				)
			}))
			DeferCleanup(server.Close)

			client := NewOIDCClient(Config{
				IdentityProviderURI: server.URL,
			})

			err := client.Setup(context.Background())

			Expect(err).To(MatchError(
				ContainSubstring("failed to create oidc provider"),
			))
		})
	})

	Describe("GenerateState", func() {
		It("generates state and nonce without PKCE when disabled", func() {
			client := NewOIDCClient(Config{PKCEEnabled: false})
			before := time.Now()

			state, err := client.GenerateState(
				"https://dashboard.example.com/callback",
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(state.State).To(HaveLen(64))
			Expect(state.Nonce).To(HaveLen(64))
			Expect(state.ReturnURL).
				To(Equal("https://dashboard.example.com/callback"))
			Expect(state.CodeVerifier).To(BeEmpty())
			Expect(state.CodeChallenge).To(BeEmpty())
			Expect(state.CodeChallengeMethod).To(BeEmpty())
			Expect(state.CreatedAt).To(BeTemporally(">=", before))
			Expect(state.CreatedAt).To(BeTemporally("<=", time.Now()))
		})

		It("generates S256 PKCE parameters when enabled", func() {
			client := NewOIDCClient(Config{PKCEEnabled: true})

			state, err := client.GenerateState(
				"http://127.0.0.1:12345/callback",
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(state.CodeVerifier).NotTo(BeEmpty())
			Expect(state.CodeChallenge).To(Equal(
				oauth2.S256ChallengeFromVerifier(state.CodeVerifier),
			))
			Expect(state.CodeChallengeMethod).To(Equal("S256"))
		})

		It("generates different state and nonce values", func() {
			client := NewOIDCClient(Config{})

			first, err := client.GenerateState("https://example.com/first")
			Expect(err).NotTo(HaveOccurred())

			second, err := client.GenerateState("https://example.com/second")
			Expect(err).NotTo(HaveOccurred())

			Expect(second.State).NotTo(Equal(first.State))
			Expect(second.Nonce).NotTo(Equal(first.Nonce))
		})
	})

	Describe("AuthCodeURL", func() {
		It("includes state, nonce and response mode", func() {
			client := NewOIDCClient(Config{PKCEEnabled: false})
			client.oauth2Config = oauth2.Config{
				ClientID:    "client-id",
				RedirectURL: "https://api.example.com/callback",
				Scopes:      []string{"openid", "profile"},
				Endpoint: oauth2.Endpoint{
					AuthURL: "https://idp.example.com/authorize",
				},
			}

			rawURL := client.AuthCodeURL(&StateData{
				State: "state-id",
				Nonce: "nonce-id",
			})

			authURL, err := url.Parse(rawURL)
			Expect(err).NotTo(HaveOccurred())
			Expect(authURL.Scheme).To(Equal("https"))
			Expect(authURL.Host).To(Equal("idp.example.com"))
			Expect(authURL.Path).To(Equal("/authorize"))
			Expect(authURL.Query()).To(HaveKeyWithValue(
				"client_id",
				[]string{"client-id"},
			))
			Expect(authURL.Query()).To(HaveKeyWithValue(
				"redirect_uri",
				[]string{"https://api.example.com/callback"},
			))
			Expect(authURL.Query()).To(HaveKeyWithValue(
				"response_type",
				[]string{"code"},
			))
			Expect(authURL.Query()).To(HaveKeyWithValue(
				"scope",
				[]string{"openid profile"},
			))
			Expect(authURL.Query()).To(HaveKeyWithValue(
				"state",
				[]string{"state-id"},
			))
			Expect(authURL.Query()).To(HaveKeyWithValue(
				"nonce",
				[]string{"nonce-id"},
			))
			Expect(authURL.Query()).To(HaveKeyWithValue(
				"response_mode",
				[]string{"query"},
			))
			Expect(authURL.Query()).NotTo(HaveKey("code_challenge"))
		})

		It("includes the PKCE S256 challenge when enabled", func() {
			client := NewOIDCClient(Config{PKCEEnabled: true})
			client.oauth2Config = oauth2.Config{
				ClientID: "client-id",
				Endpoint: oauth2.Endpoint{
					AuthURL: "https://idp.example.com/authorize",
				},
			}

			verifier := oauth2.GenerateVerifier()
			rawURL := client.AuthCodeURL(&StateData{
				State:        "state-id",
				Nonce:        "nonce-id",
				CodeVerifier: verifier,
			})

			authURL, err := url.Parse(rawURL)
			Expect(err).NotTo(HaveOccurred())
			Expect(authURL.Query()).To(HaveKeyWithValue(
				"code_challenge",
				[]string{oauth2.S256ChallengeFromVerifier(verifier)},
			))
			Expect(authURL.Query()).To(HaveKeyWithValue(
				"code_challenge_method",
				[]string{"S256"},
			))
		})
	})

	Describe("Exchange", func() {
		It("exchanges an authorization code with its PKCE verifier", func() {
			fixture := newProviderFixture()
			DeferCleanup(fixture.server.Close)

			client := NewOIDCClient(Config{
				IdentityProviderURI: fixture.server.URL,
				ClientID:            "client-id",
				ClientSecret:        "client-secret",
				RedirectURI:         "https://api.example.com/callback",
				PKCEEnabled:         true,
			})
			Expect(client.Setup(context.Background())).To(Succeed())

			state := &StateData{
				CodeVerifier: oauth2.GenerateVerifier(),
			}

			token, err := client.Exchange(
				context.Background(),
				"authorization-code",
				state,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(token.AccessToken).To(Equal("access-token"))
			Expect(token.RefreshToken).To(Equal("refresh-token"))
			Expect(token.TokenType).To(Equal("Bearer"))
			Expect(token.Expiry).To(BeTemporally(
				"~",
				time.Now().Add(time.Hour),
				5*time.Second,
			))

			Expect(fixture.tokenForm).To(HaveKeyWithValue(
				"grant_type",
				[]string{"authorization_code"},
			))
			Expect(fixture.tokenForm).To(HaveKeyWithValue(
				"code",
				[]string{"authorization-code"},
			))
			Expect(fixture.tokenForm).To(HaveKeyWithValue(
				"redirect_uri",
				[]string{"https://api.example.com/callback"},
			))
			Expect(fixture.tokenForm).To(HaveKeyWithValue(
				"code_verifier",
				[]string{state.CodeVerifier},
			))

			username, password, ok :=
				fixture.tokenHeader.Get("Authorization"), "", false
			if fixture.tokenHeader != nil {
				request := &http.Request{Header: fixture.tokenHeader}
				username, password, ok = request.BasicAuth()
			}
			Expect(ok).To(BeTrue())
			Expect(username).To(Equal("client-id"))
			Expect(password).To(Equal("client-secret"))
		})

		It("does not send a verifier when PKCE is disabled", func() {
			fixture := newProviderFixture()
			DeferCleanup(fixture.server.Close)

			client := NewOIDCClient(Config{
				IdentityProviderURI: fixture.server.URL,
				ClientID:            "client-id",
				ClientSecret:        "client-secret",
				PKCEEnabled:         false,
			})
			Expect(client.Setup(context.Background())).To(Succeed())

			_, err := client.Exchange(
				context.Background(),
				"authorization-code",
				&StateData{CodeVerifier: "unused-verifier"},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(fixture.tokenForm).NotTo(HaveKey("code_verifier"))
		})

		It("wraps token endpoint failures", func() {
			server := httptest.NewServer(http.HandlerFunc(func(
				writer http.ResponseWriter,
				_ *http.Request,
			) {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusUnauthorized)
				_, _ = writer.Write([]byte(`{"error":"invalid_grant"}`))
			}))
			DeferCleanup(server.Close)

			client := NewOIDCClient(Config{})
			client.oauth2Config = oauth2.Config{
				ClientID: "client-id",
				Endpoint: oauth2.Endpoint{
					TokenURL:  server.URL,
					AuthStyle: oauth2.AuthStyleInHeader,
				},
			}

			_, err := client.Exchange(
				context.Background(),
				"invalid-code",
				&StateData{},
			)

			Expect(err).To(MatchError(
				ContainSubstring("token exchange failed"),
			))
		})
	})

	Describe("VerifyAndGetIdToken", func() {
		var (
			privateKey *rsa.PrivateKey
			client     *Client
		)

		BeforeEach(func() {
			var err error
			privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).NotTo(HaveOccurred())

			verifier := coreoidc.NewVerifier(
				"https://idp.example.com",
				&coreoidc.StaticKeySet{
					PublicKeys: []crypto.PublicKey{
						&privateKey.PublicKey,
					},
				},
				&coreoidc.Config{ClientID: "client-id"},
			)

			client = NewOIDCClient(Config{})
			client.idTokenVerifier = *verifier
		})

		It("verifies and returns the ID token", func() {
			rawIDToken := signIDToken(privateKey, map[string]any{
				"iss":   "https://idp.example.com",
				"sub":   "user-id",
				"aud":   "client-id",
				"exp":   time.Now().Add(time.Hour).Unix(),
				"iat":   time.Now().Unix(),
				"nonce": "nonce-id",
			})

			token := (&oauth2.Token{AccessToken: "access-token"}).
				WithExtra(map[string]any{"id_token": rawIDToken})

			idToken, err := client.VerifyAndGetIdToken(
				context.Background(),
				token,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(idToken.Subject).To(Equal("user-id"))
			Expect(idToken.Nonce).To(Equal("nonce-id"))
			Expect(idToken.Audience).To(ContainElement("client-id"))
		})

		It("rejects a token response without an ID token", func() {
			_, err := client.VerifyAndGetIdToken(
				context.Background(),
				&oauth2.Token{AccessToken: "access-token"},
			)

			Expect(err).To(MatchError(
				"missing id token in token response",
			))
		})

		It("rejects an invalid ID token", func() {
			token := (&oauth2.Token{AccessToken: "access-token"}).
				WithExtra(map[string]any{"id_token": "not-a-jwt"})

			_, err := client.VerifyAndGetIdToken(
				context.Background(),
				token,
			)

			Expect(err).To(MatchError("failed to verify id token"))
		})
	})

	Describe("User information", func() {
		It("retrieves user information and parses additional claims", func() {
			fixture := newProviderFixture()
			DeferCleanup(fixture.server.Close)

			client := NewOIDCClient(Config{
				IdentityProviderURI: fixture.server.URL,
				ClientID:            "client-id",
			})
			Expect(client.Setup(context.Background())).To(Succeed())

			userInfo, err := client.GetUserInformation(
				context.Background(),
				"access-token",
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(userInfo.Subject).To(Equal("user-id"))
			Expect(fixture.userHeader.Get("Authorization")).
				To(Equal("Bearer access-token"))

			claims, _ := client.GetClaims(userInfo)

			Expect(claims.Name).NotTo(BeNil())
			Expect(*claims.Name).To(Equal("Test User"))

			Expect(claims.GivenName).NotTo(BeNil())
			Expect(*claims.GivenName).To(Equal("Test"))

			Expect(claims.FamilyName).NotTo(BeNil())
			Expect(*claims.FamilyName).To(Equal("User"))

			Expect(claims.PreferredUsername).NotTo(BeNil())
			Expect(*claims.PreferredUsername).To(Equal("test.user"))

			Expect(claims.Email).NotTo(BeNil())
			Expect(*claims.Email).To(Equal("test@example.com"))

			Expect(claims.Groups).To(ConsistOf("developers", "admins"))
		})

		It("returns an error when claims are unavailable", func() {
			client := NewOIDCClient(Config{})

			_, err := client.GetClaims(&coreoidc.UserInfo{})

			Expect(err).To(MatchError(
				ContainSubstring("failed to parse claims"),
			))
		})

		It("returns an error when the UserInfo endpoint rejects the token", func() {
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(
				writer http.ResponseWriter,
				request *http.Request,
			) {
				switch request.URL.Path {
				case "/.well-known/openid-configuration":
					writer.Header().Set(
						"Content-Type",
						"application/json",
					)
					_ = json.NewEncoder(writer).Encode(map[string]any{
						"issuer":                 server.URL,
						"authorization_endpoint": server.URL + "/authorize",
						"token_endpoint":         server.URL + "/token",
						"userinfo_endpoint":      server.URL + "/userinfo",
						"jwks_uri":               server.URL + "/keys",
					})
				case "/userinfo":
					http.Error(
						writer,
						"invalid token",
						http.StatusUnauthorized,
					)
				}
			}))
			DeferCleanup(server.Close)

			client := NewOIDCClient(Config{
				IdentityProviderURI: server.URL,
				ClientID:            "client-id",
			})
			Expect(client.Setup(context.Background())).To(Succeed())

			_, err := client.GetUserInformation(
				context.Background(),
				"invalid-access-token",
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("401 Unauthorized"))
		})
	})
})
