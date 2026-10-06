package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	kdenapi "github.com/konfidence-project/konfidence/internal/kden/apiclient"
	"github.com/konfidence-project/konfidence/internal/kden/log"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

const (
	sessionCookieDefaultName = "kden-session"
	callbackPath             = "/callback"
)

type Client struct {
	*kdenapi.ClientWithResponses
	apiEndpoint          string
	apiURL               *url.URL
	authenticationClient *kdenapi.ClientWithResponses
	httpClient           *http.Client
	cookieJar            http.CookieJar
	cookieStore          CookieStore
	openURL              func(string) error
	loginTimeout         time.Duration
	usesAccessToken      bool
}

type loginResult struct {
	code string
	err  error
}

// NewClient creates a new auth client embedding the login flow and the kden api client
func NewClient(apiEndpoint string, accessToken string, store CookieStore,
	loginTimeout time.Duration, requestTimeout time.Duration) (*Client, error) {
	apiURL, err := url.Parse(apiEndpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing API endpoint failed: %w", err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar failed: %w", err)
	}

	httpClient := &http.Client{
		Jar:     jar,
		Timeout: requestTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var options []kdenapi.ClientOption
	if accessToken != "" {
		options = append(options, kdenapi.WithRequestEditorFn(
			func(_ context.Context, req *http.Request) error {
				req.Header.Set("Authorization", "Bearer "+accessToken)
				return nil
			},
		))
	}

	// create the kden api client
	apiOptions := append(
		[]kdenapi.ClientOption{
			kdenapi.WithHTTPClient(unauthorizedDoer{doer: httpClient, usesAccessToken: accessToken != ""})},
		options...,
	)

	api, err := kdenapi.NewClientWithResponses(apiEndpoint, apiOptions...)
	if err != nil {
		return nil, fmt.Errorf("creating API client failed: %w", err)
	}

	// create a separate client for authentication
	authenticationOptions := append([]kdenapi.ClientOption{kdenapi.WithHTTPClient(httpClient)}, options...)
	authenticationClient, err := kdenapi.NewClientWithResponses(apiEndpoint, authenticationOptions...)
	if err != nil {
		return nil, fmt.Errorf("creating authentication API client failed: %w", err)
	}

	client := &Client{
		ClientWithResponses:  api,
		apiEndpoint:          apiEndpoint,
		apiURL:               apiURL,
		authenticationClient: authenticationClient,
		httpClient:           httpClient,
		cookieJar:            jar,
		cookieStore:          store,
		openURL:              browser.OpenURL,
		loginTimeout:         loginTimeout,
		usesAccessToken:      accessToken != "",
	}

	if !client.usesAccessToken {
		// load the session cookie in the cookie jar if it already exists in the keyring
		cookie, err := store.Load(apiEndpoint)
		if err != nil {
			return nil, err
		}
		if cookie != nil {
			jar.SetCookies(apiURL, []*http.Cookie{cookie})
		}
	}

	return client, nil
}

// Invalidate invalidates the session cookie in the cookie jar
func (c *Client) Invalidate() error {
	if c.usesAccessToken {
		return nil
	}

	cookieName := ""
	cookie, err := c.cookieStore.Load(c.apiEndpoint)
	if err != nil {
		return fmt.Errorf("loading cookie failed: %w", err)
	}

	if cookie != nil {
		cookieName = cookie.Name
	}

	if cookieName == "" {
		cookieName = c.sessionCookieName()
	}

	c.cookieJar.SetCookies(c.apiURL, []*http.Cookie{{
		Name:   cookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}})

	return c.cookieStore.Delete(c.apiEndpoint)
}

// Login starts the OIDC login flow with the kden api if the user has no active valid session
func (c *Client) Login(ctx context.Context) error {
	if c.usesAccessToken {
		return errors.New("login is not available when access-token authentication is active")
	}

	log.Info("Starting kden api login...")
	authenticated, err := c.hasValidSession(ctx)
	if err != nil {
		return err
	}
	if authenticated {
		log.Info("Already logged in")
		return nil
	}

	verifier := oauth2.GenerateVerifier()

	// start the local callback listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("starting login callback listener failed: %w", err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			log.Errorf("failed to close callback listener: %s\n", err)
		}
	}(listener)

	// create the callback url
	localState := oauth2.GenerateVerifier()
	callbackURL := (&url.URL{
		Scheme:   "http",
		Host:     listener.Addr().String(),
		Path:     callbackPath,
		RawQuery: url.Values{"state": []string{localState}}.Encode(),
	}).String()

	// create a temporary local server to receive callback from the api
	resultChannel := make(chan loginResult, 1)
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           loginCallbackHandler(localState, resultChannel),
	}

	// and start server
	go func() {
		_ = server.Serve(listener)
	}()
	defer func(server *http.Server, ctx context.Context) {
		err := server.Shutdown(ctx)
		if err != nil {
			log.Errorf("failed to shutdown temp server: %s\n", err)
		}
	}(server, context.Background())

	// initiate login
	challenge := oauth2.S256ChallengeFromVerifier(verifier)
	loginResponse, err := c.authenticationClient.LoginV1WithResponse(ctx, &kdenapi.LoginV1Params{
		ReturnUrl:     callbackURL,
		CodeChallenge: &challenge,
	})
	if err != nil {
		return fmt.Errorf("initiating API login failed: %w", err)
	}
	if loginResponse.StatusCode() != http.StatusFound ||
		loginResponse.Headers302 == nil ||
		loginResponse.Headers302.Location == nil {
		return fmt.Errorf("initiating API login returned HTTP %d", loginResponse.StatusCode())
	}

	// try to open idp redirect in browser window
	if err := c.openURL(*loginResponse.Headers302.Location); err != nil {
		return fmt.Errorf("opening browser login failed: %w", err)
	}

	waitContext, cancel := context.WithTimeout(ctx, c.loginTimeout)
	defer cancel()
	var exchangeCode string

	select {
	case result := <-resultChannel:
		if result.err != nil {
			return result.err
		}
		exchangeCode = result.code
	case <-waitContext.Done():
		switch {
		case errors.Is(ctx.Err(), context.Canceled):
			return errors.New("browser login was canceled")
		case errors.Is(waitContext.Err(), context.DeadlineExceeded):
			return fmt.Errorf(
				"browser login was not completed within %s; the browser may have been closed",
				c.loginTimeout,
			)
		default:
			return fmt.Errorf(
				"waiting for browser login failed: %w",
				waitContext.Err(),
			)
		}
	}

	// use exchange code and verifier to retrieve session cookie
	exchangeResponse, err := c.authenticationClient.PostExchangeCodeV1WithResponse(
		ctx,
		kdenapi.PostExchangeCodeV1JSONRequestBody{
			Code:     exchangeCode,
			Verifier: verifier,
		},
	)
	if err != nil {
		return fmt.Errorf("exchanging login code failed: %w", err)
	}
	if exchangeResponse.StatusCode() != http.StatusOK {
		return fmt.Errorf("exchanging login code returned HTTP %d", exchangeResponse.StatusCode())
	}

	if exchangeResponse.Headers200 == nil || exchangeResponse.Headers200.SetCookie == nil {
		return errors.New("api did not return a session cookie")
	}

	responseCookie, err := http.ParseSetCookie(*exchangeResponse.Headers200.SetCookie)
	if err != nil {
		return fmt.Errorf("parsing API session cookie failed: %w", err)
	}
	if responseCookie.Name == "" || responseCookie.Value == "" {
		return errors.New("api returned an invalid session cookie")
	}

	cookieInJar := false
	for _, jarCookie := range c.cookieJar.Cookies(c.apiURL) {
		if jarCookie.Name == responseCookie.Name &&
			jarCookie.Value == responseCookie.Value {
			cookieInJar = true
			break
		}
	}
	if !cookieInJar {
		return errors.New("api session cookie was not accepted by the cookie jar")
	}

	if err := c.cookieStore.Save(c.apiEndpoint, responseCookie); err != nil {
		return fmt.Errorf("storing cookie failed: %w", err)
	}

	log.Info("Successfully logged in.")
	return nil
}

func (c *Client) Logout(ctx context.Context) error {
	if c.usesAccessToken {
		return errors.New(
			"logout is not available when access-token authentication is active",
		)
	}

	log.Info("Starting kden api logout...")

	// ignore response code here, either the user was logged in (results in 200)
	// or 401 is returned if the user was not logged in or the session already expired
	_, err := c.authenticationClient.LogoutV1WithResponse(ctx)
	if err != nil {
		return fmt.Errorf("logout of Konfidence API failed: %w", err)
	}

	// remove session cookie
	err = c.Invalidate()
	if err != nil {
		return fmt.Errorf("removing session cookie failed: %w", err)
	}

	log.Info("Successfully logged out.")
	return nil
}

func loginCallbackHandler(
	localState string,
	resultChannel chan<- loginResult,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if request.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if request.URL.Path != callbackPath {
			http.NotFound(w, request)
			return
		}

		query := request.URL.Query()
		if query.Get("state") != localState {
			http.Error(w, "invalid login callback state", http.StatusBadRequest)
			return
		}

		var result loginResult
		if authError := query.Get("error"); authError != "" {
			description := query.Get("error_description")
			if description == "" {
				result.err = fmt.Errorf("authentication failed: %s", authError)
			} else {
				result.err = fmt.Errorf(
					"authentication failed: %s: %s",
					authError,
					description,
				)
			}
		} else {
			code := query.Get("code")
			if code == "" {
				http.Error(w, "missing exchange code", http.StatusBadRequest)
				return
			}

			result.code = code
		}

		select {
		case resultChannel <- result:
			if result.err != nil {
				writeLoginResultPage(w, http.StatusUnauthorized, loginFailurePage)
				return
			}

			writeLoginResultPage(w, http.StatusOK, loginSuccessPage)
		default:
			http.Error(
				w,
				"login callback already received",
				http.StatusConflict,
			)
		}
	})
}

func (c *Client) sessionCookieName() string {
	cookies := c.cookieJar.Cookies(c.apiURL)
	for _, cookie := range cookies {
		if cookie.Name != "" {
			return cookie.Name
		}
	}
	return sessionCookieDefaultName
}

func (c *Client) hasValidSession(ctx context.Context) (bool, error) {
	if c.usesAccessToken {
		return false, errors.New(
			"session validation is not available when access-token authentication is active",
		)
	}

	response, err := c.authenticationClient.GetIdentityV1WithResponse(ctx)
	if err != nil {
		return false, fmt.Errorf("checking current session failed: %w", err)
	}

	switch response.StatusCode() {
	case http.StatusOK:
		if response.JSON200 == nil {
			return false, errors.New("identity response did not contain a body")
		}
		return true, nil

	case http.StatusUnauthorized:
		if err := c.Invalidate(); err != nil {
			return false, fmt.Errorf("removing invalid session: %w", err)
		}
		return false, nil

	default:
		return false, fmt.Errorf(
			"checking current session returned HTTP %d: %s",
			response.StatusCode(),
			string(response.Body),
		)
	}
}

func (c *Client) UsesAccessToken() bool {
	return c.usesAccessToken
}

func writeLoginResultPage(
	w http.ResponseWriter,
	status int,
	page string,
) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set(
		"Content-Security-Policy",
		"default-src 'none'; "+
			"style-src 'unsafe-inline'; "+
			"script-src 'unsafe-inline'; "+
			"base-uri 'none'; "+
			"form-action 'none'; "+
			"frame-ancestors 'none'",
	)
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(page))
}
