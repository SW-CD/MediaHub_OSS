---
sidebar_position: 2
title: Database Management
---

# Database Management

MediaHub organizes media assets into logical **Databases**. Each database defines a content media type, auto-conversion rules, housekeeping cleanup schedules, and custom metadata fields.

---

## ➕ Creating a Database

1. Click **Create Database** on the Databases dashboard.
2. Configure the database parameters:

### 1. General Configuration
* **Name**: Display label for the database (1–100 characters).
* **Content Type**:
  * `image`: Image assets.
  * `audio`: Audio tracks and microphone recordings.
  * `video`: Video clips.
  * `generic`: Unstructured files or arbitrary binary documents.

### 2. Processing & Transcoding
* **Create Previews**: Check to generate web thumbnails (`.webp` / `.mp4`) automatically on entry ingestion.
* **Auto Conversion**:
  * Images: `jpeg`, `webp`, `png`, or original.
  * Audio: `mp3`, `flac`, `wav`, or original.
  * Video: `mp4`, `webm`, or original.

### 3. Housekeeping Configuration
Background housekeeping runs periodically per database based on these rules:
* **Interval**: Run frequency (e.g. `1h` for hourly, `24h` for daily).
* **Disk Space Limit**: Max disk space allocation (e.g. `100G`, `1T`). Set to `0` to disable space-based purging.
* **Max Entry Age**: Max age of entries before deletion (e.g. `30d`, `365d`). Set to `0` (default) to disable age-based purging.

---

## 🏷️ Custom Metadata Fields

You can attach typed custom fields to a database. Every entry uploaded to the database will store these fields, and MediaHub indexes them for high-performance querying.

### Available Field Types:
* `TEXT`: String metadata (e.g. camera location, operator name, description).
* `INTEGER`: Whole numbers (e.g. sample count, error code).
* `REAL`: Floating-point values (e.g. confidence score, temperature, latitude).
* `BOOLEAN`: Flag (`true`/`false`).

### Indexing:
Check **Is Indexed** when defining a custom field to enable fast index-accelerated filtering (e.g., `confidence_score > 0.85`).
