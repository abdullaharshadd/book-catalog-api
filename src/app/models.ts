// src/app/models.ts

/**
 * Book domain model and Prisma schema mapping.
 *
 * MIGRATION_NOTE: The Python source defined the Book model using SQLAlchemy's
 * declarative ORM (`Base = declarative_base()` + a `Book(Base)` class). In the
 * target stack the *persistence* shape of this model lives in `schema.prisma`
 * (Prisma is the ORM chosen in database.ts), NOT in a hand-written class.
 * Prisma generates its own `Book` type from the schema, so re-declaring an ORM
 * entity class here would be redundant and would not participate in queries.
 *
 * What this file therefore provides:
 *   1. The Prisma model definition to add to `schema.prisma` (documented below
 *      as the canonical source of truth for the table shape).
 *   2. A plain TypeScript interface mirroring the row shape, for use in service
 *      / route layers that don't want to import the generated Prisma type
 *      directly.
 *   3. Runtime Zod validation schemas derived from the same constraints
 *      (nullable summary, required title/author/published_year, string length
 *      limits from the source Column(String(255)) definitions).
 *   4. `bookToString` / `bookToRepr` helpers preserving the original
 *      `__str__` / `__repr__` behaviour.
 */

import { z } from 'zod';

/**
 * MIGRATION_NOTE: Add the following model to `prisma/schema.prisma`.
 * Field/column names are lower_snake_case (Postgres-friendly, no quoting
 * needed). The composite UniqueConstraint('title', 'author') from the source
 * `__table_args__` maps to `@@unique([title, author])`. A P2002 error from
 * Prisma on insert/update indicates a violation of this constraint and should
 * be caught in the repository/route layer and surfaced as a 409 Conflict.
 *
 * model Book {
 *   id            Int     @id @default(autoincrement())
 *   title         String  @db.VarChar(255)
 *   author        String  @db.VarChar(255)
 *   publishedYear Int     @map("published_year")
 *   summary       String? @db.Text
 *
 *   @@unique([title, author], name: "unique_title_author")
 *   @@index([title])
 *   @@index([author])
 *   @@index([publishedYear])
 *   @@map("books")
 * }
 */

/**
 * Row shape of the `books` table.
 *
 * Mirrors the Prisma-generated `Book` type. `summary` is nullable to match the
 * source `Column(Text, nullable=True)`.
 */
export interface Book {
  id: number;
  title: string;
  author: string;
  publishedYear: number;
  summary: string | null;
}

// String length limit derived from Column(String(255)) in the source model.
const MAX_STRING_LENGTH = 255;

// Reasonable bounds for a published year. The source imposed no numeric
// constraint, but validating input is idiomatic here. Adjust if unbounded
// years are required (MIGRATION_NOTE).
const MIN_PUBLISHED_YEAR = 0;
const MAX_PUBLISHED_YEAR = 9999;

/**
 * Validation schema for creating a Book.
 *
 * Enforces the source constraints:
 *   - title:          required, non-empty, <= 255 chars
 *   - author:         required, non-empty, <= 255 chars
 *   - published_year: required integer
 *   - summary:        optional / nullable text
 */
export const createBookSchema = z.object({
  title: z.string().min(1, 'title is required').max(MAX_STRING_LENGTH),
  author: z.string().min(1, 'author is required').max(MAX_STRING_LENGTH),
  publishedYear: z
    .number()
    .int()
    .min(MIN_PUBLISHED_YEAR)
    .max(MAX_PUBLISHED_YEAR),
  summary: z.string().nullable().optional(),
});

/**
 * Validation schema for updating a Book. All fields optional so callers can
 * perform partial updates; the same per-field constraints still apply.
 */
export const updateBookSchema = createBookSchema.partial();

export type CreateBookInput = z.infer<typeof createBookSchema>;
export type UpdateBookInput = z.infer<typeof updateBookSchema>;

/**
 * Human-readable representation of a Book.
 *
 * MIGRATION_NOTE: Preserves the Python `__str__` behaviour:
 *   f"{self.title} by {self.author} ({self.published_year})"
 */
export function bookToString(book: Book): string {
  return `${book.title} by ${book.author} (${book.publishedYear})`;
}

/**
 * Debug representation of a Book.
 *
 * MIGRATION_NOTE: Preserves the Python `__repr__` behaviour:
 *   f"<Book(id=..., title='...', author='...', year=...)>"
 */
export function bookToRepr(book: Book): string {
  return `<Book(id=${book.id}, title='${book.title}', author='${book.author}', year=${book.publishedYear})>`;
}
