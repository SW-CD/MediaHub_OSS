import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy, OnChanges, SimpleChanges } from '@angular/core';
import { Entry } from '../../models'; 
import { EntryService } from '../../services/entry.service';
import { CommonModule, DatePipe, DecimalPipe } from '@angular/common'; 
import { SecureImageDirective } from '../../directives/secure-image.directive';

@Component({
  selector: 'app-entry-grid',
  templateUrl: './entry-grid.component.html',
  styleUrls: ['./entry-grid.component.css'],
  standalone: true,
  imports: [
    CommonModule, 
    DatePipe,
    DecimalPipe, 
    SecureImageDirective
  ], 
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class EntryGridComponent implements OnChanges {
  @Input() entries: Entry[] = [];
  @Input() dbId: string | null = null; // UPDATED: Changed from dbName to dbId
  
  // --- SELECTION INPUTS ---
  @Input() selectedIds = new Set<number>();
  
  @Output() entryClicked = new EventEmitter<Entry>();
  @Output() toggleSelection = new EventEmitter<{ entry: Entry, event: MouseEvent }>();

  public failedImageIds = new Set<number>();

  constructor(private entryservice: EntryService) {}

  ngOnChanges(changes: SimpleChanges): void {
    // UPDATED: Check for dbId changes
    if (changes['dbId'] || changes['entries']) {
      this.failedImageIds.clear();
    }
  }

  public getPreviewUrl(entry: Entry): string {
    if (!this.dbId) return '';
    return this.entryservice.getEntryPreviewUrl(this.dbId, entry.id); // UPDATED: Pass dbId
  }

  public onEntryClick(entry: Entry): void {
    this.entryClicked.emit(entry);
  }

  public onCheckboxClick(entry: Entry, event: MouseEvent): void {
    event.stopPropagation(); 
    this.toggleSelection.emit({ entry, event });
  }

  public onImageError(entryId: number): void {
    this.failedImageIds.add(entryId);
  }

  public trackById(index: number, entry: Entry): number {
    return entry.id;
  }

  public isSelected(entry: Entry): boolean {
    return this.selectedIds.has(entry.id);
  }

  public getEntryTitle(entry: Entry): string {
    return entry.filename || `ID: ${entry.id}`; 
  }
}