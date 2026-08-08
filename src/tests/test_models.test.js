```typescript
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
  // ---------------------------------------------------------------------------
  // Book.create — creating a book with all fields including summary
  // ---------------------------------------------------------------------------
  describe('Book.create', () => {
    it('persists a book with all fields including summary and assigns a non-null unique id', async () => {
      const book = await prisma.book.create({
        data: {
          title: 'Test Book',
          author: 'Test Author',
          publishedYear: 2023,
          summary: 'A test book summary',
        },
      });

      // id must be assigned (non-null) after persistence
      expect(book.id).toBeDefined();
      expect(book.id).not.toBeNull();

      // field values must match inputs
      expect(book.title).toBe('Test Book');
      expect(book.author).toBe('Test Author');
      expect(book.publishedYear).toBe(2023);
      expect(book.summary).toBe('A test book summary');
    });

    it('inserts exactly one row into the books table when creating a book with all fields', async () => {
      const beforeCount = await prisma.book.count();

      await prisma.book.create({
        data: {
          title: 'Count Check Book',
          author: 'Count Check Author',
          publishedYear: 2021,
          summary: 'Summary here',
        },
      });

      const afterCount = await prisma.book.count();
      expect(afterCount).toBe(beforeCount + 1);
    });

    // -------------------------------------------------------------------------
    // Book.create — creating a book without a summary (optional field)
    // -------------------------------------------------------------------------
    it('persists a book without a summary and sets summary to null', async () => {
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

    it('inserts exactly one row into the books table when creating a book without a summary', async () => {
      const beforeCount = await prisma.book.count();

      await prisma.book.create({
        data: {
          title: 'No Summary Count Check',
          author: 'Author X',
          publishedYear: 2022,
        },
      });

      const afterCount = await prisma.book.count();
      expect(afterCount).toBe(beforeCount + 1);
    });

    // -------------------------------------------------------------------------
    // Book.create — duplicate (title, author) pair triggers constraint violation
    // -------------------------------------------------------------------------
    it('rejects duplicate title-author combinations with a P2002 constraint error', async () => {
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

    it('does not insert a second row when the unique constraint is violated', async () => {
      await prisma.book.create({
        data: {
          title: 'Dup Row Count',
          author: 'Dup Row Author',
          publishedYear: 2020,
        },
      });

      const countAfterFirst = await prisma.book.count();

      try {
        await prisma.book.create({
          data: {
            title: 'Dup Row Count',
            author: 'Dup Row Author',
            publishedYear: 2021,
          },
        });
        // If we reach here, the test should fail
        fail('Expected a unique constraint error but none was thrown');
      } catch {
        // expected — swallow the error
      }

      const countAfterSecond = await prisma.book.count();
      expect(countAfterSecond).toBe(countAfterFirst);
    });

    it('rejects exact duplicate (same title, author, and year) with a P2002 error', async () => {
      await prisma.book.create({
        data: {
          title: 'Exact Dup',
          author: 'Exact Dup Author',
          publishedYear: 2019,
        },
      });

      await expect(
        prisma.book.create({
          data: {
            title: 'Exact Dup',
            author: 'Exact Dup Author',
            publishedYear: 2019,
          },
        }),
      ).rejects.toMatchObject({ code: 'P2002' });
    });

    // -------------------------------------------------------------------------
    // Book.create — same title, different authors (should succeed)
    // -------------------------------------------------------------------------
    it('allows two books with the same title but different authors', async () => {
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
      expect(book1.id).not.toBeNull();
      expect(book2.id).toBeDefined();
      expect(book2.id).not.toBeNull();
      expect(book1.id).not.toBe(book2.id);
    });

    it('inserts two rows when two books share the same title but have different authors', async () => {
      const beforeCount = await prisma.book.count();

      await prisma.book.create({
        data: {
          title: 'Shared Title',
          author: 'First Author',
          publishedYear: 2023,
        },
      });

      await prisma.book.create({
        data: {
          title: 'Shared Title',
          author: 'Second Author',
          publishedYear: 2023,
        },
      });

      const afterCount = await prisma.book.count();
      expect(afterCount).toBe(beforeCount + 2);
    });

    // -------------------------------------------------------------------------
    // Book.create — same author, different titles (should succeed)
    // -------------------------------------------------------------------------
    it('allows two books with the same author but different titles', async () => {
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
      expect(book1.id).not.toBeNull();
      expect(book2.id).toBeDefined();
      expect(book2.id).not.toBeNull();
      expect(book1.id).not.toBe(book2.id);
    });

    it('inserts two rows when two books share the same author but have different titles', async () => {
      const beforeCount = await prisma.book.count();

      await prisma.book.create({
        data: {
          title: 'Alpha Title',
          author: 'Shared Author',
          publishedYear: 2023,
        },
      });

      await prisma.book.create({
        data: {
          title: 'Beta Title',
          author: 'Shared Author',
          publishedYear: 2024,
        },
      });

      const afterCount = await prisma.book.count();
      expect(afterCount).toBe(beforeCount + 2);
    });

    // -------------------------------------------------------------------------
    // Global invariants — id uniqueness across multiple books
    // -------------------------------------------------------------------------
    it('assigns unique ids to each persisted book', async () => {
      const bookA = await prisma.book.create({
        data: {
          title: 'Unique ID Book A',
          author: 'Author A',
          publishedYear: 2020,
        },
      });

      const bookB = await prisma.book.create({
        data: {
          title: 'Unique ID Book B',
          author: 'Author B',
          publishedYear: 2021,
        },
      });

      const bookC = await prisma.book.create({
        data: {
          title: 'Unique ID Book C',
          author: 'Author C',
          publishedYear: 2022,
        },
      });

      const ids = new Set([bookA.id, bookB.id, bookC.id]);
      expect(ids.size).toBe(3);
    });

    it('stores publishedYear as an integer value', async () => {
      const book = await prisma.book.create({
        data: {
          title: 'Int Year Book',
          author: 'Int Year Author',
          publishedYear: 1984,
        },
      });

      expect(typeof book.publishedYear).toBe('number');
      expect(Number.isInteger(book.publishedYear)).toBe(true);
      expect(book.publishedYear).toBe(1984);
    });
  });

  // ---------------------------------------------------------------------------
  // Book.__repr__ — developer-facing string representation
  // ---------------------------------------------------------------------------
  describe('Book.__repr__ (bookToRepr)', () => {
    it('produces the expected repr string for a persisted book', async () => {
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

    it('repr includes the book id', async () => {
      const book = await prisma.book.create({
        data: {
          title: 'Repr ID Check',
          author: 'Repr ID Author',
          publishedYear: 2000,
        },
      });

      const repr = bookToRepr(book);
      expect(repr).toContain(`id=${book.id}`);
    });

    it('repr includes the book title', async () => {
      const book = await prisma.book.create({
        data: {
          title: 'Repr Title Check',
          author: 'Some Author',
          publishedYear: 2001,
        },
      });

      const repr = bookToRepr(book);
      expect(repr).toContain("title='Repr Title Check'");
    });

    it('repr includes the book author', async () => {
      const book = await prisma.book.create({
        data: {
          title: 'Repr Author Check Book',
          author: 'Repr Author Check',
          publishedYear: 2002,
        },
      });

      const repr = bookToRepr(book);
      expect(repr).toContain("author='Repr Author Check'");
    });

    it('repr includes the published year', async () => {
      const book = await prisma.book.create({
        data: {
          title: 'Repr Year Check',
          author: 'Year Check Author',
          publishedYear: 1999,
        },
      });

      const repr = bookToRepr(book);
      expect(repr).toContain('year=1999');
    });

    it('repr matches the exact format <Book(id=..., title=\'...\', author=\'...\', year=...)>', async () => {
      const book = await prisma.book.create({
        data: {
          title: 'Format Check',
          author: 'Format Author',
          publishedYear: 2010,
        },
      });

      const reprRegex = /^<Book\(id=.+, title='[^']*', author='[^']*', year=\d+\)>$/;
      expect(bookToRepr(book)).toMatch(reprRegex);
    });

    it('repr does not include the summary field', async () => {
      const book = await prisma.book.create({
        data: {
          title: 'No Summary In Repr',
          author: 'Repr Author',
          publishedYear: 2015,
          summary: 'This should not appear in repr',
        },
      });

      const repr = bookToRepr(book);
      expect(repr).not.toContain('summary');
    });
  });

  // ---------------------------------------------------------------------------
  // Book.__str__ — human-readable string representation
  // ---------------------------------------------------------------------------
  describe('Book.__str__ (bookToString)', () => {
    it('produces the expected str string for a persisted book', async () => {
      const book = await prisma.book.create({
        data: {
          title: 'String Test',
          author: 'String Author',
          publishedYear: 2023,
        },
      });

      expect(bookToString(book)).toBe('String Test by String Author (2023)');
    });

    it('str includes the book title', async () => {
      const book = await prisma.book.create({
        data: {
          title: 'Str Title Check',
          author: 'Author',
          publishedYear: 2005,
        },
      });

      expect(bookToString(book)).toContain('Str Title Check');
    