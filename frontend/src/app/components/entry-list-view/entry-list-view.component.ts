// frontend/src/app/components/entry-list-view/entry-list-view.component.ts
import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy } from '@angular/core';
import { Entry, User } from '../../models/api.models';
import { DatabaseService } from '../../services/database.service';
import { CommonModule, DatePipe, DecimalPipe } from '@angular/common';
import { SecureImageDirective } from '../../directives/secure-image.directive'; // <-- UPDATED
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
    SecureImageDirective, // <-- UPDATED
    FormatBytesPipe
  ], 
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class EntryListViewComponent {
  @Input() entries: Entry[] = [];
  @Input() tableColumns: string[] = [];
  @Input() user: User | null = null;
  @Input() dbName: string | null = null;

  @Output() entryClicked = new EventEmitter<Entry>();
  @Output() editClicked = new EventEmitter<Entry>();
  @Output() deleteClicked = new EventEmitter<Entry>();

  constructor(private databaseService: DatabaseService) {}

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

  public trackById(index: number, entry: Entry): number {
    return entry.id;
  }

  public trackByColumn(index: number, colName: string): string {
    return colName;
  }
}