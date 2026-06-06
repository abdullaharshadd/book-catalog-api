/**
 * Book Catalog API
 *
 * A simple CRUD service for managing books.
 *
 * MIGRATION: The original Python file (`app/__init__.py`) only exposed
 * package-level metadata (`__version__`, `__author__`, `__email__`) via the
 * dunder convention. In a Node.js/TypeScript project, this information
 * conventionally lives in `package.json` rather than a source file.
 *
 * Recommended: ensure `package.json` contains:
 *   {
 *     "version": "1.0.0",
 *     "author": "Abdullah Arshad <abdullah.arshad.314@gmail.com>"
 *   }
 *
 * The constants below are re-exported here to preserve consistent access to
 * the metadata from application code (e.g. for a `/version` health endpoint).
 * Keep these values in sync with `package.json`.
 *
 * NOTE: The source docstring references FastAPI, SQLAlchemy, and Pydantic.
 * The target stack uses Express, Prisma, and Zod per the migration plan.
 */

/** PACKAGE_VERSION is the current version of the Book Catalog API package. */
export const PACKAGE_VERSION = "1.0.0";

/** PACKAGE_AUTHOR is the author of the Book Catalog API package. */
export const PACKAGE_AUTHOR = "Abdullah Arshad";

/** PACKAGE_EMAIL is the contact email for the package author. */
export const PACKAGE_EMAIL = "abdullah.arshad.314@gmail.com";

/**
 * PackageMetadata describes the static metadata for the Book Catalog API package.
 */
export interface PackageMetadata {
  readonly version: string;
  readonly author: string;
  readonly email: string;
}

/** packageMetadata is the consolidated metadata object for the package. */
export const packageMetadata: PackageMetadata = {
  version: PACKAGE_VERSION,
  author: PACKAGE_AUTHOR,
  email: PACKAGE_EMAIL,
};
