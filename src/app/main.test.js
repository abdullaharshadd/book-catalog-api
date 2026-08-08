```typescript
// src/app/main.test.ts

import request from 'supertest';
import { createApp, createBooksRouter, HttpError } from './main';

// ---------------------------------------------------------------------------
// Mock external dependencies
// ---------------------------------------------------------------------------
jest.mock('./database', () => ({
  initDb: jest.fn().mockResolvedValue(undefined),
  closeDb: jest.fn().mockResolvedValue(undefined),
  prisma: {
    book: {
      findMany: jest.fn(),
      findUnique: jest.fn(),
      create: jest.fn(),
      update: jest.fn(),
      delete: jest.fn(),
    },
  },
}));

jest.mock('../config', () => ({
  config: { port: 3000 },
}));

jest.mock('./schemas', () => {
  const actual = jest.requireActual('./schemas');
  return {
    ...actual,
    toBookResponse: jest.fn((book: Record<string, unknown>) => book),
    bookCreateSchema: {
      parse: jest.fn((data: unknown) => {
        // Basic validation simulation
        const d = data as Record<string, unknown>;
        if (!d || !d.title || !d.author || d.published_year === undefined) {
          const { ZodError, z } = jest.requireActual('zod');
          // Throw a ZodError-like object
          const schema = (jest.requireActual('./schemas') as typeof import('./schemas')).bookCreateSchema;
          return schema.parse(data);
        }
        return data;
      }),
    },
    bookUpdateSchema: {
      parse: jest.fn((data: unknown) => data),
    },
  };
});

import { prisma } from './database';
import { toBookResponse, bookCreateSchema, bookUpdateSchema } from './schemas';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const mockPrismaBook = prisma.book as jest.Mocked<typeof prisma.book>;

function makeBook(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    title: 'Test Book',
    author: 'Test Author',
    published_year: 2021,
    summary: 'A test summary',
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// Reset mocks before each test
// ---------------------------------------------------------------------------
beforeEach(() => {
  jest.clearAllMocks();
  (toBookResponse as jest.Mock).mockImplementation((book: Record<string, unknown>) => book);
});

// ---------------------------------------------------------------------------
// GET /
// ---------------------------------------------------------------------------
describe('GET /', () => {
  it('returns 200 with welcome message, version and docs_url', async () => {
    const app = createApp();
    const res = await request(app).get('/');

    expect(res.status).toBe(200);
    expect(res.body).toEqual({
      message: 'Welcome to Book Catalog API',
      version: '1.0.0',
      docs_url: '/docs',
    });
  });

  it('always returns HTTP 200 with the same static payload', async () => {
    const app = createApp();
    const res1 = await request(app).get('/');
    const res2 = await request(app).get('/');

    expect(res1.status).toBe(200);
    expect(res2.status).toBe(200);
    expect(res1.body).toEqual(res2.body);
  });
});

// ---------------------------------------------------------------------------
// GET /health
// ---------------------------------------------------------------------------
describe('GET /health', () => {
  it('returns 200 with healthy status and service name', async () => {
    const app = createApp();
    const res = await request(app).get('/health');

    expect(res.status).toBe(200);
    expect(res.body).toEqual({
      status: 'healthy',
      service: 'book-catalog-api',
    });
  });

  it('always returns the same static healthy payload', async () => {
    const app = createApp();
    const res1 = await request(app).get('/health');
    const res2 = await request(app).get('/health');

    expect(res1.status).toBe(200);
    expect(res2.status).toBe(200);
    expect(res1.body).toEqual(res2.body);
  });
});

// ---------------------------------------------------------------------------
// GET /books/
// ---------------------------------------------------------------------------
describe('GET /books/', () => {
  it('returns 200 with array of books when books exist (default pagination)', async () => {
    const books = [makeBook({ id: 1 }), makeBook({ id: 2, title: 'Book 2' })];
    mockPrismaBook.findMany.mockResolvedValue(books as never);

    const app = createApp();
    const res = await request(app).get('/books/');

    expect(res.status).toBe(200);
    expect(Array.isArray(res.body)).toBe(true);
    expect(res.body).toHaveLength(2);
    expect(mockPrismaBook.findMany).toHaveBeenCalledWith({
      skip: 0,
      take: 100,
      orderBy: { id: 'asc' },
    });
  });

  it('returns 200 with empty array when no books exist', async () => {
    mockPrismaBook.findMany.mockResolvedValue([] as never);

    const app = createApp();
    const res = await request(app).get('/books/');

    expect(res.status).toBe(200);
    expect(res.body).toEqual([]);
  });

  it('caps limit at 1000 when limit exceeds 1000', async () => {
    mockPrismaBook.findMany.mockResolvedValue([] as never);

    const app = createApp();
    const res = await request(app).get('/books/?limit=9999');

    expect(res.status).toBe(200);
    expect(mockPrismaBook.findMany).toHaveBeenCalledWith({
      skip: 0,
      take: 1000,
      orderBy: { id: 'asc' },
    });
  });

  it('passes skip offset to the query', async () => {
    const books = [makeBook({ id: 5 })];
    mockPrismaBook.findMany.mockResolvedValue(books as never);

    const app = createApp();
    const res = await request(app).get('/books/?skip=4&limit=10');

    expect(res.status).toBe(200);
    expect(mockPrismaBook.findMany).toHaveBeenCalledWith({
      skip: 4,
      take: 10,
      orderBy: { id: 'asc' },
    });
  });

  it('returns 200 with correct limit from query param', async () => {
    mockPrismaBook.findMany.mockResolvedValue([] as never);

    const app = createApp();
    const res = await request(app).get('/books/?skip=0&limit=50');

    expect(res.status).toBe(200);
    expect(mockPrismaBook.findMany).toHaveBeenCalledWith(
      expect.objectContaining({ take: 50 }),
    );
  });

  it('defaults limit to 100 when limit is not provided', async () => {
    mockPrismaBook.findMany.mockResolvedValue([] as never);

    const app = createApp();
    await request(app).get('/books/');

    expect(mockPrismaBook.findMany).toHaveBeenCalledWith(
      expect.objectContaining({ take: 100 }),
    );
  });

  it('defaults limit to 100 when limit is invalid (NaN)', async () => {
    mockPrismaBook.findMany.mockResolvedValue([] as never);

    const app = createApp();
    await request(app).get('/books/?limit=abc');

    expect(mockPrismaBook.findMany).toHaveBeenCalledWith(
      expect.objectContaining({ take: 100 }),
    );
  });

  it('returns 500 with detail when database error occurs', async () => {
    mockPrismaBook.findMany.mockRejectedValue(new Error('DB connection failed') as never);

    const app = createApp();
    const res = await request(app).get('/books/');

    expect(res.status).toBe(500);
    expect(res.body).toEqual({
      detail: 'Internal server error while retrieving books',
    });
  });

  it('response is always an array', async () => {
    mockPrismaBook.findMany.mockResolvedValue([makeBook()] as never);

    const app = createApp();
    const res = await request(app).get('/books/');

    expect(Array.isArray(res.body)).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// GET /books/:book_id
// ---------------------------------------------------------------------------
describe('GET /books/:book_id', () => {
  it('returns 200 with the book when it exists', async () => {
    const book = makeBook({ id: 1 });
    mockPrismaBook.findUnique.mockResolvedValue(book as never);

    const app = createApp();
    const res = await request(app).get('/books/1');

    expect(res.status).toBe(200);
    expect(res.body).toMatchObject({ id: 1, title: 'Test Book' });
    expect(mockPrismaBook.findUnique).toHaveBeenCalledWith({ where: { id: 1 } });
  });

  it('returns 404 with detail when book does not exist', async () => {
    mockPrismaBook.findUnique.mockResolvedValue(null as never);

    const app = createApp();
    const res = await request(app).get('/books/999');

    expect(res.status).toBe(404);
    expect(res.body).toEqual({ detail: 'Book with ID 999 not found' });
  });

  it('returns 404 when book_id is non-numeric', async () => {
    const app = createApp();
    const res = await request(app).get('/books/abc');

    expect(res.status).toBe(404);
    // NaN-parsed id → null branch → 404 (detail may vary based on NaN display)
    expect(res.body).toHaveProperty('detail');
    expect(mockPrismaBook.findUnique).not.toHaveBeenCalled();
  });

  it('returned book id matches the requested book_id', async () => {
    const book = makeBook({ id: 42 });
    mockPrismaBook.findUnique.mockResolvedValue(book as never);

    const app = createApp();
    const res = await request(app).get('/books/42');

    expect(res.status).toBe(200);
    expect(res.body.id).toBe(42);
  });

  it('returns 500 with detail when a database error occurs', async () => {
    mockPrismaBook.findUnique.mockRejectedValue(new Error('DB error') as never);

    const app = createApp();
    const res = await request(app).get('/books/1');

    expect(res.status).toBe(500);
    expect(res.body).toEqual({ detail: 'Internal server error while retrieving book' });
  });
});

// ---------------------------------------------------------------------------
// POST /books/
// ---------------------------------------------------------------------------
describe('POST /books/', () => {
  const validPayload = {
    title: 'New Book',
    author: 'Author Name',
    published_year: 2023,
    summary: 'Some summary',
  };

  beforeEach(() => {
    (bookCreateSchema.parse as jest.Mock).mockImplementation((data: unknown) => data);
  });

  it('returns 201 with created book on valid data', async () => {
    const createdBook = makeBook({ id: 10, ...validPayload });
    mockPrismaBook.create.mockResolvedValue(createdBook as never);

    const app = createApp();
    const res = await request(app).post('/books/').send(validPayload);

    expect(res.status).toBe(201);
    expect(res.body).toMatchObject({ id: 10, title: 'New Book' });
    expect(mockPrismaBook.create).toHaveBeenCalledWith({ data: validPayload });
  });

  it('returns 201 and the created book has a generated ID', async () => {
    const createdBook = makeBook({ id: 99, ...validPayload });
    mockPrismaBook.create.mockResolvedValue(createdBook as never);

    const app = createApp();
    const res = await request(app).post('/books/').send(validPayload);

    expect(res.status).toBe(201);
    expect(res.body).toHaveProperty('id');
    expect(res.body.id).toBe(99);
  });

  it('returns 400 when a unique constraint violation occurs (duplicate title+author)', async () => {
    const prismaUniqueError = { code: 'P2002', message: 'Unique constraint failed' };
    mockPrismaBook.create.mockRejectedValue(prismaUniqueError as never);

    const app = createApp();
    const res = await request(app).post('/books/').send(validPayload);

    expect(res.status).toBe(400);
    expect(res.body).toEqual({
      detail: 'Book with this title and author already exists',
    });
  });

  it('returns 422 when required fields are missing (ZodError)', async () => {
    // Make bookCreateSchema.parse throw a ZodError for missing fields
    const { ZodError } = jest.requireActual('zod') as typeof import('zod');
    (bookCreateSchema.parse as jest.Mock).mockImplementation(() => {
      // Build a minimal ZodError
      const real = jest.requireActual('./schemas') as typeof import('./schemas');
      return real.bookCreateSchema.parse({});
    });

    const app = createApp();
    const res = await request(app)
      .post('/books/')
      .send({ title: 'Missing fields' });

    expect(res.status).toBe(422);
    expect(res.body).toHaveProperty('detail');
  });

  it('returns 422 for malformed JSON body', async () => {
    const app = createApp();
    const res = await request(app)
      .post('/books/')
      .set('Content-Type', 'application/json')
      .send('{ invalid json ');

    expect(res.status).toBe(422);
    expect(res.body).toEqual({ detail: 'Invalid JSON body' });
  });

  it('returns 500 when an unexpected database error occurs', async () => {
    mockPrismaBook.create.mockRejectedValue(new Error('Unexpected DB error') as never);

    const app = createApp();
    const res = await request(app).post('/books/').send(validPayload);

    expect(res.status).toBe(500);
    expect(res.body).toEqual({ detail: 'Internal server error while creating book' });
  });
});

// ---------------------------------------------------------------------------
// PUT /books/:book_id
// ---------------------------------------------------------------------------
describe('PUT /books/:book_id', () => {
  const updatePayload = { title: 'Updated Title' };

  beforeEach(() => {
    (bookUpdateSchema.parse as jest.Mock).mockImplementation((data: unknown) => data);
  });

  it('returns 200 with updated book on valid partial data', async () => {
    const existingBook = makeBook({ id: 1 });
    const updatedBook = makeBook({ id: 1, title: 'Updated Title' });

    mockPrismaBook.findUnique.mockResolvedValue(existingBook as never);
    mockPrismaBook.update.mockResolvedValue(updatedBook as never);

    const app = createApp();
    const res = await request(app).put('/books/1').send(updatePayload);

    expect(res.status).toBe(200);
    expect(res.body).toMatchObject({ id: 1, title: 'Updated Title' });
  });

  it('only updates provided fields (partial update)', async ()