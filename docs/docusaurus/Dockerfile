# ==============================================================================
# Docusaurus Documentation Container
# ==============================================================================
# How to build the image (run from project root directory):
#   docker build -t mediahub-docs -f docs/docusaurus/Dockerfile .
#
# How to run the container:
#   docker run -d -p 8081:80 mediahub-docs
# ==============================================================================

# Stage 1: Build the static Docusaurus HTML/JS site
FROM node:24-alpine AS builder

WORKDIR /app

# Copy package files and install dependencies
COPY docs/docusaurus/package*.json ./docs/docusaurus/
WORKDIR /app/docs/docusaurus
RUN npm install

# Copy markdown docs, logo asset, and docusaurus source code
COPY docs/manual /app/docs/manual
COPY docs/docusaurus /app/docs/docusaurus
COPY frontend/src/assets/logo.svg /app/docs/docusaurus/static/img/logo.svg

# Build static output to /app/docs/docusaurus/build
RUN npm run build

# Stage 2: Serve pre-rendered documentation via Nginx
FROM nginx:alpine

# Copy static assets from build stage to nginx web root
COPY --from=builder /app/docs/docusaurus/build /usr/share/nginx/html

# Expose HTTP port 80
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
