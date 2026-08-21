import {
  Component,
  Input,
  Output,
  EventEmitter,
  ChangeDetectionStrategy,
  OnChanges,
  OnInit,
  AfterViewInit,
  OnDestroy,
  SimpleChanges,
  ChangeDetectorRef,
  ElementRef,
  NgZone
} from '@angular/core';
import { Entry } from '../../models'; 
import { EntryService } from '../../services/entry.service';
import { CommonModule } from '@angular/common'; 
import { SecureImageDirective } from '../../directives/secure-image.directive';
import { fromEvent, Subject, Subscription } from 'rxjs';
import { debounceTime, takeUntil } from 'rxjs/operators';

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
export class EntryGridComponent implements OnInit, OnChanges, AfterViewInit, OnDestroy {
  @Input() entries: Entry[] = [];
  @Input() dbId: string | null = null; // UPDATED: Changed from dbName to dbId
  
  // --- SELECTION INPUTS ---
  @Input() selectedIds = new Set<number>();
  
  @Output() entryClicked = new EventEmitter<Entry>();
  @Output() toggleSelection = new EventEmitter<{ entry: Entry, event: MouseEvent }>();

  public failedImageIds = new Set<number>();
  public dateGroups: DateGroup[] = [];
  public aspectRatios = new Map<number, number>();
  private groupAspectRatios = new Map<DateGroup, number>();
  private _maxRowAspectRatio = 8;

  private resizeObserver: ResizeObserver | null = null;
  private resizeSub: Subscription | null = null;
  private destroy$ = new Subject<void>();

  constructor(
    private entryService: EntryService,
    private cdr: ChangeDetectorRef,
    private el: ElementRef,
    private ngZone: NgZone
  ) {}

  ngOnInit(): void {
    this.updateContainerWidth();
  }

  ngAfterViewInit(): void {
    this.ngZone.runOutsideAngular(() => {
      if (typeof ResizeObserver !== 'undefined' && this.el?.nativeElement) {
        this.resizeObserver = new ResizeObserver(() => {
          this.handleResize();
        });
        this.resizeObserver.observe(this.el.nativeElement);
      } else if (typeof window !== 'undefined') {
        this.resizeSub = fromEvent(window, 'resize')
          .pipe(
            debounceTime(50),
            takeUntil(this.destroy$)
          )
          .subscribe(() => {
            this.handleResize();
          });
      }
    });

    this.updateContainerWidth();
    this.cdr.markForCheck();
  }

  ngOnChanges(changes: SimpleChanges): void {
    // Check for dbId changes
    if (changes['dbId'] || changes['entries']) {
      if (changes['dbId']) {
        this.failedImageIds.clear();
      }
      this.groupEntries();
      this.recalculateGroupAspectRatios();
    }
  }

  private groupEntries(): void {
    if (!this.entries || this.entries.length === 0) {
      this.dateGroups = [];
      this.groupAspectRatios.clear();
      return;
    }

    const groupsMap = new Map<string, Entry[]>();
    const weekdays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

    for (const entry of this.entries) {
      const date = new Date(entry.timestamp);

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

  private recalculateGroupAspectRatios(): void {
    this.groupAspectRatios.clear();
    for (const group of this.dateGroups) {
      let sum = 0;
      for (const entry of group.entries) {
        sum += this.getAspectRatio(entry);
      }
      this.groupAspectRatios.set(group, sum > 0 ? sum : 1.0);
    }
  }

  public getPreviewUrl(entry: Entry): string {
    if (!this.dbId) return '';
    return this.entryService.getEntryPreviewUrl(this.dbId, entry.id);
  }

  public onEntryClick(entry: Entry, event?: MouseEvent): void {
    // As soon as one image is selected, simply tapping other images selects those entries as well
    if (this.selectedIds.size > 0) {
      const mouseEvent = event || new MouseEvent('click', { bubbles: true, cancelable: true });
      this.toggleSelection.emit({ entry, event: mouseEvent });
      return;
    }

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

  public trackByDateStr(index: number, group: DateGroup): string {
    return group.dateStr;
  }

  public isSelected(entry: Entry): boolean {
    return this.selectedIds.has(entry.id);
  }

  public getEntryTitle(entry: Entry): string {
    return entry.filename || `ID: ${entry.id}`; 
  }

  public getAspectRatio(entry: Entry): number {
    // 1. Check if we have dynamically loaded aspect ratio
    if (this.aspectRatios.has(entry.id)) {
      return this.aspectRatios.get(entry.id)!;
    }
    // 2. Check if we have width and height in metadata
    if (entry.media_fields?.width && entry.media_fields?.height) {
      const ar = entry.media_fields.width / entry.media_fields.height;
      return this.clampAspectRatio(ar);
    }
    // 3. Fallback to square
    return 1.0;
  }

  private clampAspectRatio(ar: number): number {
    if (isNaN(ar) || ar <= 0) return 1.0;
    if (ar > 2.5) return 2.5;
    if (ar < 0.4) return 0.4;
    return ar;
  }

  public onAspectLoaded(entryId: number, ar: number): void {
    const clamped = this.clampAspectRatio(ar);
    if (this.aspectRatios.get(entryId) !== clamped) {
      this.aspectRatios.set(entryId, clamped);
      this.recalculateGroupAspectRatios();
      this.cdr.markForCheck();
    }
  }

  public getGroupAspectRatio(group: DateGroup): number {
    if (this.groupAspectRatios.has(group)) {
      return this.groupAspectRatios.get(group)!;
    }
    if (!group || !group.entries) return 1.0;
    let sum = 0;
    for (const entry of group.entries) {
      sum += this.getAspectRatio(entry);
    }
    const val = sum > 0 ? sum : 1.0;
    this.groupAspectRatios.set(group, val);
    return val;
  }

  private handleResize(): void {
    this.updateContainerWidth();
    this.ngZone.run(() => {
      this.cdr.markForCheck();
    });
  }

  private updateContainerWidth(): void {
    if (typeof window === 'undefined') {
      this._maxRowAspectRatio = 8;
      return;
    }
    let width = 0;
    if (this.el && this.el.nativeElement) {
      width = this.el.nativeElement.getBoundingClientRect().width;
    }
    if (width <= 0) {
      const isSidebarShown = window.location.pathname === '/dashboard' || window.location.pathname === '/';
      const sidebarWidth = isSidebarShown ? 260 : 0;
      const padding = 48;
      width = window.innerWidth - sidebarWidth - padding;
    }
    const tileHeight = window.innerWidth <= 768 ? 100 : 150;
    this._maxRowAspectRatio = width > 0 ? width / tileHeight : 8;
  }

  public get maxRowAspectRatio(): number {
    return this._maxRowAspectRatio;
  }

  public isLargeGroup(group: DateGroup): boolean {
    return this.getGroupAspectRatio(group) > this._maxRowAspectRatio;
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
    if (this.resizeObserver) {
      this.resizeObserver.disconnect();
      this.resizeObserver = null;
    }
    if (this.resizeSub) {
      this.resizeSub.unsubscribe();
      this.resizeSub = null;
    }
  }
}