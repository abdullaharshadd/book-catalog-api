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

describe('TestRootEndpoint', () => {
  test('test_read_root — root endpoint returns welcome message', async () => {
    const response = await request(app).get('/');
    expect(response.status).toBe(200);

    const data = response.body;
    expect(data.message).toBe('Welcome to Book Catalog API');
    expect(data.version).toBe('1.0.0');
    expect(data).toHaveProperty('docs_url');
  });
});

describe('TestHealthEndpoint', () => {
  test('test_health_check — health check endpoint', async () => {
    const response = await request(app).get('/health');
    expect(response.status).toBe(200);

    const data = response.body;
    expect(data.status).toBe('healthy');
    expect(data.service).toBe('book-catalog-api');
  });
});

describe('TestBooksAPI', () => {
  test('test_create_book — creating a new book', async () => {
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

  test('test_create_book_validation_error — creating book with validation errors', async () => {
    // Missing required fields (author and published_year)
    const missingFields = await request(app)
      .post('/books/')
      .send({ title: 'Test Book' });
    expect(missingFields.status).toBe(422);

    // Invalid published year (too early)
    const invalidYear = await request(app).post('/books/').send({
      title: 'Test Book',
      author: 'Test Author',
      published_year: 999,
    });
    expect(invalidYear.status).toBe(422);
  });

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

  test('test_get_books_with_pagination — getting books with pagination', async () => {
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
    const response = await request(app).get('/books/').query({ skip: 2, limit: 2 });
    expect(response.status).toBe(200);
    expect(Array.isArray(response.body)).toBe(true);
  });

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

  test('test_get_book_not_found — getting a book that does not exist', async () => {
    const response = await request(app).get('/books/999');
    expect(response.status).toBe(404);
    expect(String(response.body.detail).toLowerCase()).toContain('not found');
  });

  test('test_update_book — updating an existing book', async () => {
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
      title: 'Updated Title',
      published_year: 2024,
      // Not updating author or summary
    };

    const response = await request(app)
      .put(`/books/${createdBook.id}`)
      .send(updateData);
    expect(response.status).toBe(200);

    const updatedBook = response.body;
    expect(updatedBook.title).toBe('Updated Title');
    expect(updatedBook.author).toBe('Original Author'); // Unchanged
    expect(updatedBook.published_year).toBe(2024);
    expect(updatedBook.summary).toBe('Original summary'); // Unchanged
  });

  test('test_update_book_not_found — updating a book that does not exist', async () => {
    const updateData = { title: 'New Title' };

    const response = await request(app).put('/books/999').send(updateData);
    expect(response.status).toBe(404);
    expect(String(response.body.detail).toLowerCase()).toContain('not found');
  });

  test('test_update_book_validation_error — updating book with validation errors', async () => {
    const bookData = {
      title: 'Test Book',
      author: 'Test Author',
      published_year: 2023,
    };

    const createResponse = await request(app).post('/books/').send(bookData);
    const createdBook = createResponse.body;

    const response = await request(app)
      .put(`/books/${createdBook.id}`)
      .send({ published_year: 999 }); // Invalid year
    expect(response.status).toBe(422);
  });

  test('test_delete_book — deleting a book', async () => {
    const bookData = {
      title: 'Book to Delete',
      author: 'Delete Author',
      published_year: 2023,
    };

    const createResponse = await request(app).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook = createResponse.body;

    const response = await request(app).delete(`/books/${createdBook.id}`);
    expect(response.status).toBe(204);

    const getResponse = await request(app).get(`/books/${createdBook.id}`);
    expect(getResponse.status).toBe(404);
  });

  test('test_delete_book_not_found — deleting a book that does not exist', async () => {
    const response = await request(app).delete('/books/999');
    expect(response.status).toBe(404);
    expect(String(response.body.detail).toLowerCase()).toContain('not found');
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

  test('test_full_crud_workflow — complete CRUD workflow', async () => {
    // CREATE
    const bookData = {
      title: 'CRUD Test Book Unique',
      author: 'CRUD Author Unique',
      published_year: 2023,
      summary: 'Testing CRUD operations',
    };

    const createResponse = await request(app).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook = createResponse.body;
    const bookId = createdBook.id;

    // READ (single)
    const getResponse = await request(app).get(`/books/${bookId}`);
    expect(getResponse.status).toBe(200);
    const retrievedBook = getResponse.body;

    expect(retrievedBook.id).toBe(createdBook.id);
    expect(retrievedBook.title).toBe(createdBook.title);
    expect(retrievedBook.author).toBe(createdBook.author);
    expect(retrievedBook.published_year).toBe(createdBook.published_year);
    expect(retrievedBook.summary).toBe(createdBook.summary);

    // READ (list)
    const listResponse = await request(app).get('/books/');
    expect(listResponse.status).toBe(200);
    const books = listResponse.body as Array<Record<string, unknown>>;

    // MIGRATION_NOTE: source found-book assertions were commented out; the
    // lookup is preserved for parity but not asserted.
    const foundBook = books.find((book) => book.id === createdBook.id) ?? null;
    void foundBook;

    // UPDATE
    const updateData = {
      title: 'Updated CRUD Book',
      summary: 'Updated summary',
    };
    const updateResponse = await request(app).put(`/books/${bookId}`).send(updateData);
    expect(updateResponse.status).toBe(200);

    // MIGRATION_NOTE: source update assertions were commented out; preserved.

    // DELETE
    const deleteResponse = await request(app).delete(`/books/${bookId}`);
    expect(deleteResponse.status).toBe(204);

    // Verify deletion
    const finalGetResponse = await request(app).get(`/books/${bookId}`);
    expect(finalGetResponse.status).toBe(404);
  });
});
