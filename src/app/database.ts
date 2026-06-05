// src/app/database.ts
import { PrismaClient } from '@prisma/client';
import { config } from '../config';

/**
 * MIGRATION NOTES (from app/database.py — FastAPI/SQLAlchemy):
 *
 * The source file used SQLAlchemy with dual sync + async engines and
 * yield-based FastAPI dependency-injection generators (get_db / get_sync_db).
 *
 * In the idiomatic Node.js stack we replace SQLAlchemy with Prisma:
 *   - Prisma is async-first, so there is no need for a separate sync engine.
 *     The original sync engine existed only to run create_all() for table
 *     creation and for synchronous testing. With Prisma, schema/table
 *     management is handled by `prisma migrate` / `prisma db push`, so the
 *     parallel sync engine is intentionally not reproduced.
 *   - `echo=True` (SQL logging) is preserved but made environment-conditional
 *     via the typed config module (logging only enabled outside production).
 *   - SQLite-specific connect_args ({check_same_thread: False}) is a Python
 *     threading concern that does not apply to Node.js / Prisma, so it is
 *     dropped.
 *   - expire_on_commit=False has no Prisma equivalent: Prisma returns plain
 *     JS objects detached from any session, so objects remain fully
 *     accessible after a write by default. Behavior is preserved implicitly.
 *
 * MIGRATION: The original `init_db` used SQLAlchemy's Base.metadata.create_all
 * to create tables at runtime. The recommended Node.js approach is to use
 * Prisma migrations (`npx prisma migrate deploy`) instead of runtime table
 * creation. `initDb` below verifies connectivity rather than creating tables —
 * review whether you want to invoke migrations programmatically here.
 *
 * MIGRATION: The yield-based DI generators (get_db / get_sync_db) map onto a
 * single shared PrismaClient instance. Prisma manages its own connection pool,
 * so per-request session creation/close is unnecessary. If you need
 * request-scoped transactions, use `prisma.$transaction(...)` at the route
 * level instead.
 */

/**
 * DatabaseError represents a failure interacting with the database layer.
 */
export interface DatabaseError {
  error: string;
  details?: unknown;
}

/**
 * Result is a tuple of [value, error] used for all fallible operations.
 * On success, error is null. On failure, value is null.
 */
export type Result<T> = [T | null, DatabaseError | null];

/**
 * createPrismaClient builds a configured PrismaClient instance.
 *
 * SQL query logging is enabled only when not running in production,
 * mirroring the source's `echo=True  # Set to False in production`.
 */
export function createPrismaClient(): PrismaClient {
  return new PrismaClient({
    datasources: {
      db: {
        url: config.databaseUrl,
      },
    },
    log: config.isProduction
      ? ['error']
      : ['query', 'info', 'warn', 'error'],
  });
}

/**
 * prisma is the shared, application-wide database client.
 *
 * Prisma manages an internal connection pool, replacing the dual
 * sync/async engine + sessionmaker setup from the SQLAlchemy source.
 * Prefer injecting this instance via constructor parameters rather than
 * importing it directly in business logic, to keep code testable.
 */
export const prisma: PrismaClient = createPrismaClient();

/**
 * initDb verifies the database is reachable.
 *
 * In the SQLAlchemy source this created all tables via
 * Base.metadata.create_all. With Prisma, schema management is handled by
 * migrations; this function instead ensures connectivity so the app fails
 * fast on startup if the database is unavailable.
 *
 * Returns [true, null] on success or [null, DatabaseError] on failure.
 */
export async function initDb(
  client: PrismaClient = prisma,
): Promise<Result<true>> {
  try {
    await client.$connect();
    return [true, null];
  } catch (err) {
    return [
      null,
      { error: 'Failed to initialize database connection', details: err },
    ];
  }
}

/**
 * getDb returns the shared PrismaClient instance for use in request handlers.
 *
 * This replaces the FastAPI `get_db` dependency generator. Because Prisma
 * pools connections internally, no per-request open/close lifecycle is
 * required. For transactional, request-scoped work, use
 * `prisma.$transaction(...)` within the route.
 */
export function getDb(): PrismaClient {
  return prisma;
}

/**
 * closeDb gracefully disconnects the Prisma client.
 *
 * Call this on application shutdown. Returns [true, null] on success or
 * [null, DatabaseError] on failure.
 */
export async function closeDb(
  client: PrismaClient = prisma,
): Promise<Result<true>> {
  try {
    await client.$disconnect();
    return [true, null];
  } catch (err) {
    return [
      null,
      { error: 'Failed to close database connection', details: err },
    ];
  }
}
