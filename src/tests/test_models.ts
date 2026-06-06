// src/tests/test_models.ts
//
// Jest test suite for the Book model, migrated from the original SQLAlchemy/pytest
// suite. The source used an in-memory SQLite SQLAlchemy session; here we use Prisma
// against an in-memory / test SQLite database. Business logic (field validation,
// __repr__ / __str__ representations, and the composite unique constraint on
// (title, author)) is preserved exactly.
//
// MIGRATION: The original Book model defined `__repr__` and `__str__` dunder methods.
// Node.js/TypeScript has no dunder methods, so these are migrated to helper functions
// `bookRepr(book)` and `bookStr(book)` which MUST live alongside the Book model in
// `src/models/book.ts`. They are imported here. If they do not yet exist, create them
// with the exact formats asserted below.
//
// MIGRATION: The original used pytest fixtures + an in-memory SQLite engine created
// per-test. Here we use a single PrismaClient and clean the table in beforeEach to get
// equivalent test isolation. Ensure DATABASE_URL points at a disposable test database
// (e.g. file::memory:?cache=shared or a dedicated test sqlite file).

import { PrismaClient, Prisma } from '@prisma/client';
import { bookRepr, bookStr } from '../models/book';

/**
 * Shared Prisma client used by the Book model test suite.
 * MIGRATION: For true isolation per the original fixture, consider a dedicated test
 * database or transactions. Here we truncate the Book table before each test.
 */
const prisma = new PrismaClient();

beforeAll(async () => {
  await prisma.$connect();
});

afterAll(async () => {
  await prisma.$disconnect();
});

beforeEach(async () => {
  // Equivalent to the per-test fresh in-memory database in the original fixture.
  await prisma.book.deleteMany();
});

/**
 * Test cases for the Book model.
 */
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

    expect(book.id).not.toBeNull();
    expect(book.id).toBeDefined();
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

  it('produces the expected str string', async () => {
    const book = await prisma.book.create({
      data: {
        title: 'String Test',
        author: 'String Author',
        publishedYear: 2023,
      },
    });

    const expectedStr = 'String Test by String Author (2023)';
    expect(bookStr(book)).toBe(expectedStr);
  });

  it('rejects duplicate title-author combinations (unique constraint)', async () => {
    await prisma.book.create({
      data: {
        title: 'Duplicate Test',
        author: 'Duplicate Author',
        publishedYear: 2023,
      },
    });

    // Same title + author, different year — must violate the composite unique constraint.
    await expect(
      prisma.book.create({
        data: {
          title: 'Duplicate Test',
          author: 'Duplicate Author',
          publishedYear: 2024,
        },
      }),
    ).rejects.toMatchObject({
      // Prisma maps unique constraint violations to error code P2002.
      code: 'P2002',
    } as Partial<Prisma.PrismaClientKnownRequestError>);
  });

  it('allows books with the same title but different authors', async () => {
    const book1 = await prisma.book.create({
      data: { title: 'Common Title', author: 'Author One', publishedYear: 2023 },
    });
    const book2 = await prisma.book.create({
      data: { title: 'Common Title', author: 'Author Two', publishedYear: 2023 },
    });

    expect(book1.id).toBeDefined();
    expect(book2.id).toBeDefined();
    expect(book1.id).not.toBe(book2.id);
  });

  it('allows books with the same author but different titles', async () => {
    const book1 = await prisma.book.create({
      data: { title: 'First Book', author: 'Prolific Author', publishedYear: 2023 },
    });
    const book2 = await prisma.book.create({
      data: { title: 'Second Book', author: 'Prolific Author', publishedYear: 2024 },
    });

    expect(book1.id).toBeDefined();
    expect(book2.id).toBeDefined();
    expect(book1.id).not.toBe(book2.id);
  });
});
