/**
 * Unit test suite for the Book Zod validation schemas.
 *
 * MIGRATION: The original `tests/test_schemas.py` was a **pytest** suite that
 * exercised **Pydantic** schemas (`BookCreate`, `BookUpdate`, `BookResponse`)
 * — a FastAPI-style validation layer, NOT Django/DRF. In the
 * Node.js/TypeScript + Jest stack we use **Zod** schemas as the idiomatic
 * equivalent of Pydantic models:
 *
 *   - `BookCreate(**data)`        -> `bookCreateSchema.parse(data)`
 *   - `BookUpdate(**data)`        -> `bookUpdateSchema.parse(data)`
 *   - `BookResponse(**data)`      -> `bookResponseSchema.parse(data)`
 *   - `pytest.raises(ValidationError)` -> expecting `schema.parse(...)` to
 *                                    throw a `ZodError`, asserted via
 *                                    `expect(() => ...).toThrow(...)`.
 *
 * IMPORTANT MIGRATION NOTES:
 *   1. The original asserted **verbatim Pydantic v1** messages such as
 *      `'ensure this value has at most 255 characters'`. These are NOT
 *      reproducible in Zod. Instead we assert on the **custom messages**
 *      defined in the Zod schemas (see `src/schemas.ts`). For length limits
 *      we assert on a clear custom message (e.g. 'must be at most 255
 *      characters'). The exact wording depends on the migrated schema —
 *      review `src/schemas.ts` to keep these strings in sync.
 *   2. The `published_year` future check uses `new Date().getFullYear()`
 *      computed dynamically, mirroring Python's `datetime.now().year`, so the
 *      edge-case tests remain deterministic.
 *   3. `BookUpdate` is the PATCH/partial schema where all fields are optional
 *      but field-level validators still run when a value is provided.
 *   4. `BookResponse` requires a non-optional `id`.
 *
 * These tests assume `src/schemas.ts` exports `bookCreateSchema`,
 * `bookUpdateSchema`, and `bookResponseSchema` Zod schemas whose error
 * messages match the assertions below. If the schema messages differ, update
 * the expected strings here accordingly.
 */

import { ZodError } from 'zod';
import {
  bookCreateSchema,
  bookUpdateSchema,
  bookResponseSchema,
} from '../schemas';

/**
 * extractZodMessage flattens a thrown error into a single searchable string,
 * mirroring the Python pattern of `str(exc_info.value)` used to assert on
 * substrings of a Pydantic ValidationError.
 */
function extractZodMessage(error: unknown): string {
  if (error instanceof ZodError) {
    return error.issues
      .map((issue) => `${issue.path.join('.')}: ${issue.message}`)
      .join(' | ');
  }
  return String(error);
}

/**
 * expectParseError runs a parse and returns the flattened error message,
 * failing the test if the parse unexpectedly succeeds.
 */
function expectParseError(parse: () => unknown): string {
  try {
    parse();
  } catch (err) {
    return extractZodMessage(err);
  }
  throw new Error('Expected validation to throw, but it succeeded');
}

describe('bookCreateSchema (BookCreate)', () => {
  it('parses a valid book', () => {
    const book = bookCreateSchema.parse({
      title: 'Valid Book',
      author: 'Valid Author',
      published_year: 2023,
      summary: 'A valid book summary',
    });

    expect(book.title).toBe('Valid Book');
    expect(book.author).toBe('Valid Author');
    expect(book.published_year).toBe(2023);
    expect(book.summary).toBe('A valid book summary');
  });

  it('parses a book without summary (optional)', () => {
    const book = bookCreateSchema.parse({
      title: 'Book Without Summary',
      author: 'Author',
      published_year: 2023,
    });

    expect(book.title).toBe('Book Without Summary');
    expect(book.author).toBe('Author');
    expect(book.published_year).toBe(2023);
    expect(book.summary ?? null).toBeNull();
  });

  it('strips whitespace from title, author and summary', () => {
    const book = bookCreateSchema.parse({
      title: '  Whitespace Book  ',
      author: '  Whitespace Author  ',
      published_year: 2023,
      summary: '  Whitespace summary  ',
    });

    expect(book.title).toBe('Whitespace Book');
    expect(book.author).toBe('Whitespace Author');
    expect(book.summary).toBe('Whitespace summary');
  });

  it('converts a whitespace-only summary to null', () => {
    const book = bookCreateSchema.parse({
      title: 'Book',
      author: 'Author',
      published_year: 2023,
      summary: '   ',
    });

    expect(book.summary ?? null).toBeNull();
  });

  it('rejects missing required fields', () => {
    const missingTitle = expectParseError(() =>
      bookCreateSchema.parse({ author: 'Author', published_year: 2023 }),
    );
    expect(missingTitle).toContain('title');

    const missingAuthor = expectParseError(() =>
      bookCreateSchema.parse({ title: 'Title', published_year: 2023 }),
    );
    expect(missingAuthor).toContain('author');

    const missingYear = expectParseError(() =>
      bookCreateSchema.parse({ title: 'Title', author: 'Author' }),
    );
    expect(missingYear).toContain('published_year');
  });

  it('rejects an empty title', () => {
    const emptyMsg = expectParseError(() =>
      bookCreateSchema.parse({ title: '', author: 'Author', published_year: 2023 }),
    );
    expect(emptyMsg).toContain('Title cannot be empty');

    const whitespaceMsg = expectParseError(() =>
      bookCreateSchema.parse({ title: '   ', author: 'Author', published_year: 2023 }),
    );
    expect(whitespaceMsg).toContain('Title cannot be empty');
  });

  it('rejects an empty author', () => {
    const emptyMsg = expectParseError(() =>
      bookCreateSchema.parse({ title: 'Title', author: '', published_year: 2023 }),
    );
    expect(emptyMsg).toContain('Author cannot be empty');

    const whitespaceMsg = expectParseError(() =>
      bookCreateSchema.parse({ title: 'Title', author: '   ', published_year: 2023 }),
    );
    expect(whitespaceMsg).toContain('Author cannot be empty');
  });

  it('validates the published year range', () => {
    // MIGRATION: computed dynamically to match Python's datetime.now().year
    const currentYear = new Date().getFullYear();

    const tooEarly = expectParseError(() =>
      bookCreateSchema.parse({ title: 'Title', author: 'Author', published_year: 999 }),
    );
    expect(tooEarly).toContain('Published year must be after year 1000');

    const future = expectParseError(() =>
      bookCreateSchema.parse({
        title: 'Title',
        author: 'Author',
        published_year: currentYear + 1,
      }),
    );
    expect(future).toContain('cannot be in the future');

    const bookMin = bookCreateSchema.parse({
      title: 'Title',
      author: 'Author',
      published_year: 1000,
    });
    expect(bookMin.published_year).toBe(1000);

    const bookCurrent = bookCreateSchema.parse({
      title: 'Title',
      author: 'Author',
      published_year: currentYear,
    });
    expect(bookCurrent.published_year).toBe(currentYear);
  });

  it('rejects a title longer than 255 characters', () => {
    // MIGRATION: Pydantic v1 message ('ensure this value has at most 255
    // characters') is not reproducible in Zod. Asserting on the substring
    // '255 characters' which the custom Zod message in src/schemas.ts should
    // include. Review the schema message if this fails.
    const longTitle = 'A'.repeat(256);
    const msg = expectParseError(() =>
      bookCreateSchema.parse({ title: longTitle, author: 'Author', published_year: 2023 }),
    );
    expect(msg).toContain('255 characters');
  });

  it('rejects an author longer than 255 characters', () => {
    const longAuthor = 'B'.repeat(256);
    const msg = expectParseError(() =>
      bookCreateSchema.parse({ title: 'Title', author: longAuthor, published_year: 2023 }),
    );
    expect(msg).toContain('255 characters');
  });

  it('rejects a summary longer than 2000 characters', () => {
    const longSummary = 'C'.repeat(2001);
    const msg = expectParseError(() =>
      bookCreateSchema.parse({
        title: 'Title',
        author: 'Author',
        published_year: 2023,
        summary: longSummary,
      }),
    );
    expect(msg).toContain('2000 characters');
  });
});

describe('bookUpdateSchema (BookUpdate)', () => {
  it('parses a valid partial update', () => {
    const update = bookUpdateSchema.parse({
      title: 'Updated Title',
      published_year: 2024,
    });

    expect(update.title).toBe('Updated Title');
    expect(update.author ?? null).toBeNull();
    expect(update.published_year).toBe(2024);
    expect(update.summary ?? null).toBeNull();
  });

  it('parses an empty update', () => {
    const update = bookUpdateSchema.parse({});

    expect(update.title ?? null).toBeNull();
    expect(update.author ?? null).toBeNull();
    expect(update.published_year ?? null).toBeNull();
    expect(update.summary ?? null).toBeNull();
  });

  it('applies the same field validation rules as create when values are provided', () => {
    const emptyTitle = expectParseError(() => bookUpdateSchema.parse({ title: '' }));
    expect(emptyTitle).toContain('Title cannot be empty');

    const invalidYear = expectParseError(() =>
      bookUpdateSchema.parse({ published_year: 999 }),
    );
    expect(invalidYear).toContain('Published year must be after year 1000');
  });
});

describe('bookResponseSchema (BookResponse)', () => {
  it('parses a valid book response', () => {
    const book = bookResponseSchema.parse({
      id: 1,
      title: 'Response Book',
      author: 'Response Author',
      published_year: 2023,
      summary: 'Response summary',
    });

    expect(book.id).toBe(1);
    expect(book.title).toBe('Response Book');
    expect(book.author).toBe('Response Author');
    expect(book.published_year).toBe(2023);
    expect(book.summary).toBe('Response summary');
  });

  it('requires an id field', () => {
    const msg = expectParseError(() =>
      bookResponseSchema.parse({
        title: 'Title',
        author: 'Author',
        published_year: 2023,
      }),
    );
    expect(msg).toContain('id');
  });
});
