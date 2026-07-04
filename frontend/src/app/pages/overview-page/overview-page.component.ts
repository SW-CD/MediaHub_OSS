import { Component, OnInit, OnDestroy, ChangeDetectorRef } from '@angular/core';
import { DatabaseService } from '../../services/database.service';
import { EntryService } from '../../services/entry.service';
import { AuthService } from '../../services/auth.service';
import { ModalService } from '../../services/modal.service';
import { CreateDatabaseModalComponent } from '../../components/create-database-modal/create-database-modal.component';
import { Database, User } from '../../models';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { Router } from '@angular/router';

@Component({
  selector: 'app-overview-page',
  templateUrl: './overview-page.component.html',
  styleUrls: ['./overview-page.component.css'],
  standalone: false
})
export class OverviewPageComponent implements OnInit, OnDestroy {
  public databases: Database[] = [];
  public currentUser: User | null = null;
  public dbPreviews: Record<string, string> = {};
  public failedPreviews = new Set<string>();
  
  private destroy$ = new Subject<void>();

  constructor(
    private databaseService: DatabaseService,
    private entryService: EntryService,
    private authService: AuthService,
    private modalService: ModalService,
    private router: Router,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit(): void {
    this.authService.currentUser$.pipe(takeUntil(this.destroy$)).subscribe(user => {
      this.currentUser = user;
      this.cdr.markForCheck();
    });

    this.databaseService.databases$.pipe(takeUntil(this.destroy$)).subscribe(databases => {
      this.databases = databases || [];
      this.loadPreviews();
      this.cdr.markForCheck();
    });

    this.databaseService.loadDatabases().subscribe();
  }

  private loadPreviews(): void {
    for (const db of this.databases) {
      if (db.stats && db.stats.entry_count > 0) {
        // Query the latest entry to use as preview
        this.entryService.searchEntries(db.id, {
          pagination: { limit: 1 },
          sort: { field: 'timestamp', direction: 'desc' }
        }).pipe(takeUntil(this.destroy$)).subscribe({
          next: (entries) => {
            if (entries && entries.length > 0) {
              const latestEntry = entries[0];
              this.dbPreviews[db.id] = this.entryService.getEntryPreviewUrl(db.id, latestEntry.id);
            }
            this.cdr.markForCheck();
          },
          error: (err) => {
            console.error(`Failed to load latest entry for database ${db.name}`, err);
          }
        });
      }
    }
  }

  public getIconForDb(db: Database): string {
    switch (db.content_type) {
      case 'image': return 'assets/icons/image-icon.svg';
      case 'audio': return 'assets/icons/audio-icon.svg';
      case 'video': return 'assets/icons/video-icon.svg';
      case 'file': return 'assets/icons/file-icon.svg';
      default: return 'assets/icons/db-icon.svg';
    }
  }

  public handleImageError(dbId: string): void {
    this.failedPreviews.add(dbId);
    this.cdr.markForCheck();
  }

  public openCreateDatabaseModal(): void {
    this.modalService.open(CreateDatabaseModalComponent.MODAL_ID);
  }

  public goToDatabase(dbId: string): void {
    this.router.navigate(['/dashboard/db', dbId]);
  }

  public goToSettings(event: MouseEvent, dbId: string): void {
    event.preventDefault();
    event.stopPropagation();
    this.router.navigate(['/dashboard/settings', dbId]);
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}
