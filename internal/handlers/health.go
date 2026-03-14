package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// BuildInfo holds version metadata.
type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// NewHealthHandler returns a handler for GET /health and /healthz.
func NewHealthHandler(info BuildInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"time":    time.Now().UTC().Format(time.RFC3339),
			"version": info.Version,
			"commit":  info.Commit,
			"date":    info.Date,
		})
	}
}
