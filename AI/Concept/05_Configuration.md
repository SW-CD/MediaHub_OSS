## 🔧 Configuration and Startup

The application is configured via command-line (CLI) flags, environment variables, and TOML configuration files. CLI flags take precedence over environment variables, which in turn override values loaded from the `config.toml` file.

User management is handled via API endpoints, with the initial admin user being provisioned at startup by the `UserService`.

### 1\. Configuration Files

The server uses two types of configuration files:

  * **Base Configuration (`config.toml`)**: Loaded on every startup. This file defines the base settings for the server, such as the port and database paths.
  * **Initialization Configuration (`--init_config`)**: An optional file, specified by a flag, that is run *only once*. It is used to create users and databases that do not already exist, making it ideal for automated deployments.

### 2\. Command-Line Interface

The executable accepts the following flags:

  * `--help` or `help`: Prints a short description of the app's functionality and all available options.
  * `--port` (int): The port for the HTTP server. (Overrides `config.toml` and `IMS_PORT`).
  * `--log-level` (string): The logging level (`debug`, `info`, `warn`, `error`). (Overrides `config.toml` and `IMS_LOG_LEVEL`).
  * `--password` (string): The password for the 'admin' user (used on first run or with `--reset_pw`). (Overrides `IMS_PASSWORD`).
  * `--reset_pw` (bool): If `true`, resets the 'admin' password on startup to the one provided. (Overrides `IMS_RESET_PW`).
  * `--config_path` (string): Path to the base TOML configuration file. (Default: `config.toml`).
  * `--init_config` (string): Path to a TOML config file for one-time initialization of users/databases. (Default: `""`).

### 3\. Environment Variables

If a CLI flag is not provided, the application will check for these environment variables:

  * `IMS_PORT`: See `--port`.
  * `IMS_LOG_LEVEL`: See `--log-level`.
  * `IMS_PASSWORD`: See `--password`.
  * `IMS_RESET_PW`: See `--reset_pw` (e.g., `IMS_RESET_PW=true`).
  * `DATABASE_PATH`: The path to the SQLite database file. (Overrides `config.toml`).
  * `STORAGE_ROOT`: The root directory where files will be stored. (Overrides `config.toml`).
  * `IMS_CONFIG_PATH`: See `--config_path`.

### 4\. Config File Initialization (`--init_config`)

You can provide a TOML configuration file on startup using the `--init_config` flag. The server will read this file and **create any users or databases that do not already exist**.

  * This process **will not overwrite** existing users or databases.
  * After a successful run, the server will **attempt to overwrite the config file** to remove the plaintext `password` fields for security.
  * If this write fails (e.g., due to file permissions), the server will log a warning and continue, but you should **manually secure the file** to remove the passwords.

**Example `config.toml`:**

```toml
[[user]]
name = "Viewer"
roles = ["CanView"]
password = "StrongPassword"

[[database]]
name = "ImageDB1"
content_type = "image"
config = { convert_to_jpeg = true, create_previews = true }
housekeeping = {
    interval = "1h",
    disk_space = "100G",
    max_age = "365d"
}
custom_fields = [
    {name = "latitude", type = "REAL"},
    {name = "longitude", type = "REAL"}
]
```

### 5\. Admin User Startup Logic

The application follows a specific sequence on startup to ensure an administrator account exists. This logic is now handled by the `UserService`.

1.  **Check for 'admin' user:** The server queries the `users` table for a user with `username = 'admin'`.
2.  **Case 1: 'admin' user does NOT exist (First Run)**
      * The server retrieves the password from `--password`, `IMS_PASSWORD`, or generates a random 10-character string if neither is set.
      * The random password will be printed to the console (e.g., `INFO: No admin user found. Created 'admin' with password: 'aXbY12cZ34'`).
      * A new user is created with `username: 'admin'`, the securely hashed password, and all roles set to `true`.
3.  **Case 2: 'admin' user exists**
      * The server checks for `--reset_pw=true` or `IMS_RESET_PW=true`.
      * **If reset is true:**
          * The server *requires* a password to be provided via `--password` or `IMS_PASSWORD`. If neither is set, the server will exit with an error.
          * The existing 'admin' user's `password_hash` is updated with the new, securely hashed password.
      * **If reset is false (default):**
          * No action is taken. The 'admin' user's existing password remains unchanged.
