package httpserver

import (
	"net/http"
	"strings"
)

// CORSMiddleware creates a middleware that handles Cross-Origin Resource Sharing (CORS).
// It verifies the Origin header against the configured allowed origins.
func CORSMiddleware(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// If no Origin header is present (same-origin request), or if the server
			// hasn't configured any allowed origins, we skip the CORS logic.
			if origin == "" || len(allowedOrigins) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Check if the request's origin matches our allowed list
			isAllowed := false
			for _, o := range allowedOrigins {
				o = strings.TrimSpace(o)
				if o == origin || o == "*" {
					isAllowed = true
					break
				}
			}

			// If the origin is allowed, append the required CORS headers
			if isAllowed {
				// We echo back the specific origin rather than "*" to support credentials
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

				// Allow headers necessary for JSON APIs, Auth, and Media Streaming
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, Range")

				// Expose headers so the frontend JavaScript can read them (Crucial for streaming/chunking)
				w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges, Content-Disposition")

				// Allow credentials (like cookies or Authorization headers) to be sent cross-origin
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests (OPTIONS method)
			// The browser sends this before the actual request to verify permissions.
			if r.Method == http.MethodOptions {
				// If it's a preflight, we answer successfully and stop the chain here.
				w.WriteHeader(http.StatusOK)
				return
			}

			// For all other standard requests, pass the execution to the next handler
			next.ServeHTTP(w, r)
		})
	}
}
