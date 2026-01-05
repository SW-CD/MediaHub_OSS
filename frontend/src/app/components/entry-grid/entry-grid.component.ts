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
  // --- SELECTION INPUTS ---
  @Input() selectedIds = new Set<number>();
  
  @Output() entryClicked = new EventEmitter<Entry>();
  @Output() toggleSelection = new EventEmitter<{ entry: Entry, event: MouseEvent }>();

  public failedImageIds = new Set<number>();

  constructor(private databaseService: DatabaseService) {}

  ngOnChanges(changes: SimpleChanges): void {
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

  // Handle the checkbox click separately to prevent bubbling if needed,
  // or handle it in the parent div click if we want entire card to toggle?
  // Use case: Card click -> Detail. Checkbox click -> Select.
  public onCheckboxClick(entry: Entry, event: MouseEvent): void {
    event.stopPropagation(); // Don't open details
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
    return `ID: ${entry.id}`; // Simplified
  }
}