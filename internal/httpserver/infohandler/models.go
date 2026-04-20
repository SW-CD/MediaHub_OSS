package infohandler

import (
	"log/slog"
	"time"

	"mediahub_oss/internal/logging/audit"
)

// OIDCConfig represents the nested OIDC settings in the InfoResponse.
type OIDCConfig struct {
	Enabled           bool   `json:"enabled"`
	LoginPageDisabled bool   `json:"login_page_disabled"`
	IssuerURL         string `json:"oidc_issuer_url"`
	ClientID          string `json:"oidc_client_id"`
	RedirectURL       string `json:"oidc_redirect_url"`
}

type InfoHandler struct {
	Logger       *slog.Logger
	Auditor      audit.AuditLogger
	Version      string
	StartTime    time.Time
	ConversionTo map[string][]string
	OIDC         OIDCConfig
}

// InfoResponse defines the JSON structure for the /api/info endpoint.
type InfoResponse struct {
	ServiceName  string              `json:"service_name"`
	Version      string              `json:"version"`
	Uptime       string              `json:"uptime"` // Changed to reflect elapsed duration
	ConversionTo map[string][]string `json:"conversion_to"`
	OIDC         OIDCConfig          `json:"oidc"`
}
