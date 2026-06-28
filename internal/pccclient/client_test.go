package pccclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Matrix030/hackathon-nurse/internal/config"
)

func testClient(baseURL string) *Client {
	cfg := config.Load()
	cfg.BaseURL = baseURL
	cfg.RatePerSecond = 1000 // don't let the limiter slow tests
	cfg.RateBurst = 1000
	cfg.MaxRetries = 5
	return New(cfg)
}

// 429 with Retry-After, then 200 → should retry and succeed.
func TestRetryOn429ThenSuccess(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	if _, err := c.Patients(context.Background(), 101, ""); err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 calls (2 x 429 + 1 ok), got %d", got)
	}
	reqs, retries, fails := c.Stats()
	if retries != 2 || fails != 0 || reqs != 3 {
		t.Fatalf("stats: reqs=%d retries=%d fails=%d, want 3/2/0", reqs, retries, fails)
	}
}

// 5xx is retryable too.
func TestRetryOn5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	if _, err := c.Coverage(context.Background(), "FA-001"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls, got %d", got)
	}
}

// 4xx (other than 429) is NOT retried.
func TestNoRetryOn4xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	if _, err := c.Diagnoses(context.Background(), "FA-001"); err == nil {
		t.Fatal("expected error on 422, got nil")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 call (no retry), got %d", got)
	}
}

// Retries are bounded by MaxRetries, then surface a failure.
func TestRetryExhaustion(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	if _, err := c.Patients(context.Background(), 101, ""); err == nil {
		t.Fatal("expected exhaustion error, got nil")
	}
	// MaxRetries=5 → 1 initial + 5 retries = 6 calls.
	if got := atomic.LoadInt32(&calls); got != 6 {
		t.Fatalf("expected 6 calls (1 + 5 retries), got %d", got)
	}
}
