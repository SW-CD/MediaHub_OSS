// frontend/src/app/models/api.models.ts
import { ContentType, EntryStatus } from './enums';

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
}

export interface CustomField {
  name: string;
  type: 'TEXT' | 'INTEGER' | 'REAL' | 'BOOLEAN';
}

export interface Housekeeping {
  interval: string;
  disk_space: string;
  max_age: string;
}

export interface Stats {
  entry_count: number;
  total_disk_space_bytes: number;
}

export interface DatabaseConfig {
  create_preview?: boolean;
  convert_to_jpeg?: boolean;
  auto_conversion?: 'none' | 'flac' | 'opus';
}

export interface Database {
  name: string;
  content_type: ContentType; // <-- Updated to Enum
  config: DatabaseConfig;
  housekeeping: Housekeeping;
  custom_fields: CustomField[];
  stats?: Stats;
}

export interface Entry {
  id: number;
  timestamp: number;
  mime_type: string;
  filesize: number;
  filename?: string;
  status: EntryStatus; // <-- Updated to Enum
  database_name?: string;

  width?: number;
  height?: number;

  duration_sec?: number;
  channels?: number;

  [key: string]: any;
}

export interface PartialEntryResponse {
  id: number;
  timestamp: number;
  database_name: string;
  status: EntryStatus; // <-- Updated to Enum
  custom_fields: { [key: string]: any };
}

export interface User {
  id: number;
  username: string;
  can_view: boolean;
  can_create: boolean;
  can_edit: boolean;
  can_delete: boolean;
  is_admin: boolean;
}

export interface ApiError {
  error: string;
}

export interface HousekeepingReport {
  database_name: string;
  entries_deleted: number;
  space_freed_bytes: number;
  message: string;
}

export interface SearchFilter {
  operator: string;
  conditions?: SearchFilter[];
  field?: string;
  value?: any;
}

export interface SearchSort {
  field: string;
  direction: 'asc' | 'desc';
}

export interface SearchPagination {
  offset: number;
  limit: number;
}

export interface SearchRequest {
  filter?: SearchFilter;
  sort?: SearchSort;
  pagination: SearchPagination;
}

export interface AppInfo {
  service_name: string;
  version: string;
  uptime_since: string;
}