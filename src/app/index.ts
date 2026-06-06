/**
 * Book Catalog API
 *
 * A simple CRUD service for managing books using Express, Prisma, and Zod.
 *
 * This module declares package-level metadata. In the Node.js ecosystem,
 * package metadata (version, author) conventionally lives in `package.json`.
 * These constants are re-exported here to preserve the original source values
 * and provide programmatic access at runtime.
 */

// MIGRATION: The original Python `app/__init__.py` only declared package
// metadata via dunder attributes (__version__, __author__, __email__).
// In Node.js, the canonical home for this metadata is `package.json`:
//   {
//     "name": "book-catalog-api",
//     "version": "1.0.0",
//     "author": "Abdullah Arshad <abdullah.arshad.314@gmail.com>"
//   }
// Ensure these fields are added/synced in package.json. The constants below
// mirror those values for runtime access.

/** Semantic version of the application package. */
export const VERSION = "1.0.0";

/** Primary author of the application package. */
export const AUTHOR = "Abdullah Arshad";

/** Contact email for the application package author. */
export const EMAIL = "abdullah.arshad.314@gmail.com";

/** Aggregated package metadata, mirroring the original Python dunder attributes. */
export const packageMetadata = {
  version: VERSION,
  author: AUTHOR,
  email: EMAIL,
} as const;

export type PackageMetadata = typeof packageMetadata;
