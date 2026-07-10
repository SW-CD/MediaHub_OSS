import { Component, OnDestroy, OnInit, Input, Output, EventEmitter } from '@angular/core';
import { Observable, Subject } from 'rxjs';
import { map } from 'rxjs/operators';
import { User, AppInfo } from '../../models';
import { AuthService } from '../../services/auth.service';
import { AppInfoService } from '../../services/app-info.service'; 
import { ThemeService } from '../../services/theme.service';

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
  public isLightTheme$: Observable<boolean>;

  private destroy$ = new Subject<void>();

  constructor(
    private authService: AuthService,
    private appInfoService: AppInfoService,
    private themeService: ThemeService
  ) {
    this.currentUser$ = this.authService.currentUser$;
    this.appInfo$ = this.appInfoService.info$; 
    this.isLightTheme$ = this.themeService.theme$.pipe(
      map(theme => theme === 'light')
    );
  }

  toggleTheme(): void {
    this.themeService.toggleTheme();
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