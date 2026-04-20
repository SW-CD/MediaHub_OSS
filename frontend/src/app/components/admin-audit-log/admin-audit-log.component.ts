// frontend/src/app/components/admin-audit-log/admin-audit-log.component.ts

import { Component, OnInit, OnDestroy, ChangeDetectorRef } from '@angular/core';
import { Subject } from 'rxjs';
import { takeUntil, finalize } from 'rxjs/operators';
import { AuditService } from '../../services/audit.service';
import { AuditLog } from '../../models/audit.models';

@Component({
  selector: 'app-admin-audit-log',
  templateUrl: './admin-audit-log.component.html',
  styleUrls: ['./admin-audit-log.component.css'],
  standalone: false
})
export class AdminAuditLogComponent implements OnInit, OnDestroy {
  public logs: AuditLog[] = [];
  public isLoading = true;
  
  // Pagination State
  public limit = 20;
  public offset = 0;
  public currentPage = 1;
  public hasNextPage = false;

  // Filter State (Bound to HTML inputs)
  public startDate: string = '';
  public endDate: string = '';
  
  private destroy$ = new Subject<void>();

  constructor(
    private auditService: AuditService,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit(): void {
    this.fetchLogs();
  }

  /**
   * Fetches the audit logs based on current pagination and filter states.
   */
  private fetchLogs(): void {
    this.isLoading = true;
    this.cdr.markForCheck(); 

    // new Date(string) parses the string as Local Time.
    // .getTime() converts that local time into the UTC epoch in milliseconds.
    const tstart = this.startDate ? new Date(this.startDate).getTime() : undefined;
    const tend = this.endDate ? new Date(this.endDate).getTime() : undefined;

    this.auditService.getAuditLogs(this.limit, this.offset, 'desc', tstart, tend)
      .pipe(
        takeUntil(this.destroy$),
        finalize(() => {
          this.isLoading = false;
          this.cdr.markForCheck(); 
        })
      )
      .subscribe({
        next: (logs) => {
          this.logs = logs || []; 
          this.hasNextPage = this.logs.length === this.limit;
        },
        error: (err) => {
          console.error('Failed to load audit logs', err);
          this.logs = []; 
        }
      });
  }

  // --- Consolidated Actions ---

  /**
   * Called by the Refresh button. Resets pagination and applies current filters.
   */
  public refresh(): void {
    this.offset = 0;
    this.currentPage = 1;
    this.fetchLogs();
  }

  /**
   * Called by the Clear Filters button. Clears inputs and fetches again.
   */
  public clearFilters(): void {
    this.startDate = '';
    this.endDate = '';
    this.refresh();
  }

  // --- Pagination Methods ---

  public previousPage(): void {
    if (this.offset >= this.limit) {
      this.offset -= this.limit;
      this.currentPage--;
      this.fetchLogs();
    }
  }

  public nextPage(): void {
    if (this.hasNextPage) {
      this.offset += this.limit;
      this.currentPage++;
      this.fetchLogs();
    }
  }

  /**
   * Helper to format the dynamic 'details' JSON object into a readable string.
   */
  public formatDetails(details: Record<string, any>): string {
    if (!details || Object.keys(details).length === 0) {
      return '-';
    }
    
    return Object.entries(details)
      .map(([key, value]) => `${key}: ${value}`)
      .join(', ');
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}