/**
 * Book request/response schemas.
 *
 * MIGRATION: The original `app/schemas.py` defined Pydantic v2 models
 * (BookCreate / BookUpdate / BookResponse) for a FastAPI app. In the
 * Node.js/TypeScript stack we use **Zod** for request validation and plain
 * TypeScript types/serializers for responses. Mapping decisions:
 *
 *   - Pydantic `ConfigDict(str_strip_whitespace=True)` (auto-strip all string
 *     fields) -> explicit `.trim()` transforms inside each Zod schema.
 *   - Pydantic `field_validator` methods -> Zod `.refine()` / `.transform()`
 *     chained per field. Custom error messages are preserved verbatim so the
 *     API contract stays stable for existing clients.
 *   - `BookCreate` -> `bookCreateSchema` (all fields required).
 *   - `BookUpdate` -> `bookUpdateSchema` (all fields optional / partial).
 *     Pydantic treated `None` as "not provided"; here we use `.optional()`
 *     so omitted keys are simply absent from the parsed object.
 *   - `BookResponse` (Pydantic ORM mode `from_attributes=True`) -> a plain
 *     `BookResponse` interface plus a `toBookResponse` serializer that maps a
 *     Prisma model instance to the response shape. Prisma returns plain
 *     objects, so ORM-mode is implicit.
 *   - The dynamic future-year check (`datetime.now().year`) is preserved by
 *     evaluating `new Date().getFullYear()` at validation time.
 *
 * Note: the `summary` empty-string -> `null` normalization from the source is
 * preserved via a transform.
 */

import { z } from 'zod';

/** Maximum allowed length for the `title` and `author` fields. */
const MAX_TITLE_AUTHOR_LENGTH = 255;

/** Maximum allowed length for the `summary` field. */
const MAX_SUMMARY_LENGTH = 2000;

/** Earliest accepted publication year. */
const MIN_PUBLISHED_YEAR = 1000;

/**
 * Validates and normalizes a required title/author-style string field.
 *
 * Trims whitespace, rejects empty values, and enforces the max length.
 * Error messages mirror the original Pydantic/FastAPI contract.
 *
 * @param fieldLabel - Human-readable field name used in the "cannot be empty" message.
 */
function requiredNameField(fieldLabel: string): z.ZodEffects<z.ZodString, string, string> {
  return z
    .string()
    .transform((v) => v.trim())
    .refine((v) => v.length > 0, { message: `${fieldLabel} cannot be empty` })
    .refine((v) => v.length <= MAX_TITLE_AUTHOR_LENGTH, {
      message: 'ensure this value has at most 255 characters',
    });
}

/**
 * Validates a published-year value.
 *
 * Enforces a lower bound of {@link MIN_PUBLISHED_YEAR} and an upper bound of
 * the current calendar year, evaluated dynamically at validation time.
 */
function publishedYearField(): z.ZodEffects<z.ZodNumber, number, number> {
  return z
    .number()
    .int()
    .superRefine((v, ctx) => {
      const currentYear = new Date().getFullYear();
      if (v < MIN_PUBLISHED_YEAR) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'Published year must be after year 1000',
        });
      }
      if (v > currentYear) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: `Published year cannot be in the future (current year: ${currentYear})`,
        });
      }
    });
}

/**
 * Validates and normalizes the optional `summary` field.
 *
 * Trims whitespace, converts empty/whitespace-only strings to `null`, and
 * enforces the {@link MAX_SUMMARY_LENGTH} cap. Matches the source behavior
 * where an empty summary collapses to `null`.
 */
function summaryField(): z.ZodType<string | null, z.ZodTypeDef, unknown> {
  return z
    .string()
    .nullish()
    .transform((v) => {
      if (v === null || v === undefined) {
        return null;
      }
      const trimmed = v.trim();
      return trimmed.length === 0 ? null : trimmed;
    })
    .superRefine((v, ctx) => {
      if (v !== null && v.length > MAX_SUMMARY_LENGTH) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'ensure this value has at most 2000 characters',
        });
      }
    });
}

/**
 * Zod schema for creating a book. All core fields are required; `summary` is
 * optional and normalized to `null` when blank.
 */
export const bookCreateSchema = z.object({
  title: requiredNameField('Title'),
  author: requiredNameField('Author'),
  published_year: publishedYearField(),
  summary: summaryField().optional().default(null),
});

/** Parsed and validated input for creating a book. */
export type BookCreate = z.infer<typeof bookCreateSchema>;

/**
 * Zod schema for partially updating a book.
 *
 * MIGRATION: The Pydantic `BookUpdate` made every field Optional and treated
 * `None` as "not provided". Here each field is `.optional()`, so omitted keys
 * are simply absent after parsing — preserving the partial-update semantics.
 */
export const bookUpdateSchema = z.object({
  title: requiredNameField('Title').optional(),
  author: requiredNameField('Author').optional(),
  published_year: publishedYearField().optional(),
  summary: summaryField().optional(),
});

/** Parsed and validated input for updating a book. */
export type BookUpdate = z.infer<typeof bookUpdateSchema>;

/**
 * Response shape for a serialized book.
 *
 * Mirrors the Pydantic `BookResponse` model used for output serialization.
 */
export interface BookResponse {
  id: number;
  title: string;
  author: string;
  published_year: number;
  summary: string | null;
}

/**
 * Minimal structural type describing a persisted book record (e.g. a Prisma
 * model instance). Used to serialize into a {@link BookResponse}.
 */
export interface BookRecord {
  id: number;
  title: string;
  author: string;
  published_year: number;
  summary: string | null;
}

/**
 * Serializes a persisted book record into the public {@link BookResponse}
 * shape.
 *
 * MIGRATION: Replaces Pydantic's `ConfigDict(from_attributes=True)` ORM mode.
 * Prisma returns plain objects, so we map fields explicitly and coerce a
 * missing summary to `null`.
 *
 * @returns A tuple of `[response, error]`; `error` is `null` on success.
 */
export function toBookResponse(record: BookRecord): [BookResponse | null, Error | null] {
  if (record === null || record === undefined) {
    return [null, new Error('Cannot serialize a null book record')];
  }

  const response: BookResponse = {
    id: record.id,
    title: record.title,
    author: record.author,
    published_year: record.published_year,
    summary: record.summary ?? null,
  };

  return [response, null];
}
