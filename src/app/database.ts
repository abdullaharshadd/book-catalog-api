/**
 * Database configuration and lifecycle management.
 *
 * MIGRATION: The original `app/database.py` used SQLAlchemy with a dual
 * async/sync engine setup plus FastAPI-style yield-based dependency
 * generators (`get_db` / `get_sync_db`). In the Node.js/TypeScript stack we
 * standardize on **Prisma**, which abstracts away engine/session lifecycle,
 * connection pooling, and per-request connection management. As a result:
 *
 *   - `async_engine` / `sync_engine` / `async_session` / `sync_session`
 *     have no 1:1 equivalent — Prisma exposes a single `PrismaClient`
 *     instance that manages connections internally.
 *   - The `get_db` / `get_sync_db` dependency generators are removed.
 *     Instead, the shared `prisma` client is injected into services via
 *     constructor parameters (dependency injection).
 *   - `init_db` (Base.metadata.create_all) maps to Prisma migrations
 *     (`prisma migrate dev` / `prisma migrate deploy`). The runtime
 *     `initDb` here simply verifies connectivity.
 *   - `echo=True` SQL logging maps to Prisma's `log` option, which is
 *     configured from the typed config module (disable verbose logs in
 *     production).
 *
 * The declarative models from `.models` must be expressed in
 * `prisma/schema.prisma` rather than in this file.
 */

import { PrismaClient } from '@prisma/client';
import { config } from '../config';

/**
 * Determines Prisma log levels based on the environment.
 *
 * MIGRATION: Equivalent to SQLAlchemy's `echo=True`. We enable query/info
 * logging only outside production to mirror the "Set to False in production"
 * intent from the original engines.
 */
function resolveLogLevels(): Array<'query' | 'info' | 'warn' | 'error'> {
  if (config.nodeEnv === 'production') {
    return ['warn', 'error'];
  }
  return ['query', 'info', 'warn', 'error'];
}

/**
 * The shared Prisma client instance.
 *
 * This is the single source of database access for the application,
 * replacing the SQLAlchemy async/sync engines and session factories.
 * Inject this into services/repositories via their constructors rather
 * than importing it ad-hoc, to keep call sites testable.
 *
 * MIGRATION: The connection string (previously `DATABASE_URL` /
 * `ASYNC_DATABASE_URL`) is configured through Prisma's `datasource` block
 * in `schema.prisma`, which reads `DATABASE_URL` from the environment.
 * SQLite's `check_same_thread` workaround is unnecessary — Prisma handles
 * concurrency safely on its own.
 */
export const prisma = new PrismaClient({
  log: resolveLogLevels(),
});

/**
 * Initializes the database connection.
 *
 * MIGRATION: The original `init_db` created all tables via
 * `Base.metadata.create_all`. With Prisma, schema/table creation is handled
 * by the migration workflow (`prisma migrate deploy`) run as part of
 * deployment — NOT at runtime. This function instead establishes (and
 * verifies) the connection eagerly so startup fails fast if the database
 * is unreachable.
 *
 * @returns A tuple of `[connected, error]`. `connected` is `true` on success;
 *          on failure it is `false` and `error` describes the problem.
 */
export async function initDb(): Promise<[boolean, Error | null]> {
  try {
    await prisma.$connect();
    return [true, null];
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    return [false, error];
  }
}

/**
 * Gracefully disconnects the Prisma client.
 *
 * MIGRATION: There is no source equivalent — the original code relied on
 * per-request session `close()` calls. Prisma pools connections for the
 * lifetime of the process, so we expose a single disconnect hook to be
 * called during application shutdown (e.g. on SIGTERM/SIGINT).
 *
 * @returns A tuple of `[disconnected, error]`.
 */
export async function disconnectDb(): Promise<[boolean, Error | null]> {
  try {
    await prisma.$disconnect();
    return [true, null];
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    return [false, error];
  }
}
