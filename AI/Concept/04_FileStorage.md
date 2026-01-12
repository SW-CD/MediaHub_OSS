## File storage

The file storage system is designed to be simple and directly mirror the database structure. Each database created in SQLite or PostgreSQL corresponds to a main folder on the server's filesystem.

  * **Database Folder:** When a new database is created (e.g., `MyAudioDatabase`), a corresponding folder with the same name (`MyAudioDatabase/`) is created in a main storage root directory.
  * **Entry Filename:** Files are stored using their unique `id` (from the `entries_[database_name]` table) as the filename, **with no file extension**. The file's true format is tracked by the `mime_type` column in the database.
  * **Directory Structure (Year/Month):** To ensure good filesystem performance, files are stored in nested subfolders based on their `timestamp`. The directory structure is `YYYY/MM`.
  * **Folder Creation:** When a new file is uploaded via `POST /api/entry`, the system reads its `timestamp`, determines the correct year and month, and then checks if the corresponding folder path (e.g., `MyAudioDatabase/2025/10`) exists. If the folder path (including the year folder) does not exist, the system creates it automatically before saving the file.

For example, an entry with:

  * `id`: **10232**
  * `database_name`: **MyAudioDatabase**
  * `mime_type`: **audio/flac**
  * `timestamp`: **1760713653** (which is in October 2025)

...will be saved at the following file path:
`.../storage_root/MyAudioDatabase/2025/10/10232`

In the future, a MinIO/S3 compatible API is supported.

### Preview Storage

If `create_preview` is enabled for the database, a parallel directory structure is used to store the previews.

  * **Preview Root:** Previews are stored in a separate root folder (e.g., `.../storage_root/previews/`).
  * **Preview Path:** The path mirrors the original file's path: `previews/DatabaseName/YYYY/MM/`. This path is also created automatically.
  * **Preview File:** The preview file is **always** a JPEG, and it is saved using only its `id` as the filename.
  * **Generation Logic:**
      * **Image:** A JPEG thumbnail, fit within 200x200 pixels, preserving aspect ratio.
      * **Audio:** A JPEG waveform image (e.g., 200x120 pixels).
      * **File:** No preview is generated.

Using the same example entry (`id: 10232`), its audio waveform preview would be saved at:
`.../storage_root/previews/MyAudioDatabase/2025/10/10232`