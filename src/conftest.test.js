```typescript
// src/conftest.test.ts

import type { Express } from 'express';
import request from 'supertest';

// ---------------------------------------------------------------------------
// Mock external dependencies BEFORE importing the module under test
// ---------------------------------------------------------------------------

const mockInitDb = jest.fn<Promise<void>, []>();
const mockCloseDb = jest.fn<Promise<void>, []>();
const mockExecuteRawUnsafe = jest.fn<Promise<unknown>, [string]>();

jest.mock('./app/database', () => ({
  initDb: (...args: unknown[]) => mockInitDb(...(args as [])),
  closeDb: (...args: unknown[]) => mockCloseDb(...(args as [])),
  prisma: {
    $executeRawUnsafe: (...args: unknown[]) =>
      mockExecuteRawUnsafe(args[0] as string),
  },
}));

const mockCreateApp = jest.fn<Express, []>();

jest.mock('./app/main', () => ({
  createApp: () => mockCreateApp(),
}));

// ---------------------------------------------------------------------------
// Now import the module under test
// ---------------------------------------------------------------------------

import {
  setupTestDb,
  resetTestDb,
  teardownTestDb,
  createTestClient,
  useTestApp,
} from './conftest';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Minimal Express-like stub that supertest is happy wrapping. */
function makeExpressStub(): Express {
  const stub = (
    _req: unknown,
    _res: unknown,
    _next: unknown,
  ): void => void 0;
  // supertest only needs .listen(); add a no-op version
  (stub as unknown as { listen: jest.Mock }).listen = jest
    .fn()
    .mockImplementation((_port: number, cb?: () => void) => {
      cb?.();
      return { close: jest.fn(), on: jest.fn() };
    });
  return stub as unknown as Express;
}

// ---------------------------------------------------------------------------
// Test suite
// ---------------------------------------------------------------------------

beforeEach(() => {
  jest.clearAllMocks();
  mockInitDb.mockResolvedValue(undefined);
  mockCloseDb.mockResolvedValue(undefined);
  mockExecuteRawUnsafe.mockResolvedValue(undefined);
  mockCreateApp.mockReturnValue(makeExpressStub());
});

// ===========================================================================
// setupTestDb
// ===========================================================================

describe('setupTestDb', () => {
  it('initialises the DB connection (initDb)', async () => {
    await setupTestDb();

    expect(mockInitDb).toHaveBeenCalledTimes(1);
  });

  it('truncates the books table after connecting (resetTestDb side-effect)', async () => {
    await setupTestDb();

    expect(mockExecuteRawUnsafe).toHaveBeenCalledWith(
      'TRUNCATE TABLE "books" RESTART IDENTITY CASCADE',
    );
  });

  it('calls initDb before executing the TRUNCATE statement', async () => {
    const callOrder: string[] = [];
    mockInitDb.mockImplementation(async () => {
      callOrder.push('initDb');
    });
    mockExecuteRawUnsafe.mockImplementation(async () => {
      callOrder.push('truncate');
    });

    await setupTestDb();

    expect(callOrder).toEqual(['initDb', 'truncate']);
  });

  it('resolves without throwing when all operations succeed', async () => {
    await expect(setupTestDb()).resolves.toBeUndefined();
  });
});

// ===========================================================================
// resetTestDb
// ===========================================================================

describe('resetTestDb', () => {
  it('executes the TRUNCATE TABLE … RESTART IDENTITY CASCADE statement', async () => {
    await resetTestDb();

    expect(mockExecuteRawUnsafe).toHaveBeenCalledTimes(1);
    expect(mockExecuteRawUnsafe).toHaveBeenCalledWith(
      'TRUNCATE TABLE "books" RESTART IDENTITY CASCADE',
    );
  });

  it('resolves without throwing when the statement succeeds', async () => {
    await expect(resetTestDb()).resolves.toBeUndefined();
  });

  it('can be called multiple times (idempotent per call)', async () => {
    await resetTestDb();
    await resetTestDb();
    await resetTestDb();

    expect(mockExecuteRawUnsafe).toHaveBeenCalledTimes(3);
  });

  it('propagates errors thrown by prisma.$executeRawUnsafe', async () => {
    const dbError = new Error('TRUNCATE failed');
    mockExecuteRawUnsafe.mockRejectedValue(dbError);

    await expect(resetTestDb()).rejects.toThrow('TRUNCATE failed');
  });
});

// ===========================================================================
// teardownTestDb
// ===========================================================================

describe('teardownTestDb', () => {
  it('closes the DB connection via closeDb', async () => {
    await teardownTestDb();

    expect(mockCloseDb).toHaveBeenCalledTimes(1);
  });

  it('resolves without throwing when closeDb succeeds', async () => {
    await expect(teardownTestDb()).resolves.toBeUndefined();
  });

  it('propagates errors thrown by closeDb', async () => {
    const closeError = new Error('close failed');
    mockCloseDb.mockRejectedValue(closeError);

    await expect(teardownTestDb()).rejects.toThrow('close failed');
  });
});

// ===========================================================================
// createTestClient
// ===========================================================================

describe('createTestClient', () => {
  it('calls createApp() to build the Express application', () => {
    createTestClient();

    expect(mockCreateApp).toHaveBeenCalledTimes(1);
  });

  it('returns an object with an `app` property equal to the Express instance', () => {
    const fakeApp = makeExpressStub();
    mockCreateApp.mockReturnValue(fakeApp);

    const { app } = createTestClient();

    expect(app).toBe(fakeApp);
  });

  it('returns an object with a `client` supertest agent (truthy)', () => {
    const { client } = createTestClient();

    // supertest agents expose HTTP-verb methods
    expect(typeof client.get).toBe('function');
    expect(typeof client.post).toBe('function');
    expect(typeof client.put).toBe('function');
    expect(typeof client.delete).toBe('function');
  });

  it('wraps the Express app returned by createApp() in the supertest agent', () => {
    // We cannot directly inspect what supertest stored, but we can verify that
    // the returned client is bound to our app by checking it is not undefined
    // and that createApp was called exactly once.
    const { client } = createTestClient();

    expect(client).toBeDefined();
    expect(mockCreateApp).toHaveBeenCalledTimes(1);
  });

  it('creates a fresh app and client on each invocation (no singleton)', () => {
    const app1 = makeExpressStub();
    const app2 = makeExpressStub();
    mockCreateApp
      .mockReturnValueOnce(app1)
      .mockReturnValueOnce(app2);

    const result1 = createTestClient();
    const result2 = createTestClient();

    expect(result1.app).toBe(app1);
    expect(result2.app).toBe(app2);
    expect(result1.app).not.toBe(result2.app);
  });

  it('does not interact with the database', () => {
    createTestClient();

    expect(mockInitDb).not.toHaveBeenCalled();
    expect(mockCloseDb).not.toHaveBeenCalled();
    expect(mockExecuteRawUnsafe).not.toHaveBeenCalled();
  });
});

// ===========================================================================
// useTestApp — lifecycle hook wiring
// ===========================================================================

describe('useTestApp', () => {
  /**
   * useTestApp() calls jest.beforeAll / jest.beforeEach / jest.afterAll
   * internally. To exercise those hooks in isolation we capture the callbacks
   * that Jest would schedule, then invoke them manually.
   */

  let capturedBeforeAll: (() => Promise<void>) | undefined;
  let capturedBeforeEach: (() => Promise<void>) | undefined;
  let capturedAfterAll: (() => Promise<void>) | undefined;

  // Spy on the global Jest lifecycle functions so we can capture their
  // callbacks without actually scheduling real hooks inside this describe block.
  const originalBeforeAll = global.beforeAll;
  const originalBeforeEach = global.beforeEach;
  const originalAfterAll = global.afterAll;

  beforeEach(() => {
    capturedBeforeAll = undefined;
    capturedBeforeEach = undefined;
    capturedAfterAll = undefined;

    (global as unknown as Record<string, unknown>).beforeAll = jest
      .fn()
      .mockImplementation((fn: () => Promise<void>) => {
        capturedBeforeAll = fn;
      });

    (global as unknown as Record<string, unknown>).beforeEach = jest
      .fn()
      .mockImplementation((fn: () => Promise<void>) => {
        capturedBeforeEach = fn;
      });

    (global as unknown as Record<string, unknown>).afterAll = jest
      .fn()
      .mockImplementation((fn: () => Promise<void>) => {
        capturedAfterAll = fn;
      });
  });

  afterEach(() => {
    // Restore originals
    (global as unknown as Record<string, unknown>).beforeAll =
      originalBeforeAll;
    (global as unknown as Record<string, unknown>).beforeEach =
      originalBeforeEach;
    (global as unknown as Record<string, unknown>).afterAll = originalAfterAll;
  });

  // -------------------------------------------------------------------------
  // Hook registration
  // -------------------------------------------------------------------------

  it('registers a beforeAll hook', () => {
    useTestApp();

    expect(global.beforeAll).toHaveBeenCalledTimes(1);
  });

  it('registers a beforeEach hook', () => {
    useTestApp();

    expect(global.beforeEach).toHaveBeenCalledTimes(1);
  });

  it('registers an afterAll hook', () => {
    useTestApp();

    expect(global.afterAll).toHaveBeenCalledTimes(1);
  });

  // -------------------------------------------------------------------------
  // beforeAll callback — equivalent of db_session setup + client fixture setup
  // -------------------------------------------------------------------------

  it('beforeAll callback calls initDb (equivalent: create all tables)', async () => {
    useTestApp();
    await capturedBeforeAll!();

    expect(mockInitDb).toHaveBeenCalledTimes(1);
  });

  it('beforeAll callback truncates the table (initial clean state)', async () => {
    useTestApp();
    await capturedBeforeAll!();

    expect(mockExecuteRawUnsafe).toHaveBeenCalledWith(
      'TRUNCATE TABLE "books" RESTART IDENTITY CASCADE',
    );
  });

  it('beforeAll callback calls createApp() to build the Express app', async () => {
    useTestApp();
    await capturedBeforeAll!();

    expect(mockCreateApp).toHaveBeenCalledTimes(1);
  });

  // -------------------------------------------------------------------------
  // beforeEach callback — equivalent of per-test isolation (reset data)
  // -------------------------------------------------------------------------

  it('beforeEach callback truncates the table to provide per-test isolation', async () => {
    useTestApp();

    // Run beforeAll first to initialise the context
    await capturedBeforeAll!();
    mockExecuteRawUnsafe.mockClear();

    // Now run the beforeEach hook
    await capturedBeforeEach!();

    expect(mockExecuteRawUnsafe).toHaveBeenCalledTimes(1);
    expect(mockExecuteRawUnsafe).toHaveBeenCalledWith(
      'TRUNCATE TABLE "books" RESTART IDENTITY CASCADE',
    );
  });

  it('beforeEach does NOT re-initialise the DB connection', async () => {
    useTestApp();
    await capturedBeforeAll!();
    mockInitDb.mockClear();

    await capturedBeforeEach!();

    expect(mockInitDb).not.toHaveBeenCalled();
  });

  it('tables are cleared before each individual test (multiple beforeEach calls)', async () => {
    useTestApp();
    await capturedBeforeAll!();
    mockExecuteRawUnsafe.mockClear();

    await capturedBeforeEach!();
    await capturedBeforeEach!();
    await capturedBeforeEach!();

    expect(mockExecuteRawUnsafe).toHaveBeenCalledTimes(3);
  });

  // -------------------------------------------------------------------------
  // afterAll callback — equivalent of db_session teardown + clearing overrides
  // -------------------------------------------------------------------------

  it('afterAll callback closes the DB connection via closeDb', async () => {
    useTestApp();
    await capturedBeforeAll!();

    await capturedAfterAll!();

    expect(mockCloseDb).toHaveBeenCalledTimes(1);
  });

  it('afterAll callback does NOT call createApp again', async () => {
    useTestApp();
    await capturedBeforeAll!();
    mockCreateApp.mockClear();

    await capturedAfterAll!();

    expect(mockCreateApp).not.toHaveBeenCalled();
  });

  // -------------------------------------------------------------------------
  // Returned proxy — lazy guard
  // -------------------------------------------------------------------------

  it('accessing .app before beforeAll runs throws an informative error', () => {
    const ctx = useTestApp();

    expect(() => ctx.app).toThrow('Test app accessed before beforeAll() ran');
  });

  it('accessing .client before beforeAll runs throws an informative error', () => {
    const ctx = useTestApp();

    expect(() => ctx.client).toThrow(
      'Test client accessed before beforeAll() ran',
    );
  });

  it('accessing .app after beforeAll ran returns an Express instance', async () => {
    const ctx = useTestApp();
    await capturedBeforeAll!();

    expect(ctx.app).toBeDefined();
  });

  it('accessing .client after beforeAll ran returns a supertest agent', async () => {
    const ctx = useTestApp();
    await capturedBeforeAll!();

    expect(ctx.client).toBeDefined();
    expect(typeof ctx.client.get).toBe('function');
  });

  it('.app is the same reference on repeated accesses (no re-creation)', async () => {
    const ctx = useTestApp();
    await capturedBeforeAll!();

    const first = ctx.app;
    const second = ctx.app;

    expect(first).toBe(second);
  });

  it('.client is the same reference on repeated accesses', async () => {
    const ctx = useTestApp();
    await capturedBeforeAll!();

    const first = ctx.client;
    const second = ctx.client;

    expect(first).toBe(second);
  });

  // -------------------------------------------------------------------------
  // Invariant: no dependency-override leaks (supertest uses the app directly)
  // -------------------------------------------------------------------------

  it('does not mutate any global state other than registering Jest hooks', () => {
    // We simply verify no DB calls happen purely from invoking useTestApp()
    useTestApp();

    expect(mockInitDb).not.toHaveBeenCalled();
    expect(mockCloseDb).not.toHaveBeenCalled();
    expect(mockExecuteRawUnsafe).not.toHaveBeenCalled();
    expect(mockCreateApp).not.toHaveBeenCalled();
  });

  // -------------------------------------------------------------------------
  // Full lifecycle integration
  // -------------------------------------------------------------------------

  it('full lifecycle: beforeAll -> beforeEach -> afterAll calls correct helpers in order', async () => {
    const order: string[] = [];

    mockInitDb.mockImplementation(async () => { order.push('initDb'); });
    mockExecuteRawUnsafe.mockImplementation(async () => { order.push('truncate'); });
    mockCloseDb.mockImplementation(async () => {