// src/app/schemas.ts
import { z } from 'zod';
import type { Book } from '@prisma/client';

/**
 * MIGRATION NOTES
 * ----------------
 * The source `app/schemas.py` defined Pydantic v2 DTOs (`BookCreate`,
 * `BookUpdate`, `BookResponse`) with field-level validators. In the
 * Node.js/TypeScript stack we use **Zod** for request validation and a plain
 * serialization helper for responses.
 *
 * Key mappings:
 *   - Pydantic `field_validator` -> Zod `.refine` / `.transform` chains.
 *   - `ConfigDict(str_strip_whitespace=True)` -> explicit `.trim()` on string
 *     schemas. Zod's `.trim()` normalizes whitespace before validation.
 *   - `ConfigDict(from_attributes=True)` (ORM mode) -> `toBookResponse`, which
 *     reads from a Prisma `Book` model instance.
 *   - Empty/whitespace-only `summary` -> normalized to `null` (Prisma uses
 *     `null` rather than Python's `None`).
 *   - `BookUpdate` fields are all optional (partial update semantics); absent
 *     fields are simply omitted and not validated for emptiness.
 *   - `published_year` upper bound uses the current year at validation time
 *     (matching `datetime.now().year`). Note: the source used local time via
 *     `datetime.now()`, NOT UTC; we preserve that with `new Date().getFullYear()`.
 */

const MAX_TITLE_LENGTH = 255;
const MAX_AUTHOR_LENGTH = 255;
const MAX_SUMMARY_LENGTH = 2000;
const MIN_PUBLISHED_YEAR = 1000;

/**
 * currentYear returns the current year using local time, matching the source's
 * `datetime.now().year` behavior.
 */
function currentYear(): number {
  return new Date().getFullYear();
}

/**
 * requiredTrimmedString builds a Zod schema for a required, non-empty string
 * field with a maximum length. Whitespace is stripped before validation,
 * replicating Pydantic's `str_strip_whitespace=True`.
 */
function requiredTrimmedString(
  emptyMessage: string,
  maxLength: number,
): z.ZodEffects<z.ZodString, string, string> {
  // Validate length against the RAW value first (Pydantic checked len(v)
  // before stripping for the length rule), then strip for the empty check
  // and the returned value.
  return z
    .string()
    .refine((v) => v.length <= maxLength, {
      message: `ensure this value has at most ${maxLength} characters`,
    })
    .refine((v) => v.trim().length > 0, { message: emptyMessage })
    .transform((v) => v.trim());
}

/**
 * publishedYearSchema validates a publication year: it must be an integer no
 * earlier than year 1000 and not in the future (relative to the local current
 * year). Mirrors the source `validate_published_year`.
 */
const publishedYearSchema = z
  .number()
  .int()
  .refine((v) => v >= MIN_PUBLISHED_YEAR, {
    message: 'Published year must be after year 1000',
  })
  .refine((v) => v <= currentYear(), {
    message: `Published year cannot be in the future (current year: ${currentYear()})`,
  });

/**
 * summarySchema normalizes a summary: undefined/null stays null, empty or
 * whitespace-only strings become null, and overly long strings are rejected.
 * Mirrors the source `validate_summary`.
 */
const summarySchema = z
  .string()
  .max(MAX_SUMMARY_LENGTH, {
    message: `ensure this value has at most ${MAX_SUMMARY_LENGTH} characters`,
  })
  .nullish()
  .transform((v) => {
    if (v === undefined || v === null) {
      return null;
    }
    const trimmed = v.trim();
    return trimmed.length > 0 ? trimmed : null;
  });

/**
 * bookCreateSchema validates the payload for creating a Book.
 *
 * Replaces the Pydantic `BookCreate` model. All required fields are validated
 * and normalized; `summary` is optional and coerced to `null` when empty.
 */
export const bookCreateSchema = z.object({
  title: requiredTrimmedString('Title cannot be empty', MAX_TITLE_LENGTH),
  author: requiredTrimmedString('Author cannot be empty', MAX_AUTHOR_LENGTH),
  published_year: publishedYearSchema,
  summary: summarySchema,
});

/**
 * BookCreateInput is the validated, normalized shape produced by
 * `bookCreateSchema`.
 */
export type BookCreateInput = z.infer<typeof bookCreateSchema>;

/**
 * bookUpdateSchema validates the payload for partially updating a Book.
 *
 * Replaces the Pydantic `BookUpdate` model. Every field is optional; absent
 * fields are omitted and not validated for emptiness (partial-update
 * semantics). Provided fields follow the same rules as `bookCreateSchema`.
 */
export const bookUpdateSchema = z.object({
  title: requiredTrimmedString('Title cannot be empty', MAX_TITLE_LENGTH)
    .optional(),
  author: requiredTrimmedString('Author cannot be empty', MAX_AUTHOR_LENGTH)
    .optional(),
  published_year: publishedYearSchema.optional(),
  summary: summarySchema.optional(),
});

/**
 * BookUpdateInput is the validated, normalized shape produced by
 * `bookUpdateSchema`.
 */
export type BookUpdateInput = z.infer<typeof bookUpdateSchema>;

/**
 * BookResponse is the serialized representation of a Book returned by the API.
 * Replaces the Pydantic `BookResponse` model (ORM mode).
 */
export interface BookResponse {
  id: number;
  title: string;
  author: string;
  published_year: number;
  summary: string | null;
}

/**
 * toBookResponse serializes a Prisma `Book` model instance into the public
 * `BookResponse` shape. This replaces Pydantic's `from_attributes=True`
 * (ORM mode) serialization.
 *
 * MIGRATION: This assumes the Prisma `Book` model exposes fields named
 * `id`, `title`, `author`, `published_year`, and `summary`. Adjust the field
 * mapping if the Prisma schema uses different column/field names (e.g.
 * camelCase `publishedYear`).
 */
export function toBookResponse(book: Book): BookResponse {
  return {
    id: book.id,
    title: book.title,
    author: book.author,
    published_year: (book as unknown as { published_year: number })
      .published_year,
    summary: book.summary ?? null,
  };
}
