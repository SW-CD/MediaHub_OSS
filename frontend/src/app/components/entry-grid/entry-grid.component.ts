import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy, OnChanges, SimpleChanges, ChangeDetectorRef, HostListener, ElementRef } from '@angular/core';
import { Entry } from '../../models'; 
import { EntryService } from '../../services/entry.service';
import { CommonModule } from '@angular/common'; 
import { SecureImageDirective } from '../../directives/secure-image.directive';
import { ImageCacheService } from '../../services/image-cache.service';

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
  public aspectRatios = new Map<number, number>();

  // Touch gesture state for smartphone long-press & multi-select tap
  private touchTimer: any = null;
  private touchStartX = 0;
  private touchStartY = 0;
  private isLongPress = false;
  private wasTouchHandled = false;

  constructor(
    private entryservice: EntryService,
    private imageCacheService: ImageCacheService,
    private cdr: ChangeDetectorRef,
    private el: ElementRef
  ) {}

  ngOnChanges(changes: SimpleChanges): void {
    // Check for dbId changes
    if (changes['dbId'] || changes['entries']) {
      if (changes['dbId']) {
        this.failedImageIds.clear();
        this.imageCacheService.clearAll();
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
    return this.entryservice.getEntryPreviewUrl(this.dbId, entry.id);
  }

  public onEntryClick(entry: Entry, event?: MouseEvent): void {
    // If touch gesture already handled this interaction, ignore synthetic click
    if (this.wasTouchHandled) {
      this.wasTouchHandled = false;
      return;
    }

    // As soon as one image is selected, simply tapping other images selects those entries as well
    if (this.selectedIds.size > 0) {
      const mouseEvent = event || new MouseEvent('click', { bubbles: true, cancelable: true });
      this.toggleSelection.emit({ entry, event: mouseEvent });
      return;
    }

    this.entryClicked.emit(entry);
  }

  public onTouchStart(entry: Entry, event: TouchEvent): void {
    if (event.touches.length !== 1) return;
    const touch = event.touches[0];
    this.touchStartX = touch.clientX;
    this.touchStartY = touch.clientY;
    this.isLongPress = false;

    this.clearTouchTimer();
    this.touchTimer = setTimeout(() => {
      this.isLongPress = true;
      // Haptic vibration feedback on long-press selection
      if (typeof navigator !== 'undefined' && 'vibrate' in navigator) {
        try {
          navigator.vibrate(50);
        } catch (_) {}
      }
      const syntheticEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
      this.toggleSelection.emit({ entry, event: syntheticEvent });
      this.cdr.markForCheck();
    }, 500);
  }

  public onTouchMove(event: TouchEvent): void {
    if (!this.touchTimer) return;
    if (event.touches.length > 0) {
      const touch = event.touches[0];
      const deltaX = Math.abs(touch.clientX - this.touchStartX);
      const deltaY = Math.abs(touch.clientY - this.touchStartY);
      // Cancel long-press timer if user is scrolling
      if (deltaX > 10 || deltaY > 10) {
        this.clearTouchTimer();
      }
    }
  }

  public onTouchEnd(entry: Entry, event: TouchEvent): void {
    const wasLongPress = this.isLongPress;
    this.clearTouchTimer();

    if (wasLongPress) {
      this.wasTouchHandled = true;
      if (event.cancelable) event.preventDefault();
      setTimeout(() => { this.wasTouchHandled = false; }, 300);
      return;
    }

    // If selection active and this was a simple tap (not long press)
    if (this.selectedIds.size > 0) {
      this.wasTouchHandled = true;
      const syntheticEvent = new MouseEvent('click', { bubbles: true, cancelable: true });
      this.toggleSelection.emit({ entry, event: syntheticEvent });
      if (event.cancelable) event.preventDefault();
      setTimeout(() => { this.wasTouchHandled = false; }, 300);
      this.cdr.markForCheck();
    }
  }

  public onTouchCancel(): void {
    this.clearTouchTimer();
    this.isLongPress = false;
  }

  private clearTouchTimer(): void {
    if (this.touchTimer) {
      clearTimeout(this.touchTimer);
      this.touchTimer = null;
    }
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
      this.cdr.markForCheck();
    }
  }

  public getGroupAspectRatio(group: DateGroup): number {
    if (!group || !group.entries) return 1.0;
    let sum = 0;
    for (const entry of group.entries) {
      sum += this.getAspectRatio(entry);
    }
    return sum > 0 ? sum : 1.0;
  }

  @HostListener('window:resize')
  onResize() {
    this.cdr.markForCheck();
  }

  private getContainerWidth(): number {
    if (typeof window === 'undefined') return 1200;
    if (this.el && this.el.nativeElement) {
      const width = this.el.nativeElement.getBoundingClientRect().width;
      if (width > 0) return width;
    }
    const isSidebarShown = window.location.pathname === '/dashboard' || window.location.pathname === '/';
    const sidebarWidth = isSidebarShown ? 260 : 0;
    const padding = 48;
    return window.innerWidth - sidebarWidth - padding;
  }

  public get maxRowAspectRatio(): number {
    const width = this.getContainerWidth();
    const tileHeight = window.innerWidth <= 768 ? 100 : 150;
    return width / tileHeight;
  }

  public isLargeGroup(group: DateGroup): boolean {
    return this.getGroupAspectRatio(group) > this.maxRowAspectRatio;
  }
}