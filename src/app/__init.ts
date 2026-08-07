/**
 * Book Catalog API
 *
 * A simple CRUD service for managing books.
 *
 * MIGRATION_NOTE: The Python source exposed package-level dunder metadata
 * (__version__, __author__, __email__). In Node.js, package metadata is
 * conventionally sourced from package.json rather than an __init__ module.
 * These constants are re-exported here to preserve the original values and
 * keep them accessible to the application at runtime.
 */

export const PACKAGE_METADATA = {
  version: '1.0.0',
  author: 'Abdullah Arshad',
  email: 'abdullah.arshad.314@gmail.com',
} as const;

export const VERSION = PACKAGE_METADATA.version;
export const AUTHOR = PACKAGE_METADATA.author;
export const EMAIL = PACKAGE_METADATA.email;
