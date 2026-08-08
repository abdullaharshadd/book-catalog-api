```typescript
// src/tests/test_api.ts

/**
 * Integration tests for the Book Catalog API.
 *
 * MIGRATION_NOTE: The Python source used pytest with FastAPI's TestClient and
 * SQLAlchemy in-memory SQLite (StaticPool) for test isolation. In the
 * Express/TS + Jest + supertest world we:
 *   - build a fresh app via `createTestClient()` (from conftest.ts) which wires
 *     the app to a test Prisma client,
 *   - reset the DB between tests with `resetTestDb()` rather than dropping and
 *     recreating tables per pytest function-scoped fixture,
 *   - use supertest's fluent request API instead of TestClient.
 *
 * MIGRATION_NOTE: pytest's `test_db` / `client` fixtures are replaced by Jest
 * lifecycle hooks (`beforeAll` / `afterEach` / `afterAll`) calling the shared
 * helpers migrated into `src/conftest.ts`.
 *
 * MIGRATION_NOTE: The wire contract stays snake_case (`published_year`) exactly
 * as in the source — do NOT camelCase these JSON keys, the API response schema
 * (schemas.ts / toBookResponse) preserves them.
 *
 * MIGRATION_NOTE: FastAPI returns validation errors as HTTP 422 and
 * not-found/duplicate errors carry a `detail` field. The Express app mirrors
 * this: Zod validation failures -> 422, and error responses expose `detail`.
 * POST /books/ returns 201 (confirmed by source assertions).
 */

import request from 'supertest';
import type { Express } from 'express';
import {
  createTestClient,
  setupTestDb,
  resetTestDb,
  teardownTestDb,
} from '../conftest';

let app: Express;

beforeAll(async () => {
  await setupTestDb();
  app = await createTestClient();
});

afterEach(async () => {
  await resetTestDb();
});

afterAll(async () => {
  await teardownTestDb();
});

// ---------------------------------------------------------------------------
// TestRootEndpoint
// ---------------------------------------------------------------------------

describe('TestRootEndpoint', () => {
  test('test_read_root — root endpoint returns welcome message', async () => {
    const response = await request(app).get('/');
    expect(response.status).toBe(200);

    const data = response.body;
    expect(data.message).toBe('Welcome to Book Catalog API');
    expect(data.version).toBe('1.0.0');
    expect(data).toHaveProperty('docs_url');
  });

  test('invariant: GET / always returns 200 with required keys', async () => {
    const response = await request(app).get('/');
    expect(response.status).toBe(200);
    expect(response.body).toHaveProperty('message');
    expect(response.body).toHaveProperty('version');
    expect(response.body).toHaveProperty('docs_url');
  });
});

// ---------------------------------------------------------------------------
// TestHealthEndpoint
// ---------------------------------------------------------------------------

describe('TestHealthEndpoint', () => {
  test('test_health_check — health check endpoint', async () => {
    const response = await request(app).get('/health');
    expect(response.status).toBe(200);

    const data = response.body;
    expect(data.status).toBe('healthy');
    expect(data.service).toBe('book-catalog-api');
  });

  test('invariant: GET /health always returns 200 with status=healthy and correct service name', async () => {
    const response = await request(app).get('/health');
    expect(response.status).toBe(200);
    expect(response.body.status).toBe('healthy');
    expect(response.body.service).toBe('book-catalog-api');
  });
});

// ---------------------------------------------------------------------------
// TestBooksAPI
// ---------------------------------------------------------------------------

describe('TestBooksAPI', () => {
  // -------------------------------------------------------------------------
  // POST /books/
  // -------------------------------------------------------------------------

  test('test_create_book — creating a new book with all fields', async () => {
    const bookData = {
      title: 'Test Book',
      author: 'Test Author',
      published_year: 2023,
      summary: 'A test book summary',
    };

    const response = await request(app).post('/books/').send(bookData);
    expect(response.status).toBe(201);

    const data = response.body;
    expect(data.title).toBe(bookData.title);
    expect(data.author).toBe(bookData.author);
    expect(data.published_year).toBe(bookData.published_year);
    expect(data.summary).toBe(bookData.summary);
    expect(data).toHaveProperty('id');
    expect(typeof data.id).not.toBe('undefined');
  });

  test('test_create_book_without_summary — creating a book without summary', async () => {
    const bookData = {
      title: 'Book Without Summary',
      author: 'Author',
      published_year: 2023,
    };

    const response = await request(app).post('/books/').send(bookData);
    expect(response.status).toBe(201);

    const data = response.body;
    expect(data.title).toBe(bookData.title);
    expect(data.author).toBe(bookData.author);
    expect(data.published_year).toBe(bookData.published_year);
    expect(data.summary).toBeNull();
  });

  test('test_create_book_validation_error — missing required fields (only title provided)', async () => {
    const response = await request(app)
      .post('/books/')
      .send({ title: 'Test Book' });
    expect(response.status).toBe(422);
  });

  test('test_create_book_validation_error — missing author field', async () => {
    const response = await request(app).post('/books/').send({
      title: 'Test Book',
      published_year: 2023,
    });
    expect(response.status).toBe(422);
  });

  test('test_create_book_validation_error — missing published_year field', async () => {
    const response = await request(app).post('/books/').send({
      title: 'Test Book',
      author: 'Test Author',
    });
    expect(response.status).toBe(422);
  });

  test('test_create_book_validation_error — invalid published_year (too early, 999)', async () => {
    const response = await request(app).post('/books/').send({
      title: 'Test Book',
      author: 'Test Author',
      published_year: 999,
    });
    expect(response.status).toBe(422);
  });

  test('test_create_book_validation_error — empty body', async () => {
    const response = await request(app).post('/books/').send({});
    expect(response.status).toBe(422);
  });

  test('test_create_duplicate_book — creating books with same title and author should fail', async () => {
    const bookData = {
      title: 'Duplicate Book',
      author: 'Duplicate Author',
      published_year: 2023,
    };

    const response1 = await request(app).post('/books/').send(bookData);
    expect(response1.status).toBe(201);

    const response2 = await request(app).post('/books/').send(bookData);
    expect(response2.status).toBe(400);
    expect(String(response2.body.detail).toLowerCase()).toContain('already exists');
  });

  test('test_books_same_title_different_authors — same title different authors should succeed', async () => {
    const book1Data = {
      title: 'Same Title',
      author: 'Author One',
      published_year: 2023,
    };
    const book2Data = {
      title: 'Same Title',
      author: 'Author Two',
      published_year: 2023,
    };

    const response1 = await request(app).post('/books/').send(book1Data);
    expect(response1.status).toBe(201);

    const response2 = await request(app).post('/books/').send(book2Data);
    expect(response2.status).toBe(201);

    const book1 = response1.body;
    const book2 = response2.body;
    expect(book1.id).not.toBe(book2.id);
  });

  test('invariant: every successfully created book receives a unique id', async () => {
    const book1 = await request(app).post('/books/').send({
      title: 'Unique ID Book One',
      author: 'Author A',
      published_year: 2020,
    });
    const book2 = await request(app).post('/books/').send({
      title: 'Unique ID Book Two',
      author: 'Author B',
      published_year: 2021,
    });

    expect(book1.status).toBe(201);
    expect(book2.status).toBe(201);
    expect(book1.body).toHaveProperty('id');
    expect(book2.body).toHaveProperty('id');
    expect(book1.body.id).not.toBe(book2.body.id);
  });

  test('invariant: returned book data reflects the submitted values', async () => {
    const bookData = {
      title: 'Reflect Test',
      author: 'Reflect Author',
      published_year: 2015,
      summary: 'Reflection summary',
    };
    const response = await request(app).post('/books/').send(bookData);
    expect(response.status).toBe(201);
    expect(response.body.title).toBe(bookData.title);
    expect(response.body.author).toBe(bookData.author);
    expect(response.body.published_year).toBe(bookData.published_year);
    expect(response.body.summary).toBe(bookData.summary);
  });

  // -------------------------------------------------------------------------
  // GET /books/
  // -------------------------------------------------------------------------

  test('test_get_books_empty — getting books when database is empty', async () => {
    const response = await request(app).get('/books/');
    expect(response.status).toBe(200);
    expect(response.body).toEqual([]);
  });

  test('test_get_books_with_data — getting books when database has data', async () => {
    const booksData = [
      { title: 'Book 1', author: 'Author 1', published_year: 2021 },
      { title: 'Book 2', author: 'Author 2', published_year: 2022 },
      { title: 'Book 3', author: 'Author 3', published_year: 2023 },
    ];

    const createdBooks: Array<Record<string, unknown>> = [];
    for (const bookData of booksData) {
      const response = await request(app).post('/books/').send(bookData);
      expect(response.status).toBe(201);
      createdBooks.push(response.body);
    }

    const listResponse = await request(app).get('/books/');
    expect(listResponse.status).toBe(200);

    // MIGRATION_NOTE: The source left the length/ID-set assertions commented
    // out; we preserve that behaviour and only assert the request succeeds.
    const books = listResponse.body;
    expect(Array.isArray(books)).toBe(true);
  });

  test('test_get_books_with_pagination — getting books with skip and limit params', async () => {
    const createdIds: number[] = [];
    for (let i = 0; i < 5; i++) {
      const bookData = {
        title: `Pagination Book ${i + 1}`,
        author: `Pagination Author ${i + 1}`,
        published_year: 2020 + i,
      };
      const response = await request(app).post('/books/').send(bookData);
      expect(response.status).toBe(201);
      createdIds.push(response.body.id);
    }

    const allBooksResponse = await request(app).get('/books/');
    expect(allBooksResponse.status).toBe(200);

    // MIGRATION_NOTE: source length assertions were commented out; preserved.
    const response = await request(app)
      .get('/books/')
      .query({ skip: 2, limit: 2 });
    expect(response.status).toBe(200);
    expect(Array.isArray(response.body)).toBe(true);
  });

  test('invariant: GET /books/ always returns a JSON array', async () => {
    const response = await request(app).get('/books/');
    expect(response.status).toBe(200);
    expect(Array.isArray(response.body)).toBe(true);
  });

  test('GET /books/ with skip=0 limit=0 returns empty or valid array', async () => {
    await request(app).post('/books/').send({
      title: 'Limit Test Book',
      author: 'Limit Author',
      published_year: 2022,
    });

    const response = await request(app)
      .get('/books/')
      .query({ skip: 0, limit: 0 });
    expect(response.status).toBe(200);
    expect(Array.isArray(response.body)).toBe(true);
  });

  // -------------------------------------------------------------------------
  // GET /books/:id
  // -------------------------------------------------------------------------

  test('test_get_book_by_id — getting a specific book by ID', async () => {
    const bookData = {
      title: 'Specific Book',
      author: 'Specific Author',
      published_year: 2023,
      summary: 'A specific book',
    };

    const createResponse = await request(app).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook = createResponse.body;

    const response = await request(app).get(`/books/${createdBook.id}`);
    expect(response.status).toBe(200);
    expect(response.body).toEqual(createdBook);
  });

  test('invariant: returned object for an existing book equals the object returned at creation', async () => {
    const bookData = {
      title: 'Parity Check Book',
      author: 'Parity Author',
      published_year: 2019,
      summary: 'Parity summary',
    };
    const createResponse = await request(app).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook = createResponse.body;

    const getResponse = await request(app).get(`/books/${createdBook.id}`);
    expect(getResponse.status).toBe(200);
    expect(getResponse.body.id).toBe(createdBook.id);
    expect(getResponse.body.title).toBe(createdBook.title);
    expect(getResponse.body.author).toBe(createdBook.author);
    expect(getResponse.body.published_year).toBe(createdBook.published_year);
    expect(getResponse.body.summary).toBe(createdBook.summary);
  });

  test('test_get_book_not_found — getting a book that does not exist', async () => {
    const response = await request(app).get('/books/999');
    expect(response.status).toBe(404);
    expect(String(response.body.detail).toLowerCase()).toContain('not found');
  });

  // -------------------------------------------------------------------------
  // PUT /books/:id
  // -------------------------------------------------------------------------

  test('test_update_book — updating an existing book with partial fields', async () => {
    const bookData = {
      title: 'Original Title',
      author: 'Original Author',
      published_year: 2023,
      summary: 'Original summary',
    };

    const createResponse = await request(app).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook = createResponse.body;

    const updateData = {
      title: '