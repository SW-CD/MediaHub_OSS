// frontend/src/app/models/api.models.ts

/**
 * Defines the models for the API.
 * These interfaces align with the backend's API responses.
 */

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
 * UPDATED: Renamed image_count to entry_count.
 */
export interface Stats {
  entry_count: number;
  total_disk_space_bytes: number;
}

/**
 * NEW: Represents the dynamic, type-specific configuration
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
 * UPDATED: To match the new backend concept.
 */
export interface Database {
  name: string;
  content_type: 'image' | 'audio' | 'file'; // <-- ADDED
  config: DatabaseConfig; // <-- ADDED
  housekeeping: Housekeeping;
  custom_fields: CustomField[];
  stats?: Stats;
}

/**
 * Represents the metadata for a single entry (image, audio, or file).
 * Uses an index signature to allow for dynamic custom fields.
 * UPDATED: Renamed from Image to Entry and fields updated.
 */
export interface Entry {
  id: number;
  timestamp: number;
  mime_type: string;
  filesize: number;
  filename?: string; // <-- ADDED: Original filename
  status: 'processing' | 'ready' | 'error'; // <-- ADDED
  database_name?: string; // May not always be present depending on endpoint

  // Image-specific (optional)
  width?: number;
  height?: number;

  // Audio-specific (optional)
  duration_sec?: number;
  channels?: number;

  [key: string]: any; // Allows for custom fields
}

/**
 * NEW: Represents the partial response for an async upload (202 Accepted).
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
 * UPDATED: Renamed images_deleted to entries_deleted.
 */
export interface HousekeepingReport {
  database_name: string;
  entries_deleted: number;
  space_freed_bytes: number;
  message: string;
}

// --- STRUCTS FOR SEARCH ENDPOINT (Unchanged as per TODOs) ---

/**
 * Defines the structure for a single filter condition or a nested group
 * for the POST /api/database/entries/search endpoint.
 */
export interface SearchFilter {
  // For logical grouping ("and", "or")
  operator: string; // Should be 'and' or 'or' for groups, or comparison operator for conditions
  conditions?: SearchFilter[]; // Used for logical grouping

  // For single conditions ("=", ">", "<", etc.)
  field?: string; // Field name (standard or custom)
  value?: any; // The value to compare against
}

/**
 * Defines the sorting criteria for the search endpoint.
 */
export interface SearchSort {
  field: string;
  direction: 'asc' | 'desc';
}

/**
 * Defines the limit and offset for pagination in the search endpoint.
 */
export interface SearchPagination {
  offset: number;
  limit: number; // Backend requires this
}

/**
 * Top-level request body for the POST /api/database/entries/search endpoint.
 */
export interface SearchRequest {
  filter?: SearchFilter; // Optional top-level filter group
  sort?: SearchSort; // Optional sort criteria
  pagination: SearchPagination; // Required pagination
}

/**
 * Represents the response from /api/info
 */
export interface AppInfo {
  service_name: string;
  version: string;
  uptime_since: string;
}