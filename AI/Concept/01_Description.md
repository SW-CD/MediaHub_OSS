# General backend description

This document described the API and data layout of a simple database for **files (images, audio, and other binary data)**. The database stores files directly as separate files in folders in the filesystem, and related metadata in SQLite.
It should be implemented in Go and run an HTTP server which exposes REST endpoints. It also provides a simple web interface for browsing the **entries** or uploading a new **file**.

When files are uploaded, the server can be configured to automatically generate and store a small preview (e.g., a 200x200 pixel JPEG **thumbnail for an image, or a waveform image for audio**) for each original. This allows a frontend gallery to load quickly.

The server also supports advanced media processing, such as auto-converting audio files to different formats (e.g., FLAC). **This functionality has an optional dependency on the FFmpeg executable.** If FFmpeg is not available in the system's PATH, these features will be disabled, and the API will return an error if a user tries to enable them.

The API is also documented in Swagger, available under /swagger/index.html . Endpoints that return arrays of data should return an empty array rather than `null` if returning no data.

### Security & Authentication
The application implements a **hybrid authentication model**. It supports:
1.  **Basic Authentication:** Standard username/password for simple clients or initial login.
2.  **JSON Web Tokens (JWT):** A stateful token system with Access and Refresh tokens, allowing for secure sessions and server-side logout (revocation).