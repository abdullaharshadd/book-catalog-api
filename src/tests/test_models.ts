// src/tests/test_models.ts

/**
 * Unit tests for the Book model.
 *
 * MIGRATION_NOTE: The Python source used pytest with a SQLAlchemy in-memory
 * SQLite session and instantiated `Book` ORM objects directly (`Book(...)`),
 * relying on the ORM to persist rows, autoincrement `id`, and enforce the
 * composite unique constraint on (title, author).
 *
 * In the Express/TS + Prisma world:
 *   - There is no ORM "model object" to instantiate the way SQLAlchemy does.
 *     `Book` (from src/app/models.ts) is a plain TypeScript interface, so we
 *     create rows via `prisma.book.create(...)` and assert on the returned
 *     record.
 *   - The `repr()` / `str()` dunder methods map to the migrated helper
 *     functions `bookToRepr()` and `bookToString()` (src/app/models.ts).
 *   - The unique-constraint violation surfaces as a Prisma
 *     `PrismaClientKnownRequestError` with code `P2002` instead of a
 *     SQLAlchemy `IntegrityError`.
 *   - The per-test SQLite session fixture is replaced by the shared test-DB
 *     helpers (`setupTestDb` / `resetTestDb` / `teardownTestDb`) from
 *     src/conftest.ts, driving the PostgreSQL test database via Prisma.
 *
 * The target database is PostgreSQL; the composite unique constraint
 * `@@unique([title, author])` must exist in schema.prisma for the violation
 * tests to pass.
 */

import { Prisma } from '@prisma/client';
import { prisma } from '../app/database';
import { bookToRepr, bookToString } from '../app/models';
import { setupTestDb, resetTestDb, teardownTestDb } from '../conftest';

beforeAll(async () => {
  await setupTestDb();
});

afterEach(async () => {
  await resetTestDb();
});

afterAll(async () => {
  await teardownTestDb();
});

describe('Book model', () => {
  it('creates a basic book', async () => {
    const book = await prisma.book.create({
      data: {
        title: 'Test Book',
        author: 'Test Author',
        publishedYear: 2023,
        summary: 'A test book summary',
      },
    });

    expect(book.id).toBeDefined();
    expect(book.id).not.toBeNull();
    expect(book.title).toBe('Test Book');
    expect(book.author).toBe('Test Author');
    expect(book.publishedYear).toBe(2023);
    expect(book.summary).toBe('A test book summary');
  });

  it('creates a book without summary (optional field)', async () => {
    const book = await prisma.book.create({
      data: {
        title: 'Test Book No Summary',
        author: 'Test Author',
        publishedYear: 2023,
      },
    });

    expect(book.id).toBeDefined();
    expect(book.id).not.toBeNull();
    expect(book.title).toBe('Test Book No Summary');
    expect(book.author).toBe('Test Author');
    expect(book.publishedYear).toBe(2023);
    expect(book.summary).toBeNull();
  });

  it('produces the expected repr string', async () => {
    const book = await prisma.book.create({
      data: {
        title: 'Repr Test',
        author: 'Repr Author',
        publishedYear: 2023,
        summary: 'Test summary',
      },
    });

    const expectedRepr = `<Book(id=${book.id}, title='Repr Test', author='Repr Author', year=2023)>`;
    expect(bookToRepr(book)).toBe(expectedRepr);
  });

  it('produces the expected str string', async () => {
    const book = await prisma.book.create({
      data: {
        title: 'String Test',
        author: 'String Author',
        publishedYear: 2023,
      },
    });

    expect(bookToString(book)).toBe('String Test by String Author (2023)');
  });

  it('rejects duplicate title-author combinations (unique constraint)', async () => {
    await prisma.book.create({
      data: {
        title: 'Duplicate Test',
        author: 'Duplicate Author',
        publishedYear: 2023,
      },
    });

    // Same title + author, different year -> unique constraint violation.
    await expect(
      prisma.book.create({
        data: {
          title: 'Duplicate Test',
          author: 'Duplicate Author',
          publishedYear: 2024,
        },
      }),
    ).rejects.toMatchObject({ code: 'P2002' });
  });

  it('allows books with same title but different authors', async () => {
    const book1 = await prisma.book.create({
      data: {
        title: 'Common Title',
        author: 'Author One',
        publishedYear: 2023,
      },
    });

    const book2 = await prisma.book.create({
      data: {
        title: 'Common Title',
        author: 'Author Two',
        publishedYear: 2023,
      },
    });

    expect(book1.id).toBeDefined();
    expect(book2.id).toBeDefined();
    expect(book1.id).not.toBe(book2.id);
  });

  it('allows books with same author but different titles', async () => {
    const book1 = await prisma.book.create({
      data: {
        title: 'First Book',
        author: 'Prolific Author',
        publishedYear: 2023,
      },
    });

    const book2 = await prisma.book.create({
      data: {
        title: 'Second Book',
        author: 'Prolific Author',
        publishedYear: 2024,
      },
    });

    expect(book1.id).toBeDefined();
    expect(book2.id).toBeDefined();
    expect(book1.id).not.toBe(book2.id);
  });
});

// Silence unused-import lint if Prisma namespace isn't referenced elsewhere.
void Prisma;
