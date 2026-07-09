import { Component, OnDestroy, OnInit, Input, Output, EventEmitter } from '@angular/core';
import { Observable, Subject } from 'rxjs';
import { User, AppInfo } from '../../models';
import { AuthService } from '../../services/auth.service';
import { AppInfoService } from '../../services/app-info.service'; 

@Component({
  selector: 'app-sidebar',
  templateUrl: './sidebar.component.html',
  styleUrls: ['./sidebar.component.css'],
  standalone: false,
})
export class SidebarComponent implements OnInit, OnDestroy {
  @Input() isCollapsed: boolean = true;
  @Output() toggleSidebar = new EventEmitter<void>();

  public currentUser$: Observable<User | null>;
  public appInfo$: Observable<AppInfo | null>; 

  private destroy$ = new Subject<void>();

  constructor(
    private authService: AuthService,
    private appInfoService: AppInfoService 
  ) {
    this.currentUser$ = this.authService.currentUser$;
    this.appInfo$ = this.appInfoService.info$; 
  }

  ngOnInit(): void {
    this.appInfoService.loadInfo().subscribe();
  }

  onToggleSidebar(): void {
    this.toggleSidebar.emit();
  }

  logout(): void {
    this.authService.logout();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}