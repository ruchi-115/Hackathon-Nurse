// Command pipeline runs the wound-care billing data pipeline and/or the REST API.
//
// Usage:
//
//	pipeline ingest   # fetch from the mock PCC API, extract, route → SQLite
//	pipeline serve    # serve the eligibility REST API from SQLite
//	pipeline run      # ingest then serve (default)
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Matrix030/hackathon-nurse/internal/api"
	"github.com/Matrix030/hackathon-nurse/internal/config"
	"github.com/Matrix030/hackathon-nurse/internal/extract"
	"github.com/Matrix030/hackathon-nurse/internal/ingest"
	"github.com/Matrix030/hackathon-nurse/internal/models"
	"github.com/Matrix030/hackathon-nurse/internal/pccclient"
	"github.com/Matrix030/hackathon-nurse/internal/store"
)

func main() {
	cmd := "run"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	since := os.Getenv("SINCE") // optional incremental-sync watermark

	cfg := config.Load()
	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	switch cmd {
	case "ingest":
		if err := runIngest(ctx, cfg, st, since, false); err != nil {
			log.Fatalf("ingest: %v", err)
		}
	case "process":
		// Re-run extraction + routing over already-ingested data (no re-fetch).
		if err := runProcess(ctx, cfg, st); err != nil {
			log.Fatalf("process: %v", err)
		}
	case "serve":
		if err := runServe(ctx, cfg, st); err != nil {
			log.Fatalf("serve: %v", err)
		}
	case "run":
		go func() {
			if err := runIngest(ctx, cfg, st, since, true); err != nil {
				log.Printf("background ingest: %v", err)
			}
		}()
		if err := runServe(ctx, cfg, st); err != nil {
			log.Fatalf("serve: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (want: ingest | serve | run)\n", cmd)
		os.Exit(2)
	}
}

func runIngest(ctx context.Context, cfg config.Config, st *store.Store, since string, progressive bool) error {
	client := pccclient.New(cfg)
	if err := client.Health(ctx); err != nil {
		log.Printf("warning: health check failed: %v", err)
	}

	log.Printf("ingesting (concurrency=%d, rate=%.0f/s)...", cfg.Concurrency, cfg.RatePerSecond)
	in := ingest.New(client, st, cfg)
	var afterPatient ingest.PatientCallback
	if progressive {
		llm := extract.NewLLM(cfg)
		if llm == nil {
			log.Printf("progressive processing enabled; LLM fallback disabled (no ANTHROPIC_API_KEY)")
		} else {
			log.Printf("progressive processing enabled; LLM fallback enabled (model=%s)", cfg.AnthropicModel)
		}
		afterPatient = func(ctx context.Context, p models.Patient) error {
			_, err := ingest.ProcessPatient(ctx, st, llm, p)
			return err
		}
	}
	res, err := in.Run(ctx, since, afterPatient)
	if err != nil {
		return err
	}
	log.Printf("ingest done in %s: %d patients, %d diagnoses, %d coverage, %d notes, %d assessments",
		res.Elapsed.Round(1e8), res.Patients, res.Diagnoses, res.Coverage, res.Notes, res.Assessments)
	log.Printf("API calls: %d requests, %d retries, %d failures (%.0f%% retry rate)",
		res.Requests, res.Retries, res.Failures, retryPct(res.Requests, res.Retries))

	if progressive {
		log.Printf("running final extraction/routing reconciliation over stored data...")
	}
	return runProcess(ctx, cfg, st)
}

// runProcess runs extraction + routing over the stored data and writes
// eligibility rows. Safe to run repeatedly without re-fetching.
func runProcess(ctx context.Context, cfg config.Config, st *store.Store) error {
	llm := extract.NewLLM(cfg) // nil when ANTHROPIC_API_KEY is unset
	if llm == nil {
		log.Printf("LLM fallback disabled (no ANTHROPIC_API_KEY)")
	} else {
		log.Printf("LLM fallback enabled (model=%s)", cfg.AnthropicModel)
	}
	_, err := ingest.Process(ctx, st, llm)
	return err
}

func runServe(ctx context.Context, cfg config.Config, st *store.Store) error {
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: api.NewServer(st, cfg).Router(),
	}
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	log.Printf("serving REST API on http://localhost:%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func retryPct(requests, retries int64) float64 {
	if requests == 0 {
		return 0
	}
	return float64(retries) / float64(requests) * 100
}
