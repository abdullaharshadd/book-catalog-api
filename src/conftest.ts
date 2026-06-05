/**
 * Test setup utilities (migrated from app/tests/conftest.py).
 *
 * MIGRATION NOTE:
 * The source file is a pytest `conftest.py` for a FastAPI + SQLAlchemy app.
 * In the idiomatic Node.js/TypeScript + Prisma stack, the concepts map as follows:
 *   - SQLAlchemy engine/session        -> Prisma client pointed at a test database
 *   - Base.metadata.create_all/drop_all -> `prisma migrate deploy` / `prisma db push` against the test DB
 *   - FastAPI app.dependency_overrides  -> dependency injection of the PrismaClient into the Express app factory
 *   - Starlette TestClient              -> supertest wrapping the Express app
 *
 * Because Prisma manages its own connection pool and there is no per-request
 * session object to override, the `db_session` / `client` fixture split collapses
 * into a single set of helpers that build an isolated Prisma client and a
 * supertest-ready Express app.
 */

import { execSync } from 'child_process';
import { PrismaClient } from '@prisma/client';
import type { Express } from 'express';
import request from 'supertest';
import type SuperTest from 'supertest';

import { createApp } from './app';

/**
 * Connection string for the isolated test database.
 *
 * Mirrors the source `SQLALCHEMY_DATABASE_URL = "sqlite:///./test.db"`.
 * Override via the TEST_DATABASE_URL env var when running against another engine.
 *
 * MIGRATION: Prisma requires the provider in schema.prisma to match this URL
 * (e.g. `provider = "sqlite"`). Keep schema.prisma in sync with the engine used here.
 */
export const TEST_DATABASE_URL =
  process.env.TEST_DATABASE_URL ?? 'file:./test.db';

/**
 * Result tuple convention: [value, error]. `error` is null on success.
 */
export type Result<T> = [T, Error | null];

/**
 * createTestDb builds an isolated Prisma client bound to the test database and
 * ensures the schema exists (analogous to Base.metadata.create_all).
 *
 * Returns [prisma, error]; the caller must invoke the returned dispose() to tear
 * down (analogous to db.close() + Base.metadata.drop_all).
 */
export async function createTestDb(): Promise<
  Result<{ prisma: PrismaClient; dispose: () => Promise<Result<void>> }>
> {
  try {
    // Create tables for the test DB. `db push --force-reset` gives us a clean
    // schema each run, replacing SQLAlchemy's create_all.
    execSync('npx prisma db push --force-reset --skip-generate', {
      stdio: 'ignore',
      env: { ...process.env, DATABASE_URL: TEST_DATABASE_URL },
    });

    const prisma = new PrismaClient({
      datasources: { db: { url: TEST_DATABASE_URL } },
    });

    await prisma.$connect();

    const dispose = async (): Promise<Result<void>> => {
      try {
        await prisma.$disconnect();
        // Drop tables after tests are done (mirrors Base.metadata.drop_all).
        execSync('npx prisma db push --force-reset --skip-generate', {
          stdio: 'ignore',
          env: { ...process.env, DATABASE_URL: TEST_DATABASE_URL },
        });
        return [undefined, null];
      } catch (err) {
        return [undefined, err instanceof Error ? err : new Error(String(err))];
      }
    };

    return [{ prisma, dispose }, null];
  } catch (err) {
    return [
      { prisma: undefined as unknown as PrismaClient, dispose: async () => [undefined, null] },
      err instanceof Error ? err : new Error(String(err)),
    ];
  }
}

/**
 * createTestClient builds an Express app wired to the provided Prisma client and
 * returns a supertest agent for issuing HTTP requests.
 *
 * This replaces the FastAPI `app.dependency_overrides[get_db]` mechanism: instead
 * of overriding a request-scoped dependency, we inject the test PrismaClient
 * directly into the app factory (constructor-style dependency injection).
 */
export function createTestClient(prisma: PrismaClient): {
  app: Express;
  client: SuperTest.SuperTest<SuperTest.Test>;
} {
  const app = createApp({ prisma });
  return { app, client: request(app) as unknown as SuperTest.SuperTest<SuperTest.Test> };
}

/**
 * withTestContext provides a yield-style setup/teardown akin to the pytest
 * `db_session` + `client` fixtures. It creates the DB, runs the callback with a
 * Prisma client and supertest client, and guarantees teardown.
 *
 * Usage:
 *   it('GET /things', async () => {
 *     await withTestContext(async ({ client }) => {
 *       await client.get('/things').expect(200);
 *     });
 *   });
 */
export async function withTestContext(
  fn: (ctx: {
    prisma: PrismaClient;
    app: Express;
    client: SuperTest.SuperTest<SuperTest.Test>;
  }) => Promise<void>,
): Promise<Result<void>> {
  const [db, dbErr] = await createTestDb();
  if (dbErr) {
    return [undefined, dbErr];
  }

  try {
    const { app, client } = createTestClient(db.prisma);
    await fn({ prisma: db.prisma, app, client });
    return [undefined, null];
  } catch (err) {
    return [undefined, err instanceof Error ? err : new Error(String(err))];
  } finally {
    await db.dispose();
  }
}
