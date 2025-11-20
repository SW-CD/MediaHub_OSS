// filepath: internal/services/auth/middleware.go
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"mediahub/internal/services"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// writeError sends a JSON error response.
func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Middleware provides authentication and authorization middleware.
type Middleware struct {
	User  services.UserService
	Token TokenService
}

// NewMiddleware creates a new instance of Middleware.
func NewMiddleware(user services.UserService, token TokenService) *Middleware { // <-- UPDATED
	return &Middleware{
		User:  user,
		Token: token,
	}
}

// AuthMiddleware is a middleware function that checks for a valid JWT Bearer token OR Basic Auth.
func (m *Middleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Tell the client we accept both
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", Bearer realm="restricted"`)
			writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		var user *models.User
		var err error

		// 1. Check for Bearer Token (JWT)
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			user, err = m.Token.ValidateAccessToken(tokenString)
			if err != nil {
				logging.Log.Warnf("AuthMiddleware: Invalid Bearer token: %v", err)
				if strings.Contains(err.Error(), "expired") {
					// Send a specific error for expired tokens
					writeError(w, http.StatusUnauthorized, "Token expired")
				} else {
					writeError(w, http.StatusUnauthorized, "Invalid token")
				}
				return
			}
		} else if strings.HasPrefix(authHeader, "Basic ") {
			// 2. Fallback to Basic Auth
			username, password, ok := r.BasicAuth()
			if !ok {
				writeError(w, http.StatusUnauthorized, "Invalid Basic Auth header")
				return
			}
			user, err = m.validateBasicAuth(username, password)
			if err != nil {
				logging.Log.Warnf("AuthMiddleware: Invalid Basic Auth: %v", err)
				writeError(w, http.StatusUnauthorized, "Authentication failed")
				return
			}
		} else {
			writeError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		// --- Success: We have a valid user from one of the methods ---

		// Add user and roles to the context
		ctx := context.WithValue(r.Context(), "user", user)
		roles := getUserRoles(user) // getUserRoles is in roles.go
		ctx = context.WithValue(ctx, "roles", roles)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateBasicAuth is a helper to check username/password against the database.
func (m *Middleware) validateBasicAuth(username, password string) (*models.User, error) {
	user, err := m.User.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("user '%s' not found", username)
	}

	// Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("password comparison failed for user '%s'", username)
	}
	return user, nil
}

// RoleMiddleware is a middleware function that checks if the user has the required role.
func (m *Middleware) RoleMiddleware(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, userOk := r.Context().Value("user").(*models.User)
			roles, rolesOk := r.Context().Value("roles").([]string)

			if !userOk || !rolesOk {
				logging.Log.Warnf("RoleMiddleware: No user or roles found in context for %s", r.URL.Path)
				writeError(w, http.StatusForbidden, "Forbidden")
				return
			}

			//logging.Log.Debugf("RoleMiddleware: Checking for role '%s' for user '%s' on path %s", requiredRole, user.Username, r.URL.Path)
			//logging.Log.Debugf("RoleMiddleware: User '%s' has roles: %v", user.Username, roles)

			for _, role := range roles {
				if role == requiredRole {
					//logging.Log.Debugf("RoleMiddleware: Access GRANTED for user '%s' (role: %s)", user.Username, requiredRole)
					next.ServeHTTP(w, r)
					return
				}
			}

			logging.Log.Warnf("RoleMiddleware: Access DENIED for user '%s'. Missing role '%s' for %s", user.Username, requiredRole, r.URL.Path)
			writeError(w, http.StatusForbidden, "Forbidden")
		})
	}
}
