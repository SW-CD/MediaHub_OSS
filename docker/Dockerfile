# STAGE 1: The Builder
FROM golang:1.24-bookworm AS builder

# 1. Install build tools
RUN apt-get update && apt-get install -y gcc nodejs npm

WORKDIR /app

# 2. CACHE OPTIMIZATION: Download Go dependencies first
COPY go.mod go.sum ./
RUN go mod download

# 3. CACHE OPTIMIZATION: Install Node.js dependencies first
COPY frontend/package*.json ./frontend/
WORKDIR /app/frontend
RUN npm install

# 4. Copy the rest of the source code
WORKDIR /app
COPY . .

# 5. Build the Angular frontend
WORKDIR /app/frontend
RUN npm run build -- --configuration production

# 6. Build the Go backend
WORKDIR /app
RUN CGO_ENABLED=1 go build -o /app/mediahub ./cmd/mediahub


# STAGE 2: The Final Production Image
FROM debian:12-slim

# 1. Install runtime dependencies
RUN apt-get update && apt-get install -y ffmpeg ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# 2. Create user
RUN addgroup --system app && adduser --system --ingroup app app

# 3. Setup folders & Permissions
# We chown these folders so the 'app' user can write the DB file and uploads
RUN mkdir /config /storage && \
    chown -R app:app /config /storage

# 4. Copy binary
WORKDIR /app
COPY --from=builder --chown=app:app /app/mediahub .

# 5. COPY CONFIGURATION (The missing step)
# We place it in /config/ because that is where the ENV var points.
COPY --from=builder --chown=app:app /app/config.toml /config/config.toml

# 6. Environment Config
ENV FDB_DATABASE_PATH=/config/mediahub.db
ENV FDB_STORAGE_ROOT=/storage
ENV FDB_CONFIG_PATH=/config/config.toml

# 7. Finalize
USER app
EXPOSE 8080
# Declare volumes AFTER copying files and setting permissions
VOLUME ["/config", "/storage"]
ENTRYPOINT ["./mediahub"]