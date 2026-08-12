---
sidebar_position: 3
title: Uploading Media Entries
---

# Uploading Media Entries

Entries can be uploaded via the Angular Web UI or programmatically via HTTP REST endpoints.

---

## 💻 Uploading via Web UI

1. Open a database from the main dashboard.
2. Click **Upload Entry** or **Bulk Upload**.
3. Select media files from your machine.
4. Fill in custom metadata values defined for the target database.
5. Click **Submit Upload**.

---

## ⚡ Synchronous vs. Asynchronous Upload Threshold

MediaHub optimizes memory and disk utilization during uploads using the `max_sync_upload_size` configuration setting (default `4MB`):

* **Synchronous Processing (File Size ≤ `max_sync_upload_size`)**:
  * Processed entirely in RAM for maximum throughput.
  * Transcoding and preview creation occur synchronously before returning the API HTTP response.
* **Asynchronous Processing (File Size > `max_sync_upload_size`)**:
  * Streamed to temporary disk storage.
  * Transcoding runs in a background worker pool (`n_max_queued` configuration).
  * Returns an HTTP `202 Accepted` response immediately while processing completes asynchronously.

---

## 📤 Programmatic API Upload Example

Upload a file with custom metadata using `curl`:

```bash
curl -X POST "http://localhost:8080/api/database/01HGFB9Z5W7ABCDEFGHJKMNPQR/entry" \
  -H "Authorization: Bearer srv_your_api_key_here" \
  -F "file=@/path/to/camera_capture.jpg" \
  -F 'metadata={"latitude": 49.6116, "longitude": 6.1319, "defect_detected": false}'
```
