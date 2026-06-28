package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Matrix030/hackathon-nurse/internal/models"
	"github.com/Matrix030/hackathon-nurse/internal/store"
	"github.com/go-chi/chi/v5"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleEligibility lists eligibility rows with optional filters:
//
//	?facility_id=101  ?decision=auto_accept  ?mcb=true  ?limit=50  ?offset=0
func (s *Server) handleEligibility(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.EligibilityFilter{
		Decision: q.Get("decision"),
		MCBOnly:  q.Get("mcb") == "true",
	}
	if v := q.Get("facility_id"); v != "" {
		f.FacilityID, _ = strconv.Atoi(v)
	}
	if v := q.Get("limit"); v != "" {
		f.Limit, _ = strconv.Atoi(v)
	}
	if v := q.Get("offset"); v != "" {
		f.Offset, _ = strconv.Atoi(v)
	}

	rows, err := s.store.ListEligibility(f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rows == nil {
		rows = []models.EligibilityRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(rows), "results": rows})
}

// handlePatient returns the full drill-down record for one external patient id.
func (s *Server) handlePatient(w http.ResponseWriter, r *http.Request) {
	patientID := chi.URLParam(r, "patientID")
	detail, err := s.store.GetPatientDetail(patientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if detail == nil {
		writeError(w, http.StatusNotFound, "patient not found")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.Stats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}
