package pccclient

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// retryStats tracks retry activity so the ingest layer can report how hard the
// rate limiter is working (the headline scalability metric).
type retryStats struct {
	Requests int64
	Retries  int64
	Failures int64
}

// doWithRetry issues req (regenerated each attempt via build) through the shared
// rate limiter, retrying on 429 and 5xx with exponential backoff + jitter,
// honoring the Retry-After header. 4xx other than 429 are returned immediately.
// The caller is responsible for closing the returned response body.
func (c *Client) doWithRetry(ctx context.Context, build func() (*http.Request, error)) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Wait for a rate-limiter token before every attempt so we never
		// exceed the configured request rate, even across many workers.
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		req, err := build()
		if err != nil {
			return nil, err
		}
		c.stats.add(&c.stats.Requests, 1)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Transport error (timeout, connection reset) — retryable.
			lastErr = err
			if !c.sleepBackoff(ctx, attempt, 0) {
				break
			}
			c.stats.add(&c.stats.Retries, 1)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		// Non-200. Decide whether to retry.
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		status := resp.StatusCode
		resp.Body.Close()

		if status == http.StatusTooManyRequests || status >= 500 {
			lastErr = fmt.Errorf("status %d", status)
			if !c.sleepBackoff(ctx, attempt, retryAfter) {
				break
			}
			c.stats.add(&c.stats.Retries, 1)
			continue
		}

		// 4xx (other than 429) — not retryable.
		c.stats.add(&c.stats.Failures, 1)
		return nil, fmt.Errorf("non-retryable status %d", status)
	}

	c.stats.add(&c.stats.Failures, 1)
	return nil, fmt.Errorf("exhausted retries: %w", lastErr)
}

// sleepBackoff sleeps for max(retryAfter, exponential backoff + jitter) before
// the next attempt. It returns false if there are no attempts remaining or the
// context is cancelled.
func (c *Client) sleepBackoff(ctx context.Context, attempt int, retryAfter time.Duration) bool {
	if attempt >= c.maxRetries {
		return false
	}
	// Exponential backoff: base * 2^attempt, capped, plus jitter.
	base := 250 * time.Millisecond
	backoff := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
	if backoff > 8*time.Second {
		backoff = 8 * time.Second
	}
	jitter := time.Duration(rand.Int63n(int64(250 * time.Millisecond)))
	wait := backoff + jitter
	if retryAfter > wait {
		wait = retryAfter
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// parseRetryAfter parses the Retry-After header (delay in seconds per the spec).
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}
