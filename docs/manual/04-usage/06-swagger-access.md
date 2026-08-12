---
sidebar_position: 6
title: Interactive Swagger API Documentation
---

# Interactive Swagger API Documentation

MediaHub automatically serves an interactive OpenAPI 2.0 / Swagger UI directly from the running web server.

---

## 🚀 Accessing Swagger UI

You can access the interactive Swagger documentation in two ways:

1. **Via Web UI Button**:
   * Log into the MediaHub Web Interface.
   * Click the **API Documentation** (or Swagger) button located in the top navigation header / account menu.
2. **Direct Browser URL**:
   * Navigate directly to:
     ```text
     http://localhost:8080/swagger/index.html
     ```

---

## ⚡ Using Swagger UI

1. **Interactive Endpoint Explorer**: Expand any endpoint category to inspect request payloads, headers, query parameters, and response schemas.
2. **Authentication**:
   * Click the **Authorize** button at the top right of the Swagger interface.
   * Enter your JWT access token (`Bearer <token>`), Basic Auth credentials, or API Key (`Bearer srv_...`).
3. **Regenerating OpenAPI Specifications** (For Developers):
   If backend handler Go code is modified, regenerate the swagger specification prior to compiling:
   ```bash
   swag init -g ./cmd/mediahub/main.go
   ```
