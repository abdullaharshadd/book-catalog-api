// src/conftest.ts

/**
 * Shared test setup / fixtures for the Book Catalog API test suite.
 *
 * MIGRATION_NOTE: The Python source (`app/tests/conftest.py`) was a pytest
 * `conftest.py` providing two fixtures:
 *   - `db_session`: created a SQLAlchemy engine against a SQLite test DB,
 *     created all tables before a test, yielded a session, then closed the
 *     session and dropped all tables afterward.
 *   - `client`: overrode FastAPI's `get_db` dependency to use the test session
 *     and wrapped the app in Starlette's `TestClient`.
 *
 * There is no direct equivalent of `conftest.py` in the Jest/TypeScript world:
 *   - pytest auto-discovers `conftest.py` and injects fixtures by name into
 *     test functions. Jest has no fixture-injection mechanism; instead each
 *     test file imports helpers explicitly and uses `beforeAll`/`afterAll`/
 *     `beforeEach`/`afterEach` lifecycle hooks.
 *   - Starlette's `TestClient` is replaced by `supertest`, which wraps the
 *     Express app returned by `createApp()` directly — no dependency-override
 *     machinery is needed because our app uses the shared Prisma client from
 *     `src/app/database.ts` rather than FastAPI's `Depends(get_db)` injection.
 *
 * MIGRATION_NOTE: Target DB is PostgreSQL (via Prisma), NOT SQLite. The source
 * used an in-memory/file SQLite database purely for test isolation. In the
 * target stack the test database is configured through the `DATABASE_URL`
 * environment variable (a dedicated Postgres test database). We therefore do
 * NOT translate the SQLite engine/sessionmaker setup literally — that would be
 * mirroring the source dialect, which is explicitly discouraged.
 *
 * This module exports reusable helpers that test files import to replicate the
 * setup/teardown semantics of the two pytest fixtures.
 */

import type { Express } from 'express';
import request from 'supertest';
import { createApp } from './app/main';
import { initDb, closeDb, prisma } from './app/database';

/**
 * Equivalent of the pytest `db_session` fixture's *setup* phase.
 *
 * MIGRATION_NOTE: `Base.metadata.create_all(bind=engine)` created the schema.
 * With Prisma + PostgreSQL, the schema is managed by migrations
 * (`prisma migrate deploy` / `prisma db push`) run out-of-band before the test
 * process starts. Here we only ensure the DB connection is established and the
 * table is empty so each test starts from a clean state — mirroring the
 * source's per-test create/drop isolation.
 */
export async function setupTestDb(): Promise<void> {
  await initDb();
  await resetTestDb();
}

/**
 * Truncate all test data. Called between tests to preserve the source's
 * per-test isolation (source dropped & recreated all tables per fixture use).
 *
 * MIGRATION_NOTE: `Base.metadata.drop_all` dropped tables; dropping tables
 * under Prisma-managed migrations is undesirable because the schema is owned by
 * migrations. Truncating with RESTART IDENTITY + CASCADE gives the same clean
 * slate (including resetting the SERIAL/IDENTITY sequence) without destroying
 * the migrated schema.
 */
export async function resetTestDb(): Promise<void> {
  await prisma.$executeRawUnsafe(
    'TRUNCATE TABLE "books" RESTART IDENTITY CASCADE',
  );
}

/**
 * Equivalent of the pytest `db_session` fixture's *teardown* phase: close the
 * DB connection so the process can exit cleanly and connections are released.
 */
export async function teardownTestDb(): Promise<void> {
  await closeDb();
}

/**
 * Equivalent of the pytest `client` fixture.
 *
 * MIGRATION_NOTE: The source used `app.dependency_overrides[get_db]` to inject
 * the test session and wrapped `app` in Starlette's `TestClient`. In Express +
 * supertest there is no dependency-override layer: the app built by
 * `createApp()` already uses the shared Prisma client, which points at the
 * test database via `DATABASE_URL`. We simply build the app and hand back a
 * bound supertest agent.
 *
 * Returns both the raw Express app (for advanced use) and a supertest agent
 * that test files use to issue HTTP requests.
 */
export function createTestClient(): {
  app: Express;
  client: request.SuperTest<request.Test>;
} {
  const app = createApp();
  const client = request(app);
  return { app, client };
}

/**
 * Convenience: full lifecycle wiring for a test file.
 *
 * Usage in a `*.test.ts` file:
 *
 *   const ctx = useTestApp();
 *   it('creates a book', async () => {
 *     const res = await ctx.client.post('/books').send({ ... });
 *     expect(res.status).toBe(201);
 *   });
 *
 * This registers Jest lifecycle hooks that reproduce the create/yield/drop
 * semantics of the combined `db_session` + `client` pytest fixtures:
 *   - beforeAll  -> connect + build client
 *   - beforeEach -> reset data (clean slate per test)
 *   - afterAll   -> close DB connection
 */
export function useTestApp(): {
  readonly app: Express;
  readonly client: request.SuperTest<request.Test>;
} {
  const ctx: {
    app: Express | undefined;
    client: request.SuperTest<request.Test> | undefined;
  } = { app: undefined, client: undefined };

  beforeAll(async () => {
    await setupTestDb();
    const created = createTestClient();
    ctx.app = created.app;
    ctx.client = created.client;
  });

  beforeEach(async () => {
    await resetTestDb();
  });

  afterAll(async () => {
    await teardownTestDb();
  });

  return {
    get app(): Express {
      if (!ctx.app) {
        throw new Error('Test app accessed before beforeAll() ran');
      }
      return ctx.app;
    },
    get client(): request.SuperTest<request.Test> {
      if (!ctx.client) {
        throw new Error('Test client accessed before beforeAll() ran');
      }
      return ctx.client;
    },
  };
}
