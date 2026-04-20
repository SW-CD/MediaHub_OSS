import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy, OnChanges, SimpleChanges } from '@angular/core';
import { Entry } from '../../models'; // UPDATED: Import from barrel
import { EntryService } from '../../services/entry.service';
import { CommonModule, DatePipe, DecimalPipe } from '@angular/common'; // UPDATED: Added DecimalPipe
import { SecureImageDirective } from '../../directives/secure-image.directive';

@Component({
  selector: 'app-entry-grid',
  templateUrl: './entry-grid.component.html',
  styleUrls: ['./entry-grid.component.css'],
  standalone: true,
  imports: [
    CommonModule, 
    DatePipe,
    DecimalPipe, // UPDATED: Add to imports array
    SecureImageDirective
  ], 
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class EntryGridComponent implements OnChanges {
  @Input() entries: Entry[] = [];
  @Input() dbName: string | null = null;
  
  // --- SELECTION INPUTS ---
  @Input() selectedIds = new Set<number>();
  
  @Output() entryClicked = new EventEmitter<Entry>();
  @Output() toggleSelection = new EventEmitter<{ entry: Entry, event: MouseEvent }>();

  public failedImageIds = new Set<number>();

  constructor(private entryservice: EntryService) {}

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['dbName'] || changes['entries']) {
      this.failedImageIds.clear();
    }
  }

  public getPreviewUrl(entry: Entry): string {
    if (!this.dbName) return '';
    return this.entryservice.getEntryPreviewUrl(this.dbName, entry.id);
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
    return entry.filename || `ID: ${entry.id}`; // UPDATED: Prefer filename if available
  }
}