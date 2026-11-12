package auth

import (
	"net/http"
	"time"

	kdenapi "github.com/konfidence-project/konfidence/internal/kden/apiclient"
)

const (
	conflictMaxAttempts = 3
	conflictBackoff     = 100 * time.Millisecond
)

// retryDoer retries concurrent modifications. Permanent action errors use 422.
type retryDoer struct {
	doer kdenapi.HttpRequestDoer
}

func (d retryDoer) Do(request *http.Request) (*http.Response, error) {
	ctx := request.Context()
	hasBody := request.Body != nil && request.Body != http.NoBody
	for attempt := 1; ; attempt++ {
		// Keep headers isolated from changes made by the HTTP client (e.g. cookies).
		current := request.Clone(ctx)
		if attempt > 1 && hasBody {
			// The first attempt consumes and closes Body. Clone shares that stream,
			// so retries need a fresh body from GetBody.
			body, err := request.GetBody()
			if err != nil {
				return nil, err
			}
			current.Body = body
		}

		response, err := d.doer.Do(current)
		if err != nil || response.StatusCode != http.StatusConflict || attempt >= conflictMaxAttempts ||
			(hasBody && request.GetBody == nil) {
			return response, err
		}

		_ = response.Body.Close()
		timer := time.NewTimer(conflictBackoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
