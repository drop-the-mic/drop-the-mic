package handler

import (
	"encoding/json"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Handler provides HTTP handlers for the DTM API.
type Handler struct {
	client client.Client
}

// New creates a new Handler.
func New(c client.Client) *Handler {
	return &Handler{client: c}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Log only — headers are already sent, so we can't change the status code.
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
