// src/tests/test_schemas.ts
import { ZodError } from 'zod';
import {
  bookCreateSchema,
  bookUpdateSchema,
  bookResponseSchema,
  type BookCreate,
  type BookUpdate,
  type BookResponse,
} from '../schemas/book.schema';

/**
 * MIGRATION NOTE:
 * The source tests validate Pydantic schemas. In the Node.js stack we use Zod
 * schemas (`src/schemas/book.schema.ts`) as the equivalent validation layer.
 *
 * Key migration decisions:
 * - Pydantic's `BookCreate(**data)` becomes `bookCreateSchema.parse(data)`.
 * - `ValidationError` -> Zod's `ZodError`.
 * - Custom error messages ('Title cannot be empty', 'Author cannot be empty',
 *   'Published year must be after year 1000', 'cannot be in the future') are
 *   reproduced verbatim in the Zod schema using `.refine()` / custom messages
 *   so the assertions below remain literal-string matches.
 * - The Pydantic v1 length message 'ensure this value has at most N characters'
 *   does NOT exist in Zod. Zod emits 'String must contain at most N character(s)'.
 *   The assertions for length validation have been updated to match the Zod
 *   wording. If you must preserve the exact Pydantic message, override it with
 *   `.max(255, { message: 'ensure this value has at most 255 characters' })`
 *   in the schema — see MIGRATION comments in the length tests below.
 * - Whitespace stripping and empty-summary-to-None are handled with
 *   `.transform()` in the schema.
 *
 * The expected Zod schema shape (in src/schemas/book.schema.ts):
 *
 *   bookCreateSchema:
 *     title:  string, trimmed, non-empty ('Title cannot be empty'), max 255
 *     author: string, trimmed, non-empty ('Author cannot be empty'), max 255
 *     published_year: int, >= 1000 ('Published year must be after year 1000'),
 *                     <= currentYear ('...cannot be in the future')
 *     summary: optional string, trimmed, empty -> null, max 2000
 *
 *   bookUpdateSchema = bookCreateSchema.partial() with same field rules
 *   bookResponseSchema = bookCreateSchema-like + required numeric `id`
 */

/**
 * expectZodError runs a parse and returns the resulting ZodError, failing the
 * test if no error was thrown. Mirrors pytest's `pytest.raises(ValidationError)`.
 */
function expectZodError(fn: () => unknown): ZodError {
  try {
    fn();
  } catch (err) {
    expect(err).toBeInstanceOf(ZodError);
    return err as ZodError;
  }
  throw new Error('Expected ZodError to be thrown, but none was');
}

/** flatten a ZodError into a single searchable string (like str(exc_info.value)). */
function errorString(err: ZodError): string {
  return JSON.stringify(err.issues);
}

describe('BookCreate schema', () => {
  it('parses a valid book', () => {
    const book: BookCreate = bookCreateSchema.parse({
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
    const book: BookCreate = bookCreateSchema.parse({
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
    const book: BookCreate = bookCreateSchema.parse({
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
    const book: BookCreate = bookCreateSchema.parse({
      title: 'Book',
      author: 'Author',
      published_year: 2023,
      summary: '   ',
    });

    expect(book.summary ?? null).toBeNull();
  });

  it('reports validation errors for missing required fields', () => {
    const missingTitle = expectZodError(() =>
      bookCreateSchema.parse({ author: 'Author', published_year: 2023 }),
    );
    expect(errorString(missingTitle)).toContain('title');

    const missingAuthor = expectZodError(() =>
      bookCreateSchema.parse({ title: 'Title', published_year: 2023 }),
    );
    expect(errorString(missingAuthor)).toContain('author');

    const missingYear = expectZodError(() =>
      bookCreateSchema.parse({ title: 'Title', author: 'Author' }),
    );
    expect(errorString(missingYear)).toContain('published_year');
  });

  it('rejects empty title', () => {
    const emptyTitle = expectZodError(() =>
      bookCreateSchema.parse({ title: '', author: 'Author', published_year: 2023 }),
    );
    expect(errorString(emptyTitle)).toContain('Title cannot be empty');

    const whitespaceTitle = expectZodError(() =>
      bookCreateSchema.parse({ title: '   ', author: 'Author', published_year: 2023 }),
    );
    expect(errorString(whitespaceTitle)).toContain('Title cannot be empty');
  });

  it('rejects empty author', () => {
    const emptyAuthor = expectZodError(() =>
      bookCreateSchema.parse({ title: 'Title', author: '', published_year: 2023 }),
    );
    expect(errorString(emptyAuthor)).toContain('Author cannot be empty');

    const whitespaceAuthor = expectZodError(() =>
      bookCreateSchema.parse({ title: 'Title', author: '   ', published_year: 2023 }),
    );
    expect(errorString(whitespaceAuthor)).toContain('Author cannot be empty');
  });

  it('validates the published year range', () => {
    const currentYear = new Date().getFullYear();

    const tooEarly = expectZodError(() =>
      bookCreateSchema.parse({ title: 'Title', author: 'Author', published_year: 999 }),
    );
    expect(errorString(tooEarly)).toContain('Published year must be after year 1000');

    const future = expectZodError(() =>
      bookCreateSchema.parse({
        title: 'Title',
        author: 'Author',
        published_year: currentYear + 1,
      }),
    );
    expect(errorString(future)).toContain('cannot be in the future');

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

  it('validates title length (max 255)', () => {
    // MIGRATION: Pydantic v1 message was 'ensure this value has at most 255
    // characters'. Zod emits 'String must contain at most 255 character(s)'.
    // Asserting on a stable substring ('255') keeps the test robust across
    // versions. Override the schema message if exact parity is required.
    const longTitle = 'A'.repeat(256);
    const err = expectZodError(() =>
      bookCreateSchema.parse({ title: longTitle, author: 'Author', published_year: 2023 }),
    );
    expect(errorString(err)).toContain('255');
  });

  it('validates author length (max 255)', () => {
    // MIGRATION: see note on title length about message wording.
    const longAuthor = 'B'.repeat(256);
    const err = expectZodError(() =>
      bookCreateSchema.parse({ title: 'Title', author: longAuthor, published_year: 2023 }),
    );
    expect(errorString(err)).toContain('255');
  });

  it('validates summary length (max 2000)', () => {
    // MIGRATION: see note on title length about message wording.
    const longSummary = 'C'.repeat(2001);
    const err = expectZodError(() =>
      bookCreateSchema.parse({
        title: 'Title',
        author: 'Author',
        published_year: 2023,
        summary: longSummary,
      }),
    );
    expect(errorString(err)).toContain('2000');
  });
});

describe('BookUpdate schema', () => {
  it('allows a partial update', () => {
    const update: BookUpdate = bookUpdateSchema.parse({
      title: 'Updated Title',
      published_year: 2024,
    });

    expect(update.title).toBe('Updated Title');
    expect(update.author ?? null).toBeNull();
    expect(update.published_year).toBe(2024);
    expect(update.summary ?? null).toBeNull();
  });

  it('allows an empty update', () => {
    const update: BookUpdate = bookUpdateSchema.parse({});

    expect(update.title ?? null).toBeNull();
    expect(update.author ?? null).toBeNull();
    expect(update.published_year ?? null).toBeNull();
    expect(update.summary ?? null).toBeNull();
  });

  it('applies the same validation rules as create', () => {
    const emptyTitle = expectZodError(() => bookUpdateSchema.parse({ title: '' }));
    expect(errorString(emptyTitle)).toContain('Title cannot be empty');

    const invalidYear = expectZodError(() => bookUpdateSchema.parse({ published_year: 999 }));
    expect(errorString(invalidYear)).toContain('Published year must be after year 1000');
  });
});

describe('BookResponse schema', () => {
  it('parses a valid book response', () => {
    const book: BookResponse = bookResponseSchema.parse({
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

  it('requires an id', () => {
    const err = expectZodError(() =>
      bookResponseSchema.parse({
        title: 'Title',
        author: 'Author',
        published_year: 2023,
      }),
    );
    expect(errorString(err)).toContain('id');
  });
});
