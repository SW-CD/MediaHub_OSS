import { ContentType } from './enums';

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
  // UPDATED: Removed convert_to_jpeg and updated auto_conversion to accept the new mime-type string format
  auto_conversion?: string; 
}

export interface Database {
  name: string;
  content_type: ContentType; 
  config: DatabaseConfig;
  housekeeping: Housekeeping;
  custom_fields: CustomField[];
  stats?: Stats;
}

export interface HousekeepingReport {
  database_name: string;
  entries_deleted: number;
  space_freed_bytes: number;
  message: string;
}