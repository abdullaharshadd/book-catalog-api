import { z } from 'zod';
import {
  bookCreateSchema,
  bookUpdateSchema,
  bookResponseSchema,
  type BookCreate,
  type BookUpdate,
  type BookResponse,
} from '../schemas/book.schema';

/**
 * Tests for the Zod schemas (BookCreate, BookUpdate, BookResponse) used for
 * request/response validation.
 *
 * MIGRATION: The original Python tests used Pydantic, which raises a
 * `ValidationError` when the model is *constructed*. The idiomatic Node.js
 * equivalent uses Zod, where validation happens via `schema.safeParse(data)`.
 * Instead of `pytest.raises(ValidationError)`, we assert that
 * `result.success === false` and inspect the formatted error messages.
 *
 * The schemas referenced here are expected to live in
 * `src/schemas/book.schema.ts` and replicate the Pydantic behaviour:
 *   - title/author/summary trimmed; empty summary -> null
 *   - title cannot be empty ("Title cannot be empty")
 *   - author cannot be empty ("Author cannot be empty")
 *   - published_year >= 1000 ("Published year must be after year 1000")
 *   - published_year <= current year ("...cannot be in the future")
 *   - title/author max 255 chars, summary max 2000 chars
 */

/** Collect all error messages from a Zod safeParse failure into a single string. */
function errorText(error: z.ZodError): string {
  return error.issues.map((issue) => `${issue.path.join('.')}: ${issue.message}`).join(' | ');
}

describe('BookCreate schema', () => {
  it('accepts a valid book', () => {
    const result = bookCreateSchema.safeParse({
      title: 'Valid Book',
      author: 'Valid Author',
      published_year: 2023,
      summary: 'A valid book summary',
    });

    expect(result.success).toBe(true);
    if (result.success) {
      const book: BookCreate = result.data;
      expect(book.title).toBe('Valid Book');
      expect(book.author).toBe('Valid Author');
      expect(book.published_year).toBe(2023);
      expect(book.summary).toBe('A valid book summary');
    }
  });

  it('accepts a book without a summary (optional)', () => {
    const result = bookCreateSchema.safeParse({
      title: 'Book Without Summary',
      author: 'Author',
      published_year: 2023,
    });

    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.title).toBe('Book Without Summary');
      expect(result.data.author).toBe('Author');
      expect(result.data.published_year).toBe(2023);
      expect(result.data.summary ?? null).toBeNull();
    }
  });

  it('strips whitespace from title, author and summary', () => {
    const result = bookCreateSchema.safeParse({
      title: '  Whitespace Book  ',
      author: '  Whitespace Author  ',
      published_year: 2023,
      summary: '  Whitespace summary  ',
    });

    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.title).toBe('Whitespace Book');
      expect(result.data.author).toBe('Whitespace Author');
      expect(result.data.summary).toBe('Whitespace summary');
    }
  });

  it('converts an empty (whitespace-only) summary to null', () => {
    const result = bookCreateSchema.safeParse({
      title: 'Book',
      author: 'Author',
      published_year: 2023,
      summary: '   ',
    });

    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.summary ?? null).toBeNull();
    }
  });

  it('reports validation errors for missing required fields', () => {
    const missingTitle = bookCreateSchema.safeParse({ author: 'Author', published_year: 2023 });
    expect(missingTitle.success).toBe(false);
    if (!missingTitle.success) {
      expect(errorText(missingTitle.error)).toContain('title');
    }

    const missingAuthor = bookCreateSchema.safeParse({ title: 'Title', published_year: 2023 });
    expect(missingAuthor.success).toBe(false);
    if (!missingAuthor.success) {
      expect(errorText(missingAuthor.error)).toContain('author');
    }

    const missingYear = bookCreateSchema.safeParse({ title: 'Title', author: 'Author' });
    expect(missingYear.success).toBe(false);
    if (!missingYear.success) {
      expect(errorText(missingYear.error)).toContain('published_year');
    }
  });

  it('rejects an empty title', () => {
    const emptyString = bookCreateSchema.safeParse({ title: '', author: 'Author', published_year: 2023 });
    expect(emptyString.success).toBe(false);
    if (!emptyString.success) {
      expect(errorText(emptyString.error)).toContain('Title cannot be empty');
    }

    const whitespace = bookCreateSchema.safeParse({ title: '   ', author: 'Author', published_year: 2023 });
    expect(whitespace.success).toBe(false);
    if (!whitespace.success) {
      expect(errorText(whitespace.error)).toContain('Title cannot be empty');
    }
  });

  it('rejects an empty author', () => {
    const emptyString = bookCreateSchema.safeParse({ title: 'Title', author: '', published_year: 2023 });
    expect(emptyString.success).toBe(false);
    if (!emptyString.success) {
      expect(errorText(emptyString.error)).toContain('Author cannot be empty');
    }

    const whitespace = bookCreateSchema.safeParse({ title: 'Title', author: '   ', published_year: 2023 });
    expect(whitespace.success).toBe(false);
    if (!whitespace.success) {
      expect(errorText(whitespace.error)).toContain('Author cannot be empty');
    }
  });

  it('validates the published year bounds', () => {
    const currentYear = new Date().getFullYear();

    const tooEarly = bookCreateSchema.safeParse({ title: 'Title', author: 'Author', published_year: 999 });
    expect(tooEarly.success).toBe(false);
    if (!tooEarly.success) {
      expect(errorText(tooEarly.error)).toContain('Published year must be after year 1000');
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

    const min = bookCreateSchema.safeParse({ title: 'Title', author: 'Author', published_year: 1000 });
    expect(min.success).toBe(true);
    if (min.success) {
      expect(min.data.published_year).toBe(1000);
    }

    const current = bookCreateSchema.safeParse({
      title: 'Title',
      author: 'Author',
      published_year: currentYear,
    });
    expect(current.success).toBe(true);
    if (current.success) {
      expect(current.data.published_year).toBe(currentYear);
    }
  });

  it('rejects a title longer than 255 characters', () => {
    const longTitle = 'A'.repeat(256);
    const result = bookCreateSchema.safeParse({ title: longTitle, author: 'Author', published_year: 2023 });
    expect(result.success).toBe(false);
    if (!result.success) {
      // MIGRATION: Pydantic message was 'ensure this value has at most 255 characters'.
      // Zod's default max() message is 'String must contain at most 255 character(s)'.
      // We assert on the numeric bound which is stable across both wordings.
      expect(errorText(result.error)).toContain('255');
    }
  });

  it('rejects an author longer than 255 characters', () => {
    const longAuthor = 'B'.repeat(256);
    const result = bookCreateSchema.safeParse({ title: 'Title', author: longAuthor, published_year: 2023 });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(errorText(result.error)).toContain('255');
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
      expect(errorText(result.error)).toContain('2000');
    }
  });
});

describe('BookUpdate schema', () => {
  it('accepts a partial update', () => {
    const result = bookUpdateSchema.safeParse({
      title: 'Updated Title',
      published_year: 2024,
    });

    expect(result.success).toBe(true);
    if (result.success) {
      const update: BookUpdate = result.data;
      expect(update.title).toBe('Updated Title');
      expect(update.author ?? null).toBeNull();
      expect(update.published_year).toBe(2024);
      expect(update.summary ?? null).toBeNull();
    }
  });

  it('accepts an empty update', () => {
    const result = bookUpdateSchema.safeParse({});

    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.title ?? null).toBeNull();
      expect(result.data.author ?? null).toBeNull();
      expect(result.data.published_year ?? null).toBeNull();
      expect(result.data.summary ?? null).toBeNull();
    }
  });

  it('applies the same validation rules as create', () => {
    const emptyTitle = bookUpdateSchema.safeParse({ title: '' });
    expect(emptyTitle.success).toBe(false);
    if (!emptyTitle.success) {
      expect(errorText(emptyTitle.error)).toContain('Title cannot be empty');
    }

    const invalidYear = bookUpdateSchema.safeParse({ published_year: 999 });
    expect(invalidYear.success).toBe(false);
    if (!invalidYear.success) {
      expect(errorText(invalidYear.error)).toContain('Published year must be after year 1000');
    }
  });
});

describe('BookResponse schema', () => {
  it('accepts a valid book response', () => {
    const result = bookResponseSchema.safeParse({
      id: 1,
      title: 'Response Book',
      author: 'Response Author',
      published_year: 2023,
      summary: 'Response summary',
    });

    expect(result.success).toBe(true);
    if (result.success) {
      const book: BookResponse = result.data;
      expect(book.id).toBe(1);
      expect(book.title).toBe('Response Book');
      expect(book.author).toBe('Response Author');
      expect(book.published_year).toBe(2023);
      expect(book.summary).toBe('Response summary');
    }
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
