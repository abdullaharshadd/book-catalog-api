// src/conftest.ts

import { execSync } from 'child_process';
import { PrismaClient } from '@prisma/client';
import { config } from './app/config';

/**
 * MIGRATION NOTES
 * ----------------
 * The source `app/tests/conftest.py` was a **pytest** configuration file that
 * provided shared SQLAlchemy + FastAPI test fixtures:
 *
 *   - `db_session_fixture` created all tables via `Base.metadata.create_all`,
 *     yielded a session, then closed it and dropped all tables
 *     (function-scoped setup/teardown).
 *   - `client_fixture` overrode FastAPI's `get_db` dependency with the test
 *     session and yielded a Starlette `TestClient`.
 *
 * In the idiomatic Node.js/TypeScript stack we use **Jest** + **supertest** +
 * **Prisma**. The mappings are:
 *
 *   - pytest fixtures -> Jest setup/teardown helpers + lifecycle hooks
 *     (`beforeAll` / `afterAll` / `beforeEach` / `afterEach`).
 *   - SQLAlchemy `create_engine` with SQLite `check_same_thread=False` ->
 *     a `PrismaClient` pointed at a test database URL (configured via
 *     `DATABASE_URL` in the test environment / `.env.test`).
 *   - `Base.metadata.create_all` / `drop_all` -> Prisma migrations applied with
 *     `prisma migrate deploy` (or `db push`). With Prisma there is no in-process
 *     metadata reflection, so schema creation is delegated to the Prisma CLI.
 *   - FastAPI `app.dependency_overrides[get_db]` -> there is no DI override in
 *     the Express + Prisma stack. `app/main.ts` imports a single shared
 *     `prisma` client. For tests we simply point that client at the test
 *     database via the environment and clean the data between tests.
 *
 * MIGRATION: This file is a *test harness*, not application code. There is no
 * direct one-to-one translation of the pytest fixture/DI-override mechanics.
 * The helpers below expose the same semantic capabilities (a fresh DB session
 * and a configured HTTP client) for use inside Jest tests. Review the test DB
 * URL strategy and whether per-test schema recreation (function scope) vs.
 * transactional truncation is desired.
 */

/**
 * The SQLite test database URL. Mirrors the source's
 * `SQLALCHEMY_DATABASE_URL = "sqlite:///./test.db"`.
 *
 * In Prisma this is read from `DATABASE_URL`; we fall back to a file-based
 * SQLite database so tests do not touch the development/production database.
 */
export const TEST_DATABASE_URL: string =
  process.env.TEST_DATABASE_URL ?? 'file:./test.db';

/**
 * A dedicated Prisma client for tests, bound to the test database URL.
 *
 * Equivalent to the source's `TestingSessionLocal` session factory bound to a
 * SQLite engine. All tests should import this client (or use the helpers
 * below) instead of the application's default client.
 */
export const testPrisma: PrismaClient = new PrismaClient({
  datasources: {
    db: {
      url: TEST_DATABASE_URL,
    },
  },
});

/**
 * createTestSchema applies the Prisma schema to the test database.
 *
 * Replaces SQLAlchemy's `Base.metadata.create_all(bind=engine)`. Prisma has no
 * in-process schema reflection, so we delegate schema creation to the Prisma
 * CLI via `prisma db push`. Returns a tuple of `[void, Error | null]` so the
 * caller can handle failures explicitly.
 */
export async function createTestSchema(): Promise<[void, Error | null]> {
  try {
    execSync('npx prisma db push --skip-generate --accept-data-loss', {
      env: { ...process.env, DATABASE_URL: TEST_DATABASE_URL },
      stdio: 'ignore',
    });
    return [undefined, null];
  } catch (err) {
    return [undefined, err instanceof Error ? err : new Error(String(err))];
  }
}

/**
 * dropTestSchema removes all data from the test database.
 *
 * Replaces SQLAlchemy's `Base.metadata.drop_all(bind=engine)`. Rather than
 * dropping tables (which would require re-running migrations for the next
 * test), we truncate every model's data, preserving the per-test isolation
 * semantics of the original function-scoped fixture.
 *
 * MIGRATION: Add a `deleteMany()` call for every Prisma model that needs
 * clearing. Only `book` is known from the migrated source; extend as the
 * schema grows.
 */
export async function dropTestSchema(): Promise<[void, Error | null]> {
  try {
    await testPrisma.book.deleteMany();
    return [undefined, null];
  } catch (err) {
    return [undefined, err instanceof Error ? err : new Error(String(err))];
  }
}

/**
 * setupTestDb provides a fresh database for a test, returning the test Prisma
 * client. Mirrors the source `db_session_fixture` setup phase
 * (`create_all` + new session).
 *
 * Intended for use inside Jest `beforeEach`/`beforeAll`:
 *
 *   beforeAll(async () => { await createTestSchema(); });
 *   beforeEach(async () => { await dropTestSchema(); });
 *   afterAll(async () => { await teardownTestDb(); });
 */
export async function setupTestDb(): Promise<[PrismaClient, Error | null]> {
  const [, schemaErr] = await createTestSchema();
  if (schemaErr) {
    return [testPrisma, schemaErr];
  }
  return [testPrisma, null];
}

/**
 * teardownTestDb closes the test Prisma client connection.
 *
 * Mirrors the `finally: db.close()` teardown of the source `db_session_fixture`.
 */
export async function teardownTestDb(): Promise<[void, Error | null]> {
  try {
    await testPrisma.$disconnect();
    return [undefined, null];
  } catch (err) {
    return [undefined, err instanceof Error ? err : new Error(String(err))];
  }
}

/**
 * MIGRATION: The source `client_fixture` overrode FastAPI's `get_db`
 * dependency and yielded a Starlette `TestClient`. In the Express + Prisma
 * stack there is no DI-override hook; the app imports a single shared `prisma`
 * client. To exercise routes against the test database in Jest, point
 * `DATABASE_URL` at `TEST_DATABASE_URL` *before* importing `app/main.ts`, then
 * wrap the exported Express `app` with supertest, e.g.:
 *
 *   process.env.DATABASE_URL = TEST_DATABASE_URL;
 *   const { app } = await import('./app/main');
 *   const request = supertest(app);
 *
 * Because the import order matters, this client construction is left to the
 * individual test files rather than being centralized here.
 */
