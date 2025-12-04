# General backend description

This document describes the API and data layout of a robust, industrial-grade database for **files (images, audio, and other binary data)**. The database stores files directly as separate files in folders in the filesystem, and related metadata in SQLite.

It is implemented in Go as a modular application. It exposes a **CLI (Command Line Interface)** for various operations:
1.  **Default (No command)**: Runs the HTTP REST API server and the embedded Angular web interface.
2.  **`recovery`**: Runs maintenance tasks to fix inconsistent states (e.g., after a power loss).
3.  **`migrate`**: Manages database schema versions.

### Core Features
* **Media Processing:** Auto-conversion (e.g., Audio to FLAC) and preview generation (Thumbnails/Waveforms).
* **Streaming I/O:** Efficiently handles large file uploads and dataset exports using streaming to minimize RAM usage.
* **Architecture First:** The backend uses strict interface decoupling (Auditor, Repository, Storage) to verify extensibility for future commercial backends (PostgreSQL, S3) while keeping the OSS version lightweight (SQLite, Disk).

### Security & Authentication
The application implements a **hybrid authentication model**. It supports:
1.  **Basic Authentication:** Standard username/password for simple clients or initial login.
2.  **JSON Web Tokens (JWT):** A stateful token system with Access and Refresh tokens, allowing for secure sessions and server-side logout (revocation).