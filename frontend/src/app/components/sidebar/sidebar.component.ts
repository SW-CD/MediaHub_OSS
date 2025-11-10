// frontend/src/app/components/sidebar/sidebar.component.ts
import { Component, OnDestroy, OnInit, ChangeDetectorRef, Input, Output, EventEmitter } from '@angular/core';
import { Observable, Subject } from 'rxjs';
import { filter, takeUntil } from 'rxjs/operators';
import { Database, User } from '../../models/api.models';
import { AuthService } from '../../services/auth.service';
import { DatabaseService } from '../../services/database.service';
import { Router, NavigationEnd } from '@angular/router';
import { ModalService } from '../../services/modal.service';
// UPDATED: Import renamed modal
import { CreateDatabaseModalComponent } from '../create-database-modal/create-database-modal.component';
import { ChangePasswordModalComponent } from '../change-password-modal/change-password-modal.component';
import { AppInfoService } from '../../services/app-info.service'; 
import { AppInfo } from '../../models/api.models'; 

@Component({
  selector: 'app-sidebar',
  templateUrl: './sidebar.component.html',
  styleUrls: ['./sidebar.component.css'],
  standalone: false,
})
export class SidebarComponent implements OnInit, OnDestroy {
  /**
   * Receives the current collapsed state from the parent component.
   */
  @Input() isCollapsed: boolean = false;
  
  /**
   * Emits an event when the toggle button is clicked.
   */
  @Output() toggleSidebar = new EventEmitter<void>();

  public databases$: Observable<Database[]>;
  public currentUser$: Observable<User | null>;
  public appInfo$: Observable<AppInfo | null>; 
  public selectedDbName: string | null = null;

  private destroy$ = new Subject<void>();

  constructor(
    private authService: AuthService,
    private databaseService: DatabaseService,
    private router: Router,
    private modalService: ModalService,
    private cdr: ChangeDetectorRef,
    private appInfoService: AppInfoService 
  ) {
    this.databases$ = this.databaseService.databases$;
    this.currentUser$ = this.authService.currentUser$;
    this.appInfo$ = this.appInfoService.info$; 
  }

  ngOnInit(): void {
    this.appInfoService.loadInfo(); 
    this.databaseService.loadDatabases().subscribe();

    this.router.events.pipe(
      takeUntil(this.destroy$),
      filter((event): event is NavigationEnd => event instanceof NavigationEnd)
    ).subscribe((event: NavigationEnd) => {
      const urlSegments = event.urlAfterRedirects.split('/');
      const dbIndex = urlSegments.indexOf('db');
      const settingsIndex = urlSegments.indexOf('settings');

      let newDbName: string | null = null;
      if (dbIndex > -1 && urlSegments.length > dbIndex + 1) {
        newDbName = urlSegments[dbIndex + 1];
      } else if (settingsIndex > -1 && urlSegments.length > settingsIndex + 1) {
        newDbName = urlSegments[settingsIndex + 1];
      }

      this.selectedDbName = newDbName;
      this.cdr.detectChanges();
    });
  }

  /**
   * Emits the toggle event to the parent dashboard component.
   */
  onToggleSidebar(): void {
    this.toggleSidebar.emit();
  }

  /**
   * NEW: Returns the correct icon path based on db content_type.
   * (Assumes icons are added to assets/icons/)
   */
  public getIconForDb(db: Database): string {
    switch (db.content_type) {
      case 'image': return 'assets/icons/image-icon.svg';
      case 'audio': return 'assets/icons/audio-icon.svg';
      case 'file': return 'assets/icons/file-icon.svg';
      default: return 'assets/icons/db-icon.svg';
    }
  }

  logout(): void {
    this.authService.logout();
  }

  openCreateDatabaseModal(): void {
    this.modalService.open(CreateDatabaseModalComponent.MODAL_ID);
  }

  openChangePasswordModal(): void {
    this.modalService.open(ChangePasswordModalComponent.MODAL_ID);
  }

  goToSettings(event: MouseEvent, dbName: string): void {
    event.preventDefault();
    event.stopPropagation();
    this.router.navigate(['/dashboard/settings', dbName]);
  }

  trackByDbName(index: number, database: Database): string {
    return database.name;
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}
