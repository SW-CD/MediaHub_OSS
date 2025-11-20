# Build the application (backend and frontend) as a docker image
# run in the project root folder `docker build -t mediahub:1.1.0 -f ./docker/Dockerfile .`

# STAGE 1: The Builder
# Docker will automatically pull the golang image matching your host architecture (amd64 or arm64)
FROM golang:1.24-bookworm AS builder

# 1. Install build dependencies
RUN apt-get update && apt-get install -y curl gcc
# NodeSource script detects arch automatically
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get install -y nodejs

WORKDIR /app

# 2. Cache Go Modules
COPY go.mod go.sum ./
RUN go mod download

# 3. Cache Node Modules
COPY frontend/package*.json ./frontend/
WORKDIR /app/frontend
RUN npm install

# 4. Build Frontend
WORKDIR /app
COPY . .
WORKDIR /app/frontend
RUN npm run build -- --configuration production

# 5. Build Backend
WORKDIR /app
# We leave GOARCH auto-detected so it matches the container's arch
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /app/mediahub ./cmd/mediahub


# STAGE 2: The Final Production Image
FROM debian:12-slim

# 1. Install basics + Download Static FFmpeg
# We use 'dpkg --print-architecture' to dynamically grab 'amd64' or 'arm64'
RUN apt-get update && apt-get install -y ca-certificates curl xz-utils && \
    rm -rf /var/lib/apt/lists/* && \
    # Detect architecture (returns 'amd64' or 'arm64')
    ARCH=$(dpkg --print-architecture) && \
    echo "Downloading FFmpeg for architecture: $ARCH" && \
    # Download specific version based on detection
    curl -L -o ffmpeg.tar.xz "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-${ARCH}-static.tar.xz" && \
    tar -xJf ffmpeg.tar.xz && \
    # Move binaries (wildcard handles the variable folder names)
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
ENV FDB_DATABASE_PATH=/config/mediahub.db
ENV FDB_STORAGE_ROOT=/storage
ENV FDB_CONFIG_PATH=/config/config.toml

USER app
EXPOSE 8080
VOLUME ["/config", "/storage"]
ENTRYPOINT ["./mediahub"]