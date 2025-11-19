// frontend/src/app/components/entry-grid/entry-grid.component.ts
import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy } from '@angular/core';
import { Entry } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { CommonModule, DatePipe } from '@angular/common';
import { SecureImageDirective } from '../../directives/secure-image.directive'; // <-- UPDATED

@Component({
  selector: 'app-entry-grid',
  templateUrl: './entry-grid.component.html',
  styleUrls: ['./entry-grid.component.css'],
  standalone: true,
  imports: [
    CommonModule, 
    DatePipe,
    SecureImageDirective // <-- UPDATED: Import Directive
  ], 
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class EntryGridComponent {
  @Input() entries: Entry[] = [];
  @Input() dbName: string | null = null;
  @Output() entryClicked = new EventEmitter<Entry>();

  constructor(private databaseService: DatabaseService) {}

  public getPreviewUrl(entry: Entry): string {
    if (!this.dbName) return '';
    return this.databaseService.getEntryPreviewUrl(this.dbName, entry.id);
  }

  public onEntryClick(entry: Entry): void {
    this.entryClicked.emit(entry);
  }

  public trackById(index: number, entry: Entry): number {
    return entry.id;
  }

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