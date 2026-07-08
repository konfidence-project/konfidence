package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	base string
	http *http.Client
}

// New creates a Client that targets the API gateway at baseURL
// (e.g. "http://localhost:8090" or "http://konfidence-api.konfidence-system:8090").
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{
		base: baseURL,
		http: &http.Client{Timeout: timeout},
	}
}

type StatusResponse struct {
	Status string `json:"status"`
}

// Healthz calls GET /healthz and returns the response body.
// It can be used by kden commands to verify the gateway is reachable before
// sending domain requests.
func (c *Client) Healthz(ctx context.Context) (*StatusResponse, error) {
	return c.getJSON(ctx, "/healthz")
}

// Readyz calls GET /readyz and returns the response body.
func (c *Client) Readyz(ctx context.Context) (*StatusResponse, error) {
	return c.getJSON(ctx, "/readyz")
}

func (c *Client) getJSON(ctx context.Context, path string) (*StatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s%s: %w", c.base, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s%s: unexpected status %d", c.base, path, resp.StatusCode)
	}

	var body StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &body, nil
}
