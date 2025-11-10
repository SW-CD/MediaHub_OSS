// filepath: internal/api/handlers/info_handler.go
package handlers

import (
	"net/http"
)

// @Summary Get service information
// @Description Retrieves general information about the software, i.e., the service name, software version, uptime, and ffmpeg availability. This is a public endpoint.
// @Tags Info
// @Produce  json
// @Success 200 {object} models.Info
// @Router /info [get]
func (h *Handlers) GetInfo(w http.ResponseWriter, r *http.Request) {
	// --- Call InfoService ---
	info := h.Info.GetInfo()
	respondWithJSON(w, http.StatusOK, info)
}
