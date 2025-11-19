// frontend/src/app/models/api.models.ts

/**
 * Defines the models for the API.
 * These interfaces align with the backend's API responses.
 */

/**
 * Represents the response from the token generation endpoints.
 */
export interface TokenResponse {
  access_token: string;
  refresh_token: string;
}

/**
 * Represents the structure of a custom metadata field
 * for a database.
 */
export interface CustomField {
  name: string;
  type: 'TEXT' | 'INTEGER' | 'REAL' | 'BOOLEAN';
}

/**
 * Represents the housekeeping rules for a database.
 */
export interface Housekeeping {
  interval: string;
  disk_space: string;
  max_age: string;
}

/**
 * Represents live statistics for a database.
 */
export interface Stats {
  entry_count: number;
  total_disk_space_bytes: number;
}

/**
 * Represents the dynamic, type-specific configuration
 * for a database.
 */
export interface DatabaseConfig {
  create_preview?: boolean;
  // Image-specific
  convert_to_jpeg?: boolean;
  // Audio-specific
  auto_conversion?: 'none' | 'flac' | 'opus';
}

/**
 * Represents the full Database object, including its
 * configuration, schema, and stats.
 */
export interface Database {
  name: string;
  content_type: 'image' | 'audio' | 'file';
  config: DatabaseConfig;
  housekeeping: Housekeeping;
  custom_fields: CustomField[];
  stats?: Stats;
}

/**
 * Represents the metadata for a single entry (image, audio, or file).
 * Uses an index signature to allow for dynamic custom fields.
 */
export interface Entry {
  id: number;
  timestamp: number;
  mime_type: string;
  filesize: number;
  filename?: string;
  status: 'processing' | 'ready' | 'error';
  database_name?: string;

  // Image-specific (optional)
  width?: number;
  height?: number;

  // Audio-specific (optional)
  duration_sec?: number;
  channels?: number;

  [key: string]: any; // Allows for custom fields
}

/**
 * Represents the partial response for an async upload (202 Accepted).
 */
export interface PartialEntryResponse {
  id: number;
  timestamp: number;
  database_name: string;
  status: 'processing';
  custom_fields: { [key: string]: any };
}


/**
 * Represents the authenticated user's details and roles.
 * This is received from the /api/me endpoint.
 */
export interface User {
  id: number;
  username: string;
  can_view: boolean;
  can_create: boolean;
  can_edit: boolean;
  can_delete: boolean;
  is_admin: boolean;
}

/**
 * Represents the standard JSON error response from the API.
 */
export interface ApiError {
  error: string;
}

/**
 * Represents the report from a successful housekeeping run.
 */
export interface HousekeepingReport {
  database_name: string;
  entries_deleted: number;
  space_freed_bytes: number;
  message: string;
}

// --- STRUCTS FOR SEARCH ENDPOINT ---

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

/**
 * Represents the response from /api/info
 */
export interface AppInfo {
  service_name: string;
  version: string;
  uptime_since: string;
}