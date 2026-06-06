/**
 * Integration test suite for the Book Catalog API.
 *
 * MIGRATION: The original `tests/test_api.py` was a **pytest** suite using
 * FastAPI's `TestClient`, SQLAlchemy in-memory SQLite, and
 * `app.dependency_overrides` for DB injection. In the Node.js/TypeScript
 * stack we use **Jest** + **supertest** + **Prisma**:
 *
 *   - `TestClient(app)`                -> `supertest(app)` (the Express app).
 *   - `@pytest.fixture test_db`        -> `setupTestDatabase()` /
 *                                         `teardownTestDatabase()` from
 *                                         `conftest.ts`, wired into Jest
 *                                         `beforeEach` / `afterEach`.
 *   - `app.dependency_overrides`       -> environment-based test DB selection
 *                                         (the test PrismaClient points at an
 *                                         isolated SQLite test database).
 *   - pytest classes (`TestRootEndpoint`, ...) -> Jest `describe` blocks.
 *   - `def test_*` methods             -> `it(...)` test cases.
 *
 * NOTE: FastAPI/Pydantic returns **422** for validation errors. If the target
 * API is built with the same convention (e.g. Zod validation middleware mapped
 * to 422) these assertions hold. If the target API returns 400 for validation,
 * the 422 assertions below must be updated. Status codes are preserved as in
 * the source (201 create, 204 delete, 422 validation, 400 duplicate, 404 not
 * found).
 *
 * MIGRATION: Several assertions were commented out in the source due to
 * apparent test-isolation/state issues. Because the Node.js harness resets the
 * test database between each test (`beforeEach`/`afterEach`), these checks are
 * restored here where they are now safe to assert. Lines that remain risky are
 * flagged with a `// MIGRATION:` comment.
 */

import request from 'supertest';
import type { Express } from 'express';

import { app } from '../app/main';
import { setupTestDatabase, teardownTestDatabase } from './conftest';

/**
 * The Express application under test, typed for supertest.
 */
const server: Express = app;

/**
 * BookInput describes the request body accepted by the create/update endpoints.
 */
interface BookInput {
  title?: string;
  author?: string;
  published_year?: number;
  summary?: string | null;
}

/**
 * Book describes the shape returned by the API for a persisted book.
 */
interface Book {
  id: number;
  title: string;
  author: string;
  published_year: number;
  summary: string | null;
}

beforeEach(async () => {
  await setupTestDatabase();
});

afterEach(async () => {
  await teardownTestDatabase();
});

describe('TestRootEndpoint', () => {
  /** Test root endpoint returns welcome message. */
  it('test_read_root', async () => {
    const response = await request(server).get('/');
    expect(response.status).toBe(200);

    const data = response.body;
    expect(data.message).toBe('Welcome to Book Catalog API');
    expect(data.version).toBe('1.0.0');
    expect(data).toHaveProperty('docs_url');
  });
});

describe('TestHealthEndpoint', () => {
  /** Test health check endpoint. */
  it('test_health_check', async () => {
    const response = await request(server).get('/health');
    expect(response.status).toBe(200);

    const data = response.body;
    expect(data.status).toBe('healthy');
    expect(data.service).toBe('book-catalog-api');
  });
});

describe('TestBooksAPI', () => {
  /** Test creating a new book. */
  it('test_create_book', async () => {
    const bookData: BookInput = {
      title: 'Test Book',
      author: 'Test Author',
      published_year: 2023,
      summary: 'A test book summary',
    };

    const response = await request(server).post('/books/').send(bookData);
    expect(response.status).toBe(201);

    const data: Book = response.body;
    expect(data.title).toBe(bookData.title);
    expect(data.author).toBe(bookData.author);
    expect(data.published_year).toBe(bookData.published_year);
    expect(data.summary).toBe(bookData.summary);
    expect(data).toHaveProperty('id');
  });

  /** Test creating a book without summary. */
  it('test_create_book_without_summary', async () => {
    const bookData: BookInput = {
      title: 'Book Without Summary',
      author: 'Author',
      published_year: 2023,
    };

    const response = await request(server).post('/books/').send(bookData);
    expect(response.status).toBe(201);

    const data: Book = response.body;
    expect(data.title).toBe(bookData.title);
    expect(data.author).toBe(bookData.author);
    expect(data.published_year).toBe(bookData.published_year);
    expect(data.summary).toBeNull();
  });

  /** Test creating book with validation errors. */
  it('test_create_book_validation_error', async () => {
    // Missing required fields (author and published_year).
    let response = await request(server)
      .post('/books/')
      .send({ title: 'Test Book' });
    // MIGRATION: 422 is FastAPI/Pydantic-specific. Adjust to 400 if the target
    // API returns 400 for validation errors.
    expect(response.status).toBe(422);

    // Invalid published year (too early).
    response = await request(server).post('/books/').send({
      title: 'Test Book',
      author: 'Test Author',
      published_year: 999,
    });
    expect(response.status).toBe(422);
  });

  /** Test getting books when database is empty. */
  it('test_get_books_empty', async () => {
    const response = await request(server).get('/books/');
    expect(response.status).toBe(200);
    expect(response.body).toEqual([]);
  });

  /** Test getting books when database has data. */
  it('test_get_books_with_data', async () => {
    const booksData: BookInput[] = [
      { title: 'Book 1', author: 'Author 1', published_year: 2021 },
      { title: 'Book 2', author: 'Author 2', published_year: 2022 },
      { title: 'Book 3', author: 'Author 3', published_year: 2023 },
    ];

    const createdBooks: Book[] = [];
    for (const bookData of booksData) {
      const response = await request(server).post('/books/').send(bookData);
      expect(response.status).toBe(201);
      createdBooks.push(response.body);
    }

    const response = await request(server).get('/books/');
    expect(response.status).toBe(200);

    const books: Book[] = response.body;
    // Restored: with proper per-test DB isolation these checks are now safe.
    expect(createdBooks).toHaveLength(3);

    const createdIds = new Set(createdBooks.map((book) => book.id));
    const retrievedIds = new Set(books.map((book) => book.id));
    expect(retrievedIds).toEqual(createdIds);
  });

  /** Test getting books with pagination. */
  it('test_get_books_with_pagination', async () => {
    const createdIds: number[] = [];
    for (let i = 0; i < 5; i++) {
      const bookData: BookInput = {
        title: `Pagination Book ${i + 1}`,
        author: `Pagination Author ${i + 1}`,
        published_year: 2020 + i,
      };
      const response = await request(server).post('/books/').send(bookData);
      expect(response.status).toBe(201);
      createdIds.push((response.body as Book).id);
    }

    const allBooksResponse = await request(server).get('/books/');
    expect(allBooksResponse.status).toBe(200);
    const allBooks: Book[] = allBooksResponse.body;
    expect(allBooks.length).toBeGreaterThanOrEqual(5);

    const response = await request(server).get('/books/?skip=2&limit=2');
    expect(response.status).toBe(200);

    const books: Book[] = response.body;
    expect(books).toHaveLength(2);
  });

  /** Test getting a specific book by ID. */
  it('test_get_book_by_id', async () => {
    const bookData: BookInput = {
      title: 'Specific Book',
      author: 'Specific Author',
      published_year: 2023,
      summary: 'A specific book',
    };

    const createResponse = await request(server).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook: Book = createResponse.body;

    const response = await request(server).get(`/books/${createdBook.id}`);
    expect(response.status).toBe(200);

    const retrievedBook: Book = response.body;
    expect(retrievedBook).toEqual(createdBook);
  });

  /** Test getting a book that doesn't exist. */
  it('test_get_book_not_found', async () => {
    const response = await request(server).get('/books/999');
    expect(response.status).toBe(404);
    expect(response.body.detail.toLowerCase()).toContain('not found');
  });

  /** Test updating an existing book. */
  it('test_update_book', async () => {
    const bookData: BookInput = {
      title: 'Original Title',
      author: 'Original Author',
      published_year: 2023,
      summary: 'Original summary',
    };

    const createResponse = await request(server).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook: Book = createResponse.body;

    // Not updating author or summary.
    const updateData: BookInput = {
      title: 'Updated Title',
      published_year: 2024,
    };

    const response = await request(server)
      .put(`/books/${createdBook.id}`)
      .send(updateData);
    expect(response.status).toBe(200);

    const updatedBook: Book = response.body;
    expect(updatedBook.title).toBe('Updated Title');
    expect(updatedBook.author).toBe('Original Author'); // Unchanged
    expect(updatedBook.published_year).toBe(2024);
    expect(updatedBook.summary).toBe('Original summary'); // Unchanged
  });

  /** Test updating a book that doesn't exist. */
  it('test_update_book_not_found', async () => {
    const updateData: BookInput = { title: 'New Title' };

    const response = await request(server).put('/books/999').send(updateData);
    expect(response.status).toBe(404);
    expect(response.body.detail.toLowerCase()).toContain('not found');
  });

  /** Test updating book with validation errors. */
  it('test_update_book_validation_error', async () => {
    const bookData: BookInput = {
      title: 'Test Book',
      author: 'Test Author',
      published_year: 2023,
    };

    const createResponse = await request(server).post('/books/').send(bookData);
    const createdBook: Book = createResponse.body;

    const response = await request(server)
      .put(`/books/${createdBook.id}`)
      .send({ published_year: 999 }); // Invalid year
    expect(response.status).toBe(422);
  });

  /** Test deleting a book. */
  it('test_delete_book', async () => {
    const bookData: BookInput = {
      title: 'Book to Delete',
      author: 'Delete Author',
      published_year: 2023,
    };

    const createResponse = await request(server).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook: Book = createResponse.body;

    const response = await request(server).delete(`/books/${createdBook.id}`);
    expect(response.status).toBe(204);

    const getResponse = await request(server).get(`/books/${createdBook.id}`);
    expect(getResponse.status).toBe(404);
  });

  /** Test deleting a book that doesn't exist. */
  it('test_delete_book_not_found', async () => {
    const response = await request(server).delete('/books/999');
    expect(response.status).toBe(404);
    expect(response.body.detail.toLowerCase()).toContain('not found');
  });

  /** Test creating books with same title and author (should fail). */
  it('test_create_duplicate_book', async () => {
    const bookData: BookInput = {
      title: 'Duplicate Book',
      author: 'Duplicate Author',
      published_year: 2023,
    };

    const response1 = await request(server).post('/books/').send(bookData);
    expect(response1.status).toBe(201);

    // Same title and author -> duplicate.
    const response2 = await request(server).post('/books/').send(bookData);
    expect(response2.status).toBe(400);
    expect(response2.body.detail.toLowerCase()).toContain('already exists');
  });

  /** Test creating books with same title but different authors (should succeed). */
  it('test_books_same_title_different_authors', async () => {
    const book1Data: BookInput = {
      title: 'Same Title',
      author: 'Author One',
      published_year: 2023,
    };

    const book2Data: BookInput = {
      title: 'Same Title',
      author: 'Author Two',
      published_year: 2023,
    };

    const response1 = await request(server).post('/books/').send(book1Data);
    expect(response1.status).toBe(201);

    const response2 = await request(server).post('/books/').send(book2Data);
    expect(response2.status).toBe(201);

    const book1: Book = response1.body;
    const book2: Book = response2.body;
    expect(book1.id).not.toBe(book2.id);
  });

  /** Test complete CRUD workflow. */
  it('test_full_crud_workflow', async () => {
    // CREATE
    const bookData: BookInput = {
      title: 'CRUD Test Book Unique',
      author: 'CRUD Author Unique',
      published_year: 2023,
      summary: 'Testing CRUD operations',
    };

    const createResponse = await request(server).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook: Book = createResponse.body;
    const bookId = createdBook.id;

    // READ (single) - verify the book was created.
    const getResponse = await request(server).get(`/books/${bookId}`);
    expect(getResponse.status).toBe(200);
    const retrievedBook: Book = getResponse.body;

    // Compare fields individually to be more robust.
    expect(retrievedBook.id).toBe(createdBook.id);
    expect(retrievedBook.title).toBe(createdBook.title);
    expect(retrievedBook.author).toBe(createdBook.author);
    expect(retrievedBook.published_year).toBe(createdBook.published_year);
    expect(retrievedBook.summary).toBe(createdBook.summary);

    // READ (list) - verify the book appears in the list.
    const listResponse = await request(server).get('/books/');
    expect(listResponse.status).toBe(200);
    const books: Book[] = listResponse.body;

    const foundBook = books.find((book) => book.id === createdBook.id) ?? null;
    // Restored: with per-test DB isolation the created book must appear.
    expect(foundBook).not.toBeNull();
    expect(foundBook?.title).toBe(bookData.title);

    // UPDATE
    const updateData: BookInput = {
      title: 'Updated CRUD Book',
      summary: 'Updated summary',
    };
    const updateResponse = await request(server)
      .put(`/books/${bookId}`)
      .send(updateData);
    expect(updateResponse.status).toBe(200);
    const updatedBook: Book = updateResponse.body;
    // Restored from source (was commented out due to isolation issues).
    expect(updatedBook.title).toBe('Updated CRUD Book');
    expect(updatedBook.summary).toBe('Updated summary');
    expect(updatedBook.author).toBe(bookData.author); // Unchanged

    // DELETE
    const deleteResponse = await request(server).delete(`/books/${bookId}`);
    expect(deleteResponse.status).toBe(204);

    // Verify deletion.
    const finalGetResponse = await request(server).get(`/books/${bookId}`);
    expect(finalGetResponse.status).toBe(404);
  });
});
