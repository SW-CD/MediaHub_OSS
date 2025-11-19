# Development Guide: File Drag & Drop Implementation

## 1\. 🎯 Objective

Enable users to drag and drop files directly onto the Entry List/Grid views or the Upload Modal.

  * **List/Grid View:** Dropping a file validates the MIME type against the current database. If valid, it opens the **Upload Entry Modal** with the file pre-selected.
  * **Upload Modal:** Dropping a file inside the open modal replaces the currently selected file.

## 2\. 🏗️ Architectural Strategy

To maintain the project's clean architecture, we will avoid cluttering components with DOM event listeners. Instead, we will use an **Angular Directive**.

1.  **`FileDragDropDirective`:** A standalone directive that handles `dragover`, `dragleave`, and `drop` events. It emits a `fileDropped` event when a valid file is released.
2.  **Component Integration:** We will attach this directive to the main container in `EntryListComponent` and the form container in `UploadEntryModalComponent`.
3.  **Data Passing:** We will leverage the existing `ModalService` to pass the dropped file object into the modal when it opens.

-----

## 3\. 🛠️ Implementation Steps

### Step 1: Centralize MIME Type Logic

Currently, MIME type validation logic exists in the backend and partially in the `UploadEntryModal`. We should move the MIME type mapping to a shared helper to ensure the "Drop Zone" validation matches the "File Picker" validation.

**Action:** Update `frontend/src/app/models/enums.ts` (or create a new `utils/mime-types.ts`).

```typescript
// Suggestion for frontend/src/app/utils/mime-types.ts
import { ContentType } from '../models/enums';

export const ALLOWED_MIME_TYPES: Record<ContentType, string[]> = {
  [ContentType.Image]: ['image/jpeg', 'image/png', 'image/gif', 'image/webp'],
  [ContentType.Audio]: ['audio/mpeg', 'audio/wav', 'audio/flac', 'audio/opus', 'audio/ogg', 'application/ogg', 'audio/x-flac'],
  [ContentType.File]: [] // Empty array means "Allow All"
};

export function isMimeTypeAllowed(contentType: ContentType, mimeType: string): boolean {
  const allowed = ALLOWED_MIME_TYPES[contentType];
  if (!allowed || allowed.length === 0) return true; // Allow all if list is empty
  return allowed.includes(mimeType);
}
```

### Step 2: Create the Directive

Create a new directive `frontend/src/app/directives/file-drag-drop.directive.ts`.

**Key Responsibilities:**

1.  **`@HostBinding('class.file-drag-over')`**: Toggles a CSS class when a file is being dragged over the element.
2.  **`@HostListener('dragover')`**: Prevent default behavior and set the flag to true.
3.  **`@HostListener('dragleave')`**: Set the flag to false.
4.  **`@HostListener('drop')`**: Prevent default, get `files[0]`, and emit it via `@Output()`.

<!-- end list -->

```typescript
@Directive({
  selector: '[appFileDragDrop]',
  standalone: true
})
export class FileDragDropDirective {
  @Output() fileDropped = new EventEmitter<File>();
  @HostBinding('class.file-drag-over') fileOver: boolean = false;

  // ... implementations for dragover, dragleave, drop
}
```

**Don't forget to register this Directive in `AppModule` imports.**

### Step 3: Update Entry List Component (Main Drop Zone)

We want the entire list/grid area to be a drop zone.

**File:** `frontend/src/app/components/entry-list/entry-list.component.html`

1.  Add the directive to the root container `div.image-list-container`.
2.  Bind the event: `(fileDropped)="onFileDropped($event)"`.

<!-- end list -->

```html
<div class="image-list-container" 
     *ngIf="currentUser$ | async as user"
     appFileDragDrop 
     (fileDropped)="onFileDropped($event)">
     </div>
```

**File:** `frontend/src/app/components/entry-list/entry-list.component.ts`

Implement `onFileDropped(file: File)`:

1.  Check if `this.currentDb` is loaded.
2.  Use the helper from Step 1 to validate `file.type` against `this.currentDb.content_type`.
3.  If invalid, show an error via `NotificationService`.
4.  If valid, open the modal **passing the file as data**:

<!-- end list -->

```typescript
// Inside EntryListComponent
onFileDropped(file: File): void {
  if (!this.currentDb) return;

  if (!isMimeTypeAllowed(this.currentDb.content_type, file.type)) {
    this.notificationService.showError(`Invalid file type. Allowed: ${this.currentDb.content_type}`);
    return;
  }

  // Pass the file to the modal
  this.modalService.open(UploadEntryModalComponent.MODAL_ID, { droppedFile: file });
}
```

### Step 4: Update Upload Modal to Receive Data

The modal needs to handle receiving a file *on open* (from the list view) AND handling a file dropped *directly onto it*.

**File:** `frontend/src/app/components/upload-entry-modal/upload-entry-modal.component.ts`

1.  **Receive Data:** Update `ngOnInit`. Subscribe to `modalService.getModalEvents()`. Check if `event.data` contains `droppedFile`.
      * *Note:* Currently, your `ngOnInit` only listens to `databaseService`. You need to add the `modalService` subscription similar to `UserFormComponent`.

<!-- end list -->

```typescript
ngOnInit(): void {
  // Existing DB subscription...

  // NEW: Listen for modal open events to check for passed data
  this.modalService.getModalEvents(UploadEntryModalComponent.MODAL_ID)
    .pipe(takeUntil(this.destroy$))
    .subscribe(event => {
      if (event.action === 'open' && event.data?.droppedFile) {
        this.handleFile(event.data.droppedFile);
      }
    });
}

handleFile(file: File): void {
  this.selectedFile = file;
  this.selectedFileName = file.name;
  this.uploadForm.patchValue({ file: this.selectedFile });
  this.uploadForm.get('file')?.markAsTouched();
}
```

2.  **Direct Drop:** Add the `appFileDragDrop` directive to the `<form>` or a specific `div` in the HTML template.
3.  Implement `onFileDropped` in this component to call `handleFile`.

**File:** `frontend/src/app/components/upload-entry-modal/upload-entry-modal.component.html`

```html
<div *ngIf="currentDatabase && uploadForm" 
     appFileDragDrop 
     (fileDropped)="handleFile($event)">
  <form ...>
    </form>
</div>
```

### Step 5: Styling (CSS)

Add global styles for the drag-over state to give user feedback.

**File:** `frontend/src/styles.css`

```css
/* Visual cue when dragging a file over a drop zone */
.file-drag-over {
  position: relative;
}

.file-drag-over::after {
  content: "Drop file to upload";
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(66, 153, 225, 0.9); /* Primary Blue with opacity */
  color: white;
  display: flex;
  justify-content: center;
  align-items: center;
  font-size: 1.5rem;
  font-weight: bold;
  z-index: 5000; /* Ensure it's on top */
  border: 4px dashed white;
  border-radius: 0.5rem;
  pointer-events: none; /* Let the drop event pass through to the element */
}
```

## 4\. 🧪 Testing Checklist

1.  **Validation Test:**
      * Go to an **Image** database.
      * Drag a `.txt` file onto the grid.
      * **Expected:** Error toast "Invalid file type". Modal does *not* open.
2.  **List/Grid Drop Test:**
      * Go to an **Image** database.
      * Drag a `.jpg` file onto the grid.
      * **Expected:** Upload Modal opens. "File" input shows the filename. Validations pass.
3.  **Modal Drop Test:**
      * Open the Upload Modal manually (via FAB).
      * Drag a file onto the modal form.
      * **Expected:** The file input updates to the new file.
4.  **Generic File DB:**
      * Go to a "File" database.
      * Drag *any* file type.
      * **Expected:** Modal opens successfully (since all types are allowed).