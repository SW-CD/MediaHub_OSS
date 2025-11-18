# Development Plan: Stateful Hybrid JWT & Basic Authentication

## 1\. 🎯 Project Goal

The current authentication system relies exclusively on Basic Authentication, which forces the server to perform a computationally expensive `bcrypt` hash comparison on **every single API request**. This creates a performance bottleneck and is not ideal for a fast, responsive Single-Page Application (SPA).

This development plan outlines the implementation of a modern, token-based authentication system using **JSON Web Tokens (JWT)**. This will be implemented *in addition* to Basic Auth, not as a replacement.

This update introduces a **stateful, database-backed** token model to achieve two critical goals:

1.  **Security:** Allow for robust, immediate, and persistent session revocation (logout) that survives server restarts.
2.  **Scalability:** Pave the way for horizontal scaling (e.g., multiple server instances) by using a central SQL database (like the planned PostgreSQL) as the single source of truth for valid sessions.

## 2\. 📋 Core Requirements

The developer implementing this feature must adhere to all general "clean code" principles (small functions, high package cohesion, clear naming) and ensure the following functional requirements are met.

### Functional Requirements

1.  **Hybrid Authentication:** All existing protected API endpoints (e.g., `GET /api/databases`, `POST /api/entry`) **must** accept *either* a `Basic <auth>` header OR a `Bearer <token>` header for authentication.
2.  **New Token Database Table:** A new SQL table named `refresh_tokens` must be created in the database to store an "Allow List" of valid refresh tokens.
3.  **Secure Token Storage:** Refresh tokens **must not** be stored in the database in plaintext. They must be hashed (e.g., with SHA-256) before being stored in the `token_hash` column.
4.  **New Token Endpoints:**
      * `POST /api/token`: A new **public** endpoint. A client sends their `username` and `password` via Basic Auth. The server validates them, generates a new Access/Refresh token pair, **stores the hash of the refresh token in the `refresh_tokens` table**, and returns the tokens to the client.
      * `POST /api/token/refresh`: A new **public** endpoint. A client sends their Refresh Token. The server hashes it, checks if the hash exists in the `refresh_tokens` table, and, if valid, issues a new token pair (deleting the old token hash and storing the new one).
      * `POST /api/logout`: A new **protected** endpoint (requires a valid Access Token). The client sends their Refresh Token. The server hashes it and **deletes the hash from the `refresh_tokens` table**, permanently invalidating the session.
5.  **Persistent JWT Secret:** The JWT signing secret **must** be persistent across server restarts. The startup logic must be:
      * **Priority 1:** Use the secret from the `--jwt-secret` flag or `FDB_JWT_SECRET` environment variable (for production/Docker).
      * **Priority 2:** If not found, use the `secret` key from the `[jwt]` section of `config.toml`.
      * **Priority 3 (First Run):** If not found, generate a new secure 256-bit secret, **save it back to `config.toml`**, and print an `INFO` message.

### Clean Code & Structural Requirements

1.  **High Cohesion:** All new authentication logic (token generation, validation, middleware, interfaces) must be consolidated within the `internal/services/auth` package.
2.  **Repository Separation:** All direct SQL operations for the new table must be encapsulated in a new `internal/repository/token_repo.go` file.
3.  **Transactional Integrity:** When a user is deleted from the `users` table, all their associated refresh tokens must also be deleted from the `refresh_tokens` table within the same database transaction.

-----

## 3\. 🔑 New Database Schema

A new table is required to store the "Allow List" of valid refresh token hashes.

Add the following to the schema initialization:

**`internal/repository/schema.go` (in `initDB` function)**

```sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    
    -- Stores the SHA-256 hash of the refresh token
    token_hash TEXT UNIQUE NOT NULL, 
    
    expiry TIMESTAMP NOT NULL,
    
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
```

**Note:** The `ON DELETE CASCADE` is critical. It automatically removes a user's tokens when their `users` record is deleted, fulfilling the transactional integrity requirement.

-----

## 4\. 🛠️ Implementation Plan

This is the step-by-step guide for implementation, broken down by package.

### Phase 1: Configuration & Startup

Update config files and `main.go` to handle the new persistent, auto-generating secret.

1.  **Add Dependency:**

      * Run `go get github.com/golang-jwt/jwt/v5`

2.  **Update `internal/config/config.go`:**

      * Add a `Secret` field to `JWTConfig` for TOML persistence.
      * Create a `SaveConfig` function.

    <!-- end list -->

    ```go
    // filepath: internal/config/config.go
    package config

    import (
    	"fmt" // <-- ADD
    	"os"  // <-- ADD
    	"github.com/BurntSushi/toml"
    )

    type Config struct {
        // ... (existing fields)
        JWT      JWTConfig      `toml:"jwt"`
        JWTSecret          string `toml:"-"` // This remains the high-priority, in-memory secret
    }

    type JWTConfig struct {
        AccessDurationMin  int `toml:"access_duration_min"`
        RefreshDurationHours int `toml:"refresh_duration_hours"`
        Secret             string `toml:"secret"` // <-- ADD THIS (for file persistence)
    }

    // ... (LoadConfig) ...

    // ADD THIS NEW FUNCTION
    func SaveConfig(path string, cfg *Config) error {
        f, err := os.Create(path)
        if err != nil {
            return fmt.Errorf("failed to create config file for saving: %w", err)
        }
        defer f.Close()
        encoder := toml.NewEncoder(f)
        if err := encoder.Encode(cfg); err != nil {
            return fmt.Errorf("failed to encode config to file: %w", err)
        }
        return nil
    }
    ```

3.  **Update `cmd/mediahub/main.go`:**

      * Add `crypto/rand`, `encoding/hex`, and `mediahub/internal/services/auth` imports.
      * Implement the "Generate & Persist" secret logic.
      * Update service wiring to use the `auth` package.

    <!-- end list -->

    ```go
    // filepath: cmd/mediahub/main.go
    package main

    import (
    	"crypto/rand" // <-- ADD
    	"embed"
    	"encoding/hex" // <-- ADD
        // ...
    	"mediahub/internal/services"
    	"mediahub/internal/services/auth" // <-- Import 'auth' package
    )

    func main() {
        // ... (flag parsing) ...
        var jwtSecret string
        flag.StringVar(&jwtSecret, "jwt-secret", "", "Secret key for signing JWTs. (Env: FDB_JWT_SECRET)")
        // ... (flag.Parse()) ...
        
        // ... (LoadConfig) ...

        overrideConfigFromEnvAndCLI(&cliCfg, cfg)

        // --- NEW JWT SECRET LOGIC ---
        // 3a. Set JWT Secret from flag/env (Priority 1)
        if envJWTSecret := os.Getenv("FDB_JWT_SECRET"); envJWTSecret != "" {
            cfg.JWTSecret = envJWTSecret
        }
        if jwtSecret != "" { cfg.JWTSecret = jwtSecret }

        // 3b. Handle secret if not provided by flag/env
        if cfg.JWTSecret == "" {
            if cfg.JWT.Secret != "" {
                // Priority 2: Use secret from config.toml
                logging.Log.Info("Using JWT secret loaded from config.toml.")
                cfg.JWTSecret = cfg.JWT.Secret
            } else {
                // Priority 3: Generate, save, and use a new secret
                logging.Log.Info("No JWT secret found. Generating a new random secret...")
                newSecretBytes := make([]byte, 32) // 256 bits
                if _, err := rand.Read(newSecretBytes); err != nil {
                    logging.Log.Fatalf("Failed to generate new JWT secret: %v", err)
                }
                newSecretString := hex.EncodeToString(newSecretBytes)

                cfg.JWT.Secret = newSecretString
                cfg.JWTSecret = newSecretString

                if err := config.SaveConfig(cliCfg.ConfigPath, cfg); err != nil {
                    logging.Log.Warnf("Failed to save new JWT secret to %s: %v", cliCfg.ConfigPath, err)
                } else {
                    logging.Log.Infof("New JWT secret has been generated and saved to %s.", cliCfg.ConfigPath)
                }
            }
        }
        // --- END NEW LOGIC ---

        // ... (logging.Init, media.Initialize, repository.NewRepository) ...

        storageService := services.NewStorageService(cfg)
        infoService := services.NewInfoService(version, startTime, media.IsFFmpegAvailable(), media.IsFFprobeAvailable())
        userService := services.NewUserService(repo)
        
        // --- UPDATED WIRING ---
        tokenService := auth.NewTokenService(cfg, userService, repo) // <-- USE 'auth' package
        
        databaseService := services.NewDatabaseService(repo, storageService)
        // ... (other services) ...

        // Initialize Auth Middleware
        authMiddleware := auth.NewMiddleware(userService, tokenService) // <-- USE 'auth' package

        // ... (Admin User Setup, Init Config, Housekeeping) ...

        // Create Handlers struct
        h := handlers.NewHandlers(
            infoService,
            userService,
            tokenService, // <-- Pass the tokenService
            databaseService,
            entryService,
            housekeepingService,
            cfg,
        )

        // ... (rest of main) ...
    }

    // ... in overrideConfigFromEnvAndCLI() ...

        // --- ADD: Set JWT duration defaults ---
        if cfg.JWT.AccessDurationMin == 0 {
            cfg.JWT.AccessDurationMin = 5 // Default to 5 minutes
        }
        if cfg.JWT.RefreshDurationHours == 0 {
            cfg.JWT.RefreshDurationHours = 24 // Default to 24 hours
        }
    // ...
    ```

-----

### Phase 2: Database & Repository Updates

Implement the new `refresh_tokens` table and its repository logic.

1.  **Update `internal/repository/schema.go`:**

      * Add the `CREATE TABLE refresh_tokens...` SQL (from section 3) inside the `initDB` function.

2.  **Create `internal/repository/token_repo.go` (New File):**

      * This file will encapsulate all SQL for the `refresh_tokens` table.

    <!-- end list -->

    ```go
    // filepath: internal/repository/token_repo.go
    package repository

    import (
    	"database/sql"
    	"fmt"
    	"time"
    )

    // StoreRefreshToken saves the hash of a refresh token to the database.
    func (s *Repository) StoreRefreshToken(userID int64, tokenHash string, expiry time.Time) error {
    	query := "INSERT INTO refresh_tokens (user_id, token_hash, expiry) VALUES (?, ?, ?)"
    	_, err := s.DB.Exec(query, userID, tokenHash, expiry)
    	return err
    }

    // ValidateRefreshToken checks if a token hash exists and is not expired, returning the user ID.
    func (s *Repository) ValidateRefreshToken(tokenHash string) (int64, error) {
    	query := "SELECT user_id FROM refresh_tokens WHERE token_hash = ? AND expiry > ?"
    	var userID int64
    	err := s.DB.QueryRow(query, tokenHash, time.Now()).Scan(&userID)
    	if err != nil {
    		if err == sql.ErrNoRows {
    			return 0, fmt.Errorf("token not found or expired")
    		}
    		return 0, err
    	}
    	return userID, nil
    }

    // DeleteRefreshToken removes a specific refresh token hash from the database.
    func (s *Repository) DeleteRefreshToken(tokenHash string) error {
    	query := "DELETE FROM refresh_tokens WHERE token_hash = ?"
    	_, err := s.DB.Exec(query, tokenHash)
    	return err
    }
    ```

3.  **Update `internal/repository/user_repo.go`:**

      * The `ON DELETE CASCADE` in the schema handles this, so no code change is required in `DeleteUser`. This is a clean, database-level solution.

-----

### Phase 3: Auth Package (The Core Logic)

Consolidate all new authentication logic within `internal/services/auth`.

1.  **Create `internal/services/auth/interfaces.go` (New File):**

    ```go
    // filepath: internal/services/auth/interfaces.go
    package auth
    import "mediahub/internal/models"

    type TokenService interface {
        GenerateTokens(user *models.User) (accessToken string, refreshToken string, err error)
        ValidateAccessToken(tokenString string) (*models.User, error)
        ValidateRefreshToken(tokenString string) (*models.User, error)
        Logout(refreshToken string) error
    }
    ```

2.  **Create `internal/services/auth/token_service.go` (New File):**

      * This is the new *stateful* implementation.

    <!-- end list -->

    ```go
    // filepath: internal/services/auth/token_service.go
    package auth

    import (
    	"crypto/sha256"
    	"encoding/hex"
    	"errors"
    	"fmt"
    	"mediahub/internal/config"
    	"mediahub/internal/models"
    	"mediahub/internal/repository"
    	"mediahub/internal/services"
    	"time"

    	"github.com/golang-jwt/jwt/v5"
    )

    // ... (accessClaims and refreshClaims structs) ...
    type accessClaims struct {
    	Username string `json:"username"`
    	jwt.RegisteredClaims
    }
    type refreshClaims struct {
    	// We only need the ID (jti) in the claims.
        // The corresponding hash is what we store in the DB.
    	jwt.RegisteredClaims
    }

    var _ TokenService = (*tokenService)(nil)

    type tokenService struct {
    	cfg     *config.config.Config
    	userSvc services.UserService
    	repo    *repository.Repository
    }

    func NewTokenService(cfg *config.Config, userSvc services.UserService, repo *repository.Repository) TokenService {
    	return &tokenService{cfg: cfg, userSvc: userSvc, repo: repo}
    }

    // hashToken securely hashes a token string for database storage.
    func hashToken(token string) string {
        hash := sha256.Sum256([]byte(token))
        return hex.EncodeToString(hash[:])
    }

    // GenerateTokens creates, signs, and *stores* a new token pair.
    func (s *tokenService) GenerateTokens(user *models.User) (string, string, error) {
    	// 1. Create Access Token (5 min)
    	accessExpiry := time.Now().Add(time.Minute * time.Duration(s.cfg.JWT.AccessDurationMin))
    	accessClaims := &accessClaims{
    		Username: user.Username,
    		RegisteredClaims: {
    			ExpiresAt: jwt.NewNumericDate(accessExpiry),
    			Issuer:    "mediahub",
                Subject:   fmt.Sprintf("%d", user.ID),
    		},
    	}
    	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    	signedAccessToken, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
    	if err != nil {
    		return "", "", err
    	}

    	// 2. Create Refresh Token (24h)
    	refreshExpiry := time.Now().Add(time.Hour * time.Duration(s.cfg.JWT.RefreshDurationHours))
        // Use a random ID for the token
        jtiBytes := make([]byte, 16)
        rand.Read(jtiBytes)
        jti := hex.EncodeToString(jtiBytes)

    	refreshClaims := &refreshClaims{
    		RegisteredClaims: {
    			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
    			Issuer:    "mediahub",
                Subject:   fmt.Sprintf("%d", user.ID),
                ID:        jti,
    		},
    	}
    	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    	signedRefreshToken, err := refreshToken.SignedString([]byte(s.cfg.JWTSecret))
    	if err != nil {
    		return "", "", err
    	}

    	// 3. Store the hash of the refresh token in the database
    	tokenHash := hashToken(signedRefreshToken)
    	if err := s.repo.StoreRefreshToken(user.ID, tokenHash, refreshExpiry); err != nil {
    		return "", "", fmt.Errorf("failed to store refresh token: %w", err)
    	}

    	return signedAccessToken, signedRefreshToken, nil
    }

    // ValidateAccessToken checks an access token (stateless)
    func (s *tokenService) ValidateAccessToken(tokenString string) (*models.User, error) {
    	claims := &accessClaims{}
    	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
    		return []byte(s.cfg.JWTSecret), nil
    	})

    	if err != nil { return nil, err }
    	if !token.Valid { return nil, errors.New("invalid access token") }

    	user, err := s.userSvc.GetUserByUsername(claims.Username)
    	if err != nil { return nil, errors.New("user not found for token") }
    	return user, nil
    }

    // ValidateRefreshToken checks a refresh token (stateful)
    func (s *tokenService) ValidateRefreshToken(tokenString string) (*models.User, error) {
        // 1. Check signature first (fast check)
        claims := &refreshClaims{}
    	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
    		return []byte(s.cfg.JWTSecret), nil
    	})
        if err != nil { return nil, err }
        if !token.Valid { return nil, errors.New("invalid refresh token signature or claims") }

    	// 2. Hash the token and check the database "Allow List"
    	tokenHash := hashToken(tokenString)
    	userID, err := s.repo.ValidateRefreshToken(tokenHash)
    	if err != nil {
    		return nil, fmt.Errorf("token not found in database: %w", err)
    	}

        // 3. Fetch the user
    	user, err := s.userSvc.GetUserByID(int(userID))
        if err != nil {
            return nil, errors.New("user not found for valid token")
        }
    	return user, nil
    }

    // Logout invalidates a refresh token by deleting it from the DB.
    func (s *tokenService) Logout(refreshToken string) error {
    	tokenHash := hashToken(refreshToken)
    	return s.repo.DeleteRefreshToken(tokenHash)
    }
    ```

3.  **Update `internal/services/interfaces.go`:**

      * **Remove** the `TokenService` interface from this file. It now lives in `internal/services/auth/interfaces.go`.

-----

### Phase 4: Handlers & Routing

Create the new HTTP handlers and wire up the routes.

1.  **Update `internal/api/handlers/main.go`:**

      * Update `Handlers` struct and `NewHandlers` to import and accept `auth.TokenService`.

    <!-- end list -->

    ```go
    // filepath: internal/api/handlers/main.go
    package handlers

    import (
    	"mediahub/internal/config"
    	"mediahub/internal/services"
    	"mediahub/internal/services/auth" // <-- ADD this import
    	"time"
    )

    type Handlers struct {
    	Info         services.InfoService
    	User         services.UserService
    	Token        auth.TokenService // <-- UPDATE this type
    	Database     services.DatabaseService
    	Entry        services.EntryService
    	Housekeeping services.HousekeepingService
        // ...
    }

    func NewHandlers(
    	info services.InfoService,
    	user services.UserService,
    	token auth.TokenService, // <-- UPDATE this type
    	database services.DatabaseService,
        // ...
    ) *Handlers {
    	return &Handlers{
    		Info:         info,
    		User:         user,
    		Token:        token, // <-- UPDATE this
    		Database:     database,
            // ...
    	}
    }
    ```

2.  **Create `internal/api/handlers/token_handler.go` (New File):**

      * Add this new file for the three token endpoints.

    <!-- end list -->

    ```go
    // filepath: internal/api/handlers/token_handler.go
    package handlers

    import (
    	"encoding/json"
    	"mediahub/internal/logging"
    	"net/http"

    	"golang.org/x/crypto/bcrypt"
    )

    type tokenRequest struct {
    	RefreshToken string `json:"refresh_token"`
    }

    type tokenResponse struct {
    	AccessToken  string `json:"access_token"`
    	RefreshToken string `json:"refresh_token"`
    }

    // @Summary Get JWT tokens
    // @Description Authenticate using Basic Auth to receive an access and refresh token.
    // @Tags Auth
    // @Produce  json
    // @Success 200 {object} tokenResponse
    // @Failure 401 {object} ErrorResponse "Authentication failed"
    // @Failure 500 {object} ErrorResponse "Token generation failed"
    // @Router /token [post]
    func (h *Handlers) GetToken(w http.ResponseWriter, r *http.Request) {
    	username, password, ok := r.BasicAuth()
    	if !ok {
    		respondWithError(w, http.StatusUnauthorized, "Authentication failed")
    		return
    	}

    	// Validate user
    	user, err := h.User.GetUserByUsername(username)
    	if err != nil {
    		respondWithError(w, http.StatusUnauthorized, "Authentication failed")
    		return
    	}

    	// Compare password
    	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
    		respondWithError(w, http.StatusUnauthorized, "Authentication failed")
    		return
    	}

    	// Generate tokens (now stateful)
    	accessToken, refreshToken, err := h.Token.GenerateTokens(user) // <-- Pass user object
    	if err != nil {
    		logging.Log.Errorf("Token generation failed for %s: %v", username, err)
    		respondWithError(w, http.StatusInternalServerError, "Could not generate tokens")
    		return
    	}

    	respondWithJSON(w, http.StatusOK, tokenResponse{
    		AccessToken:  accessToken,
    		RefreshToken: refreshToken,
    	})
    }

    // @Summary Refresh JWT access token
    // @Description Provide a valid refresh token to receive a new access token.
    // @Tags Auth
    // @Accept   json
    // @Produce  json
    // @Param   token  body  tokenRequest  true  "Refresh Token"
    // @Success 200 {object} tokenResponse
    // @Failure 400 {object} ErrorResponse "Invalid request body"
    // @Failure 401 {object} ErrorResponse "Invalid or expired token"
    // @Failure 500 {object} ErrorResponse "Token generation failed"
    // @Router /token/refresh [post]
    func (h *Handlers) RefreshToken(w http.ResponseWriter, r *http.Request) {
    	var req tokenRequest
    	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    		respondWithError(w, http.StatusBadRequest, "Invalid request body")
    		return
    	}

    	// Validate the refresh token (returns user)
    	user, err := h.Token.ValidateRefreshToken(req.RefreshToken)
    	if err != nil {
    		respondWithError(w, http.StatusUnauthorized, err.Error())
    		return
    	}

        // We must invalidate the old refresh token
        // This makes the refresh token a one-time-use token, which is more secure.
        if err := h.Token.Logout(req.RefreshToken); err != nil {
            logging.Log.Warnf("Failed to invalidate old refresh token during refresh for user %s: %v", user.Username, err)
        }

    	// Issue new tokens for this user
    	accessToken, refreshToken, err := h.Token.GenerateTokens(user)
    	if err != nil {
    		logging.Log.Errorf("Token refresh failed for %s: %v", user.Username, err)
    		respondWithError(w, http.StatusInternalServerError, "Could not generate tokens")
    		return
    	}

    	respondWithJSON(w, http.StatusOK, tokenResponse{
    		AccessToken:  accessToken,
    		RefreshToken: refreshToken,
    	})
    }

    // @Summary Logout
    // @Description Invalidates a refresh token. This endpoint is protected.
    // @Tags Auth
    // @Accept   json
    // @Produce  json
    // @Param   token  body  tokenRequest  true  "Refresh Token to invalidate"
    // @Success 200 {object} MessageResponse
    // @Failure 400 {object} ErrorResponse "Invalid request body"
    // @Failure 401 {object} ErrorResponse "Authentication required (invalid access token)"
    // @Failure 500 {object} ErrorResponse "Could not process token"
    // @Security BasicAuth
    // @Router /logout [post]
    func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
    	var req tokenRequest
    	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    		respondWithError(w, http.StatusBadRequest, "Invalid request body")
    		return
    	}

    	if err := h.Token.Logout(req.RefreshToken); err != nil {
    		respondWithError(w, http.StatusInternalServerError, err.Error())
    		return
    	}

    	respondWithJSON(w, http.StatusOK, MessageResponse{Message: "Logged out successfully."})
    }
    ```

3.  **Update `internal/api/router.go`:**

      * Register the new public token endpoints *before* the `apiRouter`.
      * Register the new protected logout endpoint *inside* the `apiRouter`.

    <!-- end list -->

    ```go
    // filepath: internal/api/router.go
    // ...
    func SetupRouter(
    	h *handlers.Handlers,
    	authMiddleware *auth.Middleware,
    	cfg *config.Config,
    	frontendFS embed.FS,
    ) *mux.Router {
    	r := mux.NewRouter()

    	// --- Public Endpoints ---
    	r.HandleFunc("/health", handlers.HealthCheck).Methods("GET")
    	r.HandleFunc("/api/info", h.GetInfo).Methods("GET")
    	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

    	// --- NEW: Public Token Endpoints (Not protected by authMiddleware) ---
    	r.HandleFunc("/api/token", h.GetToken).Methods("POST")
    	r.HandleFunc("/api/token/refresh", h.RefreshToken).Methods("POST")

    	// --- Authenticated API Routes ---
    	apiRouter := r.PathPrefix("/api").Subrouter()
    	apiRouter.Use(authMiddleware.AuthMiddleware) // <-- This will now check for JWT *or* Basic

    	// --- NEW: Authenticated Logout Endpoint (Protected by authMiddleware) ---
    	apiRouter.HandleFunc("/logout", h.Logout).Methods("POST")

    	// Attach resource-specific routes
    	addDatabaseRoutes(apiRouter, h, authMiddleware)
    	// ... (rest of function)
    }
    ```

-----

### Phase 5: Modify Auth Middleware (The Hybrid Logic)

Implement the dual-auth (Bearer/Basic) check.

1.  **Update `internal/services/auth/middleware.go`:**

      * Modify `NewMiddleware` to accept the `TokenService`.
      * Rewrite `AuthMiddleware` to check for ` Bearer  ` first, then fall back to ` Basic  `.
      * Add `bcrypt` import back.

    <!-- end list -->

    ```go
    // filepath: internal/services/auth/middleware.go
    package auth

    import (
    	"context"
    	"encoding/json"
    	"fmt" // <-- ADD
    	"mediahub/internal/logging"
    	"mediahub/internal/models"
    	"mediahub/internal/services" // <-- Import services
    	"net/http"
    	"strings" // <-- ADD

    	"golang.org/x/crypto/bcrypt" // <-- ADD
    )

    // ... (writeError remains the same) ...

    // Middleware provides authentication and authorization middleware.
    type Middleware struct {
    	User  services.UserService // <-- Dependency on main services
    	Token TokenService         // <-- Dependency internal to 'auth' package
    }

    // NewMiddleware creates a new instance of Middleware.
    func NewMiddleware(user services.UserService, token TokenService) *Middleware { // <-- UPDATE
    	return &Middleware{
    		User:  user,  // <-- UPDATE
    		Token: token, // <-- ADD
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

    		// Check for Bearer token first
    		if strings.HasPrefix(authHeader, "Bearer ") {
    			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
    			user, err = m.validateJwtToken(tokenString)
    			if err != nil {
    				logging.Log.Warnf("AuthMiddleware: Invalid Bearer token: %v", err)
    				if strings.Contains(err.Error(), "token is expired") {
    					writeError(w, http.StatusUnauthorized, "Token expired")
    				} else {
    					writeError(w, http.StatusUnauthorized, "Invalid token")
    				}
    				return
    			}
    		} else if strings.HasPrefix(authHeader, "Basic ") {
    			// Fallback to Basic Auth
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
    		roles := getUserRoles(user)
    		ctx = context.WithValue(ctx, "roles", roles)

    		next.ServeHTTP(w, r.WithContext(ctx))
    	})
    }

    // --- NEW HELPER: validateJwtToken ---
    func (m *Middleware) validateJwtToken(tokenString string) (*models.User, error) {
    	// Validate the access token
    	user, err := m.Token.ValidateAccessToken(tokenString)
    	if err != nil {
    		return nil, err
    	}
    	return user, nil
    }

    // --- NEW HELPER: validateBasicAuth ---
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

    // ... (RoleMiddleware and getUserRoles remain the same in this file) ...
    ```

-----

### Phase 6: Frontend (Angular) Implications

Finally, the backend developer must inform the frontend developer of the new, *preferred* auth flow. This is outside the scope of the backend code but is critical for the feature to work.

1.  **On Login:** The UI should call `POST /api/token` (with Basic Auth).
2.  **Store Tokens:** On success, the UI must store the `access_token` and `refresh_token` (e.g., in `localStorage`).
3.  **Auth Interceptor:** The UI's `AuthInterceptor` must be updated to add `Authorization: Bearer <access_token>` to all subsequent API requests.
4.  **Token Refresh Logic:** The `AuthInterceptor` must also be updated to catch `401 Unauthorized` responses.
      * If a 401 is received (and it's not from the refresh endpoint itself), it should *pause* the failed request and send a `POST /api/token/refresh` request with the stored `refresh_token`.
      * **Success:** Save the *new* tokens, and retry the original failed request with the *new* access token.
      * **Failure:** The refresh token is invalid/expired. Delete all stored tokens and redirect the user to the login page.
5.  **Logout:** The logout button must call `POST /api/logout` (using the Bearer token) and send its `refresh_token` in the body. On success (or 401), it must clear all local tokens and redirect to login.