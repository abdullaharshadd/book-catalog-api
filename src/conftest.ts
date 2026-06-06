// src/conftest.ts
//
// Test harness setup for the application's Jest + supertest integration tests.
//
// MIGRATION: The Python source is a pytest conftest.py using FastAPI's
// `app.dependency_overrides` + SQLAlchemy's in-memory SQLite engine. There is no
// direct Node.js equivalent of pytest fixtures, so this file exposes reusable
// helper functions (`createTestDb`, `createTestClient`) that test files can
// call in their `beforeAll`/`afterAll`/`beforeEach`/`afterEach` hooks.
//
// The original FastAPI `get_db` dependency override is modelled here by swapping
// the Prisma client used by the app at test time. This assumes the application
// is structured with an injectable Prisma client rather than a global singleton.
// If your app uses a global Prisma instance, you must refactor it to accept an
// injected client (constructor / app.locals) for this override to work.

import { PrismaClient } from '@prisma/client';
import { execSync } from 'child_process';
import supertest, { SuperTest, Test } from 'supertest';
import type { Application } from 'express';

// MIGRATION: `import { app } from '../app'` — adjust this path to wherever your
// Express application instance is exported. The original imported `app` from
// `app.main`.
import { createApp } from './app';

/**
 * Database URL used exclusively by the test suite.
 *
 * MIGRATION: The Python source used SQLite (`sqlite:///./test.db`). With Prisma
 * the connection string lives in the `DATABASE_URL` environment variable. We set
 * a dedicated test database URL here. For an in-memory equivalent you can point
 * this at a disposable SQLite file (`file:./test.db`) when your Prisma schema
 * provider is `sqlite`.
 */
export const TEST_DATABASE_URL: string =
  process.env.TEST_DATABASE_URL ?? 'file:./test.db';

/**
 * Result of {@link createTestDb}: the live Prisma client plus a teardown
 * function. Following the project convention, fallible operations return a
 * `[value, error]` tuple.
 */
export interface TestDb {
  prisma: PrismaClient;
  /** Drops all data / disconnects — call from `afterAll`/`afterEach`. */
  teardown: () => Promise<Error | null>;
}

/**
 * createTestDb provisions an isolated Prisma client backed by the test database
 * and runs the schema migrations so all tables exist before tests run.
 *
 * It mirrors the SQLAlchemy `Base.metadata.create_all(bind=engine)` setup and
 * the `db_session` fixture from the original conftest.py.
 *
 * @returns a `[TestDb, error]` tuple — `error` is non-null on failure.
 */
export async function createTestDb(): Promise<[TestDb | null, Error | null]> {
  try {
    // Apply the Prisma schema to the test database (equivalent of create_all).
    // MIGRATION: `prisma db push` is used instead of SQLAlchemy's create_all.
    // For real migration-based projects prefer `prisma migrate deploy`.
    execSync('npx prisma db push --skip-generate --accept-data-loss', {
      env: { ...process.env, DATABASE_URL: TEST_DATABASE_URL },
      stdio: 'ignore',
    });

    const prisma = new PrismaClient({
      datasources: { db: { url: TEST_DATABASE_URL } },
    });

    await prisma.$connect();

    const teardown = async (): Promise<Error | null> => {
      try {
        // Equivalent of Base.metadata.drop_all — reset the schema between runs.
        execSync('npx prisma db push --force-reset --skip-generate --accept-data-loss', {
          env: { ...process.env, DATABASE_URL: TEST_DATABASE_URL },
          stdio: 'ignore',
        });
        await prisma.$disconnect();
        return null;
      } catch (err) {
        return err instanceof Error ? err : new Error(String(err));
      }
    };

    return [{ prisma, teardown }, null];
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    return [null, error];
  }
}

/**
 * Result of {@link createTestClient}: a supertest agent bound to an Express app
 * that has had its Prisma client overridden with the test database client.
 */
export interface TestClient {
  client: SuperTest<Test>;
  app: Application;
}

/**
 * createTestClient builds an Express application wired to the provided test
 * Prisma client and returns a supertest agent for issuing HTTP requests.
 *
 * This replaces the FastAPI `client` fixture which used
 * `app.dependency_overrides[get_db] = override_get_db` to inject the test DB
 * session, followed by `TestClient(app)`.
 *
 * MIGRATION: `createApp` is expected to accept a Prisma client so the test DB
 * can be injected. If your app exposes a pre-built singleton instead, refactor
 * `createApp` to take a dependencies object `{ prisma }`.
 *
 * @param prisma the Prisma client created by {@link createTestDb}.
 * @returns a `[TestClient, error]` tuple — `error` is non-null on failure.
 */
export async function createTestClient(
  prisma: PrismaClient,
): Promise<[TestClient | null, Error | null]> {
  try {
    const app = createApp({ prisma });
    const client = supertest(app);
    return [{ client, app }, null];
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    return [null, error];
  }
}

/**
 * setupTestContext is a convenience helper that combines {@link createTestDb}
 * and {@link createTestClient} for the common case where a test needs both a
 * database and an HTTP client. Use it inside a `beforeAll`/`beforeEach` block:
 *
 * ```ts
 * let ctx: TestContext;
 * beforeAll(async () => {
 *   const [c, err] = await setupTestContext();
 *   if (err) throw err;
 *   ctx = c!;
 * });
 * afterAll(async () => { await ctx.teardown(); });
 * ```
 *
 * @returns a `[TestContext, error]` tuple — `error` is non-null on failure.
 */
export interface TestContext {
  prisma: PrismaClient;
  client: SuperTest<Test>;
  app: Application;
  teardown: () => Promise<Error | null>;
}

export async function setupTestContext(): Promise<
  [TestContext | null, Error | null]
> {
  const [db, dbErr] = await createTestDb();
  if (dbErr || !db) {
    return [null, dbErr ?? new Error('failed to create test db')];
  }

  const [tc, tcErr] = await createTestClient(db.prisma);
  if (tcErr || !tc) {
    // Best-effort cleanup of the DB we just created.
    await db.teardown();
    return [null, tcErr ?? new Error('failed to create test client')];
  }

  return [
    {
      prisma: db.prisma,
      client: tc.client,
      app: tc.app,
      teardown: db.teardown,
    },
    null,
  ];
}
