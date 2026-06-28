// Package ingest orchestrates the concurrent fan-out fetch from the mock PCC API
// into SQLite. This is the scalability core: a bounded worker pool, fed by a
// shared rate limiter inside the client, pulls every patient's child records in
// parallel while gracefully absorbing the API's ~30% 429 rate.
package ingest

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Matrix030/hackathon-nurse/internal/config"
	"github.com/Matrix030/hackathon-nurse/internal/models"
	"github.com/Matrix030/hackathon-nurse/internal/pccclient"
	"github.com/Matrix030/hackathon-nurse/internal/store"
	"golang.org/x/sync/errgroup"
)

// Ingestor pulls data from the API into the store.
type Ingestor struct {
	client *pccclient.Client
	store  *store.Store
	cfg    config.Config
}

// New builds an Ingestor.
func New(client *pccclient.Client, st *store.Store, cfg config.Config) *Ingestor {
	return &Ingestor{client: client, store: st, cfg: cfg}
}

// Result summarizes an ingest run.
type Result struct {
	Patients    int
	Diagnoses   int
	Coverage    int
	Notes       int
	Assessments int
	Requests    int64
	Retries     int64
	Failures    int64
	Elapsed     time.Duration
}

// PatientCallback runs after a patient's child records have been attempted.
// It is best-effort: callback failures are logged and do not abort ingestion.
type PatientCallback func(context.Context, models.Patient) error

// Run resolves all patients per facility, then fans out across them to fetch
// diagnoses, coverage, notes, and assessments concurrently. `since` is optional
// (ISO 8601) for incremental sync.
func (in *Ingestor) Run(ctx context.Context, since string, afterPatient PatientCallback) (*Result, error) {
	start := time.Now()
	res := &Result{}

	// Phase 1: resolve all patients (sequential per facility — only 3 calls).
	var patients []models.Patient
	for _, fid := range config.Facilities {
		ps, err := in.client.Patients(ctx, fid, since)
		if err != nil {
			return nil, fmt.Errorf("patients facility %d: %w", fid, err)
		}
		for _, p := range ps {
			if err := in.store.UpsertPatient(p); err != nil {
				return nil, fmt.Errorf("store patient %s: %w", p.PatientID, err)
			}
		}
		patients = append(patients, ps...)
		log.Printf("facility %d: %d patients", fid, len(ps))
	}
	res.Patients = len(patients)

	// Phase 2: bounded-concurrency fan-out over patients. The errgroup limit
	// caps in-flight patients; the client's shared rate limiter caps the actual
	// request rate. Store writes are serialized via the mutex (SQLite single writer).
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(in.cfg.Concurrency)
	var mu sync.Mutex

	for _, p := range patients {
		p := p
		g.Go(func() error {
			counts, err := in.fetchPatient(gctx, p, since, &mu)
			if err != nil {
				// Log and continue: one patient's failure shouldn't abort the run.
				log.Printf("patient %s: %v", p.PatientID, err)
			} else {
				mu.Lock()
				res.Diagnoses += counts.diagnoses
				res.Coverage += counts.coverage
				res.Notes += counts.notes
				res.Assessments += counts.assessments
				mu.Unlock()
			}
			if afterPatient != nil && gctx.Err() == nil {
				if err := afterPatient(gctx, p); err != nil {
					log.Printf("patient %s: process: %v", p.PatientID, err)
				}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	res.Requests, res.Retries, res.Failures = in.client.Stats()
	res.Elapsed = time.Since(start)
	return res, nil
}

type patientCounts struct {
	diagnoses, coverage, notes, assessments int
}

// fetchPatient pulls all four child record sets for one patient and persists them.
// The four endpoints are fetched concurrently within the patient.
func (in *Ingestor) fetchPatient(ctx context.Context, p models.Patient, since string, mu *sync.Mutex) (patientCounts, error) {
	var counts patientCounts
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		ds, err := in.client.Diagnoses(gctx, p.PatientID)
		if err != nil {
			return fmt.Errorf("diagnoses: %w", err)
		}
		mu.Lock()
		defer mu.Unlock()
		for _, d := range ds {
			if err := in.store.UpsertDiagnosis(d); err != nil {
				return err
			}
		}
		counts.diagnoses = len(ds)
		return nil
	})

	g.Go(func() error {
		cs, err := in.client.Coverage(gctx, p.PatientID)
		if err != nil {
			return fmt.Errorf("coverage: %w", err)
		}
		mu.Lock()
		defer mu.Unlock()
		for _, c := range cs {
			if err := in.store.UpsertCoverage(c); err != nil {
				return err
			}
		}
		counts.coverage = len(cs)
		return nil
	})

	g.Go(func() error {
		ns, err := in.client.Notes(gctx, p.ID, since)
		if err != nil {
			return fmt.Errorf("notes: %w", err)
		}
		mu.Lock()
		defer mu.Unlock()
		for _, n := range ns {
			if err := in.store.UpsertNote(n); err != nil {
				return err
			}
		}
		counts.notes = len(ns)
		return nil
	})

	g.Go(func() error {
		as, err := in.client.Assessments(gctx, p.ID, since)
		if err != nil {
			return fmt.Errorf("assessments: %w", err)
		}
		mu.Lock()
		defer mu.Unlock()
		for _, a := range as {
			if err := in.store.UpsertAssessment(a); err != nil {
				return err
			}
		}
		counts.assessments = len(as)
		return nil
	})

	return counts, g.Wait()
}
