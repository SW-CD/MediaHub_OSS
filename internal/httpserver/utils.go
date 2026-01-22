package httpserver

import (
	"encoding/json"
	"log"
	"net/http"
)

// errorResponse matches the JSON structure used by the API handlers.
// Defined locally to avoid circular dependencies with the handlers package.
type errorResponse struct {
	Error string `json:"error"`
}

// respondWithError writes a JSON error response to ensure consistency with the API.
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(errorResponse{Error: message}); err != nil {
		log.Printf("ERROR: Failed to encode web error response: %v", err)
	}
}
