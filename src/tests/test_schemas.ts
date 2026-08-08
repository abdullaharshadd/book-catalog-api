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

describe('BookCreate schema', () => {
  it('accepts a valid book', () => {
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

  it('accepts a book without a summary (optional)', () => {
    const result = bookCreateSchema.parse({
      title: 'Book Without Summary',
      author: 'Author',
      published_year: 2023,
    });

    expect(result.title).toBe('Book Without Summary');
    expect(result.author).toBe('Author');
    expect(result.published_year).toBe(2023);
    expect(result.summary ?? null).toBeNull();
  });

  it('strips whitespace from title, author, and summary', () => {
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

  it('rejects missing required fields', () => {
    const missingTitle = bookCreateSchema.safeParse({
      author: 'Author',
      published_year: 2023,
    });
    expect(missingTitle.success).toBe(false);
    if (!missingTitle.success) {
      expect(errorText(missingTitle.error)).toContain('title');
    }

    const missingAuthor = bookCreateSchema.safeParse({
      title: 'Title',
      published_year: 2023,
    });
    expect(missingAuthor.success).toBe(false);
    if (!missingAuthor.success) {
      expect(errorText(missingAuthor.error)).toContain('author');
    }

    const missingYear = bookCreateSchema.safeParse({
      title: 'Title',
      author: 'Author',
    });
    expect(missingYear.success).toBe(false);
    if (!missingYear.success) {
      expect(errorText(missingYear.error)).toContain('published_year');
    }
  });

  it('rejects an empty title', () => {
    const empty = bookCreateSchema.safeParse({
      title: '',
      author: 'Author',
      published_year: 2023,
    });
    expect(empty.success).toBe(false);
    if (!empty.success) {
      expect(errorText(empty.error)).toContain('Title cannot be empty');
    }

    const whitespace = bookCreateSchema.safeParse({
      title: '   ',
      author: 'Author',
      published_year: 2023,
    });
    expect(whitespace.success).toBe(false);
    if (!whitespace.success) {
      expect(errorText(whitespace.error)).toContain('Title cannot be empty');
    }
  });

  it('rejects an empty author', () => {
    const empty = bookCreateSchema.safeParse({
      title: 'Title',
      author: '',
      published_year: 2023,
    });
    expect(empty.success).toBe(false);
    if (!empty.success) {
      expect(errorText(empty.error)).toContain('Author cannot be empty');
    }

    const whitespace = bookCreateSchema.safeParse({
      title: 'Title',
      author: '   ',
      published_year: 2023,
    });
    expect(whitespace.success).toBe(false);
    if (!whitespace.success) {
      expect(errorText(whitespace.error)).toContain('Author cannot be empty');
    }
  });

  it('validates the published year range', () => {
    const currentYear = new Date().getFullYear();

    const tooEarly = bookCreateSchema.safeParse({
      title: 'Title',
      author: 'Author',
      published_year: 999,
    });
    expect(tooEarly.success).toBe(false);
    if (!tooEarly.success) {
      expect(errorText(tooEarly.error)).toContain(
        'Published year must be after year 1000',
      );
    }

    const future = bookCreateSchema.safeParse({
      title: 'Title',
      author: 'Author',
      published_year: currentYear + 1,
    });
    expect(future.success).toBe(false);
    if (!future.success) {
      expect(errorText(future.error)).toContain('cannot be in the future');
    }

    const min = bookCreateSchema.parse({
      title: 'Title',
      author: 'Author',
      published_year: 1000,
    });
    expect(min.published_year).toBe(1000);

    const current = bookCreateSchema.parse({
      title: 'Title',
      author: 'Author',
      published_year: currentYear,
    });
    expect(current.published_year).toBe(currentYear);
  });

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
});

describe('BookUpdate schema', () => {
  it('accepts a partial update', () => {
    const result = bookUpdateSchema.parse({
      title: 'Updated Title',
      published_year: 2024,
    });

    expect(result.title).toBe('Updated Title');
    expect(result.author ?? null).toBeNull();
    expect(result.published_year).toBe(2024);
    expect(result.summary ?? null).toBeNull();
  });

  it('accepts an empty update (no fields provided)', () => {
    const result = bookUpdateSchema.parse({});

    expect(result.title ?? null).toBeNull();
    expect(result.author ?? null).toBeNull();
    expect(result.published_year ?? null).toBeNull();
    expect(result.summary ?? null).toBeNull();
  });

  it('applies the same field-level validation as create', () => {
    const emptyTitle = bookUpdateSchema.safeParse({ title: '' });
    expect(emptyTitle.success).toBe(false);
    if (!emptyTitle.success) {
      expect(errorText(emptyTitle.error)).toContain('Title cannot be empty');
    }

    const invalidYear = bookUpdateSchema.safeParse({ published_year: 999 });
    expect(invalidYear.success).toBe(false);
    if (!invalidYear.success) {
      expect(errorText(invalidYear.error)).toContain(
        'Published year must be after year 1000',
      );
    }
  });
});

describe('BookResponse schema', () => {
  it('accepts a valid book response', () => {
    const data: BookResponse = {
      id: 1,
      title: 'Response Book',
      author: 'Response Author',
      published_year: 2023,
      summary: 'Response summary',
    };

    const result = bookResponseSchema.parse(data);

    expect(result.id).toBe(1);
    expect(result.title).toBe('Response Book');
    expect(result.author).toBe('Response Author');
    expect(result.published_year).toBe(2023);
    expect(result.summary).toBe('Response summary');
  });

  it('requires an id', () => {
    const result = bookResponseSchema.safeParse({
      title: 'Title',
      author: 'Author',
      published_year: 2023,
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain('id');
    }
  });
});
