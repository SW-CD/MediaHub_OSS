import { Component, OnDestroy, OnInit, ChangeDetectorRef, Input, Output, EventEmitter } from '@angular/core';
import { Observable, Subject } from 'rxjs';
import { filter, takeUntil } from 'rxjs/operators';
import { Database, User, AppInfo } from '../../models';
import { AuthService } from '../../services/auth.service';
import { DatabaseService } from '../../services/database.service';
import { Router, NavigationEnd } from '@angular/router';
import { ModalService } from '../../services/modal.service';
import { CreateDatabaseModalComponent } from '../create-database-modal/create-database-modal.component';
import { ChangePasswordModalComponent } from '../change-password-modal/change-password-modal.component';
import { AppInfoService } from '../../services/app-info.service'; 

@Component({
  selector: 'app-sidebar',
  templateUrl: './sidebar.component.html',
  styleUrls: ['./sidebar.component.css'],
  standalone: false,
})
export class SidebarComponent implements OnInit, OnDestroy {
  @Input() isCollapsed: boolean = false;
  @Output() toggleSidebar = new EventEmitter<void>();

  public databases$: Observable<Database[]>;
  public currentUser$: Observable<User | null>;
  public appInfo$: Observable<AppInfo | null>; 
  public selectedDbId: string | null = null;

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

      let newDbId: string | null = null;
      if (dbIndex > -1 && urlSegments.length > dbIndex + 1) {
        newDbId = urlSegments[dbIndex + 1];
      } else if (settingsIndex > -1 && urlSegments.length > settingsIndex + 1) {
        newDbId = urlSegments[settingsIndex + 1];
      }

      this.selectedDbId = newDbId;
      this.cdr.detectChanges();
    });
  }

  onToggleSidebar(): void {
    this.toggleSidebar.emit();
  }

  /**
   * Returns the correct icon path based on db content_type.
   */
  public getIconForDb(db: Database): string {
    switch (db.content_type) {
      case 'image': return 'assets/icons/image-icon.svg';
      case 'audio': return 'assets/icons/audio-icon.svg';
      case 'video': return 'assets/icons/video-icon.svg';
      case 'file': return 'assets/icons/file-icon.svg';
      default: return 'assets/icons/db-icon.svg';
    }
  }

  /**
   * Helper to determine if the user should see the settings gear icon.
   * True if the user is an admin OR has edit/delete permissions for this specific db.
   */
  public canManageDatabase(dbId: string, user: User): boolean { // UPDATED: parameter
    if (user.is_admin) return true;
    
    const dbPermission = user.permissions?.find(p => p.database_id === dbId);
    return !!dbPermission && (dbPermission.can_edit || dbPermission.can_delete);
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

  goToSettings(event: MouseEvent, dbId: string): void { 
    event.preventDefault();
    event.stopPropagation();
    this.router.navigate(['/dashboard/settings', dbId]);
  }

  trackByDbId(index: number, database: Database): string {
    return database.id;
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}