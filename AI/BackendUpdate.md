# 🛠️ Implementation Plan: MediaHub v1.2

**Objective:** Refactor the backend for architectural extensibility, reliability, and efficiency.
**Strategy:** Implement changes sequentially to maintain a working build at every step.

-----

## 📅 Phase 1:  Architecture & Reliability

### Step 1: CLI Framework Adoption (Cobra)

**Goal:** Replace standard `flag` parsing with `cobra` to support subcommands (`serve`, `recovery`).
**Reference:** `cmd/mediahub/main.go`

1.  **Dependencies:**
      * Run `go get -u github.com/spf13/cobra`.
2.  **Restructure `cmd/mediahub`:**
      * Create a new package `internal/cli`.
      * **`internal/cli/root.go`**: Define the `RootCmd`. This command will host the **PersistentFlags** (flags that apply globally, like `--config_path` or `--log-level`).
      * **`internal/cli/serve.go`**: Move the main HTTP server logic from `main.go` into a `serveCmd`.
          * *Action:* Move the specific flags like `--port`, `--max-sync-upload` from `main.go` to this command's flags.
      * **`internal/cli/recovery.go`**: Create a placeholder `recoveryCmd` (logic to be added in Step 5).
3.  **Update `main.go`:**
      * Wipe the existing logic.
      * Replace it with a simple call to `cli.Execute()`.
4.  **Validation:**
      * Run `go run ./cmd/mediahub serve --port 8090` to confirm the server still starts.

### Step 2: Database Migration System

**Goal:** Replace hardcoded `initDB` strings with versioned SQL files using `pressly/goose`.
**Reference:** `internal/repository/schema.go`

1.  **Dependencies:**
      * Run `go get -u github.com/pressly/goose/v3`.
2.  **Create Migrations:**
      * Create `internal/db/migrations/`.
      * Create `001_initial_schema.sql`. Copy the SQL content from `internal/repository/schema.go` into this file.
      * **Delete** `internal/repository/schema.go`.
3.  **Embed Migrations:**
      * In `internal/repository/repository.go`, add:
        ```go
        //go:embed migrations/*.sql
        var embedMigrations embed.FS
        ```
4.  **Migration Runner:**
      * In `internal/repository`, create a helper `RunMigrations(db *sql.DB) error` that uses `goose.SetBaseFS(embedMigrations)` and `goose.Up(db, "migrations")`.
      * Call this in `NewRepository` right after opening the connection.

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
      * Inject `Auditor` into `Handlers` struct in `internal/api/handlers/main.go`.
      * Update `CreateDatabase`, `DeleteDatabase`, `CreateEntry`, `DeleteEntry` handlers to call `h.Auditor.Log(...)`.

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
      * In `internal/cli/serve.go`:
          * Create a `srv := &http.Server{...}`.
          * Run `srv.ListenAndServe()` in a goroutine.
          * Listen for `os.Interrupt`. On signal, call `srv.Shutdown(ctx)`.
2.  **Recovery Command:**
      * In `internal/cli/recovery.go`, implement the `Run` function:
          * Initialize `Repository` (this runs migrations implicitly).
          * **Zombie Fix:** Execute `UPDATE entries_X SET status='error' WHERE status='processing'`.
          * **Integrity Check (Optional):** Scan `databases` table, check if storage folders exist.
      * This command is now run via `./mediahub recovery` thanks to Cobra.

-----

## 📅 Phase 2: Efficiency & Features

### Step 1: Bulk Delete

**Goal:** Efficiently delete multiple entries.

1.  **API Handler:**
      * Add `POST /api/entry/bulk-delete` to `internal/api/handlers/entry_handler.go`.
      * Body: `{"database_name": "X", "ids": [1, 2, 3]}`.
2.  **Service Layer:**
      * Add `DeleteEntries(dbName string, ids []int64)` to `EntryService`.
3.  **Repository Layer:**
      * Use `squirrel` to generate `DELETE FROM entries_X WHERE id IN (?, ?, ?)`.
      * *Performance:* Calculate total size of files to be deleted first, then update stats *once* at the end of the transaction.

### Step 2: Streaming ZIP Export

**Goal:** Low-memory dataset export.

1.  **API Handler:**
      * Add `GET /api/database/export` to `internal/api/handlers/database_handler.go`.
2.  **Implementation:**
      * Use `io.Pipe()`. Connect `PipeWriter` to `zip.NewWriter`.
      * Run the zip writing loop in a **goroutine**.
      * Inside the loop: Query all files in DB. For each file, `os.Open` (disk) -\> `io.Copy` (to zip).
      * Stream the `PipeReader` directly to `http.ResponseWriter`.
      * Set header `Content-Type: application/zip`.
