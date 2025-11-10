# STAGE 1: The Builder
#
# This stage uses a full Go + Node.js image to build the Angular
# frontend and compile the Go backend, including CGO dependencies.
#
FROM golang:1.24-bookworm AS builder

# 1. Install dependencies for CGO (gcc) and Angular (nodejs/npm)
RUN apt-get update && apt-get install -y gcc nodejs npm

# 2. Set up the working directory and copy all source code
WORKDIR /app
COPY . .

# 3. Build the Angular frontend
# This will place the output in 'cmd/mediahub/frontend_embed/'
WORKDIR /app/frontend
RUN npm install
RUN npm run build -- --configuration production

# 4. Build the Go backend
WORKDIR /app
# We enable CGO for go-sqlite3 and build the binary
RUN CGO_ENABLED=1 go build -o /app/mediahub ./cmd/mediahub


#
# STAGE 2: The Final Production Image
#
# This stage uses a minimal Debian image, installs *only*
# runtime dependencies, and copies the built binary from Stage 1.
#
FROM debian:12-slim

# 1. Install runtime dependencies (FFmpeg, FFprobe, and SSL certs)
# ca-certificates is required for any HTTPS requests (e.g., webhooks)
RUN apt-get update && apt-get install -y ffmpeg ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# 2. Create the persistent folders for configuration and storage
# These will be set as volumes.
RUN mkdir /config
RUN mkdir /storage

# 3. Create a non-root user to run the application
# Running as 'root' in a container is a security risk.
RUN addgroup --system app && adduser --system --ingroup app app

# 4. Set up the application directory
WORKDIR /app

# 5. Copy the compiled binary from the 'builder' stage
# Also change its ownership to the new 'app' user
COPY --from=builder --chown=app:app /app/mediahub .

# 6. Set environment variables to point to our new folders
# This tells the app to create its DB in /config and store files in /storage
ENV FDB_DATABASE_PATH=/config/mediahub.db
ENV FDB_STORAGE_ROOT=/storage
ENV FDB_CONFIG_PATH=/config/config.toml

# 7. Switch to the non-root user
USER app

# 8. Expose the default port
EXPOSE 8080

# 9. Define mount points for persistent data
# This allows users to -v /path/on/host:/config
VOLUME ["/config", "/storage"]

# 10. Run the application
ENTRYPOINT ["./mediahub"]
