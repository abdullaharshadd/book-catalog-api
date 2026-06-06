// src/tests/test_schemas.ts

import { ZodError } from 'zod';
import {
  bookCreateSchema,
  bookUpdateSchema,
  bookResponseSchema,
} from '../app/schemas';

/**
 * MIGRATION NOTES
 * ----------------
 * The source `tests/test_schemas.py` was a **pytest** unit-test suite for
 * **Pydantic** DTOs (`BookCreate`, `BookUpdate`, `BookResponse`) used by a
 * FastAPI "Book Catalog API" (despite the Django-style filename, this is NOT a
 * Django project).
 *
 * In the idiomatic Node.js/TypeScript stack we replace Pydantic schemas with
 * **Zod** schemas (`src/app/schemas.ts`) and run the suite under **Jest**.
 *
 * Mapping decisions:
 *   - `BookCreate(**data)`            -> `bookCreateSchema.parse(data)`
 *   - `pytest.raises(ValidationError)` -> expect the parse to throw a `ZodError`
 *     (asserted via a small `parseExpectingError` helper).
 *   - Pydantic's automatic whitespace stripping + "empty summary becomes None"
 *     is reproduced by Zod `.transform()` logic inside `bookCreateSchema`.
 *   - `published_year` dynamic upper bound (`datetime.now().year`) is replicated
 *     with a Zod `.refine()` that reads `new Date().getFullYear()` at validation
 *     time.
 *   - `BookUpdate` (partial, all-optional but still validated) maps to a Zod
 *     schema where every field is `.optional()` but the per-field refinements
 *     still run when a value is supplied.
 *
 * IMPORTANT — error message wording:
 *   The custom business messages are preserved verbatim:
 *     - 'Title cannot be empty'
 *     - 'Author cannot be empty'
 *     - 'Published year must be after year 1000'
 *     - 'cannot be in the future'
 *   The Pydantic v1 length wording ('ensure this value has at most N
 *   characters') does NOT exist in Zod. Zod's default max-length message is
 *   'String must contain at most N character(s)'. Rather than asserting on
 *   framework-specific wording, the length tests assert that the offending
 *   field path is flagged as `too_big`. This keeps the assertions robust and
 *   tied to business intent rather than library copy.
 */

/**
 * parseExpectingError runs a Zod parse that is expected to fail and returns the
 * resulting ZodError. Fails the test if parsing unexpectedly succeeds.
 */
function parseExpectingError(fn: () => unknown): ZodError {
  try {
    fn();
  } catch (err) {
    if (err instanceof ZodError) {
      return err;
    }
    throw err;
  }
  throw new Error('Expected validation to throw a ZodError, but it succeeded');
}

/** Flatten all ZodError messages into a single searchable string. */
function errorText(err: ZodError): string {
  return err.issues.map((i) => i.message).join(' | ');
}

/** True if any issue path includes the given field name. */
function hasFieldIssue(err: ZodError, field: string): boolean {
  return err.issues.some((i) => i.path.includes(field));
}

/** True if any issue for the given field is a 'too_big' (max length) issue. */
function hasTooBigIssue(err: ZodError, field: string): boolean {
  return err.issues.some(
    (i) => i.path.includes(field) && i.code === 'too_big',
  );
}

describe('BookCreate schema', () => {
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

  it('converts an empty/whitespace-only summary to null', () => {
    const book = bookCreateSchema.parse({
      title: 'Book',
      author: 'Author',
      published_year: 2023,
      summary: '   ',
    });

    expect(book.summary ?? null).toBeNull();
  });

  it('reports validation errors for missing required fields', () => {
    const missingTitle = parseExpectingError(() =>
      bookCreateSchema.parse({ author: 'Author', published_year: 2023 }),
    );
    expect(hasFieldIssue(missingTitle, 'title')).toBe(true);

    const missingAuthor = parseExpectingError(() =>
      bookCreateSchema.parse({ title: 'Title', published_year: 2023 }),
    );
    expect(hasFieldIssue(missingAuthor, 'author')).toBe(true);

    const missingYear = parseExpectingError(() =>
      bookCreateSchema.parse({ title: 'Title', author: 'Author' }),
    );
    expect(hasFieldIssue(missingYear, 'published_year')).toBe(true);
  });

  it('rejects an empty title', () => {
    const emptyTitle = parseExpectingError(() =>
      bookCreateSchema.parse({
        title: '',
        author: 'Author',
        published_year: 2023,
      }),
    );
    expect(errorText(emptyTitle)).toContain('Title cannot be empty');

    const whitespaceTitle = parseExpectingError(() =>
      bookCreateSchema.parse({
        title: '   ',
        author: 'Author',
        published_year: 2023,
      }),
    );
    expect(errorText(whitespaceTitle)).toContain('Title cannot be empty');
  });

  it('rejects an empty author', () => {
    const emptyAuthor = parseExpectingError(() =>
      bookCreateSchema.parse({
        title: 'Title',
        author: '',
        published_year: 2023,
      }),
    );
    expect(errorText(emptyAuthor)).toContain('Author cannot be empty');

    const whitespaceAuthor = parseExpectingError(() =>
      bookCreateSchema.parse({
        title: 'Title',
        author: '   ',
        published_year: 2023,
      }),
    );
    expect(errorText(whitespaceAuthor)).toContain('Author cannot be empty');
  });

  it('validates the published year range', () => {
    const currentYear = new Date().getFullYear();

    const tooEarly = parseExpectingError(() =>
      bookCreateSchema.parse({
        title: 'Title',
        author: 'Author',
        published_year: 999,
      }),
    );
    expect(errorText(tooEarly)).toContain(
      'Published year must be after year 1000',
    );

    const future = parseExpectingError(() =>
      bookCreateSchema.parse({
        title: 'Title',
        author: 'Author',
        published_year: currentYear + 1,
      }),
    );
    expect(errorText(future)).toContain('cannot be in the future');

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
    const longTitle = 'A'.repeat(256);
    const err = parseExpectingError(() =>
      bookCreateSchema.parse({
        title: longTitle,
        author: 'Author',
        published_year: 2023,
      }),
    );
    // MIGRATION: Pydantic v1 wording 'ensure this value has at most 255
    // characters' has no Zod equivalent; assert on the structured too_big issue.
    expect(hasTooBigIssue(err, 'title')).toBe(true);
  });

  it('rejects an author longer than 255 characters', () => {
    const longAuthor = 'B'.repeat(256);
    const err = parseExpectingError(() =>
      bookCreateSchema.parse({
        title: 'Title',
        author: longAuthor,
        published_year: 2023,
      }),
    );
    expect(hasTooBigIssue(err, 'author')).toBe(true);
  });

  it('rejects a summary longer than 2000 characters', () => {
    const longSummary = 'C'.repeat(2001);
    const err = parseExpectingError(() =>
      bookCreateSchema.parse({
        title: 'Title',
        author: 'Author',
        published_year: 2023,
        summary: longSummary,
      }),
    );
    expect(hasTooBigIssue(err, 'summary')).toBe(true);
  });
});

describe('BookUpdate schema', () => {
  it('accepts a valid partial update', () => {
    const update = bookUpdateSchema.parse({
      title: 'Updated Title',
      published_year: 2024,
    });

    expect(update.title).toBe('Updated Title');
    expect(update.author ?? null).toBeNull();
    expect(update.published_year).toBe(2024);
    expect(update.summary ?? null).toBeNull();
  });

  it('accepts an empty update with no fields', () => {
    const update = bookUpdateSchema.parse({});

    expect(update.title ?? null).toBeNull();
    expect(update.author ?? null).toBeNull();
    expect(update.published_year ?? null).toBeNull();
    expect(update.summary ?? null).toBeNull();
  });

  it('applies the same validation rules as create', () => {
    const emptyTitle = parseExpectingError(() =>
      bookUpdateSchema.parse({ title: '' }),
    );
    expect(errorText(emptyTitle)).toContain('Title cannot be empty');

    const invalidYear = parseExpectingError(() =>
      bookUpdateSchema.parse({ published_year: 999 }),
    );
    expect(errorText(invalidYear)).toContain(
      'Published year must be after year 1000',
    );
  });
});

describe('BookResponse schema', () => {
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

  it('requires the id field', () => {
    const err = parseExpectingError(() =>
      bookResponseSchema.parse({
        title: 'Title',
        author: 'Author',
        published_year: 2023,
      }),
    );
    expect(hasFieldIssue(err, 'id')).toBe(true);
  });
});
