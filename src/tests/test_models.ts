// src/tests/test_models.ts

import { PrismaClient, Prisma, type Book } from '@prisma/client';
import { prismaTestClient, resetDatabase } from '../conftest';
import { bookRepr, bookStr } from '../app/models';

/**
 * MIGRATION NOTES
 * ----------------
 * The source `tests/test_models.py` was a **pytest** unit-test suite for a
 * **SQLAlchemy** `Book` model (despite the Django-style filename, it is NOT a
 * Django project). It verified:
 *   - basic book creation
 *   - optional `summary` field (nullable)
 *   - `__repr__` / `__str__` business behaviour
 *   - the composite `(title, author)` unique constraint
 *
 * In the idiomatic Node.js/TypeScript stack we use **Jest** + **Prisma**.
 * Mapping decisions:
 *
 *   - The pytest `db_session` fixture (in-memory SQLite engine + sessionmaker +
 *     `Base.metadata.create_all`) is replaced by the shared Prisma test client
 *     from `conftest.ts`, with `resetDatabase()` run before each test to give
 *     each case an isolated, clean schema state (equivalent to a fresh session).
 *   - SQLAlchemy `session.add` / `commit` / `refresh` map to
 *     `prisma.book.create({ data })`, which returns the persisted row
 *     (including the auto-generated `id` and DB defaults) directly.
 *   - The SQLAlchemy `__repr__` / `__str__` methods live as the pure helper
 *     functions `bookRepr` / `bookStr` in `app/models.ts`, so we assert against
 *     those rather than instance methods.
 *   - SQLAlchemy raises `IntegrityError` on a unique-constraint violation; Prisma
 *     raises `PrismaClientKnownRequestError` with code `P2002`. The original
 *     test caught a generic `Exception`; here we assert the create rejects
 *     (and, where useful, that it is the unique-constraint error).
 *
 * PREREQUISITE: `prisma/schema.prisma` must define a composite unique
 * constraint on `(title, author)` (e.g. `@@unique([title, author])`) for the
 * unique-constraint tests to behave as the source expects.
 */

let prisma: PrismaClient;

beforeAll(() => {
  prisma = prismaTestClient;
});

beforeEach(async () => {
  await resetDatabase(prisma);
});

afterAll(async () => {
  await prisma.$disconnect();
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

  it('creates a book without a summary (optional field)', async () => {
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
    expect(bookRepr(book)).toBe(expectedRepr);
  });

  it('produces the expected str representation', async () => {
    const book = await prisma.book.create({
      data: {
        title: 'String Test',
        author: 'String Author',
        publishedYear: 2023,
      },
    });

    expect(bookStr(book)).toBe('String Test by String Author (2023)');
  });

  it('rejects duplicate title-author combinations (unique constraint)', async () => {
    await prisma.book.create({
      data: {
        title: 'Duplicate Test',
        author: 'Duplicate Author',
        publishedYear: 2023,
      },
    });

    // Same title + author, different year -> must violate the composite unique
    // constraint and reject with Prisma's P2002 error.
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

  it('allows same title with different authors', async () => {
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

    expect(book1.id).not.toBeNull();
    expect(book2.id).not.toBeNull();
    expect(book1.id).not.toBe(book2.id);
  });

  it('allows same author with different titles', async () => {
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

    expect(book1.id).not.toBeNull();
    expect(book2.id).not.toBeNull();
    expect(book1.id).not.toBe(book2.id);
  });
});
