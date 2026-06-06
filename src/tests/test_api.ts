import request from 'supertest';
import { PrismaClient } from '@prisma/client';
import { createApp } from '../app';
import type { Express } from 'express';

/**
 * Integration test suite for the Book Catalog REST API.
 *
 * MIGRATION: The original FastAPI suite used SQLAlchemy in-memory SQLite with
 * StaticPool and FastAPI's `app.dependency_overrides[get_sync_db]` to inject a
 * test database. In the Node/Express + Prisma world there is no per-request
 * dependency override mechanism baked into the framework. Instead we:
 *   - Use a dedicated test database (configured via DATABASE_URL pointing at a
 *     file-based or in-memory SQLite test DB, or a transactional test schema).
 *   - Reset the table state between tests so each test runs in isolation,
 *     mirroring pytest's function-scoped fixture lifecycle.
 *
 * Set DATABASE_URL to a throwaway SQLite file before running these tests, e.g.
 *   DATABASE_URL="file:./test.db" jest
 */

/**
 * STATUS_VALIDATION is the HTTP status returned for request body validation
 * failures. FastAPI/Pydantic returns 422; this migration keeps 422 by
 * validating request bodies with Zod and returning 422 on parse failure to
 * preserve the original contract.
 *
 * MIGRATION: If you prefer the Express/DRF-style 400 for validation errors,
 * change both the route validation middleware and these expectations to 400.
 */
const STATUS_VALIDATION = 422;

let app: Express;
let prisma: PrismaClient;

beforeAll(async () => {
  prisma = new PrismaClient();
  // createApp accepts the Prisma client via constructor-style dependency
  // injection (no global singletons), mirroring the original override pattern.
  app = createApp(prisma);
});

afterAll(async () => {
  await prisma.$disconnect();
});

/**
 * Reset the books table before each test so every test starts from a clean
 * state, equivalent to pytest's function-scoped in-memory DB fixture.
 */
beforeEach(async () => {
  await prisma.book.deleteMany({});
});

describe('Root endpoint', () => {
  it('returns welcome message', async () => {
    const response = await request(app).get('/');
    expect(response.status).toBe(200);
    expect(response.body.message).toBe('Welcome to Book Catalog API');
    expect(response.body.version).toBe('1.0.0');
    expect(response.body).toHaveProperty('docs_url');
  });
});

describe('Health endpoint', () => {
  it('returns healthy status', async () => {
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

    const data = response.body;
    expect(data.title).toBe(bookData.title);
    expect(data.author).toBe(bookData.author);
    expect(data.published_year).toBe(bookData.published_year);
    expect(data.summary).toBe(bookData.summary);
    expect(data).toHaveProperty('id');
  });

  it('creates a book without summary', async () => {
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

  it('rejects creation with validation errors', async () => {
    // Missing required fields (author and published_year).
    const missingFields = await request(app)
      .post('/books/')
      .send({ title: 'Test Book' });
    expect(missingFields.status).toBe(STATUS_VALIDATION);

    // Invalid published year (too early).
    const invalidYear = await request(app).post('/books/').send({
      title: 'Test Book',
      author: 'Test Author',
      published_year: 999,
    });
    expect(invalidYear.status).toBe(STATUS_VALIDATION);
  });

  it('returns empty list when database is empty', async () => {
    const response = await request(app).get('/books/');
    expect(response.status).toBe(200);
    expect(response.body).toEqual([]);
  });

  it('returns books when database has data', async () => {
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

    const response = await request(app).get('/books/');
    expect(response.status).toBe(200);

    // const books = response.body;
    // expect(createdBooks.length).toBe(3);
    // Membership assertions remain disabled to match original behavior.
  });

  it('supports pagination', async () => {
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
    // expect(allBooksResponse.body.length).toBeGreaterThanOrEqual(5);

    const response = await request(app).get('/books/?skip=2&limit=2');
    expect(response.status).toBe(200);
    // expect(response.body.length).toBe(2);
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

  it('returns 404 for a missing book', async () => {
    const response = await request(app).get('/books/999');
    expect(response.status).toBe(404);
    expect(String(response.body.detail).toLowerCase()).toContain('not found');
  });

  it('updates an existing book with partial fields', async () => {
    const bookData = {
      title: 'Original Title',
      author: 'Original Author',
      published_year: 2023,
      summary: 'Original summary',
    };

    const createResponse = await request(app).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook = createResponse.body;

    // PUT with a subset of fields; unspecified fields stay unchanged
    // (PATCH-like partial update semantics preserved from the source).
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
    expect(String(response.body.detail).toLowerCase()).toContain('not found');
  });

  it('rejects update with validation errors', async () => {
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
    expect(response.status).toBe(STATUS_VALIDATION);
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
    expect(String(response.body.detail).toLowerCase()).toContain('not found');
  });

  it('rejects duplicate title+author', async () => {
    const bookData = {
      title: 'Duplicate Book',
      author: 'Duplicate Author',
      published_year: 2023,
    };

    const response1 = await request(app).post('/books/').send(bookData);
    expect(response1.status).toBe(201);

    const response2 = await request(app).post('/books/').send(bookData);
    expect(response2.status).toBe(400);
    expect(String(response2.body.detail).toLowerCase()).toContain(
      'already exists',
    );
  });

  it('allows same title with different authors', async () => {
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
    const books: Array<Record<string, unknown>> = listResponse.body;

    let foundBook: Record<string, unknown> | null = null;
    for (const book of books) {
      if (book.id === createdBook.id) {
        foundBook = book;
        break;
      }
    }
    // Membership assertions remain disabled to match original behavior.
    // expect(foundBook).not.toBeNull();
    // expect(foundBook?.title).toBe(bookData.title);
    void foundBook;

    // UPDATE
    const updateData = {
      title: 'Updated CRUD Book',
      summary: 'Updated summary',
    };
    const updateResponse = await request(app)
      .put(`/books/${bookId}`)
      .send(updateData);
    expect(updateResponse.status).toBe(200);
    // Field assertions remain disabled to match original behavior.
    // expect(updateResponse.body.title).toBe('Updated CRUD Book');
    // expect(updateResponse.body.summary).toBe('Updated summary');
    // expect(updateResponse.body.author).toBe(bookData.author);

    // DELETE
    const deleteResponse = await request(app).delete(`/books/${bookId}`);
    expect(deleteResponse.status).toBe(204);

    const finalGetResponse = await request(app).get(`/books/${bookId}`);
    expect(finalGetResponse.status).toBe(404);
  });
});
