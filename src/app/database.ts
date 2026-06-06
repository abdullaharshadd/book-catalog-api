// src/app/database.ts
import { PrismaClient } from '@prisma/client';
import { config } from './config';

/**
 * MIGRATION NOTES
 * ----------------
 * The source `app/database.py` configured SQLAlchemy with BOTH an async and a
 * sync engine/session factory, plus FastAPI-style dependency-injection
 * generators (`get_db` / `get_sync_db`) that yielded a session per request and
 * handled rollback/close in a try/except/finally block.
 *
 * In the idiomatic Node.js/TypeScript stack we use **Prisma** as the ORM:
 *
 *   - A single `PrismaClient` instance manages the connection pool. There is no
 *     need for separate "async" and "sync" engines — all Prisma calls are
 *     promise-based (async).
 *   - Per-request session lifecycle (open/rollback/close) is handled
 *     automatically by Prisma. The FastAPI `get_db` / `get_sync_db` DI
 *     generators therefore have no direct equivalent. Instead we export the
 *     shared client and a `getDb()` accessor that controllers/services receive
 *     via constructor injection.
 *   - For explicit transactional rollback semantics (the try/except/rollback in
 *     the source), use `prisma.$transaction(...)` at the call site.
 *   - `echo=True` (SQL logging) maps to Prisma's `log` option, made conditional
 *     on the environment via the typed config module.
 *   - `Base.metadata.create_all` (runtime table creation) maps to Prisma
 *     migrations (`prisma migrate deploy`), NOT runtime DDL. `initDb()` is kept
 *     for parity but only verifies connectivity; schema creation must be done
 *     via migrations. See MIGRATION comment on `initDb`.
 *   - DATABASE_URL / sqlite `check_same_thread` handling is delegated to Prisma
 *     and the `datasource` block in `schema.prisma`.
 */

/**
 * Result tuple returned by fallible operations: `[value, error]`.
 * On success `error` is `null`; on failure `value` is `null`.
 */
export type Result<T> = [T, null] | [null, Error];

/**
 * Build the Prisma log levels based on the current environment.
 *
 * MIGRATION: equivalent to SQLAlchemy's `echo=True`. Verbose query logging is
 * enabled outside production; disabled in production to match the source's
 * "Set to False in production" intent.
 */
function resolveLogLevels(): ('query' | 'info' | 'warn' | 'error')[] {
  if (config.nodeEnv === 'production') {
    return ['warn', 'error'];
  }
  return ['query', 'info', 'warn', 'error'];
}

/**
 * Singleton PrismaClient.
 *
 * MIGRATION: replaces both `async_engine`/`async_session` and
 * `sync_engine`/`sync_session`. Prisma exposes a single promise-based client;
 * a separate synchronous engine is unnecessary in Node.js.
 *
 * The connection URL is taken from the typed config module (which reads
 * `DATABASE_URL`), mirroring the source `os.getenv("DATABASE_URL", ...)`.
 */
export const prisma = new PrismaClient({
  log: resolveLogLevels(),
  datasources: {
    db: {
      url: config.databaseUrl,
    },
  },
});

/**
 * Returns the shared PrismaClient instance.
 *
 * MIGRATION: replaces the FastAPI `get_db` dependency generator. Prisma manages
 * the connection pool and per-query lifecycle automatically, so there is no
 * per-request session to open/rollback/close here. Inject this client into
 * services/controllers via their constructors.
 *
 * For the explicit rollback semantics that `get_db` provided around a unit of
 * work, wrap related operations in `prisma.$transaction(...)` at the call site.
 */
export function getDb(): PrismaClient {
  return prisma;
}

/**
 * Initialize the database connection.
 *
 * MIGRATION: the source `init_db` called `Base.metadata.create_all` to create
 * tables at runtime. In a Prisma project, schema/table creation is handled by
 * migrations (`prisma migrate deploy`) and must NOT be performed at runtime.
 * This function therefore only establishes/verifies the connection so startup
 * fails fast if the database is unreachable.
 *
 * Returns a `Result<void>`: `[undefined, null]` on success or `[null, Error]`
 * on failure.
 */
export async function initDb(): Promise<Result<void>> {
  try {
    await prisma.$connect();
    return [undefined as unknown as void, null];
  } catch (err) {
    return [null, err instanceof Error ? err : new Error(String(err))];
  }
}

/**
 * Gracefully disconnect the PrismaClient.
 *
 * MIGRATION: there was no explicit shutdown hook in the source (FastAPI/
 * SQLAlchemy engines are torn down implicitly). In Node.js it is good practice
 * to disconnect on application shutdown to drain the connection pool.
 *
 * Returns a `Result<void>`.
 */
export async function closeDb(): Promise<Result<void>> {
  try {
    await prisma.$disconnect();
    return [undefined as unknown as void, null];
  } catch (err) {
    return [null, err instanceof Error ? err : new Error(String(err))];
  }
}
