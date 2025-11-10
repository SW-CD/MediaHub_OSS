// internal/api/handlers/responses.go
package handlers

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is a standard format for API error messages.
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse is a standard format for simple API messages.
type MessageResponse struct {
	Message string `json:"message"`
}

// respondWithError sends a JSON error response.
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, ErrorResponse{Error: message})
}

// respondWithJSON sends a JSON response.
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, `{"error":"Failed to marshal JSON response"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
