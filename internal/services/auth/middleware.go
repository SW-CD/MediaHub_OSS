// filepath: internal/services/auth/middleware.go
package auth

import (
	"context"
	"encoding/json"
	"mediahub/internal/logging"
	"mediahub/internal/models"
	"mediahub/internal/services" // <-- IMPORT SERVICES
	"net/http"

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
	User services.UserService // <-- REFACTOR: Depend on the interface
}

// NewMiddleware creates a new instance of Middleware.
func NewMiddleware(user services.UserService) *Middleware { // <-- REFACTOR: Accept the interface
	return &Middleware{User: user}
}

// AuthMiddleware is a middleware function that checks for a valid user session.
func (m *Middleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		username, password, ok := r.BasicAuth()

		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := m.User.GetUserByUsername(username)
		if err != nil {
			logging.Log.Warnf("AuthMiddleware: GetUserByUsername failed for '%s': %v", username, err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Compare the provided password with the stored hash
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			logging.Log.Warnf("AuthMiddleware: Password comparison FAILED for user '%s'", username)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user and roles to the context
		ctx := context.WithValue(r.Context(), "user", user)
		roles := getUserRoles(user)
		ctx = context.WithValue(ctx, "roles", roles)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
