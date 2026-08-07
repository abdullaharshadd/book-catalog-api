```typescript
// src/tests/test_schemas.ts

/**
 * Unit tests for the Book schemas (BookCreate / BookUpdate / BookResponse).
 *
 * MIGRATION_NOTE: The Python source used Pydantic models and asserted on
 * `ValidationError` objects via `pytest.raises`. In the Zod world:
 *   - `bookCreateSchema` / `bookUpdateSchema` (src/app/schemas.ts) validate
 *     input and, on failure, throw a `ZodError` (via `.parse`) or return a
 *     discriminated result (via `.safeParse`).
 *   - Instead of `with pytest.raises(ValidationError)` we call `.safeParse`
 *     and assert `result.success === false`, then grep the aggregated issue
 *     messages for the exact literal strings the source asserted on. These
 *     literals directly gate the Zod `.refine()` messages and the 422
 *     response shape, so they are preserved verbatim.
 *
 * MIGRATION_NOTE: Pydantic's built-in length-limit message
 *   "ensure this value has at most N characters"
 * is not emitted by Zod out of the box. The migrated schemas in
 * src/app/schemas.ts are expected to attach that exact message to their
 * `.max(...)` constraints so the assertions below match. If the schemas emit
 * a different literal, update the `.max()` message in schemas.ts (not the
 * assertion here) — the assertion mirrors the original contract.
 *
 * MIGRATION_NOTE: Pydantic strips whitespace on `str` fields and coerces an
 * all-whitespace summary to `null`. The migrated Zod schemas perform the same
 * via `.transform(...)` / `.refine(...)`. These tests assert that behavior.
 */

import {
  bookCreateSchema,
  bookUpdateSchema,
  type BookResponse,
} from '../app/schemas';
import { z } from 'zod';

/**
 * Collect all Zod issue messages (and the path segments) into a single
 * searchable string, mirroring `str(exc_info.value)` in pytest which contains
 * both the field names and the error messages.
 */
function errorText(error: z.ZodError): string {
  return error.issues
    .map((issue) => `${issue.path.join('.')}: ${issue.message}`)
    .join('\n');
}

/**
 * A minimal BookResponse schema for validating response-shape assertions.
 *
 * MIGRATION_NOTE: `BookResponse` in src/app/schemas.ts is a TypeScript
 * *interface* plus a `toBookResponse` mapper — there is no runtime validator
 * exported for it (responses are constructed from trusted DB rows, not parsed
 * from untrusted input). The Python test validated that `id` is required on
 * `BookResponse`. To preserve that contract as an executable test we declare a
 * local Zod schema matching the `BookResponse` interface and assert against
 * it. Keep this in sync with the interface in schemas.ts.
 */
const bookResponseSchema = z.object({
  id: z.number().int(),
  title: z.string(),
  author: z.string(),
  published_year: z.number().int(),
  summary: z.string().nullable().optional(),
});

// ---------------------------------------------------------------------------
// BookCreate schema
// ---------------------------------------------------------------------------

describe('BookCreate schema', () => {
  // -------------------------------------------------------------------------
  // Happy-path / success cases
  // -------------------------------------------------------------------------

  it('accepts a valid book with all fields including summary', () => {
    const result = bookCreateSchema.parse({
      title: 'Valid Book',
      author: 'Valid Author',
      published_year: 2023,
      summary: 'A valid book summary',
    });

    expect(result.title).toBe('Valid Book');
    expect(result.author).toBe('Valid Author');
    expect(result.published_year).toBe(2023);
    expect(result.summary).toBe('A valid book summary');
  });

  it('accepts a book without a summary (optional field)', () => {
    const result = bookCreateSchema.parse({
      title: 'Book Without Summary',
      author: 'Author',
      published_year: 2023,
    });

    expect(result.title).toBe('Book Without Summary');
    expect(result.author).toBe('Author');
    expect(result.published_year).toBe(2023);
    // summary absent → normalised to null / undefined
    expect(result.summary ?? null).toBeNull();
  });

  it('strips leading and trailing whitespace from title, author, and summary', () => {
    const result = bookCreateSchema.parse({
      title: '  Whitespace Book  ',
      author: '  Whitespace Author  ',
      published_year: 2023,
      summary: '  Whitespace summary  ',
    });

    expect(result.title).toBe('Whitespace Book');
    expect(result.author).toBe('Whitespace Author');
    expect(result.summary).toBe('Whitespace summary');
  });

  it('coerces an all-whitespace summary to null', () => {
    const result = bookCreateSchema.parse({
      title: 'Book',
      author: 'Author',
      published_year: 2023,
      summary: '   ',
    });

    expect(result.summary ?? null).toBeNull();
  });

  it('accepts published_year exactly equal to 1000 (minimum boundary)', () => {
    const result = bookCreateSchema.parse({
      title: 'Title',
      author: 'Author',
      published_year: 1000,
    });

    expect(result.published_year).toBe(1000);
  });

  it('accepts published_year exactly equal to the current year (maximum boundary)', () => {
    const currentYear = new Date().getFullYear();

    const result = bookCreateSchema.parse({
      title: 'Title',
      author: 'Author',
      published_year: currentYear,
    });

    expect(result.published_year).toBe(currentYear);
  });

  // -------------------------------------------------------------------------
  // Missing required fields
  // -------------------------------------------------------------------------

  it('rejects when title field is missing', () => {
    const result = bookCreateSchema.safeParse({
      author: 'Author',
      published_year: 2023,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain('title');
    }
  });

  it('rejects when author field is missing', () => {
    const result = bookCreateSchema.safeParse({
      title: 'Title',
      published_year: 2023,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain('author');
    }
  });

  it('rejects when published_year field is missing', () => {
    const result = bookCreateSchema.safeParse({
      title: 'Title',
      author: 'Author',
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain('published_year');
    }
  });

  // -------------------------------------------------------------------------
  // Empty / whitespace-only title
  // -------------------------------------------------------------------------

  it('rejects an empty title with "Title cannot be empty"', () => {
    const result = bookCreateSchema.safeParse({
      title: '',
      author: 'Author',
      published_year: 2023,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain('Title cannot be empty');
    }
  });

  it('rejects a whitespace-only title with "Title cannot be empty"', () => {
    const result = bookCreateSchema.safeParse({
      title: '   ',
      author: 'Author',
      published_year: 2023,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain('Title cannot be empty');
    }
  });

  // -------------------------------------------------------------------------
  // Empty / whitespace-only author
  // -------------------------------------------------------------------------

  it('rejects an empty author with "Author cannot be empty"', () => {
    const result = bookCreateSchema.safeParse({
      title: 'Title',
      author: '',
      published_year: 2023,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain('Author cannot be empty');
    }
  });

  it('rejects a whitespace-only author with "Author cannot be empty"', () => {
    const result = bookCreateSchema.safeParse({
      title: 'Title',
      author: '   ',
      published_year: 2023,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain('Author cannot be empty');
    }
  });

  // -------------------------------------------------------------------------
  // published_year range validation
  // -------------------------------------------------------------------------

  it('rejects published_year below 1000 (e.g. 999) with "Published year must be after year 1000"', () => {
    const result = bookCreateSchema.safeParse({
      title: 'Title',
      author: 'Author',
      published_year: 999,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain(
        'Published year must be after year 1000',
      );
    }
  });

  it('rejects published_year greater than the current year with "cannot be in the future"', () => {
    const currentYear = new Date().getFullYear();

    const result = bookCreateSchema.safeParse({
      title: 'Title',
      author: 'Author',
      published_year: currentYear + 1,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain('cannot be in the future');
    }
  });

  // -------------------------------------------------------------------------
  // Max-length constraints
  // -------------------------------------------------------------------------

  it('rejects a title longer than 255 characters', () => {
    const longTitle = 'A'.repeat(256);
    const result = bookCreateSchema.safeParse({
      title: longTitle,
      author: 'Author',
      published_year: 2023,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain(
        'ensure this value has at most 255 characters',
      );
    }
  });

  it('accepts a title of exactly 255 characters (boundary)', () => {
    const maxTitle = 'A'.repeat(255);
    const result = bookCreateSchema.parse({
      title: maxTitle,
      author: 'Author',
      published_year: 2023,
    });

    expect(result.title).toBe(maxTitle);
  });

  it('rejects an author longer than 255 characters', () => {
    const longAuthor = 'B'.repeat(256);
    const result = bookCreateSchema.safeParse({
      title: 'Title',
      author: longAuthor,
      published_year: 2023,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain(
        'ensure this value has at most 255 characters',
      );
    }
  });

  it('accepts an author of exactly 255 characters (boundary)', () => {
    const maxAuthor = 'B'.repeat(255);
    const result = bookCreateSchema.parse({
      title: 'Title',
      author: maxAuthor,
      published_year: 2023,
    });

    expect(result.author).toBe(maxAuthor);
  });

  it('rejects a summary longer than 2000 characters', () => {
    const longSummary = 'C'.repeat(2001);
    const result = bookCreateSchema.safeParse({
      title: 'Title',
      author: 'Author',
      published_year: 2023,
      summary: longSummary,
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain(
        'ensure this value has at most 2000 characters',
      );
    }
  });

  it('accepts a summary of exactly 2000 characters (boundary)', () => {
    const maxSummary = 'C'.repeat(2000);
    const result = bookCreateSchema.parse({
      title: 'Title',
      author: 'Author',
      published_year: 2023,
      summary: maxSummary,
    });

    expect(result.summary).toBe(maxSummary);
  });

  // -------------------------------------------------------------------------
  // Invariant: title/author/summary always stripped
  // -------------------------------------------------------------------------

  it('invariant: title is always stripped of leading/trailing whitespace', () => {
    const result = bookCreateSchema.parse({
      title: '\t  My Book \n',
      author: 'Author',
      published_year: 2023,
    });

    expect(result.title).toBe('My Book');
  });

  it('invariant: author is always stripped of leading/trailing whitespace', () => {
    const result = bookCreateSchema.parse({
      title: 'Title',
      author: '\t  My Author \n',
      published_year: 2023,
    });

    expect(result.author).toBe('My Author');
  });

  it('invariant: summary is either null or a non-empty stripped string', () => {
    // Non-empty → stripped and returned
    const withSummary = bookCreateSchema.parse({
      title: 'Title',
      author: 'Author',
      published_year: 2023,
      summary: '  valid summary  ',
    });
    expect(withSummary.summary).toBe('valid summary');

    // Whitespace-only → null
    const withWhitespaceSummary = bookCreateSchema.parse({
      title: 'Title',
      author: 'Author',
      published_year: 2023,
      summary: '   ',
    });
    expect(withWhitespaceSummary.summary ?? null).toBeNull();

    // Absent → null
    const withoutSummary = bookCreateSchema.parse({
      title: 'Title',
      author: 'Author',
      published_year: 2023,
    });
    expect(withoutSummary.summary ?? null).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// BookUpdate schema
// ---------------------------------------------------------------------------

describe('BookUpdate schema', () => {
  // -------------------------------------------------------------------------
  // Happy-path / success cases
  // -------------------------------------------------------------------------

  it('accepts a partial update with only some fields provided', () => {
    const result = bookUpdateSchema.parse({
      title: 'Updated Title',
      published_year: 2024,
    });

    expect(result.title).toBe('Updated Title');
    expect(result.author ?? null).toBeNull();
    expect(result.published_year).toBe(2024);
    expect(result.summary ?? null).toBeNull();
  });

  it('accepts an empty update object (no fields provided)', () => {
    const result = bookUpdateSchema.parse({});

    expect(result.title ?? null).toBeNull();
    expect(result.author ?? null).toBeNull();
    expect(result.published_year ?? null).toBeNull();
    expect(result.summary ?? null).toBeNull();
  });

  it('accepts a full update with all fields provided', () => {
    const currentYear = new Date().getFullYear();

    const result = bookUpdateSchema.parse({
      title: 'Full Update',
      author: 'Full Author',
      published_year: currentYear,
      summary: 'Full summary',
    });

    expect(result.title).toB