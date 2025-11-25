// frontend/src/app/components/entry-grid/entry-grid.component.ts
import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy, OnChanges, SimpleChanges } from '@angular/core';
import { Entry } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { CommonModule, DatePipe } from '@angular/common';
import { SecureImageDirective } from '../../directives/secure-image.directive';

@Component({
  selector: 'app-entry-grid',
  templateUrl: './entry-grid.component.html',
  styleUrls: ['./entry-grid.component.css'],
  standalone: true,
  imports: [
    CommonModule, 
    DatePipe,
    SecureImageDirective
  ], 
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class EntryGridComponent implements OnChanges {
  @Input() entries: Entry[] = [];
  @Input() dbName: string | null = null;
  @Output() entryClicked = new EventEmitter<Entry>();

  // Track IDs of entries where the preview image failed to load (404)
  public failedImageIds = new Set<number>();

  constructor(private databaseService: DatabaseService) {}

  ngOnChanges(changes: SimpleChanges): void {
    // If the database or the list of entries changes, reset the error tracking
    // to allow retries or clear stale errors.
    if (changes['dbName'] || changes['entries']) {
      this.failedImageIds.clear();
    }
  }

  public getPreviewUrl(entry: Entry): string {
    if (!this.dbName) return '';
    return this.databaseService.getEntryPreviewUrl(this.dbName, entry.id);
  }

  public onEntryClick(entry: Entry): void {
    this.entryClicked.emit(entry);
  }

  /**
   * Called by the SecureImageDirective when the image fails to load (e.g. 404).
   */
  public onImageError(entryId: number): void {
    this.failedImageIds.add(entryId);
  }

  public trackById(index: number, entry: Entry): number {
    return entry.id;
  }

  public getEntryTitle(entry: Entry): string {
    if (this.failedImageIds.has(entry.id)) {
      return `Entry ID: ${entry.id} (No Preview Available)`;
    }

    switch (entry.status) {
      case 'ready':
        return `View details for entry ID: ${entry.id}`;
      case 'processing':
        return `Entry ID: ${entry.id} (Processing...)`;
      case 'error':
        return `Entry ID: ${entry.id} (Processing Failed)`;
      default:
        return `Entry ID: ${entry.id}`;
    }
  }
}