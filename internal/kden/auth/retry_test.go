package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryDoerResponses(t *testing.T) {
	for _, test := range []struct {
		name     string
		statuses []int
	}{
		{"conflict then success", []int{http.StatusConflict, http.StatusNoContent}},
		{"permanent error", []int{http.StatusUnprocessableEntity}},
		{"conflict then permanent error", []int{http.StatusConflict, http.StatusUnprocessableEntity}},
		{"exhausted retries", []int{http.StatusConflict, http.StatusConflict, http.StatusConflict}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempt := int(attempts.Add(1))
				status := test.statuses[min(attempt-1, len(test.statuses)-1)]
				w.WriteHeader(status)
				if status != http.StatusNoContent {
					_, _ = io.WriteString(w, "server message")
				}
			}))
			defer server.Close()
			request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := (retryDoer{doer: server.Client()}).Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if int(attempts.Load()) != len(test.statuses) || response.StatusCode != test.statuses[len(test.statuses)-1] {
				t.Fatalf("got %d attempts and HTTP %d", attempts.Load(), response.StatusCode)
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusNoContent && string(body) != "server message" {
				t.Fatalf("final response body was lost: %q", body)
			}
		})
	}
}

func TestClientRetriesPromotionActions(t *testing.T) {
	for _, action := range []string{"approve", "revoke-approval"} {
		t.Run(action, func(t *testing.T) {
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempt := attempts.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/projects/project/vectorPromotions/promotion/"+action ||
					r.Header.Get("Authorization") != "Bearer token" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if attempt == 1 {
					w.WriteHeader(http.StatusConflict)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			client, err := NewClient(server.URL+"/api", "token", nil, time.Second, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			var status int
			if action == "approve" {
				response, callErr := client.ApproveVectorPromotionV1WithResponse(context.Background(), "project", "promotion")
				err = callErr
				if response != nil {
					status = response.StatusCode()
				}
			} else {
				response, callErr := client.RevokeApprovalVectorPromotionV1WithResponse(context.Background(), "project", "promotion")
				err = callErr
				if response != nil {
					status = response.StatusCode()
				}
			}
			if err != nil || status != http.StatusNoContent || attempts.Load() != 2 {
				t.Fatalf("got HTTP %d, %d attempts, error %v", status, attempts.Load(), err)
			}
		})
	}
}

type retryTestDoer func(*http.Request) (*http.Response, error)

func (d retryTestDoer) Do(request *http.Request) (*http.Response, error) { return d(request) }

type retryTestBody struct {
	io.Reader
	closed bool
}

func (b *retryTestBody) Close() error { b.closed = true; return nil }

func TestRetryDoerClosesOnlyDiscardedResponses(t *testing.T) {
	var bodies []*retryTestBody
	doer := retryDoer{doer: retryTestDoer(func(*http.Request) (*http.Response, error) {
		if len(bodies) > 0 && !bodies[len(bodies)-1].closed {
			t.Error("previous response was not closed before retrying")
		}
		body := &retryTestBody{Reader: strings.NewReader("conflict")}
		bodies = append(bodies, body)
		return &http.Response{StatusCode: http.StatusConflict, Body: body}, nil
	})}
	request, _ := http.NewRequest(http.MethodPost, "http://example.test", nil)
	response, err := doer.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if len(bodies) != 3 || !bodies[0].closed || !bodies[1].closed || bodies[2].closed {
		t.Fatal("discarded responses must be closed and the final response left open")
	}
}

func TestRetryDoerStopsOnErrors(t *testing.T) {
	transportErr := errors.New("transport failed")
	for _, test := range []struct {
		name string
		err  error
		doer retryTestDoer
	}{
		{"transport error", transportErr, func(*http.Request) (*http.Response, error) { return nil, transportErr }},
		{"session unauthorized", ErrUnauthorized, func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusUnauthorized, Body: http.NoBody}, nil
		}},
		{"token rejected", ErrAccessTokenRejected, func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusUnauthorized, Body: http.NoBody}, nil
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			attempts := 0
			doer := retryDoer{doer: unauthorizedDoer{
				doer: retryTestDoer(func(r *http.Request) (*http.Response, error) {
					attempts++
					return test.doer(r)
				}),
				usesAccessToken: test.err == ErrAccessTokenRejected,
			}}
			request, _ := http.NewRequest(http.MethodPost, "http://example.test", nil)
			response, err := doer.Do(request)
			if response != nil || !errors.Is(err, test.err) || attempts != 1 {
				t.Fatalf("got %d attempts, response %v, error %v", attempts, response, err)
			}
		})
	}
}

func TestRetryDoerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	attempts := 0
	body := &retryTestBody{Reader: strings.NewReader("conflict")}
	doer := retryDoer{doer: retryTestDoer(func(*http.Request) (*http.Response, error) {
		attempts++
		cancel()
		return &http.Response{StatusCode: http.StatusConflict, Body: body}, nil
	})}
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://example.test", nil)
	response, err := doer.Do(request)
	if response != nil || !errors.Is(err, context.Canceled) || attempts != 1 || !body.closed {
		t.Fatalf("got %d attempts, closed=%t, response %v, error %v", attempts, body.closed, response, err)
	}
}

func TestRetryDoerRequestBodies(t *testing.T) {
	for _, replayable := range []bool{true, false} {
		t.Run(map[bool]string{true: "replayable", false: "stream"}[replayable], func(t *testing.T) {
			attempts := 0
			doer := retryDoer{doer: retryTestDoer(func(request *http.Request) (*http.Response, error) {
				attempts++
				body, err := io.ReadAll(request.Body)
				_ = request.Body.Close()
				if err != nil || string(body) != "payload" {
					t.Fatalf("request body was not replayed: %q, %v", body, err)
				}
				return &http.Response{StatusCode: http.StatusConflict, Body: http.NoBody}, nil
			})}
			var body io.Reader = strings.NewReader("payload")
			if !replayable {
				body = io.NopCloser(body)
			}
			request, _ := http.NewRequest(http.MethodPost, "http://example.test", body)
			response, err := doer.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			expected := 1
			if replayable {
				expected = 3
			}
			if attempts != expected {
				t.Fatalf("got %d attempts, want %d", attempts, expected)
			}
		})
	}
}
