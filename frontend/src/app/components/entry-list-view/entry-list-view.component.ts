// frontend/src/app/components/entry-list-view/entry-list-view.component.ts
import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy, OnChanges, SimpleChanges } from '@angular/core';
import { Entry, User } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { CommonModule, DatePipe, DecimalPipe } from '@angular/common';
import { SecureImageDirective } from '../../directives/secure-image.directive';
import { FormatBytesPipe } from '../../pipes/format-bytes.pipe';

@Component({
  selector: 'app-entry-list-view',
  templateUrl: './entry-list-view.component.html',
  styleUrls: ['./entry-list-view.component.css'],
  standalone: true,
  imports: [
    CommonModule, 
    DatePipe,
    DecimalPipe,
    SecureImageDirective,
    FormatBytesPipe
  ], 
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class EntryListViewComponent implements OnChanges {
  @Input() entries: Entry[] = [];
  @Input() tableColumns: string[] = [];
  @Input() user: User | null = null;
  @Input() dbName: string | null = null;

  @Output() entryClicked = new EventEmitter<Entry>();
  @Output() editClicked = new EventEmitter<Entry>();
  @Output() deleteClicked = new EventEmitter<Entry>();

  // Track IDs of entries where the preview image failed to load (404)
  public failedImageIds = new Set<number>();

  constructor(private databaseService: DatabaseService) {}

  ngOnChanges(changes: SimpleChanges): void {
    // If the database or the list of entries changes, reset the error tracking
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

  public onEditClick(entry: Entry): void {
    this.editClicked.emit(entry);
  }

  public onDeleteClick(entry: Entry): void {
    this.deleteClicked.emit(entry);
  }

  /**
   * Called by the SecureImageDirective when the image fails to load.
   */
  public onImageError(entryId: number): void {
    this.failedImageIds.add(entryId);
  }

  public trackById(index: number, entry: Entry): number {
    return entry.id;
  }

  public trackByColumn(index: number, colName: string): string {
    return colName;
  }
}