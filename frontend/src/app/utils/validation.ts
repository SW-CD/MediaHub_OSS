// frontend/src/app/utils/validation.ts

export const CUSTOM_FIELD_NAME_PATTERN = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

/**
 * Validates whether a custom field name starts with a letter or underscore
 * and contains only alphanumeric characters or underscores.
 */
export function isValidCustomFieldName(name: string): boolean {
  return CUSTOM_FIELD_NAME_PATTERN.test(name);
}
