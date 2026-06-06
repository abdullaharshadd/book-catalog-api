// src/app/database.ts
import { PrismaClient } from '@prisma/client';
import { config } from './config';

/**
 * MIGRATION: The original Python file used SQLAlchemy with BOTH an async and a
 * sync engine/session factory. In the Node.js/TypeScript ecosystem, Prisma is
 * fully async by design, so there is a single client rather than two engines.
 *
 * Mapping of original concepts -> Node/Prisma:
 *   - DATABASE_URL / ASYNC_DATABASE_URL  -> single `datasource db { url }` in
 *     schema.prisma, surfaced through the typed config module.
 *   - async_engine / sync_engine          -> a single PrismaClient instance.
 *   - async_session / sync_session         -> Prisma manages connection
 *     pooling internally; there is no per-request session factory.
 *   - get_db / get_sync_db (FastAPI deps)  -> dependency-injected PrismaClient
 *     passed into route handlers / services via constructor.
 *   - Base.metadata.create_all (init_db)   -> Prisma migrations
 *     (`prisma migrate deploy`). Runtime table creation is NOT idiomatic;
 *     see initDb() note below.
 *   - echo=True (SQL logging)              -> PrismaClient `log` option, driven
 *     by config (verbose only outside production).
 *   - check_same_thread (SQLite-specific)  -> handled by Prisma's SQLite driver;
 *     no equivalent flag needed.
 */

/**
 * Result tuple used for all fallible operations: [value, error].
 * On success `error` is null; on failure `value` is null.
 */
export type Result<T> = [T, null] | [null, Error];

/**
 * Singleton-style PrismaClient holder. We instantiate lazily so the log level
 * (the equivalent of SQLAlchemy `echo`) can be derived from config.
 *
 * MIGRATION: SQLAlchemy exposed module-level engine/session objects. Per the
 * dependency-injection requirement, prefer passing `prisma` into your services
 * rather than importing this global directly.
 */
let prismaInstance: PrismaClient | null = null;

/**
 * getPrisma returns the shared PrismaClient, creating it on first use.
 *
 * The `log` option mirrors the original `echo=True` flag: verbose query
 * logging is enabled only outside production.
 */
export function getPrisma(): PrismaClient {
  if (prismaInstance === null) {
    prismaInstance = new PrismaClient({
      log: config.nodeEnv === 'production'
        ? ['warn', 'error']
        : ['query', 'info', 'warn', 'error'],
    });
  }
  return prismaInstance;
}

/**
 * prisma is the default exported client instance, provided for convenience.
 * Prefer injecting it via constructor parameters where possible.
 */
export const prisma: PrismaClient = getPrisma();

/**
 * initDb initializes database connectivity.
 *
 * MIGRATION: The original `init_db` called `Base.metadata.create_all`, which
 * creates tables at runtime. In the Prisma/Node workflow, schema creation is
 * handled by migrations (`npx prisma migrate deploy`) run as a separate step,
 * NOT at application startup. This function therefore only validates that the
 * database connection can be established (equivalent intent: "the DB is ready").
 *
 * Also note the original was declared `async` but performed only synchronous
 * work; here connecting is genuinely async.
 *
 * @returns a Result tuple — [void, null] on success or [null, Error] on failure.
 */
export async function initDb(): Promise<Result<void>> {
  try {
    await getPrisma().$connect();
    return [undefined as unknown as void, null];
  } catch (err) {
    return [null, err instanceof Error ? err : new Error(String(err))];
  }
}

/**
 * withSession runs `fn` against the Prisma client inside a transaction,
 * preserving the original try/rollback/finally lifecycle of `get_db`.
 *
 * MIGRATION: SQLAlchemy's yield-based FastAPI dependency (get_db) opened a
 * session, yielded it, rolled back on error, and always closed it. Express has
 * no generator-based DI; the closest idiomatic equivalent is to wrap a unit of
 * work in a transaction. Prisma automatically rolls back if the callback
 * throws, matching the `await session.rollback()` behaviour. Connection
 * release (the `session.close()` step) is managed by Prisma's pool.
 *
 * @param fn callback receiving a transactional Prisma client.
 * @returns a Result tuple with the callback's value or an Error.
 */
export async function withSession<T>(
  fn: (tx: PrismaClient) => Promise<T>,
): Promise<Result<T>> {
  try {
    const value = await getPrisma().$transaction(
      (tx) => fn(tx as unknown as PrismaClient),
    );
    return [value, null];
  } catch (err) {
    return [null, err instanceof Error ? err : new Error(String(err))];
  }
}

/**
 * disconnectDb cleanly closes the Prisma connection pool. Call this during
 * graceful shutdown. There was no direct equivalent in the source file, but
 * it represents the engine/session disposal that SQLAlchemy left implicit.
 *
 * @returns a Result tuple — [void, null] on success or [null, Error] on failure.
 */
export async function disconnectDb(): Promise<Result<void>> {
  try {
    if (prismaInstance !== null) {
      await prismaInstance.$disconnect();
      prismaInstance = null;
    }
    return [undefined as unknown as void, null];
  } catch (err) {
    return [null, err instanceof Error ? err : new Error(String(err))];
  }
}
