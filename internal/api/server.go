// Package api exposes the eligibility data as a JSON REST API for the frontend.
package api

import (
	"net/http"
	"time"

	"github.com/Matrix030/hackathon-nurse/internal/config"
	"github.com/Matrix030/hackathon-nurse/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server holds API dependencies.
type Server struct {
	store *store.Store
	cfg   config.Config
}

// NewServer builds the API server.
func NewServer(st *store.Store, cfg config.Config) *Server {
	return &Server{store: st, cfg: cfg}
}

// Router builds the chi router with middleware and routes.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{s.cfg.CORSOrigin},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		MaxAge:           300,
	}))

	r.Get("/health", s.handleHealth)
	r.Get("/eligibility", s.handleEligibility)
	r.Get("/patients/{patientID}", s.handlePatient)
	r.Get("/stats", s.handleStats)
	return r
}
