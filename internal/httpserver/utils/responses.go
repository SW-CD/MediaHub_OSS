package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse matches the JSON structure used by the API handlers.
// Defined locally to avoid circular dependencies with the handlers package.
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse is a standard format for simple API messages.
type MessageResponse struct {
	Message string `json:"message"`
}

// respondWithError writes a JSON error response to ensure consistency with the API.
func RespondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Error: message}); err != nil {
		log.Printf("ERROR: Failed to encode web error response: %v", err)
	}
}

// respondWithJSON writes a JSON response to ensure consistency with the API.
func RespondWithJSON(w http.ResponseWriter, code int, payload any) {
	response, err := json.Marshal(payload)
	if err != nil {
		// Fallback to manual JSON string if marshaling fails, keeping format consistent
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Failed to marshal JSON response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
