import { z } from 'zod';

/**
 * Maximum allowed length for the book title.
 */
const MAX_TITLE_LENGTH = 255;

/**
 * Maximum allowed length for the book author.
 */
const MAX_AUTHOR_LENGTH = 255;

/**
 * Maximum allowed length for the book summary.
 */
const MAX_SUMMARY_LENGTH = 2000;

/**
 * Minimum allowed published year (must be after year 1000).
 */
const MIN_PUBLISHED_YEAR = 1000;

/**
 * Validates and normalizes a required string field (title/author).
 *
 * Strips surrounding whitespace, rejects empty values and enforces the
 * provided maximum length. Mirrors the Pydantic `field_validator` behavior
 * for `title` and `author` in the source schema.
 *
 * NOTE: Preserves the original Pydantic-style error message
 * 'ensure this value has at most N characters'.
 */
const requiredStringField = (fieldName: string, maxLength: number) =>
  z
    .string()
    .transform((v) => v.trim())
    .superRefine((v, ctx) => {
      if (!v) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: `${fieldName} cannot be empty`,
        });
        return;
      }
      if (v.length > maxLength) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: `ensure this value has at most ${maxLength} characters`,
        });
      }
    });

/**
 * Validates a published year value.
 *
 * Enforces the lower bound (after year 1000) and rejects future years based
 * on the current system year. Mirrors the source `validate_published_year`.
 *
 * MIGRATION: The source uses `datetime.now().year` (local time). This uses
 * `new Date().getFullYear()` (local time as well). If timezone-correct
 * behavior is required, replace with a UTC-aware computation.
 */
const publishedYearSchema = z.number().int().superRefine((v, ctx) => {
  const currentYear = new Date().getFullYear();
  if (v < MIN_PUBLISHED_YEAR) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'Published year must be after year 1000',
    });
    return;
  }
  if (v > currentYear) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: `Published year cannot be in the future (current year: ${currentYear})`,
    });
  }
});

/**
 * Validates and normalizes an optional summary field.
 *
 * Trims whitespace and normalizes empty/blank values to `null`, then enforces
 * the maximum length. Mirrors the source `validate_summary`.
 */
const summarySchema = z
  .union([z.string(), z.null()])
  .optional()
  .transform((v) => {
    if (v === null || v === undefined) {
      return null;
    }
    const trimmed = v.trim();
    return trimmed === '' ? null : trimmed;
  })
  .superRefine((v, ctx) => {
    if (v !== null && v.length > MAX_SUMMARY_LENGTH) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: `ensure this value has at most ${MAX_SUMMARY_LENGTH} characters`,
      });
    }
  });

/**
 * Schema for creating a Book. All required fields must be present.
 *
 * Equivalent to the Pydantic `BookCreate` model. Whitespace stripping and
 * field-level validation are applied; an empty summary is normalized to null.
 */
export const bookCreateSchema = z.object({
  title: requiredStringField('Title', MAX_TITLE_LENGTH),
  author: requiredStringField('Author', MAX_AUTHOR_LENGTH),
  published_year: publishedYearSchema,
  summary: summarySchema,
});

/**
 * Parsed and validated payload for creating a Book.
 */
export type BookCreate = z.infer<typeof bookCreateSchema>;

/**
 * Schema for partially updating a Book. All fields are optional.
 *
 * Equivalent to the Pydantic `BookUpdate` model. Provided fields are validated
 * and normalized identically to create; omitted fields are left undefined.
 */
export const bookUpdateSchema = z.object({
  title: requiredStringField('Title', MAX_TITLE_LENGTH).optional(),
  author: requiredStringField('Author', MAX_AUTHOR_LENGTH).optional(),
  published_year: publishedYearSchema.optional(),
  summary: summarySchema,
});

/**
 * Parsed and validated payload for updating a Book.
 */
export type BookUpdate = z.infer<typeof bookUpdateSchema>;

/**
 * Shape of a Book as returned to API clients.
 *
 * Equivalent to the Pydantic `BookResponse` model (ORM/serialization DTO).
 */
export interface BookResponse {
  id: number;
  title: string;
  author: string;
  published_year: number;
  summary: string | null;
}

/**
 * Result tuple type used by fallible parse helpers: [value, error].
 */
export type Result<T> = [T, null] | [null, z.ZodError];

/**
 * Parses and validates input against the BookCreate schema.
 *
 * Returns a tuple of [value, error]; exactly one element is non-null.
 */
export function parseBookCreate(input: unknown): Result<BookCreate> {
  const result = bookCreateSchema.safeParse(input);
  if (result.success) {
    return [result.data, null];
  }
  return [null, result.error];
}

/**
 * Parses and validates input against the BookUpdate schema.
 *
 * Returns a tuple of [value, error]; exactly one element is non-null.
 */
export function parseBookUpdate(input: unknown): Result<BookUpdate> {
  const result = bookUpdateSchema.safeParse(input);
  if (result.success) {
    return [result.data, null];
  }
  return [null, result.error];
}

/**
 * Serializes an ORM-like Book entity into a BookResponse DTO.
 *
 * Equivalent to Pydantic's `from_attributes=True` behavior: reads attributes
 * directly from the source object.
 */
export function toBookResponse(entity: {
  id: number;
  title: string;
  author: string;
  published_year: number;
  summary?: string | null;
}): BookResponse {
  return {
    id: entity.id,
    title: entity.title,
    author: entity.author,
    published_year: entity.published_year,
    summary: entity.summary ?? null,
  };
}
