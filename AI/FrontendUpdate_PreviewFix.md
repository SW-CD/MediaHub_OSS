# Feature Guide: Frontend Preview & Status Handling (v2)

## 1. Context & Backend Changes
The backend has been updated to fix a race condition during file uploads.
* **Previous Behavior:** Synchronous uploads immediately returned `status: "ready"`, even if the preview was still generating. This caused 404s on the image tag.
* **New Behavior:**
    * If `create_preview` is **enabled**: New entries return `status: "processing"`. A background worker generates the preview and then updates status to `"ready"`.
    * If `create_preview` is **disabled**: New entries return `status: "ready"` immediately.

**Frontend Goal:**
We need to update the UI to respect these states and provide a robust fallback for when an image fails to load (e.g., a 404 on a "ready" entry).

## 2. Visual State Logic

We will implement a priority system for displaying thumbnails in the Grid and List views.

| Entry Status | Image Load Result | **Visual Output** | Description |
| :--- | :--- | :--- | :--- |
| **`processing`** | N/A | **Spinner Icon** 🔄 | Backend is working. Do not attempt to load image. |
| **`error`** | N/A | **Error Icon** ⚠️ | Backend explicitly reported a failure (e.g., ffmpeg error). |
| **`ready`** | **Success** | **The Image** 🖼️ | Normal state. |
| **`ready`** | **Error (404)** | **"No Preview" Icon** 🚫 | Entry is valid, but preview file is missing or was never generated. |

## 3. Implementation Steps

### Step 1: Update `SecureImageDirective`
Currently, the directive logs errors to the console but doesn't tell the parent component. We need it to emit an event so the parent can switch to the "No Preview" placeholder.

* **File:** `src/app/directives/secure-image.directive.ts`
* **Changes:**
    1.  Add an `@Output() imageError = new EventEmitter<void>();`
    2.  In the `subscribe({ error: ... })` block of the HTTP request, emit `this.imageError.emit()`.
    3.  Ensure the directive doesn't keep the "loading" class if an error occurs.

### Step 2: Standardize Icons & Placeholders
We need consistent SVG icons for the different states.
* **Processing:** Use the existing spinner SVG logic (already in `entry-grid`).
* **Error (Backend):** Use an exclamation triangle SVG. This file is available as `frontend\src\assets\icons\error-icon.svg`
* **No Preview (404):** Use the file available as `frontend\src\assets\icons\no-preview-icon.svg`

### Step 3: Update `EntryGridComponent`
This component needs to track which specific image IDs have failed to load.

* **File:** `src/app/components/entry-grid/entry-grid.component.ts`
    * **Add Property:** `failedImageIds = new Set<number>();`
    * **Add Method:** `onImageError(id: number)` which adds the ID to the set.
    * **Update Method:** In `ngOnChanges` (or a setter for `entries`), clear the set to reset state when filters change.

* **File:** `src/app/components/entry-grid/entry-grid.component.html`
    * Refactor the `[ngSwitch="entry.status"]`.
    * **Case `ready`:**
        * Check `!failedImageIds.has(entry.id)`.
        * If true: Render `<img [secureSrc]... (imageError)="onImageError(entry.id)">`.
        * If false (it failed): Render the **"No Preview"** placeholder div.

### Step 4: Update `EntryListViewComponent`
Apply the same logic as the Grid component.

* **File:** `src/app/components/entry-list-view/entry-list-view.component.ts`
    * Add `failedImageIds` set and error handler.

* **File:** `src/app/components/entry-list-view/entry-list-view.component.html`
    * Update the "Preview" column logic to handle the 404 fallback.

### Step 5: Polling for "Processing" Items (Optional but Recommended)
The `DatabaseService` already has logic for `processingEntries$`. Ensure that the `EntryListComponent` (the parent) triggers a refresh of the list when the `DatabaseService` announces a processing entry has finished.
* *Current state check:* The `DatabaseService` seems to trigger `refreshNotifier`. Verify this correctly updates the grid without full page reload flickering.

## 4. Code Snippets

**SecureImageDirective Update:**
```typescript
@Output() imageError = new EventEmitter<void>();

// ... inside error block
this.renderer.removeClass(this.el.nativeElement, 'loading-image');
this.imageError.emit();
```

**Grid Template Logic (Pseudo-code):**

```html
<div [ngSwitch]="entry.status">
  
  <!-- PROCESSING -->
  <div *ngSwitchCase="'processing'" class="placeholder">
     <spinner-icon></spinner-icon>
  </div>

  <!-- ERROR (Backend) -->
  <div *ngSwitchCase="'error'" class="placeholder">
     <error-icon></error-icon>
  </div>

  <!-- READY -->
  <ng-container *ngSwitchCase="'ready'">
     <!-- If we haven't detected a load error yet -->
     <img *ngIf="!failedImageIds.has(entry.id)"
          [secureSrc]="url" 
          (imageError)="onImageError(entry.id)" />
     
     <!-- If load failed (404) -->
     <div *ngIf="failedImageIds.has(entry.id)" class="placeholder">
        <no-preview-icon></no-preview-icon>
        <span>No Preview</span>
     </div>
  </ng-container>

</div>
```
