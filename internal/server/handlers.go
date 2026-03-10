package server

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/mrowling/carelink-go/internal/logger"
)

type LatestResponse struct {
	Trend      *int    `json:"trend,omitempty"`
	Direction  string  `json:"direction,omitempty"`
	SGVMmol    float64 `json:"sgv_mmol"`
	AgeMinutes int     `json:"age_minutes"`
}

type HealthResponse struct {
	Status              string `json:"status"`
	LastFetch           string `json:"last_fetch,omitempty"`
	LastFetchAgeMinutes int    `json:"last_fetch_age_minutes,omitempty"`
	DatabaseEntries     int    `json:"database_entries,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entries, err := s.db.GetRecentGlucoseEntries(1)
	if err != nil {
		logger.Error("Server", "Failed to query database: %v", err)
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to query database"})
		return
	}

	if len(entries) == 0 {
		respondJSON(w, http.StatusNotFound, ErrorResponse{Error: "No glucose entries found in database"})
		return
	}

	entry := entries[0]
	entryTime := time.UnixMilli(entry.Date)
	ageMinutes := int(math.Round(time.Since(entryTime).Minutes()))

	directionToTrend := map[string]int{
		"NONE":          0,
		"TripleUp":      1,
		"DoubleUp":      1,
		"SingleUp":      2,
		"FortyFiveUp":   3,
		"Flat":          4,
		"FortyFiveDown": 5,
		"SingleDown":    6,
		"DoubleDown":    7,
		"TripleDown":    7,
	}

	response := LatestResponse{
		SGVMmol:    entry.SGVMmol,
		AgeMinutes: ageMinutes,
	}

	if entry.Direction != "" {
		response.Direction = entry.Direction
		if trend, ok := directionToTrend[entry.Direction]; ok {
			response.Trend = &trend
		}
	}

	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get configurable staleness threshold (default 15 minutes)
	staleThreshold := 15
	if envVal := os.Getenv("CARELINK_HEALTH_CHECK_STALE_MINS"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil {
			staleThreshold = val
		}
	}

	response := HealthResponse{
		Status: "ok",
	}

	// Check last fetch time
	if !s.lastFetch.IsZero() {
		ageMinutes := int(time.Since(s.lastFetch).Minutes())
		response.LastFetch = s.lastFetch.Format(time.RFC3339)
		response.LastFetchAgeMinutes = ageMinutes

		if ageMinutes > staleThreshold {
			response.Status = "stale"
			respondJSON(w, http.StatusServiceUnavailable, response)
			return
		}
	}

	// Check database has entries
	entries, err := s.db.GetRecentGlucoseEntries(1)
	if err == nil && len(entries) > 0 {
		response.DatabaseEntries = len(entries)
	}

	respondJSON(w, http.StatusOK, response)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("handlers", "Failed to encode JSON response: %v", err)
	}
}
