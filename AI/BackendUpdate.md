# 🛠️ Implementation Plan: MediaHub v1.2

**Objective:** Refactor the backend for architectural extensibility, reliability, and efficiency, while delivering new bulk management features.
**Strategy:** Implement changes sequentially to maintain a working build at every step.

-----

## 📅 Part 1: Architecture & Reliability

### Step 1: CLI Framework Adoption (Cobra) & Compatibility

**Goal:** Replace standard `flag` parsing with `cobra`.
**Constraint:** Must maintain backward compatibility. Running `./mediahub` (with or without flags) must still start the server.

1.  **Dependencies:**
    * Run `go get -u github.com/spf13/cobra`.
2.  **Restructure `cmd/mediahub`:**
    * Create package `internal/cli`.
    * **`internal/cli/server_logic.go`**: Extract the `http.ListenAndServe` logic (currently in `main.go`) into a reusable function `func RunServer(cfg *config.Config)`.
    * **`internal/cli/root.go`**: Define the `RootCmd`.
        * **Behavior:** Set `RootCmd.Run` to execute `RunServer()`. This makes the server the default action.
        * **Flags:** Bind the existing server flags (`--port`, `--max-sync-upload`, `--password`, `--reset_pw`) directly to `RootCmd.Flags()`.
        * **Persistent Flags:** Bind `--config_path` and `--log-level` to `RootCmd.PersistentFlags()` so they apply to all subcommands.
    * **`internal/cli/serve.go`**: (Optional) Create a `serveCmd` that aliases to `RunServer`, for users who prefer explicit syntax.
    * **`internal/cli/recovery.go`**: Create a placeholder `recoveryCmd` (logic to be added in Step 5).
    * **`internal/cli/migrate.go`**: Create a placeholder `migrateCmd` (logic to be added in Step 2).
3.  **Update `main.go`:**
    * Wipe the existing logic.
    * Replace it with a single call to `cli.Execute()`.
4.  **Validation:**
    * Run `go run ./cmd/mediahub --port 9090` to confirm the server still starts (Backward Compatibility).
    * Run `go run ./cmd/mediahub serve` to confirm explicit command works.

### Step 2: Database Migration System

**Goal:** Replace hardcoded `initDB` strings with versioned SQL files using `pressly/goose`.
**Reference:** `internal/repository/schema.go`

1.  **Dependencies:**
    * Run `go get -u github.com/pressly/goose/v3`.
2.  **Create Migrations:**
    * Create directory `internal/db/migrations/`.
    * Create `001_initial_schema.sql`. Copy the SQL content from `internal/repository/schema.go` into this file.
    * **Delete** `internal/repository/schema.go`.
3.  **Embed Migrations:**
    * In `internal/repository/repository.go`, add:
        ```go
        //go:embed migrations/*.sql
        var embedMigrations embed.FS
        ```
4.  **Migration Logic:**
    * In `internal/repository`, create a helper `RunMigrations(db *sql.DB) error`.
    * Use `goose.SetBaseFS(embedMigrations)` and `goose.Up(db, "migrations")`.
    * Call this in `NewRepository` right after opening the connection.
5.  **CLI Integration:**
    * Update `internal/cli/migrate.go` to expose this logic via `./mediahub migrate [up|down|status]`.

### Step 3: The `Auditor` Interface

**Goal:** Decouple audit logging to avoid SQLite write locks and prepare for Commercial SQL logging.

1.  **Define Interface:**
    * Create `internal/services/audit.go` (or `internal/audit/audit.go`):
        ```go
        type Auditor interface {
            Log(event string, actor string, resource string, details map[string]interface{})
        }
        ```
2.  **Implement OSS Adapter:**
    * Create `internal/audit/stdout_adapter.go`.
    * Implement `Log` using `logrus.WithFields(...).Info("AUDIT")`. This writes to stdout (safe for concurrency).
3.  **Integration:**
    * Inject `Auditor` into the `Handlers` struct in `internal/api/handlers/main.go`.
    * Update `CreateDatabase`, `DeleteDatabase`, `CreateEntry`, and `DeleteEntry` handlers to call `h.Auditor.Log(...)` on success.

### Step 4: SQL Dialect Abstraction (Squirrel)

**Goal:** Abstract SQL generation to support Postgres later without maintaining duplicate repositories.
**Reference:** `internal/repository/query_repo.go`

1.  **Dependencies:**
    * Run `go get -u github.com/Masterminds/squirrel`.
2.  **Repository Update:**
    * Add `Builder squirrel.StatementBuilderType` to the `Repository` struct in `internal/repository/repository.go`.
    * Initialize it in `NewRepository`: `Builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)` (SQLite format).
3.  **Refactor Queries:**
    * **Refactor 1:** Update `SearchEntries` in `internal/repository/query_repo.go`. Replace string concatenation with `sq.Select("*").From(tableName).Where(...)`.
    * **Refactor 2:** Update `CreateEntryInTx` in `internal/repository/dbtx.go`. Use `sq.Insert(tableName).Columns(...).Values(...)`.
    * *Note:* `squirrel` handles the conditional logic cleanly and prevents SQL injection.

### Step 5: Graceful Shutdown & Recovery Command

**Goal:** Prevent corruption during restarts and fix "stuck" uploads from crashes.

1.  **Shutdown (Server):**
    * In `internal/cli/server_logic.go`:
        * Create a `srv := &http.Server{...}`.
        * Run `srv.ListenAndServe()` in a goroutine.
        * Listen for `os.Interrupt` and `syscall.SIGTERM`.
        * On signal, call `srv.Shutdown(ctx)` with a 30-second timeout.
2.  **Recovery Command:**
    * In `internal/cli/recovery.go`, implement the `Run` function:
        * Initialize `Repository` (this runs migrations implicitly).
        * **Zombie Fix:** Execute `UPDATE entries_X SET status='error' WHERE status='processing'`.
        * **Integrity Check (Optional):** Scan `databases` table, check if storage folders exist.
    * This command is now run via `./mediahub recovery` thanks to Cobra.

-----

## 📅 Part 2: Efficiency & Features

### Step 6: Bulk Delete

**Goal:** Efficiently delete multiple entries.

1.  **API Handler:**
    * Add `POST /api/entry/bulk-delete` to `internal/api/handlers/entry_handler.go`.
    * Body: `{"database_name": "X", "ids": [1, 2, 3]}`.
2.  **Service Layer:**
    * Add `DeleteEntries(dbName string, ids []int64)` to `EntryService`.
3.  **Repository Layer:**
    * Use `squirrel` to generate `DELETE FROM entries_X WHERE id IN (?, ?, ?)`.
    * *Performance:* Calculate total size of files to be deleted first, then update stats (`entry_count`, `disk_space`) *once* at the end of the transaction.

### Step 7: Streaming ZIP Export

**Goal:** Low-memory dataset export.

1.  **API Handler:**
    * Add `GET /api/database/export` to `internal/api/handlers/database_handler.go`.
2.  **Implementation:**
    * Use `io.Pipe()`. Connect `PipeWriter` to `zip.NewWriter`.
    * Run the zip writing loop in a **goroutine**.
    * Inside the loop: Query all files in DB. For each file, `os.Open` (disk) -> `io.Copy` (to zip).
    * Stream the `PipeReader` directly to `http.ResponseWriter`.
    * Set header `Content-Type: application/zip`.