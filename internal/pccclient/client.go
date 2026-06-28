// Package pccclient is a typed, rate-limited, retry-aware HTTP client for the
// mock PointClickCare API. Every request flows through a shared rate limiter and
// the retry/backoff logic in retry.go, which is what lets the pipeline complete
// ~1,500 calls against an API that returns 429 on ~30% of requests.
package pccclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"

	"github.com/Matrix030/hackathon-nurse/internal/config"
	"github.com/Matrix030/hackathon-nurse/internal/models"
	"golang.org/x/time/rate"
)

// Client talks to the mock PCC API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	limiter    *rate.Limiter
	maxRetries int
	stats      retryStats
}

// New builds a Client from config. The rate limiter and HTTP client are shared
// across all goroutines that use this Client.
func New(cfg config.Config) *Client {
	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: &http.Client{Timeout: cfg.HTTPTimeout},
		limiter:    rate.NewLimiter(rate.Limit(cfg.RatePerSecond), cfg.RateBurst),
		maxRetries: cfg.MaxRetries,
	}
}

func (s *retryStats) add(field *int64, n int64) { atomic.AddInt64(field, n) }

// Stats returns a snapshot of request/retry/failure counts.
func (c *Client) Stats() (requests, retries, failures int64) {
	return atomic.LoadInt64(&c.stats.Requests),
		atomic.LoadInt64(&c.stats.Retries),
		atomic.LoadInt64(&c.stats.Failures)
}

// getJSON fetches path?query and decodes the JSON array body into out.
func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	resp, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		return req, nil
	})
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// Health checks the service is up.
func (c *Client) Health(ctx context.Context) error {
	resp, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Patients fetches all patients for a facility, optionally modified since a timestamp.
func (c *Client) Patients(ctx context.Context, facilityID int, since string) ([]models.Patient, error) {
	q := url.Values{"facility_id": {strconv.Itoa(facilityID)}}
	if since != "" {
		q.Set("since", since)
	}
	var out []models.Patient
	return out, c.getJSON(ctx, "/pcc/patients", q, &out)
}

// Diagnoses fetches ICD-10 diagnoses for a patient (string patient_id).
func (c *Client) Diagnoses(ctx context.Context, patientID string) ([]models.Diagnosis, error) {
	var out []models.Diagnosis
	return out, c.getJSON(ctx, "/pcc/diagnoses", url.Values{"patient_id": {patientID}}, &out)
}

// Coverage fetches insurance coverage for a patient (string patient_id).
func (c *Client) Coverage(ctx context.Context, patientID string) ([]models.Coverage, error) {
	var out []models.Coverage
	return out, c.getJSON(ctx, "/pcc/coverage", url.Values{"patient_id": {patientID}}, &out)
}

// Notes fetches current progress notes for a patient (integer internal id).
func (c *Client) Notes(ctx context.Context, internalID int, since string) ([]models.Note, error) {
	q := url.Values{"patient_id": {strconv.Itoa(internalID)}}
	if since != "" {
		q.Set("since", since)
	}
	var out []models.Note
	return out, c.getJSON(ctx, "/pcc/notes", q, &out)
}

// Assessments fetches current wound assessments for a patient (integer internal id).
func (c *Client) Assessments(ctx context.Context, internalID int, since string) ([]models.Assessment, error) {
	q := url.Values{"patient_id": {strconv.Itoa(internalID)}}
	if since != "" {
		q.Set("since", since)
	}
	var out []models.Assessment
	return out, c.getJSON(ctx, "/pcc/assessments", q, &out)
}
