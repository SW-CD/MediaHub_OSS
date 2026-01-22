package httpserver

import (
	"mediahub/internal/httpserver/auth"
	"mediahub/internal/httpserver/handlers"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

// SetupRouter configures the main router and its sub-routers.
// It sets up API endpoints, authentication, and the frontend server.
func SetupRouter(h *handlers.Handlers) *mux.Router {
	r := mux.NewRouter()

	// Public Endpoints
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")
	r.HandleFunc("/api/info", h.GetInfo).Methods("GET")
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Public Token Endpoints (Not protected by authMiddleware)
	r.HandleFunc("/api/token", h.GetToken).Methods("POST")
	r.HandleFunc("/api/token/refresh", h.RefreshToken).Methods("POST")

	// Authenticated API Routes
	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(auth.AuthMiddleware) // This will check for JWT *or* Basic

	// Authenticated Logout Endpoint (Protected by authMiddleware) ---
	apiRouter.HandleFunc("/logout", h.Logout).Methods("POST")

	// Attach resource-specific routes
	addDatabaseRoutes(apiRouter, h, authMiddleware)
	addEntryRoutes(apiRouter, h, authMiddleware)
	addUserRoutes(apiRouter, h)
	addAdminRoutes(apiRouter, h, authMiddleware)

	// Frontend web server (public)
	web.AddRoutes(r, frontendFS, "index.html")

	return r
}

// addDatabaseRoutes configures routes related to database management.
func addDatabaseRoutes(r *mux.Router, h *handlers.Handlers, am *auth.Middleware) {
	// Sub-router for endpoints requiring 'CanView' permissions
	viewRouter := r.PathPrefix("").Subrouter()
	viewRouter.Use(am.RoleMiddleware("CanView"))
	viewRouter.HandleFunc("/database", h.GetDatabase).Methods("GET")
	viewRouter.HandleFunc("/databases", h.GetDatabases).Methods("GET")
	viewRouter.HandleFunc("/database/entries", h.QueryEntries).Methods("GET")
	viewRouter.HandleFunc("/database/entries/search", h.SearchEntries).Methods("POST")
	viewRouter.HandleFunc("/database/entries/export", h.ExportEntries).Methods("POST")

	// Sub-router for endpoints requiring 'CanCreate' permissions
	createRouter := r.PathPrefix("").Subrouter()
	createRouter.Use(am.RoleMiddleware("CanCreate"))
	createRouter.HandleFunc("/database", h.CreateDatabase).Methods("POST")

	// Sub-router for endpoints requiring 'CanEdit' permissions
	editRouter := r.PathPrefix("").Subrouter()
	editRouter.Use(am.RoleMiddleware("CanEdit"))
	editRouter.HandleFunc("/database", h.UpdateDatabase).Methods("PUT")

	// Sub-router for endpoints requiring 'CanDelete' permissions
	deleteRouter := r.PathPrefix("").Subrouter()
	deleteRouter.Use(am.RoleMiddleware("CanDelete"))
	deleteRouter.HandleFunc("/database", h.DeleteDatabase).Methods("DELETE")
	deleteRouter.HandleFunc("/database/housekeeping", h.TriggerHousekeeping).Methods("POST")
	deleteRouter.HandleFunc("/database/entries/delete", h.DeleteEntries).Methods("POST")
}

// addEntryRoutes configures routes related to entry management.
func addEntryRoutes(r *mux.Router, h *handlers.Handlers, am *auth.Middleware) {
	// Sub-router for endpoints requiring 'CanView' permissions
	viewRouter := r.PathPrefix("").Subrouter()
	viewRouter.Use(am.RoleMiddleware("CanView"))
	viewRouter.HandleFunc("/entry/file", h.GetEntry).Methods("GET")
	viewRouter.HandleFunc("/entry/meta", h.GetEntryMeta).Methods("GET")
	viewRouter.HandleFunc("/entry/preview", h.GetEntryPreview).Methods("GET")

	// Sub-router for endpoints requiring 'CanCreate' permissions
	createRouter := r.PathPrefix("").Subrouter()
	createRouter.Use(am.RoleMiddleware("CanCreate"))
	createRouter.HandleFunc("/entry", h.UploadEntry).Methods("POST")

	// Sub-router for endpoints requiring 'CanEdit' permissions
	editRouter := r.PathPrefix("").Subrouter()
	editRouter.Use(am.RoleMiddleware("CanEdit"))
	editRouter.HandleFunc("/entry", h.UpdateEntry).Methods("PATCH")

	// Sub-router for endpoints requiring 'CanDelete' permissions
	deleteRouter := r.PathPrefix("").Subrouter()
	deleteRouter.Use(am.RoleMiddleware("CanDelete"))
	deleteRouter.HandleFunc("/entry", h.DeleteEntry).Methods("DELETE")
}

// addUserRoutes configures routes for non-admin user actions (e.g., managing their own profile).
func addUserRoutes(r *mux.Router, h *handlers.Handlers) {
	// These endpoints only require a valid login, which AuthMiddleware already checks.
	r.HandleFunc("/me", h.GetUserMe).Methods("GET")
	r.HandleFunc("/me", h.UpdateUserMe).Methods("PATCH")
}

// addAdminRoutes configures routes for administrative actions on users.
func addAdminRoutes(r *mux.Router, h *handlers.Handlers, am *auth.Middleware) {
	adminRouter := r.PathPrefix("").Subrouter()
	adminRouter.Use(am.RoleMiddleware("IsAdmin"))
	adminRouter.HandleFunc("/users", h.GetUsers).Methods("GET")
	adminRouter.HandleFunc("/user", h.CreateUser).Methods("POST")
	adminRouter.HandleFunc("/user", h.UpdateUser).Methods("PATCH")
	adminRouter.HandleFunc("/user", h.DeleteUser).Methods("DELETE")
}
