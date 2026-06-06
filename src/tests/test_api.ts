// src/tests/test_api.ts

import request from 'supertest';
import type { Express } from 'express';
import { PrismaClient } from '@prisma/client';
import { createApp } from '../app/main';
import { prismaTestClient, resetDatabase } from '../conftest';

/**
 * MIGRATION NOTES
 * ----------------
 * The source `tests/test_api.py` was a **pytest** integration test suite for a
 * FastAPI + SQLAlchemy "Book Catalog API". It exercised the root/health
 * endpoints and full CRUD + validation flows against an in-memory SQLite DB.
 *
 * In the idiomatic Node.js/TypeScript stack we use **Jest** + **supertest** +
 * **Prisma**. Key migration mappings:
 *
 *   - pytest `test_db` fixture (in-memory SQLite + StaticPool + dependency
 *     override) -> Prisma test client from `conftest.ts` + `resetDatabase()`
 *     in a `beforeEach` hook so every test starts with a clean table.
 *     There is no `app.dependency_overrides` equivalent; the Express app is
 *     built once with the (test) Prisma client injected via `createApp(...)`.
 *   - FastAPI `TestClient` -> `supertest(app)`.
 *   - Class-based test grouping (`TestRootEndpoint`, etc.) -> `describe(...)`.
 *   - `response.json()` -> `response.body`.
 *   - FastAPI returns `422` for Pydantic validation errors. The migrated
 *     Express stack uses Zod validation middleware that returns `422` as well
 *     (see the route layer), so the original status codes are PRESERVED here.
 *     If the validation layer is later switched to a framework that emits 400
 *     (e.g. DRF), these assertions must be updated.
 *   - FastAPI error bodies use `{ detail: string }`. The migrated error
 *     middleware returns `{ error: string, details?: unknown }`. Assertions on
 *     the not-found / duplicate messages therefore read `response.body.error`.
 *   - Several assertions in the source were commented out (length / id-set
 *     checks) due to test-isolation issues. With `resetDatabase()` per test we
 *     get true isolation, so those checks are RE-ENABLED here.
 */

let app: Express;
let prisma: PrismaClient;

beforeAll(() => {
  prisma = prismaTestClient;
  app = createApp(prisma);
});

beforeEach(async () => {
  // Function-scoped isolation: clear all rows before each test.
  await resetDatabase(prisma);
});

afterAll(async () => {
  await prisma.$disconnect();
});

describe('Root endpoint', () => {
  it('returns the welcome message', async () => {
    const response = await request(app).get('/');
    expect(response.status).toBe(200);
    expect(response.body.message).toBe('Welcome to Book Catalog API');
    expect(response.body.version).toBe('1.0.0');
    expect(response.body).toHaveProperty('docs_url');
  });
});

describe('Health endpoint', () => {
  it('reports a healthy status', async () => {
    const response = await request(app).get('/health');
    expect(response.status).toBe(200);
    expect(response.body.status).toBe('healthy');
    expect(response.body.service).toBe('book-catalog-api');
  });
});

describe('Books API', () => {
  it('creates a new book', async () => {
    const bookData = {
      title: 'Test Book',
      author: 'Test Author',
      published_year: 2023,
      summary: 'A test book summary',
    };

    const response = await request(app).post('/books/').send(bookData);
    expect(response.status).toBe(201);
    expect(response.body.title).toBe(bookData.title);
    expect(response.body.author).toBe(bookData.author);
    expect(response.body.published_year).toBe(bookData.published_year);
    expect(response.body.summary).toBe(bookData.summary);
    expect(response.body).toHaveProperty('id');
  });

  it('creates a book without a summary', async () => {
    const bookData = {
      title: 'Book Without Summary',
      author: 'Author',
      published_year: 2023,
    };

    const response = await request(app).post('/books/').send(bookData);
    expect(response.status).toBe(201);
    expect(response.body.title).toBe(bookData.title);
    expect(response.body.author).toBe(bookData.author);
    expect(response.body.published_year).toBe(bookData.published_year);
    expect(response.body.summary).toBeNull();
  });

  it('returns 422 on validation errors', async () => {
    // Missing required fields (author, published_year).
    const missingFields = await request(app)
      .post('/books/')
      .send({ title: 'Test Book' });
    expect(missingFields.status).toBe(422);

    // Invalid published year (too early).
    const invalidYear = await request(app).post('/books/').send({
      title: 'Test Book',
      author: 'Test Author',
      published_year: 999,
    });
    expect(invalidYear.status).toBe(422);
  });

  it('returns an empty list when there are no books', async () => {
    const response = await request(app).get('/books/');
    expect(response.status).toBe(200);
    expect(response.body).toEqual([]);
  });

  it('returns books when the database has data', async () => {
    const booksData = [
      { title: 'Book 1', author: 'Author 1', published_year: 2021 },
      { title: 'Book 2', author: 'Author 2', published_year: 2022 },
      { title: 'Book 3', author: 'Author 3', published_year: 2023 },
    ];

    const createdBooks: Array<{ id: number }> = [];
    for (const bookData of booksData) {
      const response = await request(app).post('/books/').send(bookData);
      expect(response.status).toBe(201);
      createdBooks.push(response.body);
    }

    const response = await request(app).get('/books/');
    expect(response.status).toBe(200);

    const books: Array<{ id: number }> = response.body;
    // Re-enabled (test isolation guaranteed by resetDatabase).
    expect(createdBooks).toHaveLength(3);

    const createdIds = new Set(createdBooks.map((b) => b.id));
    const retrievedIds = new Set(books.map((b) => b.id));
    expect(retrievedIds).toEqual(createdIds);
  });

  it('supports pagination', async () => {
    const createdIds: number[] = [];
    for (let i = 0; i < 5; i += 1) {
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
    const allBooks: unknown[] = allBooksResponse.body;
    expect(allBooks.length).toBeGreaterThanOrEqual(5);

    const response = await request(app).get('/books/?skip=2&limit=2');
    expect(response.status).toBe(200);
    const books: unknown[] = response.body;
    expect(books).toHaveLength(2);
  });

  it('gets a specific book by id', async () => {
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

  it('returns 404 when a book is not found', async () => {
    const response = await request(app).get('/books/999');
    expect(response.status).toBe(404);
    expect(String(response.body.error).toLowerCase()).toContain('not found');
  });

  it('updates an existing book', async () => {
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

  it('returns 404 when updating a missing book', async () => {
    const response = await request(app)
      .put('/books/999')
      .send({ title: 'New Title' });
    expect(response.status).toBe(404);
    expect(String(response.body.error).toLowerCase()).toContain('not found');
  });

  it('returns 422 when updating with invalid data', async () => {
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

  it('deletes a book', async () => {
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

  it('returns 404 when deleting a missing book', async () => {
    const response = await request(app).delete('/books/999');
    expect(response.status).toBe(404);
    expect(String(response.body.error).toLowerCase()).toContain('not found');
  });

  it('rejects a duplicate title + author', async () => {
    const bookData = {
      title: 'Duplicate Book',
      author: 'Duplicate Author',
      published_year: 2023,
    };

    const response1 = await request(app).post('/books/').send(bookData);
    expect(response1.status).toBe(201);

    const response2 = await request(app).post('/books/').send(bookData);
    expect(response2.status).toBe(400);
    expect(String(response2.body.error).toLowerCase()).toContain(
      'already exists',
    );
  });

  it('allows the same title with different authors', async () => {
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

    expect(response1.body.id).not.toBe(response2.body.id);
  });

  it('runs a full CRUD workflow', async () => {
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
    const books: Array<{ id: number; title: string }> = listResponse.body;

    const foundBook = books.find((b) => b.id === createdBook.id);
    // Re-enabled (test isolation guaranteed by resetDatabase).
    expect(foundBook).toBeDefined();
    expect(foundBook?.title).toBe(bookData.title);

    // UPDATE
    const updateData = {
      title: 'Updated CRUD Book',
      summary: 'Updated summary',
    };
    const updateResponse = await request(app)
      .put(`/books/${bookId}`)
      .send(updateData);
    expect(updateResponse.status).toBe(200);
    const updatedBook = updateResponse.body;
    // Re-enabled (test isolation guaranteed by resetDatabase).
    expect(updatedBook.title).toBe('Updated CRUD Book');
    expect(updatedBook.summary).toBe('Updated summary');
    expect(updatedBook.author).toBe(bookData.author); // Unchanged

    // DELETE
    const deleteResponse = await request(app).delete(`/books/${bookId}`);
    expect(deleteResponse.status).toBe(204);

    // Verify deletion
    const finalGetResponse = await request(app).get(`/books/${bookId}`);
    expect(finalGetResponse.status).toBe(404);
  });
});
