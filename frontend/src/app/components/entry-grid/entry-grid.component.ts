// frontend/src/app/components/entry-grid/entry-grid.component.ts
import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy } from '@angular/core';
// UPDATED: Image to Entry
import { Entry } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { CommonModule, DatePipe } from '@angular/common';
import { SecureImagePipe } from '../../pipes/secure-image-pipe';

@Component({
  selector: 'app-entry-grid', // RENAMED
  templateUrl: './entry-grid.component.html', // RENAMED
  styleUrls: ['./entry-grid.component.css'], // RENAMED
  standalone: true, // This component is self-contained
  imports: [
    CommonModule, 
    DatePipe,
    SecureImagePipe
  ], 
  changeDetection: ChangeDetectionStrategy.OnPush // Optimize performance
})
export class EntryGridComponent { // RENAMED
  // --- Inputs ---
  /** The list of entries to display */
  // UPDATED: images to entries, Image to Entry
  @Input() entries: Entry[] = [];
  /** The name of the database, required for generating preview URLs */
  @Input() dbName: string | null = null;

  // --- Outputs ---
  /** Emits the entry object when a card is clicked */
  // UPDATED: imageClicked to entryClicked, Image to Entry
  @Output() entryClicked = new EventEmitter<Entry>();

  constructor(private databaseService: DatabaseService) {}

  /**
   * Gets the full preview URL for an entry.
   */
  // UPDATED: image to entry, Image to Entry, getImagePreviewUrl to getEntryPreviewUrl
  public getPreviewUrl(entry: Entry): string {
    if (!this.dbName) return '';
    return this.databaseService.getEntryPreviewUrl(this.dbName, entry.id);
  }

  /**
   * Emits the entryClicked event.
   */
  // UPDATED: image to entry, Image to Entry, onImageClick to onEntryClick
  public onEntryClick(entry: Entry): void {
    this.entryClicked.emit(entry);
  }

  /**
   * TrackBy function for the entry loop to improve performance.
   */
  // UPDATED: image to entry, Image to Entry
  public trackById(index: number, entry: Entry): number {
    return entry.id;
  }

  /**
   * --- ADDED: This function was missing ---
   * Returns a descriptive title for the grid card based on entry status.
   */
  public getEntryTitle(entry: Entry): string {
    switch (entry.status) {
      case 'ready':
        return `View details for entry ID: ${entry.id}`;
      case 'processing':
        return `Entry ID: ${entry.id} (Processing...)`;
      case 'error':
        return `Entry ID: ${entry.id} (Error)`;
      default:
        return `Entry ID: ${entry.id}`;
    }
  }
}