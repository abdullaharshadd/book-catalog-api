/**
 * Book Catalog API
 *
 * A simple CRUD service for managing books.
 *
 * MIGRATION: The original source was a Python package initializer (app/__init__.py)
 * containing only package-level metadata. The docstring referenced FastAPI,
 * SQLAlchemy, and Pydantic — these framework references have been removed since
 * the target stack is Node.js/TypeScript (Express + Prisma + Zod). Update the
 * description here if the target framework details differ.
 *
 * This module is the TypeScript equivalent of the Python package metadata. In
 * Node.js, version/author/email conventionally live in package.json. The values
 * are re-exported here so existing references to package metadata continue to
 * resolve cleanly.
 */

/** Package version, mirroring the Python `__version__` dunder. */
export const VERSION = '1.0.0';

/** Package author, mirroring the Python `__author__` dunder. */
export const AUTHOR = 'Abdullah Arshad';

/** Package author email, mirroring the Python `__email__` dunder. */
export const EMAIL = 'abdullah.arshad.314@gmail.com';

/**
 * Aggregated package metadata for convenient single-import access.
 */
export const packageMetadata = {
  version: VERSION,
  author: AUTHOR,
  email: EMAIL,
} as const;

export type PackageMetadata = typeof packageMetadata;
