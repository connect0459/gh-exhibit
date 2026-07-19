package github

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

const (
	maxRetryAttempts = 3
	fixedBackoffBase = 1 * time.Second
)

// requester is the subset of *api.RESTClient's interface doWithRetry needs;
// *api.RESTClient satisfies it directly, and tests substitute a fake.
type requester interface {
	RequestWithContext(ctx context.Context, method, path string, body io.Reader) (*http.Response, error)
}

// sleeper waits d, honoring ctx cancellation, so a caller can interrupt an
// in-progress rate-limit wait (X-RateLimit-Reset can be up to an hour out)
// instead of blocking uninterruptibly. realSleep (evidence_fetcher.go) is
// the production implementation; tests substitute a spy that returns
// immediately.
type sleeper func(ctx context.Context, d time.Duration) error

// doWithRetry issues method/path via req, retrying rate-limit responses
// (403 identified as rate-limited, or 429) up to maxRetryAttempts times,
// waiting between attempts as decided by retryDelay.
func doWithRetry(ctx context.Context, req requester, sleep sleeper, method, path string) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetryAttempts; attempt++ {
		resp, err := req.RequestWithContext(ctx, method, path, nil)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		wait, retryable := retryDelay(err, attempt)
		if !retryable || attempt == maxRetryAttempts-1 {
			break
		}
		if sleepErr := sleep(ctx, wait); sleepErr != nil {
			return nil, sleepErr
		}
	}

	return nil, lastErr
}

// retryDelay decides whether err (from a REST call) warrants a retry and,
// if so, how long to wait first. 429 is always retried. 403 is retried only
// when its headers identify it as a rate limit (Retry-After present, or
// X-RateLimit-Remaining: 0) rather than a permission error, since GitHub
// returns 403 for both and only the headers distinguish them. Any other
// status, or a non-HTTP error (network failure), is not retried.
func retryDelay(err error, attempt int) (time.Duration, bool) {
	var httpErr *api.HTTPError
	if !errors.As(err, &httpErr) {
		return 0, false
	}

	switch {
	case httpErr.StatusCode == http.StatusTooManyRequests:
	case httpErr.StatusCode == http.StatusForbidden && isRateLimitResponse(httpErr.Headers):
	default:
		return 0, false
	}

	if d, ok := retryAfterDelay(httpErr.Headers); ok {
		return d, true
	}
	if d, ok := rateLimitResetDelay(httpErr.Headers); ok {
		return d, true
	}

	return fixedBackoffBase * time.Duration(1<<attempt), true
}

func isRateLimitResponse(h http.Header) bool {
	return h.Get("Retry-After") != "" || h.Get("X-RateLimit-Remaining") == "0"
}

func retryAfterDelay(h http.Header) (time.Duration, bool) {
	raw := h.Get("Retry-After")
	if raw == "" {
		return 0, false
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}

func rateLimitResetDelay(h http.Header) (time.Duration, bool) {
	raw := h.Get("X-RateLimit-Reset")
	if raw == "" {
		return 0, false
	}
	epoch, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	wait := time.Until(time.Unix(epoch, 0))
	if wait < 0 {
		wait = 0
	}
	return wait, true
}
