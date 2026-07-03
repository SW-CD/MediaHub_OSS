import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy, OnChanges, SimpleChanges } from '@angular/core';
import { Entry } from '../../models'; 
import { EntryService } from '../../services/entry.service';
import { CommonModule } from '@angular/common'; 
import { SecureImageDirective } from '../../directives/secure-image.directive';

export interface DateGroup {
  dateStr: string;
  entries: Entry[];
}

@Component({
  selector: 'app-entry-grid',
  templateUrl: './entry-grid.component.html',
  styleUrls: ['./entry-grid.component.css'],
  standalone: true,
  imports: [
    CommonModule, 
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
  public dateGroups: DateGroup[] = [];

  constructor(private entryservice: EntryService) {}

  ngOnChanges(changes: SimpleChanges): void {
    // UPDATED: Check for dbId changes
    if (changes['dbId'] || changes['entries']) {
      if (changes['dbId']) {
        this.failedImageIds.clear();
      }
      this.groupEntries();
    }
  }

  private groupEntries(): void {
    if (!this.entries || this.entries.length === 0) {
      this.dateGroups = [];
      return;
    }

    const groupsMap = new Map<string, Entry[]>();
    const weekdays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

    for (const entry of this.entries) {
      const ts = entry.timestamp;
      // Handle both seconds and milliseconds Unix epoch timestamps safely
      const ms = ts < 10000000000 ? ts * 1000 : ts;
      const date = new Date(ms);

      const dayName = weekdays[date.getDay()];
      const dayVal = date.getDate();
      const monthName = months[date.getMonth()];
      const yearVal = date.getFullYear();
      const dateStr = `${dayName} ${dayVal} ${monthName} ${yearVal}`;

      if (!groupsMap.has(dateStr)) {
        groupsMap.set(dateStr, []);
      }
      groupsMap.get(dateStr)!.push(entry);
    }

    this.dateGroups = Array.from(groupsMap.entries()).map(([dateStr, entries]) => ({
      dateStr,
      entries
    }));
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