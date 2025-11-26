# Build the application (backend and frontend) as a docker image
# -> run in the project root folder `docker build -t mediahub:1.1.0 -f ./docker/Dockerfile .`
# Start a container with, e.g., 
# docker run -d \
#   --name mediahub \
#   -p 8080:8080 \
#   -v $(pwd)/mediahub_data:/config \
#   -v $(pwd)/mediahub_storage:/storage \
#   -e FDB_PASSWORD="admin-password-123" \
#   mediahub:1.1.0

# STAGE 1: The Builder
# We use a multi-stage approach to grab Node binaries cleanly
FROM node:22-bookworm AS node_source
FROM golang:1.24-bookworm AS builder

# Prevent generic apt warnings
ARG DEBIAN_FRONTEND=noninteractive

# 1. Install minimal build dependencies
# We assume gcc and standard libs are enough.
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# 2. Install Node.js by copying from the official image
# This avoids the "apt does not have a stable CLI interface" warning completely
COPY --from=node_source /usr/local/lib/node_modules /usr/local/lib/node_modules
COPY --from=node_source /usr/local/bin/node /usr/local/bin/node
RUN ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm && \
    ln -s /usr/local/lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx

WORKDIR /app

# 3. Cache Go Modules
COPY go.mod go.sum ./
RUN go mod download

# 4. Cache Node Modules
COPY frontend/package*.json ./frontend/
WORKDIR /app/frontend
# 'npm ci' is faster and cleaner for builds
RUN npm ci --no-audit --no-fund

# 5. Build Frontend
WORKDIR /app
COPY . .
WORKDIR /app/frontend
RUN npm run build -- --configuration production

# 6. Build Backend
WORKDIR /app
# -trimpath removes file system paths from the binary (security + size)
# -ldflags="-s -w" strips debug info (size)
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /app/mediahub ./cmd/mediahub


# STAGE 2: The Final Production Image
FROM debian:12-slim

ARG DEBIAN_FRONTEND=noninteractive

# 1. Install basics + Download Static FFmpeg
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    xz-utils \
    && rm -rf /var/lib/apt/lists/* && \
    # Detect architecture
    ARCH=$(dpkg --print-architecture) && \
    # Map debian arch to John Van Sickle arch naming if necessary (usually they match for amd64/arm64)
    echo "Downloading FFmpeg for architecture: $ARCH" && \
    curl -L -o ffmpeg.tar.xz "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-${ARCH}-static.tar.xz" && \
    tar -xJf ffmpeg.tar.xz && \
    # Move binaries
    mv ffmpeg-*-static/ffmpeg /usr/local/bin/ && \
    mv ffmpeg-*-static/ffprobe /usr/local/bin/ && \
    # Cleanup
    rm -rf ffmpeg.tar.xz ffmpeg-*-static

# 2. Create user
RUN addgroup --system app && adduser --system --ingroup app app

# 3. Setup folders
RUN mkdir /config /storage && \
    chown -R app:app /config /storage

# 4. Copy artifacts
WORKDIR /app
COPY --from=builder --chown=app:app /app/mediahub .
COPY --from=builder --chown=app:app /app/config.toml /config/config.toml

# 5. Config
ENV FDB_DATABASE_PATH=/config/mediahub.db \
    FDB_STORAGE_ROOT=/storage \
    FDB_CONFIG_PATH=/config/config.toml

USER app
EXPOSE 8080
VOLUME ["/config", "/storage"]
ENTRYPOINT ["./mediahub"]
