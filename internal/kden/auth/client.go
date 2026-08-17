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

const loginSuccessPage = `<!DOCTYPE html>
<html>
<head>
    <title>Konfidence Login Successful</title>
    <style>
        body { font-family: sans-serif; text-align: center; padding: 50px; background: #f9f9f9; }
        .card { background: white; padding: 30px; border-radius: 8px; display: inline-block; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        h1 { color: #2e7d32; }
    </style>
</head>
<body>
    <div class="card">
        <h1>✓ Login Successful</h1>
        <p>You can now safely close this browser window and return to your terminal.</p>
    </div>
    <script>
        setTimeout(() => {
            window.open('', '_self').close();
        }, 1000);
    </script>
</body>
</html>`

const loginFailurePage = `<!DOCTYPE html>
<html lang="de">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Konfidence Login Failed</title>
    <style>
        :root {
            --error-color: #d32f2f;
            --bg-color: #fcf8f8;
            --text-color: #333333;
            --card-bg: #ffffff;
        }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; 
            text-align: center; 
            padding: 50px 20px; 
            background-color: var(--bg-color); 
            color: var(--text-color);
        }
        .card { 
            background: var(--card-bg); 
            padding: 40px 30px; 
            border-radius: 12px; 
            display: inline-block; 
            max-width: 450px;
            width: 100%;
            box-shadow: 0 10px 25px rgba(211, 47, 47, 0.1); 
            border: 1px solid rgba(211, 47, 47, 0.2);
        }
        .icon {
            font-size: 48px;
            color: var(--error-color);
            margin-bottom: 15px;
        }
        h1 { 
            color: var(--error-color); 
            font-size: 24px;
            margin-top: 0;
            margin-bottom: 10px;
        }
        p {
            font-size: 16px;
            line-height: 1.5;
            color: #555;
            margin-bottom: 25px;
        }
        .error-details {
            background: #f5f5f5;
            padding: 12px;
            border-radius: 6px;
            font-family: monospace;
            font-size: 13px;
            color: #666;
            text-align: left;
            word-break: break-all;
            border-left: 4px solid var(--error-color);
            margin-bottom: 25px;
        }
        .instructions {
            font-weight: bold;
            color: var(--text-color);
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon">✕</div>
        <h1>Login failed</h1>
        <p>Authentication process could not be completed. See terminal for details.</p>

        <p class="instructions">You can now close this browser window and return to your terminal.</p>
    </div>

    <script>
        setTimeout(() => {
            window.open('', '_self').close();
        }, 5000);
    </script>
</body>
</html>
`

type Client struct {
	apiEndpoint  string
	apiURL       *url.URL
	apiClient    *kdenapi.ClientWithResponses
	httpClient   *http.Client
	cookieJar    http.CookieJar
	cookieStore  CookieStore
	openURL      func(string) error
	loginTimeout time.Duration
}

type loginResult struct {
	code string
	err  error
}

// NewClient creates a new auth client embedding the login flow and the kden api client
func NewClient(apiEndpoint string, store CookieStore, loginTimeout time.Duration, requestTimeout time.Duration) (*Client, error) {
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

	// create the kden api client
	api, err := kdenapi.NewClientWithResponses(
		apiEndpoint,
		kdenapi.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("creating API client failed: %w", err)
	}

	client := &Client{
		apiEndpoint:  apiEndpoint,
		apiURL:       apiURL,
		apiClient:    api,
		httpClient:   httpClient,
		cookieJar:    jar,
		cookieStore:  store,
		openURL:      browser.OpenURL,
		loginTimeout: loginTimeout,
	}

	// load the session cookie in the cookie jar if it already exists in the keyring
	cookie, err := store.Load(apiEndpoint)
	if err != nil {
		return nil, err
	}
	if cookie != nil {
		jar.SetCookies(apiURL, []*http.Cookie{cookie})
	}

	return client, nil
}

// KdenApiClient returns the kden api client
func (c *Client) KdenApiClient() *kdenapi.ClientWithResponses {
	return c.apiClient
}

// Invalidate invalidates the session cookie in the cookie jar
func (c *Client) Invalidate() error {
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
	loginResponse, err := c.apiClient.LoginV1WithResponse(ctx, &kdenapi.LoginV1Params{
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
	exchangeResponse, err := c.apiClient.PostExchangeCodeV1WithResponse(
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
	log.Info("Starting kden api logout...")

	// ignore response code here, either the user was logged in (results in 200)
	// or 401 is returned if the user was not logged in or the session already expired
	_, err := c.apiClient.LogoutV1WithResponse(ctx)
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
	response, err := c.apiClient.GetIdentityV1WithResponse(ctx)
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

func writeLoginResultPage(
	w http.ResponseWriter,
	status int,
	page string,
) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy",
		"default-src 'none'; script-src 'unsafe-inline'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline';")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(page))
}
