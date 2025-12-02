## 🔧 Configuration and CLI

The application uses the **Cobra** framework for CLI commands. Configuration is layered, with specific precedence rules to ensure flexibility across different deployment environments (Docker, Systemd, Local).

### 1. Configuration Precedence

The application resolves configuration values in the following order (highest priority first):

1.  **CLI Flags** (e.g., `--port 9090`)
2.  **Environment Variables** (e.g., `FDB_PORT=9090`)
3.  **Base Configuration File** (`config.toml`)
4.  **Application Defaults** (Hardcoded safety defaults)

---

### 2. CLI Commands & Flags

The application is structured around subcommands.

#### Root Command (Global Flags)
These flags apply to **all** subcommands (`serve`, `recovery`, `migrate`).

| Flag | Env Variable | Description | Default |
| :--- | :--- | :--- | :--- |
| `--config_path` | `FDB_CONFIG_PATH` | Path to the base TOML configuration file. | `config.toml` |
| `--log-level` | `FDB_LOG_LEVEL` | Logging verbosity (`debug`, `info`, `warn`, `error`). | `info` |

#### `serve` Command
Runs the main HTTP API server and the embedded Angular web interface.

```bash
./mediahub serve [flags]
```

| Flag | Env Variable | Description | Default |
| :--- | :--- | :--- | :--- |
| `--port` | `FDB_PORT` | The HTTP port to bind to. | `8080` |
| `--max-sync-upload` | `FDB_MAX_SYNC_UPLOAD` | RAM threshold for uploads (e.g., "8MB"). Larger files use disk. | `8MB` |
| `--password` | `FDB_PASSWORD` | The password for the 'admin' user (used on first run or with reset). | `""` |
| `--reset_pw` | `FDB_RESET_PW` | If `true`, resets the 'admin' password on startup to the one provided. | `false` |
| `--init_config` | `FDB_INIT_CONFIG` | Path to a TOML config file for one-time initialization of users/databases. | `""` |

#### `recovery` Command

**New in v1.2**: Runs maintenance tasks to fix data inconsistencies (e.g., after a power loss or crash). Does not start the HTTP server.

```bash
./mediahub recovery [flags]
```

  * **Zombie Fix:** Scans all database tables for entries stuck in `status: "processing"` (caused by interrupted async uploads) and marks them as `error`.
  * **Integrity Check (Planned):** Verifies that file records in SQLite have corresponding files on disk.

#### `migrate` Command

**New in v1.2**: Manually manages database schema versions. (Note: The `serve` command runs migrations automatically on startup, so this is rarely needed manually).

```bash
./mediahub migrate [status|up|down]
```

-----

### 3\. Base Configuration (`config.toml`)

On startup, the application looks for a `config.toml` file. This defines the persistent settings for the server.

**Example `config.toml`:**

```toml
[server]
host = "0.0.0.0"   # The host address to bind to
port = 8080        # Default port (can be overridden by flag/env)
max_sync_upload_size = "8MB" # Threshold for switching from RAM to Disk processing

[database]
path = "mediahub.db"      # Path to the SQLite database file
storage_root = "storage_root" # Root directory for file storage

[logging]
level = "info" # Logging level
# audit_enabled = false # (v1.2+) Enable SQL audit logging (Commercial) or specific Audit file (OSS)

[media]
# Optional: Path to the FFmpeg executable.
# If empty, the server will check the system PATH.
ffmpeg_path = ""

# Optional: Path to the FFprobe executable.
# If empty, the server will check near ffmpeg_path, then the system PATH.
ffprobe_path = ""

[jwt]
# Token expiration settings
access_duration_min = 5
refresh_duration_hours = 24
# Secret is auto-generated and saved here if missing
secret = "..."
```

-----

### 4\. One-Time Initialization (`--init_config`)

You can provide a *separate* TOML configuration file on startup using the `--init_config` flag (or `FDB_INIT_CONFIG` env var). The server will read this file and **create any users or databases that do not already exist**.

  * **Behavior:** It creates missing resources. It does **not** overwrite existing users or databases.
  * **Security:** After a successful run, the server will **attempt to overwrite the init config file** to remove the plaintext `password` fields. If this fails (permissions), a warning is logged.

**Example Init Config (`my-init.toml`):**

```toml
[[user]]
name = "Viewer"
roles = ["CanView"]
password = "StrongPassword"

[[user]]
name = "MaxMustermann"
roles = ["CanView", "CanCreate", "CanEdit", "CanDelete"]
password = "DifferentPassword"

[[database]]
name = "ImageDB1"
content_type = "image"
config = { convert_to_jpeg = true, create_previews = true }
housekeeping = {
    interval = "1h",
    disk_space = "100G",
    max_age = "365d"
}
# Custom metadata schema
custom_fields = [
    {name = "latitude", type = "REAL"},
    {name = "longitude", type = "REAL"},
    {name = "ml_score", type = "REAL"},
    {name = "sensor_id", type = "TEXT"},
    {name = "description", type = "TEXT"}
]

[[database]]
name = "Audio_Archive"
content_type = "audio"
config = { create_previews = true, auto_conversion = "flac" }
housekeeping = {
    interval = "24h",
    disk_space = "500G",
    max_age = "0" # Disable age-based cleanup
}
custom_fields = [
    {name = "source", type = "TEXT"}
]
```

-----

### 5\. Admin User Startup Logic

The application ensures an administrator account exists on every startup of the `serve` command.

1.  **Check for 'admin' user:** The server queries the `users` table for a user with `username = 'admin'`.
2.  **Case 1: 'admin' user does NOT exist (First Run)**
      * The server retrieves the password from `--password`, `FDB_PASSWORD`, or generates a **random 10-character string** if neither is set.
      * The random password is **printed to the console**.
      * A new user is created with `username: 'admin'`, the securely hashed password, and all roles set to `true`.
3.  **Case 2: 'admin' user exists**
      * The server checks for `--reset_pw=true` (or `FDB_RESET_PW=true`).
      * **If reset is true:**
          * The server *requires* a password via `--password` or `FDB_PASSWORD`. If missing, it exits with an error.
          * The existing 'admin' password is updated.
      * **If reset is false (default):**
          * No action is taken.
