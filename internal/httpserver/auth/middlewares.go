package auth

import (
	"context"
	"fmt"
	"log"
	"mediahub_oss/internal/httpserver/utils"
	"mediahub_oss/internal/repository"
	"net/http"
	"strings"
)

// AuthMiddleware holds dependencies required for authentication/authorization.
type AuthMiddleware struct {
	Repo      repository.Repository
	JWTSecret []byte
}

// New creates a new AuthMiddleware service.
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
func (am *AuthMiddleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var schema, value string

		// Try to extract the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 {
				http.Error(w, "Unauthorized: Invalid Authorization header format", http.StatusUnauthorized)
				return
			}
			schema = parts[0]
			value = parts[1]
		} else {
			// Fallback: Try to extract from the URL query parameter (for native media streaming)
			queryToken := r.URL.Query().Get("token")
			if queryToken != "" {
				schema = "Bearer" // Implicitly treat query tokens as JWT Bearer tokens
				value = queryToken
			}
		}

		// 3. If both were empty, reject the request
		if schema == "" || value == "" {
			http.Error(w, "Unauthorized: Missing Authorization header or query token", http.StatusUnauthorized)
			return
		}

		var user repository.User
		var err error

		// 4. Proceed with validation exactly as before
		switch schema {
		case "Bearer":
			user, err = am.validateJWT(value)
		case "Basic":
			user, err = am.validateBasicAuth(value)
		default:
			http.Error(w, "Unauthorized: Unsupported authentication scheme", http.StatusUnauthorized)
			return
		}

		if err != nil {
			// Log the error for debugging but don't leak details to the client
			log.Printf("Auth failure: %v", err)
			http.Error(w, "Unauthorized: Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Success! Store user in context and proceed
		ctx := context.WithValue(r.Context(), utils.UserKey, &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ---------------------------------------------------------------------
// 2. Authorization Middlewares
// ---------------------------------------------------------------------

// RequireGlobalRole returns a middleware that checks if the authenticated user
// has a specific global flag (currently only "IsAdmin").
func (am *AuthMiddleware) RequireGlobalRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := utils.GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Admin check
			if role == "IsAdmin" && !user.IsAdmin {
				http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
				return
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

			// 1. Admins bypass all database permission checks
			if user.IsAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// 2. Extract database ID from the URL path (Go 1.22 feature)
			// This expects the route to be registered like "/api/database/{database_id}/..."
			dbID := r.PathValue("database_id")
			if dbID == "" {
				http.Error(w, "Bad Request: Missing database context", http.StatusBadRequest)
				return
			}

			// 3. Check specific permission
			if !am.hasPerm(user, dbID, perm) {
				http.Error(w, fmt.Sprintf("Forbidden: You lack '%s' rights on database '%s'", perm, dbID), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
