package auth

import (
	"context"
	"fmt"
	"log"
	"mediahub_oss/internal/httpserver/utils"
	"mediahub_oss/internal/repository"
	"net/http"
	"strings"
	"time"
)

// ---------------------------------------------------------------------
// Core Middleware Struct
// ---------------------------------------------------------------------

// AuthMiddleware holds dependencies required for authentication/authorization.
type AuthMiddleware struct {
	Repo             repository.Repository
	JWTSecret        []byte
	apiKeyUpdateChan chan APIKeyUpdateRequest // Buffered channel for debouncing and precision timing
}

// APIKeyUpdateRequest holds the exact timestamp the key was used for precise tracking.
type APIKeyUpdateRequest struct {
	KeyID  repository.ULID
	UsedAt time.Time
}

// NewAuthMiddleware creates a new AuthMiddleware service and starts background workers.
func NewAuthMiddleware(repo repository.Repository, secret string) *AuthMiddleware {
	am := &AuthMiddleware{
		Repo:             repo,
		JWTSecret:        []byte(secret),
		apiKeyUpdateChan: make(chan APIKeyUpdateRequest, 5000), // Generous buffer
	}

	// Start the background worker for API key debouncing
	go am.apiKeyUpdateWorker()

	return am
}

// ---------------------------------------------------------------------
// 1. Authentication Middleware
// ---------------------------------------------------------------------

func (am *AuthMiddleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		schema, value, err := am.extractAuthCredentials(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		user, apiKey, err := am.authenticateRequest(schema, value)
		if err != nil {
			log.Printf("Auth failure: %v", err)
			http.Error(w, "Unauthorized: Invalid credentials", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), utils.UserKey, &user)

		isAPIKey := !apiKey.CreatedAt.IsZero()
		if isAPIKey {
			am.asyncUpdateAPIKeyLastUsed(apiKey.ID) // Now uses the worker channel
		}

		ctx = am.cacheUserPermissions(ctx, user, apiKey, isAPIKey)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (am *AuthMiddleware) extractAuthCredentials(r *http.Request) (string, string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("Unauthorized: Invalid Authorization header format")
		}
		return parts[0], parts[1], nil
	}

	queryToken := r.URL.Query().Get("token")
	if queryToken != "" {
		return "Bearer", queryToken, nil
	}

	return "", "", fmt.Errorf("Unauthorized: Missing Authorization header or query token")
}

func (am *AuthMiddleware) authenticateRequest(schema, value string) (repository.User, repository.APIKey, error) {
	switch schema {
	case "Bearer":
		if strings.HasPrefix(value, "srv_") {
			user, apiKey, err := am.validateAPIKey(value) // Assumed implemented
			return user, apiKey, err
		}
		user, err := am.validateJWT(value) // Assumed implemented
		return user, repository.APIKey{}, err
	case "Basic":
		user, err := am.validateBasicAuth(value) // Assumed implemented
		return user, repository.APIKey{}, err
	default:
		return repository.User{}, repository.APIKey{}, fmt.Errorf("Unsupported scheme: %s", schema)
	}
}

// asyncUpdateAPIKeyLastUsed sends the update to a non-blocking channel with a precise timestamp.
func (am *AuthMiddleware) asyncUpdateAPIKeyLastUsed(keyID repository.ULID) {
	select {
	case am.apiKeyUpdateChan <- APIKeyUpdateRequest{KeyID: keyID, UsedAt: time.Now()}:
		// Queued successfully
	default:
		// Channel is full (extreme load), drop this specific timestamp update
		// to prevent the middleware from blocking HTTP requests.
		log.Printf("API key update channel full, dropping update for key %s", keyID)
	}
}

// cacheUserPermissions lazily loads bitmasked permissions into the context.
func (am *AuthMiddleware) cacheUserPermissions(ctx context.Context, user repository.User, apiKey repository.APIKey, isAPIKey bool) context.Context {
	var holder utils.PermissionHolder

	isEffectiveAdmin := user.IsAdmin
	if isAPIKey {
		isEffectiveAdmin = isEffectiveAdmin && apiKey.Scope.HasAccess(repository.AccessAdmin)
	}

	if isEffectiveAdmin {
		holder = &utils.GlobalAdmin{
			UserULID: user.ID,
			Repo:     am.Repo,
		}
		return context.WithValue(ctx, utils.PermissionHolderKey, holder)
	}

	if isAPIKey {
		if user.IsAdmin {
			holder = &utils.APIKeyOfAdmin{
				UserULID: user.ID,
				Scope:    apiKey.Scope,
				Repo:     am.Repo,
			}
		} else {
			holder = &utils.UserPermissions{
				UserULID: user.ID,
				Scope:    apiKey.Scope,
				Repo:     am.Repo,
			}
		}
	} else {
		// Full scope for normal session
		holder = &utils.UserPermissions{
			UserULID: user.ID,
			Scope:    repository.NewAccessGrant(true, true, true, true, true),
			Repo:     am.Repo,
		}
	}

	return context.WithValue(ctx, utils.PermissionHolderKey, holder)
}

// HasPermission performs a check against the cached PermissionHolder.
func (am *AuthMiddleware) HasPermission(ctx context.Context, requiredPerm string, dbID string) bool {
	holder := utils.GetPermissionHolderFromContext(ctx)

	// 1. Global Admins bypass all database permission checks
	if holder.IsGlobalAdmin() {
		return true
	}

	ulidDbID := repository.ULID(dbID)
	switch requiredPerm {
	case "CanView":
		return holder.CanView(ulidDbID)
	case "CanCreate":
		return holder.CanCreate(ulidDbID)
	case "CanEdit":
		return holder.CanEdit(ulidDbID)
	case "CanDelete":
		return holder.CanDelete(ulidDbID)
	case "CanAdmin":
		return holder.CanAdmin(ulidDbID)
	}
	return false
}

// ---------------------------------------------------------------------
// 2. Authorization Middlewares
// ---------------------------------------------------------------------

func (am *AuthMiddleware) RequireGlobalRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if role == "IsAdmin" {
				holder := utils.GetPermissionHolderFromContext(r.Context())
				if !holder.IsGlobalAdmin() {
					http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (am *AuthMiddleware) RequireDatabasePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = utils.GetUserFromContext(r.Context()) // Ensure user is authenticated

			dbID := r.PathValue("database_id")
			if dbID == "" {
				http.Error(w, "Bad Request: Missing database context", http.StatusBadRequest)
				return
			}

			if !am.HasPermission(r.Context(), perm, dbID) {
				http.Error(w, fmt.Sprintf("Forbidden: You lack '%s' rights on database '%s'", perm, dbID), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (am *AuthMiddleware) RequireSelfOrAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := utils.GetUserFromContext(r.Context())
			holder := utils.GetPermissionHolderFromContext(r.Context())

			if holder.IsGlobalAdmin() {
				next.ServeHTTP(w, r)
				return
			}

			userULID := r.PathValue("user_ulid")
			if userULID == "" {
				userULID = r.PathValue("user_id")
			}

			if userULID == "" || repository.ULID(userULID) != user.ID {
				http.Error(w, "Forbidden: You are not authorized to manage this resource", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
