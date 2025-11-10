// frontend/src/app/components/entry-list-view/entry-list-view.component.ts
import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy } from '@angular/core';
// UPDATED: Image to Entry
import { Entry, User } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { CommonModule, DatePipe, DecimalPipe } from '@angular/common'; // <-- Import DecimalPipe
import { SecureImagePipe } from '../../pipes/secure-image-pipe';
import { FormatBytesPipe } from '../../pipes/format-bytes.pipe'; // <-- ADDED

@Component({
  selector: 'app-entry-list-view', // RENAMED
  templateUrl: './entry-list-view.component.html', // RENAMED
  styleUrls: ['./entry-list-view.component.css'], // RENAMED
  standalone: true, // This component is self-contained
  imports: [
    CommonModule, 
    DatePipe,
    DecimalPipe, // <-- ADD DecimalPipe
    SecureImagePipe,
    FormatBytesPipe // <-- ADDED
  ], 
  changeDetection: ChangeDetectionStrategy.OnPush // Optimize performance
})
export class EntryListViewComponent { // RENAMED
  // --- Inputs ---
  /** The list of entries to display */
  // UPDATED: images to entries, Image to Entry
  @Input() entries: Entry[] = [];
  @Input() tableColumns: string[] = [];
  @Input() user: User | null = null;
  @Input() dbName: string | null = null;

  // --- Outputs ---
  /** Emits the entry object when a row/id is clicked */
  // UPDATED: imageClicked to entryClicked, Image to Entry
  @Output() entryClicked = new EventEmitter<Entry>();
  /** Emits the entry object when its edit button is clicked */
  @Output() editClicked = new EventEmitter<Entry>();
  /** Emits the entry object when its delete button is clicked */
  @Output() deleteClicked = new EventEmitter<Entry>();

  constructor(private databaseService: DatabaseService) {}

  /**
   * Gets the full preview URL for an entry.
   */
  // UPDATED: image to entry, Image to Entry, getImagePreviewUrl to getEntryPreviewUrl
  public getPreviewUrl(entry: Entry): string {
    if (!this.dbName) return '';
    return this.databaseService.getEntryPreviewUrl(this.dbName, entry.id);
  }

  // --- REMOVED formatBytes function ---

  // --- Event Emitters ---
  // UPDATED: image to entry, Image to Entry, onImageClick to onEntryClick, imageClicked to entryClicked
  public onEntryClick(entry: Entry): void {
    this.entryClicked.emit(entry);
  }

  public onEditClick(entry: Entry): void {
    this.editClicked.emit(entry);
  }

  public onDeleteClick(entry: Entry): void {
    this.deleteClicked.emit(entry);
  }

  /**
   * TrackBy function for the entry loop to improve performance.
   */
  // UPDATED: image to entry, Image to Entry
  public trackById(index: number, entry: Entry): number {
    return entry.id;
  }

  /**
   * TrackBy function for the column loop.
   */
  public trackByColumn(index: number, colName: string): string {
    return colName;
  }
}