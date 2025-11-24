# Feature Guide: Intelligent Preview Handling

## 1. Context & Problem Statement
The backend utilizes a "Hybrid Upload Strategy." For small files, it processes them synchronously and returns a `201 Created` status with `status: "ready"` immediately. However, the generation of the preview thumbnail happens in a background goroutine.

**The Race Condition:**
1.  Frontend receives `status: "ready"`.
2.  Frontend immediately requests the preview URL.
3.  Backend has not finished writing the JPEG to disk yet.
4.  Backend returns `404 Not Found`.
5.  Frontend displays a broken image icon.

We need a frontend-only solution that "smooths over" this race condition without adding blocking logic to the API. Additionally, we must respect the database configuration (some databases have preview generation disabled entirely).

## 2. The Logic Matrix
The frontend will decide what to render based on four factors:
1.  **DB Config:** Is `create_preview` enabled for this database?
2.  **Entry Status:** Is the entry `processing` or `ready`?
3.  **Session History:** Was this entry uploaded during the current user session?
4.  **Network Result:** Did the preview image request succeed or 404?

| `create_preview` | Entry Status | Is Recent Upload? | Image Request | **UI Result** |
| :--- | :--- | :--- | :--- | :--- |
| **False** | Any | Any | N/A | **Generic Icon** (File/Audio/Image type icon) |
| **True** | `processing` | N/A | N/A | **Spinner** (Backend is explicitly busy) |
| **True** | `ready` | **Yes** | **Error (404)** | **Spinner** (Assume race condition; wait for user refresh) |
| **True** | `ready` | No | **Error (404)** | **"No Preview" Placeholder** (Generation permanently failed/lost) |
| **True** | `ready` | Any | **Success** | **The Preview Image** |

## 3. Implementation Guide

### Step 1: Centralized State Tracking (`DatabaseService`)
We need to track which IDs were uploaded in the current session to distinguish "just uploaded" files from "broken old files."

* **File:** `src/app/services/database.service.ts`
* **Action:**
    * Add a private property: `private recentUploads = new Set<number>();`
    * Add method: `markAsRecentUpload(id: number): void`
    * Add method: `isRecentUpload(id: number): boolean`

### Step 2: Flagging New Entries (`UploadEntryModalComponent`)
When an upload is successful, flag the ID immediately.

* **File:** `src/app/components/upload-entry-modal/upload-entry-modal.component.ts`
* **Action:**
    * In `onSubmit()`, inside the `next` callback for `uploadEntry`.
    * If status is `201` (Created) or `202` (Accepted), call `this.databaseService.markAsRecentUpload(response.id)`.

### Step 3: Reactive Image Loading (`SecureImageDirective`)
Since we fetch images via `HttpClient` (to attach Bearer tokens), standard `<img>` error events do not fire reliably for 404s on the blob request.

* **File:** `src/app/directives/secure-image.directive.ts`
* **Action:**
    * Add `@Output() imageLoadError = new EventEmitter<void>();`
    * In the `http.get(...).subscribe({ error: ... })` block, emit this event. This allows the parent component to react specifically to authorization or 404 errors during the blob fetch.

### Step 4: Updating View Logic (`EntryList` & `EntryGrid`)
The components currently assume that if `status === 'ready'`, the image exists. We must decouple this.

* **Files:**
    * `src/app/components/entry-list-view/entry-list-view.component.ts` (and `.html`)
    * `src/app/components/entry-grid/entry-grid.component.ts` (and `.html`)
* **Logic Updates:**
    * **Inputs:** Add `@Input() createPreview: boolean = true;` to receive DB config.
    * **State:** Add a local tracker for load errors (e.g., `failedPreviewIds: Set<number>`).
    * **Method:** Implement `onImageError(entry: Entry)`:
        * Check `databaseService.isRecentUpload(entry.id)`.
        * If recent, the UI state remains "loading" (Spinner).
        * If NOT recent, the UI state becomes "error" (Placeholder).
* **Template Updates:**
    * Refactor the `ngSwitch` logic.
    * **Priority 1:** If `!createPreview` -> Show Generic Icon.
    * **Priority 2:** If `status === 'processing'` -> Show Spinner.
    * **Priority 3:** If `status === 'ready'`:
        * Render `<img>` with `[secureSrc]` and `(imageLoadError)`.
        * If the image previously errored (check local state):
            * Show Spinner (if Recent).
            * Show "No Preview" (if Old).

### Step 5: Parent Controller Update (`EntryList`)
Pass the configuration down to the children.

* **File:** `src/app/components/entry-list/entry-list.component.html`
* **Action:**
    * Pass `[createPreview]="currentDb?.config?.create_preview ?? true"` to `<app-entry-grid>` and `<app-entry-list-view>`.

## 4. Summary of Visual States
1.  **Spinner:** Used for `status='processing'` OR (`status='ready'` + Recent Upload + 404).
2.  **Generic Icon:** Used when database `create_preview = false`.
3.  **Broken Image / No Preview Icon:** Used for (`status='ready'` + Old Upload + 404).
4.  **Actual Image:** Used for `status='ready'` + 200 OK.