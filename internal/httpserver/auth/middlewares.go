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

// AuthMiddleware holds dependencies required for authentication/authorization.
type AuthMiddleware struct {
	Repo      repository.Repository
	JWTSecret []byte
}

// NewAuthMiddleware creates a new AuthMiddleware service.
func NewAuthMiddleware(repo repository.Repository, secret string) *AuthMiddleware {
	return &AuthMiddleware{
		Repo:      repo,
		JWTSecret: []byte(secret),
	}
}

// ---------------------------------------------------------------------
// 1. Authentication Middleware
// ---------------------------------------------------------------------

// AuthMiddleware is the main entry point. It checks for a Bearer Token (JWT)
// or Basic Auth credentials. If valid, it adds the User object to the request Context.
// In addition, if it detects an API key (starting with `srv_`), it performs key validation,
// and it pre-loads/caches the user's database permissions in the context.
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

		// Build context with user and optional API Key details
		isEffectiveAdmin := user.IsAdmin
		if !apiKey.CreatedAt.IsZero() {
			isEffectiveAdmin = isEffectiveAdmin && apiKey.ScopeAdmin
		}

		ctx := context.WithValue(r.Context(), utils.UserKey, &user)
		ctx = context.WithValue(ctx, utils.IsAdminKey, isEffectiveAdmin)

		if !apiKey.CreatedAt.IsZero() {
			ctx = context.WithValue(ctx, utils.APIKeyKey, &apiKey)
			am.asyncUpdateAPIKeyLastUsed(apiKey.ID)
		}

		// Cache user database permissions in request context
		ctx, err = am.cacheUserPermissions(ctx, user, apiKey)
		if err != nil {
			log.Printf("Failed to cache user permissions: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractAuthCredentials retrieves the schema and credential value from headers or query parameters.
func (am *AuthMiddleware) extractAuthCredentials(r *http.Request) (string, string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("Unauthorized: Invalid Authorization header format")
		}
		return parts[0], parts[1], nil
	}

	// Fallback to query parameter token (used for streaming media)
	queryToken := r.URL.Query().Get("token")
	if queryToken != "" {
		return "Bearer", queryToken, nil
	}

	return "", "", fmt.Errorf("Unauthorized: Missing Authorization header or query token")
}

// authenticateRequest authenticates the client using the appropriate credentials schema.
func (am *AuthMiddleware) authenticateRequest(schema, value string) (repository.User, repository.APIKey, error) {
	switch schema {
	case "Bearer":
		if strings.HasPrefix(value, "srv_") {
			user, apiKey, err := am.validateAPIKey(value)
			if err != nil {
				return repository.User{}, repository.APIKey{}, err
			}
			return user, apiKey, nil
		}
		user, err := am.validateJWT(value)
		return user, repository.APIKey{}, err
	case "Basic":
		user, err := am.validateBasicAuth(value)
		return user, repository.APIKey{}, err
	default:
		return repository.User{}, repository.APIKey{}, fmt.Errorf("Unsupported authentication scheme: %s", schema)
	}
}

// asyncUpdateAPIKeyLastUsed triggers a background update of the last_used_at timestamp.
func (am *AuthMiddleware) asyncUpdateAPIKeyLastUsed(keyID repository.ULID) {
	go func() {
		// Use Background context to prevent cancellation if client request ends early
		if updateErr := am.Repo.UpdateAPIKeyLastUsed(context.Background(), keyID, time.Now()); updateErr != nil {
			log.Printf("Failed to update last_used_at for api_key %s: %v", keyID, updateErr)
		}
	}()
}

// cacheUserPermissions pre-loads and caches user database permissions (intersected with API key scopes) in request context.
func (am *AuthMiddleware) cacheUserPermissions(ctx context.Context, user repository.User, apiKey repository.APIKey) (context.Context, error) {
	if utils.IsAdminFromContext(ctx) {
		return ctx, nil
	}

	perms, err := am.Repo.GetAllUserPermissions(ctx, user.ID)
	if err != nil {
		return ctx, err
	}

	permsMap := make(map[repository.ULID]repository.UserPermissions, len(perms))
	for _, p := range perms {
		if !apiKey.CreatedAt.IsZero() {
			p.Roles = intersectRolesWithScopes(p.Roles, apiKey)
		}
		permsMap[p.DatabaseID] = p
	}

	return context.WithValue(ctx, utils.PermissionsKey, permsMap), nil
}

// intersectRolesWithScopes filters user roles against the scopes of the active API key.
func intersectRolesWithScopes(roles string, apiKey repository.APIKey) string {
	var activeRoles []string
	if strings.Contains(roles, "CanView") && apiKey.ScopeView {
		activeRoles = append(activeRoles, "CanView")
	}
	if strings.Contains(roles, "CanCreate") && apiKey.ScopeCreate {
		activeRoles = append(activeRoles, "CanCreate")
	}
	if strings.Contains(roles, "CanEdit") && apiKey.ScopeEdit {
		activeRoles = append(activeRoles, "CanEdit")
	}
	if strings.Contains(roles, "CanDelete") && apiKey.ScopeDelete {
		activeRoles = append(activeRoles, "CanDelete")
	}
	return strings.Join(activeRoles, ",")
}

// ---------------------------------------------------------------------
// 2. Authorization Middlewares
// ---------------------------------------------------------------------

// RequireGlobalRole returns a middleware that checks if the authenticated user
// has a specific global flag (currently only "IsAdmin").
func (am *AuthMiddleware) RequireGlobalRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Admin check
			if role == "IsAdmin" {
				if !utils.IsAdminFromContext(r.Context()) {
					http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireDatabasePermission returns a middleware that checks if the user has
// a specific permission (e.g., "CanView", "CanCreate") for the database
// specified in the URL path parameter "{database_id}".
func (am *AuthMiddleware) RequireDatabasePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := utils.GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Extract database ID from the URL path (Go 1.22 feature)
			dbID := r.PathValue("database_id")
			if dbID == "" {
				http.Error(w, "Bad Request: Missing database context", http.StatusBadRequest)
				return
			}

			// Check specific permission (using HasPermission to handle intersection logic and cached permissions)
			if !am.HasPermission(r.Context(), perm, dbID) {
				http.Error(w, fmt.Sprintf("Forbidden: You lack '%s' rights on database '%s'", perm, dbID), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireSelfOrAdmin returns a middleware that checks if the authenticated user
// is either a global admin OR if their user ID matches the "{user_ulid}" path parameter.
func (am *AuthMiddleware) RequireSelfOrAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := utils.GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if utils.IsAdminFromContext(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}

			// Not admin, verify owner ID matches URL path parameter {user_ulid}
			userULID := r.PathValue("user_ulid")
			if userULID == "" {
				userULID = r.PathValue("user_id") // Fallback
			}

			if userULID == "" || repository.ULID(userULID) != user.ID {
				http.Error(w, "Forbidden: You are not authorized to manage this resource", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
