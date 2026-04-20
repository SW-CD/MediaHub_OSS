package tokenhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mediahub_oss/internal/repository"
	"mediahub_oss/internal/shared/customerrors"
	"net/http"
)

type OidcTokenRequest struct {
	IdpToken string `json:"idp_token"`
	// TODO add access_token and check for access role?
}

// handleOIDCValidationAndProvisioning validates the external token and returns the internal User ID.
func (h *TokenHandler) handleOIDCValidationAndProvisioning(ctx context.Context, idpToken string) (repository.User, error) {
	// 1. Verify the signature and claims of the external idpToken against Keycloak.
	// 2. Extract the username or email from the token claims.
	// 3. Look up the user in h.Repo using the extracted username.
	// 4. If the user doesn't exist, create a new internal user record assigning the 'default_user_rights' from config.
	// 5. Return the internal user

	return repository.User{}, customerrors.ErrNotImplemented
}

func checkOIDC(r *http.Request) (OidcTokenRequest, bool) {

	var hasSsoAuth bool = false
	var ssoReq OidcTokenRequest

	if r.Header.Get("Content-Type") == "application/json" {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			// Restore the body so it can be read again if needed
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			if err := json.Unmarshal(bodyBytes, &ssoReq); err == nil && ssoReq.IdpToken != "" {
				hasSsoAuth = true
			}
		}
	}
	return ssoReq, hasSsoAuth
}
