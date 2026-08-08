```typescript
// src/app/database.test.ts
import { PrismaClient } from '@prisma/client';

// ---------------------------------------------------------------------------
// Mock @prisma/client BEFORE importing the module under test so that the
// PrismaClient constructor receives our spy methods.
// ---------------------------------------------------------------------------
const mockConnect = jest.fn();
const mockDisconnect = jest.fn();
const mockQueryRaw = jest.fn();

jest.mock('@prisma/client', () => {
  return {
    PrismaClient: jest.fn().mockImplementation(() => ({
      $connect: mockConnect,
      $disconnect: mockDisconnect,
      $queryRaw: mockQueryRaw,
    })),
  };
});

// Mock config so DATABASE_URL and NODE_ENV are controlled in tests.
jest.mock('../config', () => ({
  config: {
    DATABASE_URL: 'postgresql://test:test@localhost:5432/testdb',
    NODE_ENV: 'test',
  },
}));

// ---------------------------------------------------------------------------
// Import after mocks are in place.
// ---------------------------------------------------------------------------
import { initDb, getDb, closeDb, prisma } from './database';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

describe('database module', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  // -------------------------------------------------------------------------
  // Module-level: PrismaClient instantiation
  // -------------------------------------------------------------------------
  describe('PrismaClient singleton (prisma)', () => {
    it('creates exactly one PrismaClient instance', () => {
      expect(PrismaClient).toHaveBeenCalledTimes(1);
    });

    it('passes the DATABASE_URL from config as the datasource url', () => {
      expect(PrismaClient).toHaveBeenCalledWith(
        expect.objectContaining({
          datasources: {
            db: { url: 'postgresql://test:test@localhost:5432/testdb' },
          },
        }),
      );
    });

    it('includes query/info/warn/error log levels outside production', () => {
      expect(PrismaClient).toHaveBeenCalledWith(
        expect.objectContaining({
          log: expect.arrayContaining(['query', 'info', 'warn', 'error']),
        }),
      );
    });

    it('exports the same prisma instance on repeated imports (singleton)', async () => {
      // Re-require within the same Jest module registry — both imports resolve
      // to the same cached module, so the same object reference is returned.
      const { prisma: prisma2 } = await import('./database');
      expect(prisma2).toBe(prisma);
    });
  });

  // -------------------------------------------------------------------------
  // initDb
  // -------------------------------------------------------------------------
  describe('initDb()', () => {
    describe('scenario: called when tables do not yet exist / normal path', () => {
      it('resolves to undefined (void) on success', async () => {
        mockConnect.mockResolvedValueOnce(undefined);
        const result = await initDb();
        expect(result).toBeUndefined();
      });

      it('calls prisma.$connect() exactly once', async () => {
        mockConnect.mockResolvedValueOnce(undefined);
        await initDb();
        expect(mockConnect).toHaveBeenCalledTimes(1);
      });

      it('does not call $disconnect during successful init', async () => {
        mockConnect.mockResolvedValueOnce(undefined);
        await initDb();
        expect(mockDisconnect).not.toHaveBeenCalled();
      });
    });

    describe('scenario: called when tables already exist', () => {
      it('still resolves without error (idempotent connectivity check)', async () => {
        mockConnect.mockResolvedValueOnce(undefined);
        await expect(initDb()).resolves.toBeUndefined();
      });
    });

    describe('error case: database/connection error during $connect', () => {
      it('propagates the error thrown by $connect', async () => {
        const connectionError = new Error('Connection refused');
        mockConnect.mockRejectedValueOnce(connectionError);
        await expect(initDb()).rejects.toThrow('Connection refused');
      });

      it('propagates any error type (e.g. timeout)', async () => {
        const timeoutError = new Error('ETIMEDOUT');
        mockConnect.mockRejectedValueOnce(timeoutError);
        await expect(initDb()).rejects.toThrow('ETIMEDOUT');
      });
    });
  });

  // -------------------------------------------------------------------------
  // getDb
  // -------------------------------------------------------------------------
  describe('getDb()', () => {
    describe('scenario: consumer retrieves the shared client successfully', () => {
      it('returns the shared prisma singleton', () => {
        const db = getDb();
        expect(db).toBe(prisma);
      });

      it('returns a PrismaClient instance', () => {
        const db = getDb();
        // The mock constructor returns an object with $connect/$disconnect
        expect(db).toHaveProperty('$connect');
        expect(db).toHaveProperty('$disconnect');
      });

      it('returns the same instance on repeated calls (no new sessions created)', () => {
        const db1 = getDb();
        const db2 = getDb();
        expect(db1).toBe(db2);
      });
    });

    describe('invariants', () => {
      it('does not call $connect implicitly', () => {
        getDb();
        expect(mockConnect).not.toHaveBeenCalled();
      });

      it('does not call $disconnect implicitly', () => {
        getDb();
        expect(mockDisconnect).not.toHaveBeenCalled();
      });
    });
  });

  // -------------------------------------------------------------------------
  // closeDb
  // -------------------------------------------------------------------------
  describe('closeDb()', () => {
    it('calls prisma.$disconnect() once', async () => {
      mockDisconnect.mockResolvedValueOnce(undefined);
      await closeDb();
      expect(mockDisconnect).toHaveBeenCalledTimes(1);
    });

    it('resolves to undefined on success', async () => {
      mockDisconnect.mockResolvedValueOnce(undefined);
      const result = await closeDb();
      expect(result).toBeUndefined();
    });

    it('propagates errors from $disconnect', async () => {
      const disconnectError = new Error('Disconnect failed');
      mockDisconnect.mockRejectedValueOnce(disconnectError);
      await expect(closeDb()).rejects.toThrow('Disconnect failed');
    });
  });

  // -------------------------------------------------------------------------
  // Behavioral specs mapped to migrated semantics
  // -------------------------------------------------------------------------

  // Original spec: get_db — yields a fresh session per invocation
  // Migrated: getDb() returns the shared PrismaClient; the test validates that
  // it is always available and is the same singleton (Prisma pools internally).
  describe('get_db spec — single shared client / no per-request session', () => {
    it('returns an active client with expected interface', () => {
      const db = getDb();
      expect(typeof db.$connect).toBe('function');
      expect(typeof db.$disconnect).toBe('function');
    });

    it('does not create a new PrismaClient on each call', () => {
      const callsBefore = (PrismaClient as jest.Mock).mock.calls.length;
      getDb();
      getDb();
      getDb();
      expect((PrismaClient as jest.Mock).mock.calls.length).toBe(callsBefore);
    });
  });

  // Original spec: get_sync_db — independent sessions, always closed
  // Migrated: Prisma provides a single async-only client; the sync session
  // factory is superseded. We verify the singleton is stable.
  describe('get_sync_db spec — client stability (sync session factory replaced by singleton)', () => {
    it('always returns the same client reference (no independent sessions needed)', () => {
      const refs = Array.from({ length: 5 }, () => getDb());
      refs.forEach((ref) => expect(ref).toBe(refs[0]));
    });
  });

  // -------------------------------------------------------------------------
  // Lifecycle sequence
  // -------------------------------------------------------------------------
  describe('lifecycle sequence: initDb → getDb → closeDb', () => {
    it('executes the full lifecycle without errors', async () => {
      mockConnect.mockResolvedValueOnce(undefined);
      mockDisconnect.mockResolvedValueOnce(undefined);

      await initDb();
      const db = getDb();
      expect(db).toBe(prisma);
      await closeDb();

      expect(mockConnect).toHaveBeenCalledTimes(1);
      expect(mockDisconnect).toHaveBeenCalledTimes(1);
    });
  });
});

// ---------------------------------------------------------------------------
// Separate describe block for NODE_ENV=production log-level behavior.
// We need a fresh module import with a different config mock, so we use
// jest.isolateModules.
// ---------------------------------------------------------------------------
describe('PrismaClient log levels in production mode', () => {
  it('uses only warn/error log levels when NODE_ENV is production', async () => {
    let capturedArgs: unknown;

    await jest.isolateModulesAsync(async () => {
      jest.doMock('../config', () => ({
        config: {
          DATABASE_URL: 'postgresql://prod:prod@localhost:5432/proddb',
          NODE_ENV: 'production',
        },
      }));

      const MockedPrismaClient = jest.fn().mockImplementation(() => ({
        $connect: jest.fn(),
        $disconnect: jest.fn(),
      }));

      jest.doMock('@prisma/client', () => ({
        PrismaClient: MockedPrismaClient,
      }));

      await import('./database');
      capturedArgs = MockedPrismaClient.mock.calls[0]?.[0];
    });

    const logArg = (capturedArgs as { log: string[] })?.log ?? [];
    expect(logArg).toContain('warn');
    expect(logArg).toContain('error');
    expect(logArg).not.toContain('query');
    expect(logArg).not.toContain('info');
  });
});
```