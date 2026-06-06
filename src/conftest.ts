/**
 * Test harness / fixtures for the Book Catalog API.
 *
 * MIGRATION: The original `app/tests/conftest.py` was a pytest configuration
 * file providing FastAPI/SQLAlchemy test fixtures. In the Node.js/TypeScript
 * stack we use **Jest** + **supertest** + **Prisma**, so the mapping is:
 *
 *   - SQLAlchemy `create_engine` + `sessionmaker` + `Base.metadata.create_all`
 *     -> a dedicated test `PrismaClient` pointed at an isolated SQLite test
 *     database. Schema setup/teardown is performed with `prisma db push`
 *     (or `prisma migrate deploy`) rather than per-test `create_all`/`drop_all`.
 *   - The `db_session` pytest fixture -> `setupTestDatabase()` /
 *     `teardownTestDatabase()` helpers, typically wired into Jest's
 *     `beforeAll`/`afterAll` (or `beforeEach`/`afterEach`) lifecycle hooks.
 *   - FastAPI's `app.dependency_overrides[get_db] = override_get_db` pattern
 *     has no Prisma equivalent: in this stack the `PrismaClient` is injected
 *     directly into route factories (constructor-style DI in `app/main.ts`),
 *     so we simply build the Express `app` with the *test* client and hand a
 *     ready-to-use supertest agent to tests.
 *   - The Starlette `TestClient` -> a `supertest` request agent bound to the
 *     Express app.
 *
 * Because Prisma manages connection lifecycle internally, there is no
 * `check_same_thread` / engine / session bookkeeping to replicate.
 */

import { execSync } from 'child_process';
import { PrismaClient } from '@prisma/client';
import supertest, { SuperTest, Test } from 'supertest';
import { Express } from 'express';

import { createApp } from './app/main';

/**
 * SQLite test database URL.
 *
 * MIGRATION: mirrors the original `sqlite:///./test.db`. An on-disk file is
 * used (rather than `:memory:`) because Prisma's SQLite connector spins up a
 * fresh connection per query — an in-memory DB would not be shared across
 * those connections. Override via the `TEST_DATABASE_URL` env var if needed.
 */
export const TEST_DATABASE_URL: string =
  process.env.TEST_DATABASE_URL ?? 'file:./test.db';

/**
 * TestContext bundles everything a test suite needs: the isolated Prisma
 * client, the Express app it is wired into, and a ready supertest agent.
 */
export interface TestContext {
  prisma: PrismaClient;
  app: Express;
  client: SuperTest<Test>;
}

/**
 * createTestPrismaClient builds a PrismaClient pointed at the isolated test
 * database. Equivalent to the SQLAlchemy `engine` + `TestingSessionLocal`
 * setup in the original fixture.
 */
export function createTestPrismaClient(): PrismaClient {
  return new PrismaClient({
    datasources: { db: { url: TEST_DATABASE_URL } },
  });
}

/**
 * setupTestDatabase provisions an isolated test database, applies the Prisma
 * schema, connects a fresh client, and returns a fully wired TestContext.
 *
 * MIGRATION: replaces the `db_session` + `client` pytest fixtures. The schema
 * is materialized via `prisma db push` (analogous to `Base.metadata.create_all`).
 *
 * Returns a `[TestContext, Error | null]` tuple per the project's fallible
 * operation convention.
 */
export async function setupTestDatabase(): Promise<[TestContext | null, Error | null]> {
  try {
    // Materialize the schema in the isolated test DB. `--force-reset` ensures
    // a clean slate per setup, replicating create_all/drop_all isolation.
    execSync('npx prisma db push --force-reset --skip-generate', {
      stdio: 'ignore',
      env: { ...process.env, DATABASE_URL: TEST_DATABASE_URL },
    });

    const prisma = createTestPrismaClient();
    await prisma.$connect();

    const app = createApp(prisma);
    const client = supertest(app);

    return [{ prisma, app, client }, null];
  } catch (err) {
    return [null, err instanceof Error ? err : new Error(String(err))];
  }
}

/**
 * teardownTestDatabase disconnects the Prisma client and clears all data,
 * replacing the `Base.metadata.drop_all` + `db.close()` teardown in the
 * original fixture's `finally` block.
 *
 * Returns an `Error | null` per the project's fallible operation convention.
 */
export async function teardownTestDatabase(
  ctx: TestContext,
): Promise<Error | null> {
  try {
    // Best-effort wipe of all rows so subsequent suites start clean even if
    // the same DB file is reused.
    await ctx.prisma.book.deleteMany();
    await ctx.prisma.$disconnect();
    return null;
  } catch (err) {
    return err instanceof Error ? err : new Error(String(err));
  }
}
