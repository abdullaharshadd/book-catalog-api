/**
 * Unit test suite for the Book model.
 *
 * MIGRATION: The original `tests/test_models.py` was a **pytest** suite that
 * exercised a **SQLAlchemy** declarative `Book` model against an in-memory
 * SQLite database. The mapping to the Node.js/TypeScript + Jest + Prisma stack
 * is as follows:
 *
 *   - `@pytest.fixture db_session` (in-memory SQLite engine + sessionmaker)
 *                                  -> `setupTestDatabase()` /
 *                                     `teardownTestDatabase()` from
 *                                     `conftest.ts`, driven by Jest
 *                                     `beforeEach` / `afterEach`.
 *   - `session.add` + `session.commit` + `session.refresh`
 *                                  -> `prisma.book.create({ data })`, which
 *                                     persists and returns the populated row in
 *                                     one call (no separate refresh needed).
 *   - `repr(book)` / `str(book)`   -> `bookRepr()` / `bookStr()` helpers from
 *                                     `app/models.ts`, since Prisma models are
 *                                     plain data objects with no methods.
 *   - SQLAlchemy `IntegrityError` on the unique (title, author) constraint
 *                                  -> Prisma `PrismaClientKnownRequestError`
 *                                     with code `P2002` (unique constraint
 *                                     violation).
 *
 * Business logic preserved verbatim:
 *   - `summary` is optional and defaults to `null`.
 *   - The unique constraint applies to the (title, author) *combination*,
 *     so duplicate titles with different authors (and vice versa) are allowed.
 *   - `repr` / `str` formats are asserted against the exact original strings.
 */

import { PrismaClient, Prisma, Book } from '@prisma/client';
import { bookRepr, bookStr } from '../app/models';
import { setupTestDatabase, teardownTestDatabase, getTestPrisma } from './conftest';

describe('Book model', () => {
  let prisma: PrismaClient;

  beforeEach(async () => {
    // MIGRATION: replaces the per-test in-memory SQLite `db_session` fixture.
    // `setupTestDatabase` is expected to provision an isolated test database
    // (e.g. a fresh schema / truncated tables) and return/expose a client.
    await setupTestDatabase();
    prisma = getTestPrisma();
  });

  afterEach(async () => {
    await teardownTestDatabase();
  });

  /**
   * test_create_book: a fully-populated book is persisted and read back with
   * an auto-generated id and all field values intact.
   */
  it('creates a basic book', async () => {
    const book: Book = await prisma.book.create({
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

  /**
   * test_create_book_without_summary: the optional `summary` field defaults to
   * null when omitted.
   */
  it('creates a book without a summary (optional field)', async () => {
    const book: Book = await prisma.book.create({
      data: {
        title: 'Test Book No Summary',
        author: 'Test Author',
        publishedYear: 2023,
      },
    });

    expect(book.id).not.toBeNull();
    expect(book.id).toBeDefined();
    expect(book.title).toBe('Test Book No Summary');
    expect(book.author).toBe('Test Author');
    expect(book.publishedYear).toBe(2023);
    expect(book.summary).toBeNull();
  });

  /**
   * test_book_repr: the repr helper reproduces the SQLAlchemy `__repr__` format.
   */
  it('produces the expected repr string', async () => {
    const book: Book = await prisma.book.create({
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

  /**
   * test_book_str: the str helper reproduces the SQLAlchemy `__str__` format.
   */
  it('produces the expected str string', async () => {
    const book: Book = await prisma.book.create({
      data: {
        title: 'String Test',
        author: 'String Author',
        publishedYear: 2023,
      },
    });

    const expectedStr = 'String Test by String Author (2023)';
    expect(bookStr(book)).toBe(expectedStr);
  });

  /**
   * test_unique_constraint_violation: a duplicate (title, author) combination
   * must be rejected by the unique constraint.
   *
   * MIGRATION: SQLAlchemy raised `IntegrityError`; Prisma raises a
   * `PrismaClientKnownRequestError` with code `P2002` for unique violations.
   */
  it('rejects duplicate title-author combinations', async () => {
    await prisma.book.create({
      data: {
        title: 'Duplicate Test',
        author: 'Duplicate Author',
        publishedYear: 2023,
      },
    });

    // Same title + author, different year -> must violate the unique constraint.
    const attempt = prisma.book.create({
      data: {
        title: 'Duplicate Test',
        author: 'Duplicate Author',
        publishedYear: 2024,
      },
    });

    await expect(attempt).rejects.toMatchObject({ code: 'P2002' });
  });

  /**
   * test_books_with_same_title_different_authors: same title but distinct
   * authors are allowed because the constraint is on the combination.
   */
  it('allows books with the same title but different authors', async () => {
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

  /**
   * test_books_with_same_author_different_titles: same author but distinct
   * titles are allowed because the constraint is on the combination.
   */
  it('allows books with the same author but different titles', async () => {
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
