// frontend/src/app/utils/mime-types.ts
import { ContentType } from '../models/enums';

/**
 * A centralized map of allowed MIME types for each ContentType.
 * This mirrors the validation logic found in the Go backend.
 */
export const ALLOWED_MIME_TYPES: Record<ContentType, string[]> = {
  [ContentType.Image]: [
    'image/jpeg',
    'image/png',
    'image/gif',
    'image/webp'
  ],
  [ContentType.Audio]: [
    'audio/mpeg',
    'audio/wav',
    'audio/flac',
    'audio/opus',
    'audio/ogg',
    'application/ogg',
    'audio/x-flac'
  ],
  // Empty array implies all types are allowed for Generic Files
  [ContentType.File]: [] 
};

/**
 * Validates if a given MIME type is allowed for the specific ContentType.
 * @param contentType The database content type (image, audio, file).
 * @param mimeType The file's MIME type string (e.g., 'image/jpeg').
 * @returns True if allowed, false otherwise.
 */
export function isMimeTypeAllowed(contentType: ContentType, mimeType: string): boolean {
  const allowed = ALLOWED_MIME_TYPES[contentType];
  // If list is empty or undefined, we assume it's a generic file DB allowing everything.
  if (!allowed || allowed.length === 0) {
    return true;
  }
  return allowed.includes(mimeType);
}