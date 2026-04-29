import { Component, OnInit, OnDestroy, Input, Output, EventEmitter, ElementRef, ViewChild, HostListener, ChangeDetectorRef } from '@angular/core';
import { Subject, Subscription, timer } from 'rxjs';
import { switchMap, takeUntil } from 'rxjs/operators';
import { Entry, SearchRequest } from '../../models'; 
import { EntryService } from '../../services/entry.service'; 
import { NotificationService } from '../../services/notification.service'; 
import { FullscreenSettings } from '../fullscreen-settings-modal/fullscreen-settings-modal.component';
import { isMimeTypeStreamable } from '../../utils/mime-types'; 

@Component({
  selector: 'app-fullscreen-player',
  templateUrl: './fullscreen-player.component.html',
  styleUrls: ['./fullscreen-player.component.css'],
  standalone: false
})
export class FullscreenPlayerComponent implements OnInit, OnDestroy {
  @Input() settings!: FullscreenSettings;
  @Input() dbId!: string; // UPDATED: Changed from dbName to dbId
  @Input() contentType!: string;
  
  @Output() exit = new EventEmitter<void>();

  // Use ViewChild to get access to the main container for the Fullscreen API
  @ViewChild('playerContainer', { static: true }) playerContainer!: ElementRef;
  
  // To access native video/audio elements to play/pause programmatically
  @ViewChild('mediaElement') mediaElement?: ElementRef<HTMLMediaElement>;

  public currentEntry: Entry | null = null;
  public mediaUrl: string | null = null;
  public isLoading = true;
  
  private playlist: Entry[] = [];
  private currentIndex: number = -1;
  private knownEntryIds = new Set<number>();
  
  private delayTimerSub?: Subscription;
  private pollingSub?: Subscription;
  private destroy$ = new Subject<void>();

  constructor(
    private entryService: EntryService,
    private notificationService: NotificationService,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit(): void {
    console.log('--- DEBUG: 5. FullscreenPlayerComponent initialized in the DOM! ---');
    this.requestNativeFullscreen();
    this.startPolling();
  }

  /**
   * Listens for the browser's native fullscreen exit event (e.g., user presses ESC)
   */
  @HostListener('document:fullscreenchange')
  onFullscreenChange(): void {
    if (!document.fullscreenElement) {
      this.closePlayer();
    }
  }

  /**
   * Requests the browser to take the container full screen.
   */
  private requestNativeFullscreen(): void {
    const el = this.playerContainer.nativeElement;
    if (el.requestFullscreen) {
      el.requestFullscreen().catch((err: any) => {
        console.error('Error attempting to enable fullscreen:', err);
        this.notificationService.showError('Fullscreen API is blocked or not supported by your browser.');
      });
    }
  }

  /**
   * Starts a background polling loop to fetch the latest N entries.
   */
  private startPolling(): void {
    // Poll every 5 seconds to check for new entries
    this.pollingSub = timer(0, 5000).pipe(
      takeUntil(this.destroy$),
      switchMap(() => {
        const searchPayload: SearchRequest = {
          pagination: { limit: this.settings.entryLimit, offset: 0 },
          sort: { field: 'timestamp', direction: 'desc' }
        };
        return this.entryService.searchEntries(this.dbId, searchPayload); // UPDATED: Pass dbId
      })
    ).subscribe({
      next: (entries) => this.handleFetchedEntries(entries),
      error: (err) => console.error('Polling error:', err)
    });
  }

  /**
   * Compares newly fetched entries against the current queue and updates if necessary.
   */
  private handleFetchedEntries(entries: Entry[]): void {
    // Filter out entries that aren't fully processed yet to prevent broken media
    const readyEntries = entries.filter(e => e.status === 'ready'); 
    
    if (readyEntries.length === 0) {
      this.isLoading = false;
      return;
    }

    // Check if we have any truly *new* entries that we haven't seen before
    const hasNewEntries = readyEntries.some(e => !this.knownEntryIds.has(e.id));

    if (hasNewEntries || this.playlist.length === 0) {
      // Update our tracking set
      this.knownEntryIds = new Set(readyEntries.map(e => e.id));
      
      this.playlist = [...readyEntries];
      if (this.settings.shuffle) {
        this.shuffleArray(this.playlist);
      }

      // If this is the very first load, or we are sitting idle waiting for a new file (N=1 scenario), start playback
      if (this.currentEntry === null || (this.settings.entryLimit === 1 && !this.settings.repeat)) {
        this.currentIndex = -1;
        this.playNext();
      }
    }
    this.isLoading = false;
  }

  /**
   * Advances the playlist.
   */
private playNext(): void {
    this.clearDelayTimer();

    if (this.playlist.length === 0) return;

    this.currentIndex++;
    console.log(`--- DEBUG: playNext() called. Advancing to index ${this.currentIndex} out of ${this.playlist.length} ---`);

    // Handle reaching the end of the playlist
    if (this.currentIndex >= this.playlist.length) {
      if (this.settings.repeat) {
        console.log('--- DEBUG: Reached end of queue. Repeating. ---');
        this.currentIndex = 0; // Loop back to start
        if (this.settings.shuffle) {
            this.shuffleArray(this.playlist);
        }
      } else {
        if (this.settings.entryLimit === 1) {
            this.currentIndex = 0; 
            if (this.contentType !== 'image') { return; }
        } else {
            this.notificationService.showInfo('Presentation finished.');
            this.closePlayer();
            return;
        }
      }
    }

    this.currentEntry = this.playlist[this.currentIndex];
    this.loadMedia(this.currentEntry);
  }

  private loadMedia(entry: Entry): void {
    const mime = entry.mime_type || 'file';
    const isStreamable = isMimeTypeStreamable(mime); 

    if (isStreamable || this.contentType === 'image') {
      this.mediaUrl = this.entryService.getEntryFileUrl(this.dbId, entry.id); // UPDATED: Pass dbId
      console.log('--- DEBUG: Loading media URL:', this.mediaUrl, '---');
    } else {
       this.mediaUrl = null;
    }

    // Force the screen to redraw with the new URL!
    this.cdr.detectChanges();

    if (this.contentType === 'image' || !this.mediaUrl) {
       this.startDelayTimer();
    }
  }

  /**
   * Called by the (ended) event of the <video> or <audio> tags in the HTML template.
   */
  public onMediaEnded(): void {
    // Video/Audio finished. Now we wait the user-defined delay before switching.
    this.startDelayTimer();
  }

  /**
   * Starts the countdown to the next slide.
   */
  private startDelayTimer(): void {
    this.clearDelayTimer();
    // Convert seconds to milliseconds
    const delayMs = this.settings.delaySeconds * 1000; 
    
    this.delayTimerSub = timer(delayMs).subscribe(() => {
      this.playNext();
    });
  }

  private clearDelayTimer(): void {
    if (this.delayTimerSub) {
      this.delayTimerSub.unsubscribe();
      this.delayTimerSub = undefined;
    }
  }

  /**
   * Standard Fisher-Yates shuffle algorithm.
   */
  private shuffleArray(array: any[]): void {
    for (let i = array.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [array[i], array[j]] = [array[j], array[i]];
    }
  }

  /**
   * Cleans up fullscreen and notifies the parent to destroy this component.
   */
  public closePlayer(): void {
    if (document.fullscreenElement) {
      document.exitFullscreen().catch(err => console.error(err));
    }
    this.exit.emit();
  }

  ngOnDestroy(): void {
    this.clearDelayTimer();
    if (this.pollingSub) {
      this.pollingSub.unsubscribe();
    }
    this.destroy$.next();
    this.destroy$.complete();
  }
}