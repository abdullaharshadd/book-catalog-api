/**
 * Book Catalog API
 *
 * A simple CRUD service for managing books.
 *
 * This module mirrors the metadata previously declared in the Python
 * package initializer (`app/__init__.py`). In an idiomatic Node.js project,
 * package-level metadata (version, author, email) belongs in `package.json`.
 * These constants are re-exported here so application code can reference
 * them programmatically without parsing `package.json` at runtime.
 *
 * MIGRATION: The original docstring referenced FastAPI, SQLAlchemy, and
 * Pydantic — confirming this is a FastAPI project (not Django, despite the
 * `app/` path). The Node.js equivalents are Express (routing), Prisma
 * (ORM, replacing SQLAlchemy), and Zod (validation, replacing Pydantic).
 * The canonical source of `version`/`author` should be `package.json`;
 * keep these values in sync there. Consider importing from package.json
 * instead of hardcoding once the build/packaging setup is finalized.
 */

/** APP_NAME is the human-readable name of the service. */
export const APP_NAME = "Book Catalog API" as const;

/** APP_DESCRIPTION summarizes the purpose of the service. */
export const APP_DESCRIPTION =
  "A simple CRUD service for managing books." as const;

/** APP_VERSION is the semantic version of the application. */
export const APP_VERSION = "1.0.0" as const;

/** APP_AUTHOR is the primary author of the application. */
export const APP_AUTHOR = "Abdullah Arshad" as const;

/** APP_AUTHOR_EMAIL is the contact email of the primary author. */
export const APP_AUTHOR_EMAIL = "abdullah.arshad.314@gmail.com" as const;

/**
 * appMetadata bundles all package-level metadata into a single immutable
 * object for convenient consumption (e.g., exposing a `/version` endpoint
 * or populating OpenAPI/Swagger info fields).
 */
export const appMetadata = {
  name: APP_NAME,
  description: APP_DESCRIPTION,
  version: APP_VERSION,
  author: APP_AUTHOR,
  email: APP_AUTHOR_EMAIL,
} as const;

export type AppMetadata = typeof appMetadata;
