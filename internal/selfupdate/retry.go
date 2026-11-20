package selfupdate

import (
	"net/http"
	"time"
)

// retryTransport retries transient GitHub API failures (5xx and 429) with
// bounded exponential backoff. It is used by the go-github production path so
// [network] max_attempts actually takes effect for self-update.
type retryTransport struct {
	base      http.RoundTripper
	attempts  int
	baseDelay time.Duration
	sleep     func(time.Duration)
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	attempts := t.attempts
	if attempts < 1 {
		attempts = 1
	}
	delay := t.baseDelay
	if delay <= 0 {
		delay = 300 * time.Millisecond
	}
	sleep := t.sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			sleep(delay)
			if delay < 4*time.Second {
				delay *= 2
			}
		}
		resp, err := base.RoundTrip(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = &transientStatusError{status: resp.StatusCode}
			_ = resp.Body.Close()
			continue
		}
		return resp, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, &transientStatusError{status: http.StatusServiceUnavailable}
}

type transientStatusError struct{ status int }

func (e *transientStatusError) Error() string {
	return "github api transient status " + http.StatusText(e.status)
}

func withRetry(client *http.Client, attempts int) *http.Client {
	if attempts <= 1 || client == nil {
		return client
	}
	clone := *client
	clone.Transport = &retryTransport{base: client.Transport, attempts: attempts}
	return &clone
}
