package handlers

import (
	"fmt"
	"net/http"
)

// HealthCheck is a simple public endpoint to confirm the server is running.
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}
