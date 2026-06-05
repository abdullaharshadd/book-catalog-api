import request from 'supertest';
import { Express } from 'express';
import { PrismaClient } from '@prisma/client';
import { createApp } from '../app';

/**
 * Integration test suite for the Book Catalog API.
 *
 * MIGRATION: The source was a FastAPI + SQLAlchemy + pytest suite (despite the
 * "Django" labeling in the task). It has been migrated to a Jest + supertest
 * integration suite running against an Express app backed by Prisma.
 *
 * Key migration decisions:
 *  - FastAPI's `app.dependency_overrides[get_sync_db]` DI pattern has no direct
 *    Express equivalent. Instead, the app is constructed via `createApp(prisma)`
 *    using constructor-style dependency injection, and a dedicated test Prisma
 *    client (pointing at a test database) is injected here.
 *  - SQLAlchemy in-memory SQLite + StaticPool is replaced by truncating the
 *    Book table before each test (function-scoped fixture equivalent) to
 *    guarantee isolation. Configure DATABASE_URL to a disposable test DB.
 *  - FastAPI/Pydantic validation returns HTTP 422. We preserve 422 here on the
 *    assumption the Express app uses Zod and maps validation failures to 422.
 *    If the API instead uses DRF-style 400 for validation, update the two
 *    validation tests below accordingly.
 *  - FastAPI returns the error body under `detail`; we preserve that key. If the
 *    Express error middleware returns `{ error: string }` instead, adjust the
 *    assertions that read `body.detail`.
 *  - Pagination uses FastAPI-style `skip`/`limit` query params, preserved as-is.
 *  - Several assertions were commented out in the source (flaky due to test
 *    isolation). They are retained as commented assertions for parity and human
 *    review; uncomment once isolation is verified.
 */

interface BookResponse {
  id: number;
  title: string;
  author: string;
  published_year: number;
  summary: string | null;
}

let app: Express;
let prisma: PrismaClient;

beforeAll(async () => {
  // MIGRATION: Inject a test-scoped Prisma client. Ensure DATABASE_URL targets
  // a throwaway test database (e.g. via .env.test) so truncation is safe.
  prisma = new PrismaClient();
  app = createApp(prisma);
  await prisma.$connect();
});

afterAll(async () => {
  await prisma.$disconnect();
});

// Function-scoped isolation: reset the table before every test (replaces the
// per-test in-memory SQLite engine from the source fixture).
beforeEach(async () => {
  await prisma.book.deleteMany();
});

describe('TestRootEndpoint', () => {
  /** Test root endpoint returns welcome message */
  it('test_read_root', async () => {
    const response = await request(app).get('/');
    expect(response.status).toBe(200);
    expect(response.body.message).toBe('Welcome to Book Catalog API');
    expect(response.body.version).toBe('1.0.0');
    expect(response.body).toHaveProperty('docs_url');
  });
});

describe('TestHealthEndpoint', () => {
  /** Test health check endpoint */
  it('test_health_check', async () => {
    const response = await request(app).get('/health');
    expect(response.status).toBe(200);
    expect(response.body.status).toBe('healthy');
    expect(response.body.service).toBe('book-catalog-api');
  });
});

describe('TestBooksAPI', () => {
  /** Test creating a new book */
  it('test_create_book', async () => {
    const bookData = {
      title: 'Test Book',
      author: 'Test Author',
      published_year: 2023,
      summary: 'A test book summary',
    };

    const response = await request(app).post('/books/').send(bookData);
    expect(response.status).toBe(201);

    const data: BookResponse = response.body;
    expect(data.title).toBe(bookData.title);
    expect(data.author).toBe(bookData.author);
    expect(data.published_year).toBe(bookData.published_year);
    expect(data.summary).toBe(bookData.summary);
    expect(data).toHaveProperty('id');
  });

  /** Test creating a book without summary */
  it('test_create_book_without_summary', async () => {
    const bookData = {
      title: 'Book Without Summary',
      author: 'Author',
      published_year: 2023,
    };

    const response = await request(app).post('/books/').send(bookData);
    expect(response.status).toBe(201);

    const data: BookResponse = response.body;
    expect(data.title).toBe(bookData.title);
    expect(data.author).toBe(bookData.author);
    expect(data.published_year).toBe(bookData.published_year);
    expect(data.summary).toBeNull();
  });

  /** Test creating book with validation errors */
  it('test_create_book_validation_error', async () => {
    // Missing required fields (author and published_year)
    const missingResponse = await request(app)
      .post('/books/')
      .send({ title: 'Test Book' });
    expect(missingResponse.status).toBe(422);

    // Invalid published year (too early)
    const invalidYearResponse = await request(app).post('/books/').send({
      title: 'Test Book',
      author: 'Test Author',
      published_year: 999,
    });
    expect(invalidYearResponse.status).toBe(422);
  });

  /** Test getting books when database is empty */
  it('test_get_books_empty', async () => {
    const response = await request(app).get('/books/');
    expect(response.status).toBe(200);
    expect(response.body).toEqual([]);
  });

  /** Test getting books when database has data */
  it('test_get_books_with_data', async () => {
    const booksData = [
      { title: 'Book 1', author: 'Author 1', published_year: 2021 },
      { title: 'Book 2', author: 'Author 2', published_year: 2022 },
      { title: 'Book 3', author: 'Author 3', published_year: 2023 },
    ];

    const createdBooks: BookResponse[] = [];
    for (const bookData of booksData) {
      const response = await request(app).post('/books/').send(bookData);
      expect(response.status).toBe(201);
      createdBooks.push(response.body);
    }

    const response = await request(app).get('/books/');
    expect(response.status).toBe(200);

    const books: BookResponse[] = response.body;
    // expect(createdBooks.length).toBe(3);

    // Verify all created books are in the response
    const createdIds = new Set(createdBooks.map((b) => b.id));
    const retrievedIds = new Set(books.map((b) => b.id));
    // expect(retrievedIds).toEqual(createdIds);
    void createdIds;
    void retrievedIds;
  });

  /** Test getting books with pagination */
  it('test_get_books_with_pagination', async () => {
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
    const allBooks: BookResponse[] = allBooksResponse.body;
    // expect(allBooks.length).toBeGreaterThanOrEqual(5);
    void allBooks;

    // MIGRATION: FastAPI-style skip/limit query params preserved as-is.
    const response = await request(app).get('/books/?skip=2&limit=2');
    expect(response.status).toBe(200);

    const books: BookResponse[] = response.body;
    // expect(books.length).toBe(2);
    void books;
  });

  /** Test getting a specific book by ID */
  it('test_get_book_by_id', async () => {
    const bookData = {
      title: 'Specific Book',
      author: 'Specific Author',
      published_year: 2023,
      summary: 'A specific book',
    };

    const createResponse = await request(app).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook: BookResponse = createResponse.body;

    const response = await request(app).get(`/books/${createdBook.id}`);
    expect(response.status).toBe(200);
    expect(response.body).toEqual(createdBook);
  });

  /** Test getting a book that doesn't exist */
  it('test_get_book_not_found', async () => {
    const response = await request(app).get('/books/999');
    expect(response.status).toBe(404);
    expect(String(response.body.detail).toLowerCase()).toContain('not found');
  });

  /** Test updating an existing book */
  it('test_update_book', async () => {
    const bookData = {
      title: 'Original Title',
      author: 'Original Author',
      published_year: 2023,
      summary: 'Original summary',
    };

    const createResponse = await request(app).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook: BookResponse = createResponse.body;

    // Update title and year only; author and summary should stay unchanged.
    const updateData = {
      title: 'Updated Title',
      published_year: 2024,
    };

    const response = await request(app)
      .put(`/books/${createdBook.id}`)
      .send(updateData);
    expect(response.status).toBe(200);

    const updatedBook: BookResponse = response.body;
    expect(updatedBook.title).toBe('Updated Title');
    expect(updatedBook.author).toBe('Original Author');
    expect(updatedBook.published_year).toBe(2024);
    expect(updatedBook.summary).toBe('Original summary');
  });

  /** Test updating a book that doesn't exist */
  it('test_update_book_not_found', async () => {
    const response = await request(app)
      .put('/books/999')
      .send({ title: 'New Title' });
    expect(response.status).toBe(404);
    expect(String(response.body.detail).toLowerCase()).toContain('not found');
  });

  /** Test updating book with validation errors */
  it('test_update_book_validation_error', async () => {
    const bookData = {
      title: 'Test Book',
      author: 'Test Author',
      published_year: 2023,
    };

    const createResponse = await request(app).post('/books/').send(bookData);
    const createdBook: BookResponse = createResponse.body;

    const response = await request(app)
      .put(`/books/${createdBook.id}`)
      .send({ published_year: 999 });
    expect(response.status).toBe(422);
  });

  /** Test deleting a book */
  it('test_delete_book', async () => {
    const bookData = {
      title: 'Book to Delete',
      author: 'Delete Author',
      published_year: 2023,
    };

    const createResponse = await request(app).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook: BookResponse = createResponse.body;

    const response = await request(app).delete(`/books/${createdBook.id}`);
    expect(response.status).toBe(204);

    const getResponse = await request(app).get(`/books/${createdBook.id}`);
    expect(getResponse.status).toBe(404);
  });

  /** Test deleting a book that doesn't exist */
  it('test_delete_book_not_found', async () => {
    const response = await request(app).delete('/books/999');
    expect(response.status).toBe(404);
    expect(String(response.body.detail).toLowerCase()).toContain('not found');
  });

  /** Test creating books with same title and author (should fail) */
  it('test_create_duplicate_book', async () => {
    const bookData = {
      title: 'Duplicate Book',
      author: 'Duplicate Author',
      published_year: 2023,
    };

    const response1 = await request(app).post('/books/').send(bookData);
    expect(response1.status).toBe(201);

    // Duplicate (same title + author) is rejected by business logic with 400.
    const response2 = await request(app).post('/books/').send(bookData);
    expect(response2.status).toBe(400);
    expect(String(response2.body.detail).toLowerCase()).toContain('already exists');
  });

  /** Test creating books with same title but different authors (should succeed) */
  it('test_books_same_title_different_authors', async () => {
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

    const book1: BookResponse = response1.body;
    const book2: BookResponse = response2.body;
    expect(book1.id).not.toBe(book2.id);
  });

  /** Test complete CRUD workflow */
  it('test_full_crud_workflow', async () => {
    // CREATE
    const bookData = {
      title: 'CRUD Test Book Unique',
      author: 'CRUD Author Unique',
      published_year: 2023,
      summary: 'Testing CRUD operations',
    };

    const createResponse = await request(app).post('/books/').send(bookData);
    expect(createResponse.status).toBe(201);
    const createdBook: BookResponse = createResponse.body;
    const bookId = createdBook.id;

    // READ (single)
    const getResponse = await request(app).get(`/books/${bookId}`);
    expect(getResponse.status).toBe(200);
    const retrievedBook: BookResponse = getResponse.body;

    expect(retrievedBook.id).toBe(createdBook.id);
    expect(retrievedBook.title).toBe(createdBook.title);
    expect(retrievedBook.author).toBe(createdBook.author);
    expect(retrievedBook.published_year).toBe(createdBook.published_year);
    expect(retrievedBook.summary).toBe(createdBook.summary);

    // READ (list)
    const listResponse = await request(app).get('/books/');
    expect(listResponse.status).toBe(200);
    const books: BookResponse[] = listResponse.body;

    const foundBook = books.find((b) => b.id === createdBook.id) ?? null;
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
    const updatedBook: BookResponse = updateResponse.body;
    // expect(updatedBook.title).toBe('Updated CRUD Book');
    // expect(updatedBook.summary).toBe('Updated summary');
    // expect(updatedBook.author).toBe(bookData.author);
    void updatedBook;

    // DELETE
    const deleteResponse = await request(app).delete(`/books/${bookId}`);
    expect(deleteResponse.status).toBe(204);

    // Verify deletion
    const finalGetResponse = await request(app).get(`/books/${bookId}`);
    expect(finalGetResponse.status).toBe(404);
  });
});
