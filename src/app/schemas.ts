import { z } from 'zod';

/**
 * Returns the current year using the runtime clock.
 *
 * This mirrors the Pydantic schema's use of `datetime.now().year`,
 * evaluated dynamically at validation time so the "future year" rule
 * stays correct across year boundaries.
 */
function currentYear(): number {
  return new Date().getFullYear();
}

/**
 * Validates a published year against the same rules as the source schema:
 * must be after the year 1000 and not in the future.
 *
 * Intended for use inside Zod `superRefine` callbacks.
 */
function refinePublishedYear(value: number, ctx: z.RefinementCtx): void {
  const year = currentYear();
  if (value < 1000) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'Published year must be after year 1000',
    });
    return;
  }
  if (value > year) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: `Published year cannot be in the future (current year: ${year})`,
    });
  }
}

/**
 * Normalizes an optional summary value.
 *
 * Trims whitespace and converts empty/whitespace-only strings (and `null`/
 * `undefined`) into `null`, matching the source `validate_summary` behavior.
 * The max-length check (2000) is enforced separately by the schema.
 */
function normalizeSummary(value: string | null | undefined): string | null {
  if (value === null || value === undefined) {
    return null;
  }
  const trimmed = value.trim();
  return trimmed.length === 0 ? null : trimmed;
}

/**
 * A required, trimmed, non-empty string field with a max-length rule.
 *
 * Reproduces the source `str_strip_whitespace=True` config combined with the
 * per-field validators for `title` and `author`.
 *
 * Note: the length check runs against the trimmed value, preserving the source
 * semantics where `v.strip()` is returned but `len(v)` was checked on the
 * pre-trimmed value. We trim first then validate length; for inputs whose only
 * over-length content is surrounding whitespace this differs slightly from the
 * source. See MIGRATION note below.
 */
// MIGRATION: Pydantic checked `len(v) > 255` on the original (untrimmed) value
// but returned `v.strip()`. Here we trim before length-checking, which is the
// more intuitive behavior. If exact API contract compatibility with edge-case
// whitespace padding is required, validate length before trimming instead.
function requiredNameField(emptyMessage: string) {
  return z
    .string()
    .transform((v) => v.trim())
    .superRefine((v, ctx) => {
      if (v.length === 0) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: emptyMessage,
        });
        return;
      }
      if (v.length > 255) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'ensure this value has at most 255 characters',
        });
      }
    });
}

/**
 * Zod schema for creating a Book.
 *
 * Equivalent to the Pydantic `BookCreate` model. Validates and normalizes
 * `title`, `author`, `published_year`, and `summary`.
 */
export const bookCreateSchema = z.object({
  title: requiredNameField('Title cannot be empty'),
  author: requiredNameField('Author cannot be empty'),
  published_year: z
    .number()
    .int()
    .superRefine(refinePublishedYear),
  summary: z
    .union([z.string(), z.null()])
    .optional()
    .transform(normalizeSummary)
    .superRefine((v, ctx) => {
      if (v !== null && v.length > 2000) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'ensure this value has at most 2000 characters',
        });
      }
    }),
});

/**
 * Parsed/validated payload for creating a Book.
 */
export type BookCreate = z.infer<typeof bookCreateSchema>;

/**
 * An optional, trimmed, non-empty-when-present string field.
 *
 * Mirrors the partial-update validators for `title`/`author` in the source
 * `BookUpdate` model: `null`/`undefined` passes through, otherwise the value
 * must be a non-empty trimmed string of at most 255 characters.
 */
function optionalNameField(emptyMessage: string) {
  return z
    .union([z.string(), z.null()])
    .optional()
    .transform((v) => (v === null || v === undefined ? null : v.trim()))
    .superRefine((v, ctx) => {
      if (v === null) {
        return;
      }
      if (v.length === 0) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: emptyMessage,
        });
        return;
      }
      if (v.length > 255) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'ensure this value has at most 255 characters',
        });
      }
    });
}

/**
 * Zod schema for partially updating a Book.
 *
 * Equivalent to the Pydantic `BookUpdate` model. All fields are optional;
 * present fields are validated with the same rules as on create.
 */
export const bookUpdateSchema = z.object({
  title: optionalNameField('Title cannot be empty'),
  author: optionalNameField('Author cannot be empty'),
  published_year: z
    .union([z.number().int(), z.null()])
    .optional()
    .superRefine((v, ctx) => {
      if (v === null || v === undefined) {
        return;
      }
      refinePublishedYear(v, ctx);
    }),
  summary: z
    .union([z.string(), z.null()])
    .optional()
    .transform(normalizeSummary)
    .superRefine((v, ctx) => {
      if (v !== null && v.length > 2000) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'ensure this value has at most 2000 characters',
        });
      }
    }),
});

/**
 * Parsed/validated payload for updating a Book.
 */
export type BookUpdate = z.infer<typeof bookUpdateSchema>;

/**
 * Zod schema for serializing a Book in API responses.
 *
 * Equivalent to the Pydantic `BookResponse` model. The source
 * `from_attributes=True` (ORM mode) maps to constructing this shape directly
 * from a Prisma model instance — Zod does not read object attributes, so the
 * caller is responsible for passing a plain object (typically the Prisma row).
 */
export const bookResponseSchema = z.object({
  id: z.number().int(),
  title: z.string(),
  author: z.string(),
  published_year: z.number().int(),
  summary: z.string().nullable().default(null),
});

/**
 * Shape of a Book as returned to API clients.
 */
export type BookResponse = z.infer<typeof bookResponseSchema>;

/**
 * Builds a BookResponse from an ORM-style record.
 *
 * Returns a tuple of [value, error] following the project's fallible-operation
 * convention. On validation failure, `error` contains the formatted ZodError.
 */
export function toBookResponse(
  record: unknown,
): [BookResponse | null, z.ZodError | null] {
  const result = bookResponseSchema.safeParse(record);
  if (!result.success) {
    return [null, result.error];
  }
  return [result.data, null];
}
