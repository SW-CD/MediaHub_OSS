package infohandler

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"mediahub_oss/internal/httpserver/utils"
	"mediahub_oss/internal/logging/audit"
	"mediahub_oss/internal/media"
)

func NewInfoHandler(
	logger *slog.Logger,
	auditor audit.AuditLogger,
	version string,
	mc media.MediaConverter,
	oidcEnabled bool,
	loginPageDisabled bool,
	oidcIssuerURL string,
	oidcClientID string,
	oidcRedirectURL string,
) *InfoHandler {

	convertTo := make(map[string][]string)
	for _, contentType := range media.GetContentTypes() {
		convertTo[contentType] = mc.GetOutputMimeTypes(contentType)
	}

	handler := &InfoHandler{
		Logger:       logger,
		Auditor:      auditor,
		Version:      version,
		StartTime:    time.Now(),
		ConversionTo: convertTo,
		OIDC: OIDCConfig{
			Enabled:           oidcEnabled,
			LoginPageDisabled: loginPageDisabled,
			IssuerURL:         oidcIssuerURL,
			ClientID:          oidcClientID,
			RedirectURL:       oidcRedirectURL,
		},
	}
	return handler
}

// HealthCheck is a simple public endpoint to confirm the server is running.
func (h *InfoHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// @Summary Get server info
// @Description Retrieves general information about the software, including version, uptime, and media tool availability.
// @Tags info
// @Produce json
// @Success 200 {object} InfoResponse "Returns general backend information"
// @Router /info [get]
func (h *InfoHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	// Calculate the duration since StartTime and round it to the nearest second for a cleaner output
	elapsed := time.Since(h.StartTime).Round(time.Second)

	resp := InfoResponse{
		ServiceName:  "SWCD MediaHub-API",
		Version:      h.Version,
		Uptime:       elapsed.String(), // Returns format like "1h5m30s"
		ConversionTo: h.ConversionTo,
		OIDC:         h.OIDC,
	}

	// h.Auditor.Log(r.Context(), "system.info", "anonymous", "server", nil) // this is public, not audit logging
	utils.RespondWithJSON(w, http.StatusOK, resp)
}
