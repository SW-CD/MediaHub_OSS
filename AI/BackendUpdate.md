# 📦 Backend Implementation Plan: v1.X

**Target Versions:** v1.2 (Architecture) & v1.3 (Efficiency)
**Strategy:** Refactor for modularity first (Architecture), then implement new capabilities (Features).
**Constraint:** No breaking changes to existing HTTP API behavior. CLI commands must remain backward compatible (default behavior starts server).

-----

## 📅 Phase 1: Architecture & Reliability (v1.2)

### Step 1: CLI Framework Adoption (Cobra)

**Goal:** Replace the standard `flag` package with `spf13/cobra` to support subcommands (`migrate`, `recovery`) while ensuring `./mediahub` still launches the server by default.

1.  **Dependency:** `go get -u github.com/spf13/cobra`
2.  **Refactor `cmd/mediahub/main.go`:**
      * Wipe existing logic. It should only call `cli.Execute()`.
3.  **Create `internal/cli` package:**
      * **`root.go` (The Server Command):**
          * Define `RootCmd`.
          * **Important:** Move the logic currently in `main.go` (config loading, service init, `http.ListenAndServe`) into the `RunE` field of `RootCmd`.
          * **Flags:** Migrate all flags from `main.go` (`--port`, `--password`, etc.) to `RootCmd.Flags()`.
          * **Config Loading:** Implement `PersistentPreRunE` to handle configuration loading (`config.LoadConfig`) and environment variable overrides *before* the command runs. This ensures `cfg` is available for all subcommands.
      * **Benefit:** This structure guarantees that running `./mediahub` without arguments executes the `RootCmd`, preserving v1.1 behavior.

### Step 2: Manual Migration System (Goose)

**Goal:** Replace hardcoded schema strings (`internal/repository/schema.go`) with versioned SQL files. Enforce a **"Check-Only"** policy on startup (server fails if DB is outdated).

1.  **Dependency:** `go get -u github.com/pressly/goose/v3`
2.  **Directory Structure:** Create `internal/db/migrations`.
3.  **Migration Files:**
      * **`001_init_v1_1.sql`:** Copy the `CREATE TABLE` SQL from `internal/repository/schema.go`.
          * **Critical:** Wrap every statement in `IF NOT EXISTS`. This makes the migration "idempotent," allowing v1.1 databases to apply it without error.
      * **`002_v1_2_schema.sql`:** Add new columns:
        ```sql
        ALTER TABLE databases ADD COLUMN entry_count INTEGER NOT NULL DEFAULT 0;
        ALTER TABLE databases ADD COLUMN total_disk_space_bytes INTEGER NOT NULL DEFAULT 0;
        ```
4.  **Repository Refactor (`internal/repository/repository.go`):**
      * Embed the migrations:
        ```go
        //go:embed ../db/migrations/*.sql
        var embedMigrations embed.FS
        ```
      * **Delete** `internal/repository/schema.go` and the `initDB()` method.
      * **Add Method `ValidateSchema() error`:**
          * Use `goose.SetBaseFS(embedMigrations)`.
          * Use `goose.GetDBVersion(db)` to check the DB state.
          * Compare against the latest embedded file version.
          * **Logic:**
              * If `DB < App`: Return error: *"Database schema is outdated (vX). Please run './mediahub migrate up' to update to vY."*
              * If `DB > App`: Return error: *"Application binary is too old for this database. Please upgrade."*
              * If `DB == App`: Return `nil` (Success).
5.  **CLI Integration:**
      * Call `repo.ValidateSchema()` inside the `RootCmd.RunE` (Server startup).
      * Create `internal/cli/migrate.go` with subcommands `up`, `down`, and `status` that wrap `goose.Up`, `goose.Down`, etc.

### Step 3: SQL Dialect Abstraction (Squirrel)

**Goal:** Prepare for PostgreSQL by removing raw string concatenation.

1.  **Dependency:** `go get -u github.com/Masterminds/squirrel`
2.  **Repository Update:**
      * In `internal/repository/repository.go`, add a builder to the struct:
        ```go
        type Repository struct {
            // ...
            Builder squirrel.StatementBuilderType
        }
        ```
      * Initialize it in `NewRepository`:
        ```go
        // Use 'Question' for SQLite. Later, this can be switched to 'Dollar' for Postgres.
        repo.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)
        ```
3.  **Refactor Queries (`internal/repository/query_repo.go`):**
      * Refactor `SearchEntries` to use the builder.
      * **Example Transformation:**
          * *Old:* `sbQuery.WriteString("SELECT * FROM " + tableName + " WHERE ...")`
          * *New:*
            ```go
            query := repo.Builder.Select("*").From(tableName)
            if filter != nil {
                // Apply nested logic using squirrel.And{}, squirrel.Or{}, squirrel.Eq{}
            }
            sql, args, err := query.ToSql()
            ```
      * **Guideline:** Do not try to refactor *every* simple query (like `GetByID`) immediately. Focus on dynamic queries (`SearchEntries`, `GetEntries`) first.

### Step 4: Auditor Interface

**Goal:** Decouple audit logging from the request path.

1.  **Interface Definition (`internal/services/interfaces.go`):**
    ```go
    type Auditor interface {
        // Log records an event. 'details' captures specific metadata (e.g., entry_id, file_size).
        Log(ctx context.Context, action string, actor string, resource string, details map[string]interface{})
    }
    ```
2.  **Implementation (`internal/audit/logger_auditor.go`):**
      * Implement the interface using the existing `logrus` logger.
      * Use `logrus.WithFields(details).Info(...)` to ensure structured JSON output.
3.  **Injection:**
      * Add `Auditor` to the `Handlers` struct in `internal/api/handlers/main.go`.
      * Update `cmd/mediahub/main.go` (or `internal/cli/root.go`) to inject the `LoggerAuditor`.
      * Update critical handlers (`CreateDatabase`, `DeleteEntry`, `UploadEntry`) to call `h.Auditor.Log(...)` upon success.

### Step 5: Recovery Command & Graceful Shutdown

**Goal:** Integrity checks and safe restarts.

1.  **Graceful Shutdown (`internal/cli/root.go`):**
      * Wrap `http.ListenAndServe` in a goroutine.
      * Use `signal.Notify` to listen for `os.Interrupt` and `syscall.SIGTERM`.
      * On signal, call `server.Shutdown(ctx)` with a 30-second timeout to allow active uploads to finish or abort cleanly.
2.  **Recovery Command (`internal/cli/recovery.go`):**
      * Create a new Cobra command `recovery`.
      * **Zombie Fix:**
          * Iterate all entry tables.
          * `UPDATE entries_X SET status='error' WHERE status='processing'`.
          * Log the count of fixed entries.
      * **Startup Check:** Ensure this command also runs `ValidateSchema` first. You cannot recover a database if the schema is invalid.

-----

## 📅 Phase 2: Efficiency (v1.3)

### Step 6: Bulk Operations

**Goal:** Execute multiple deletions in one transaction for performance.

1.  **API Handler:**
      * New Endpoint: `POST /api/database/entries/delete?name=MyImageDB`.
      * Payload: `{"ids": [101, 102, 103]}`.
2.  **Logic:**
      * Add `DeleteEntries(dbName string, ids []int64) error`.
      * **Optimization:**
          * Start Transaction.
          * Query the total filesize of all IDs in the list: `SELECT SUM(filesize) FROM ... WHERE id IN (...)`.
          * Delete rows: `DELETE FROM ... WHERE id IN (...)`.
          * Update `databases` stats: `entry_count = entry_count - len(ids)`, `total_disk_space = total_disk_space - totalSize`.
          * Commit.
      * *Note:* Even if file deletion on disk fails partially (e.g., file locked), the DB record should be removed to maintain consistency. Log disk errors as warnings.

### Step 7: Streaming Export

**Goal:** Low-memory ZIP export for multiple entries.

1.  **API Handler:**
      * New Endpoint: `POST /api/database/entries/export?name=MyAudioDB`.
      * Headers: `Content-Type: application/zip`, `Content-Disposition: attachment`.
      * Payload: `{"ids": [101, 102, 103]}`
2.  **Implementation Pattern:**
      * Use `io.Pipe()`.
      * **Goroutine:**
          * Create `zip.NewWriter(pipeWriter)`.
          * Query chosen entries from DB (using `rows.Next()` to stream results, do not load all into slice).
          * For each entry:
              * Open file on disk (`storageService`).
              * Create zip entry.
              * `io.Copy` from disk to zip.
          * Close zip writer.
          * Close pipe writer (with error if any occurred).
      * **Main Thread:**
          * `io.Copy(http.ResponseWriter, pipeReader)`.
      * **Error Handling:** If the zip process fails mid-stream, the HTTP response has already started (200 OK). The only way to signal error is to abort the stream (log error on server side).

-----

## 🧪 Testing Guidelines

  * **Unit Tests:** Since the logic is now heavily interface-based (`Auditor`, `Repository`, `Storage`), use `stretchr/testify/mock` to mock dependencies in Handler tests.
  * **Integration Tests:** For the `migrate` command, create a temporary SQLite file, apply migrations, assert table existence, and then delete the file.
  * **Safety:** When testing Bulk Delete, ensure your test setup creates temporary folders so you don't delete actual dev data.