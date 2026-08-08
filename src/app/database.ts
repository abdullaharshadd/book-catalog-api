// src/app/database.ts
import { PrismaClient } from '@prisma/client';
import { config } from '../config';

/**
 * Database access layer.
 *
 * MIGRATION_NOTE: The Python source used SQLAlchemy with BOTH an async engine
 * and a sync engine (the sync engine was only used to create tables via
 * `Base.metadata.create_all`). In an idiomatic Node.js/TypeScript setup we use
 * Prisma, which:
 *   - is async-only (no separate sync client is needed), so the sync
 *     engine/session pair collapses into a single PrismaClient.
 *   - manages its own connection pool, so the SQLAlchemy engine/sessionmaker
 *     distinction disappears — each request shares one PrismaClient.
 *
 * MIGRATION_NOTE: The source defaulted to SQLite. The target database is
 * PostgreSQL; the connection string comes from config.DATABASE_URL (validated
 * in src/config.ts). SQLite-specific `connect_args={check_same_thread: False}`
 * has no PostgreSQL equivalent and is intentionally dropped.
 *
 * MIGRATION_NOTE: `Base.metadata.create_all` created tables from ORM models at
 * runtime. With Prisma, schema/table creation is handled out-of-band by
 * `prisma migrate` / `prisma db push` against schema.prisma. `initDb()` below
 * therefore only verifies connectivity rather than issuing DDL — running
 * migrations at runtime is NOT recommended for production.
 */

// Enable verbose query logging outside production (mirrors SQLAlchemy echo=True).
const logLevels =
  config.NODE_ENV === 'production'
    ? (['warn', 'error'] as const)
    : (['query', 'info', 'warn', 'error'] as const);

/**
 * Single shared PrismaClient instance.
 *
 * MIGRATION_NOTE: The Python code exposed a `get_db` dependency generator that
 * yielded a fresh session per request. Prisma's client is designed to be a
 * long-lived singleton that internally pools connections, so per-request
 * session creation is unnecessary and discouraged. Consumers should import
 * `prisma` directly (or receive it via constructor injection).
 */
export const prisma = new PrismaClient({
  datasources: {
    db: { url: config.DATABASE_URL },
  },
  log: [...logLevels],
});

/**
 * Initialize the database connection.
 *
 * Replaces the Python `init_db()` which created tables. Table creation is now
 * a migration-time concern; here we simply establish and validate the
 * connection so startup fails fast on a bad DATABASE_URL.
 */
export async function initDb(): Promise<void> {
  await prisma.$connect();
}

/**
 * Accessor for the shared database client.
 *
 * Replaces the Python `get_db` / `get_sync_db` dependency generators. Because
 * Prisma manages its own pool and lifecycle, there is no per-request session
 * to open/rollback/close — transaction scoping is done explicitly via
 * `prisma.$transaction(...)` where needed.
 */
export function getDb(): PrismaClient {
  return prisma;
}

/**
 * Gracefully close the database connection. Call on application shutdown.
 */
export async function closeDb(): Promise<void> {
  await prisma.$disconnect();
}
