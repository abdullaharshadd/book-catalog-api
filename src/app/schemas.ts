// src/app/schemas.ts

/**
 * Request/response DTO validation schemas for Book entities.
 *
 * MIGRATION_NOTE: The Python source used Pydantic v2 BaseModel classes with
 * `field_validator` decorators. In the target stack we use Zod, which is the
 * idiomatic TypeScript validation library and integrates cleanly with Express
 * route-level validation.
 *
 * Fidelity notes preserved from the source wire contract:
 *   - All fields keep snake_case naming (`published_year`) to preserve the
 *     JSON wire contract exactly.
 *   - Object key declaration order matches the source field order.
 *   - The source did NOT set `extra='forbid'`, so these schemas are NOT
 *     `.strict()` — unknown keys are silently ignored, matching Pydantic's
 *     default "ignore" behavior.
 *   - Pydantic's `str_strip_whitespace=True` config trimmed all string inputs
 *     before validation. This is reproduced with `.transform(v => v.trim())`
 *     applied at the appropriate points (see per-field notes below).
 *   - The Python validators raised ValueError with specific messages; those
 *     exact messages are preserved via Zod refinements so the wire error
 *     contract is unchanged.
 *
 * MIGRATION_NOTE: `BookResponse` in Pydantic used `from_attributes=True` to
 * build a response DTO from an ORM object. In Node.js there is no equivalent
 * "validate-on-serialize" step needed for output; the response shape is
 * expressed as a plain TypeScript type plus a helper that projects a Book row
 * into the response object.
 */

import { z } from 'zod';
import type { Book } from './models';

const MAX_TITLE = 255;
const MAX_AUTHOR = 255;
const MAX_SUMMARY = 2000;
const MIN_YEAR = 1000;

/**
 * Reproduce Pydantic's str_strip_whitespace + non-empty validation for a
 * required string field. Trim happens first (mirroring the config), then the
 * emptiness and length checks run against the trimmed value.
 */
function requiredTrimmedString(fieldLabel: string, max: number) {
  return z
    .string()
    .transform((v) => v.trim())
    .superRefine((v, ctx) => {
      if (!v) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: `${fieldLabel} cannot be empty`,
        });
        return;
      }
      if (v.length > max) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: `ensure this value has at most ${max} characters`,
        });
      }
    });
}

/**
 * MIGRATION_NOTE: The source length check ran on the *original* (pre-strip)
 * length for the 255 case (`len(v) > 255` before `v.strip()` is returned).
 * However Pydantic's `str_strip_whitespace` strips BEFORE the validator runs,
 * so `v` inside the validator is already trimmed. We therefore check the
 * trimmed length, which matches actual Pydantic v2 runtime behavior.
 */

/**
 * Optional summary handling: strip, coerce empty -> null, enforce max length.
 * Mirrors Pydantic's validate_summary exactly (None passthrough, empty -> None).
 */
function summaryValidator() {
  return z
    .string()
    .nullish()
    .transform((v) => {
      if (v === null || v === undefined) return null;
      const trimmed = v.trim();
      return trimmed === '' ? null : trimmed;
    })
    .superRefine((v, ctx) => {
      if (v !== null && v.length > MAX_SUMMARY) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: `ensure this value has at most ${MAX_SUMMARY} characters`,
        });
      }
    });
}

function publishedYearRefine(v: number, ctx: z.RefinementCtx): void {
  const currentYear = new Date().getFullYear();
  if (v < MIN_YEAR) {
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
}

/**
 * BookCreate — schema for creating a new Book.
 *
 * Field order preserved: title, author, published_year, summary.
 */
export const bookCreateSchema = z.object({
  title: requiredTrimmedString('Title', MAX_TITLE),
  author: requiredTrimmedString('Author', MAX_AUTHOR),
  published_year: z
    .number()
    .int()
    .superRefine((v, ctx) => publishedYearRefine(v, ctx)),
  summary: summaryValidator(),
});

export type BookCreate = z.infer<typeof bookCreateSchema>;

/**
 * BookUpdate — schema for partial updates.
 *
 * Every field is optional (partial-update pattern). When present, each field
 * runs the same validation as BookCreate. `undefined`/absent fields are left
 * out entirely; an explicit `null` summary is treated as "clear" (-> null),
 * mirroring the source's None handling.
 */
export const bookUpdateSchema = z.object({
  title: requiredTrimmedString('Title', MAX_TITLE).optional(),
  author: requiredTrimmedString('Author', MAX_AUTHOR).optional(),
  published_year: z
    .number()
    .int()
    .superRefine((v, ctx) => publishedYearRefine(v, ctx))
    .optional(),
  summary: summaryValidator().optional(),
});

export type BookUpdate = z.infer<typeof bookUpdateSchema>;

/**
 * BookResponse — the outbound shape for a Book.
 *
 * Field order preserved: id, title, author, published_year, summary.
 */
export interface BookResponse {
  id: number;
  title: string;
  author: string;
  published_year: number;
  summary: string | null;
}

/**
 * Project a persisted Book row into the response DTO.
 *
 * MIGRATION_NOTE: Replaces Pydantic's `from_attributes=True` behavior — instead
 * of validating an ORM object on serialization, we explicitly map the known
 * fields, which is both faster and type-safe.
 */
export function toBookResponse(book: Book): BookResponse {
  return {
    id: book.id,
    title: book.title,
    author: book.author,
    published_year: book.published_year,
    summary: book.summary ?? null,
  };
}
