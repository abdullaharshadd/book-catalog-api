// src/app/models.ts

import type { Book as PrismaBook } from '@prisma/client';

/**
 * MIGRATION NOTES
 * ----------------
 * The source `app/models.py` defined a **SQLAlchemy** ORM model (`Book`) using
 * `declarative_base()` and `Column` definitions — despite the Django-style
 * filename, it is NOT a Django model.
 *
 * In the idiomatic Node.js/TypeScript + Prisma stack, the persistence schema
 * lives in `prisma/schema.prisma` (see the model below). This file therefore
 * does NOT redefine an ORM mapping; instead it provides:
 *
 *   1. A reference copy of the Prisma model (in a comment) so the schema is
 *      discoverable alongside the rest of the migrated `app` code.
 *   2. Domain helpers that preserve the SQLAlchemy `__str__` / `__repr__`
 *      business behaviour, operating on the Prisma-generated `Book` type.
 *
 * Column / constraint mappings (SQLAlchemy -> Prisma):
 *   - id            Column(Integer, primary_key, autoincrement) -> Int @id @default(autoincrement())
 *   - title         String(255), nullable=False, index=True      -> String  @db.VarChar(255)  + @@index([title])
 *   - author        String(255), nullable=False, index=True      -> String  @db.VarChar(255)  + @@index([author])
 *   - published_year Integer, nullable=False, index=True          -> Int                       + @@index([publishedYear])
 *   - summary       Text, nullable=True                          -> String? @db.Text
 *   - __tablename__ = 'books'                                     -> @@map("books")
 *   - UniqueConstraint('title','author','unique_title_author')    -> @@unique([title, author], map: "unique_title_author")
 *
 * MIGRATION: The Prisma schema below must be added to `prisma/schema.prisma`
 * and migrated with `npx prisma migrate dev`. Prisma does not read schema from
 * `.ts` files, so this block is a documentation copy only.
 *
 *   model Book {
 *     id            Int     @id @default(autoincrement())
 *     title         String  @db.VarChar(255)
 *     author        String  @db.VarChar(255)
 *     publishedYear Int     @map("published_year")
 *     summary       String? @db.Text
 *
 *     @@unique([title, author], map: "unique_title_author")
 *     @@index([title])
 *     @@index([author])
 *     @@index([publishedYear])
 *     @@map("books")
 *   }
 *
 * Note: SQLAlchemy used `snake_case` column names. The Prisma model uses
 * `camelCase` field names with `@map`/`@@map` to preserve the original DB
 * column/table names (`published_year`, `books`).
 */

/**
 * Book is the domain representation of a catalog book.
 *
 * It is structurally identical to the Prisma-generated `Book` type and is
 * re-exported here so callers can depend on `./models` rather than directly on
 * `@prisma/client`.
 */
export type Book = PrismaBook;

/**
 * bookToString reproduces the SQLAlchemy `__str__` behaviour:
 *   `"{title} by {author} ({published_year})"`
 *
 * @param book - the book to render
 * @returns a human-friendly one-line description
 */
export function bookToString(book: Book): string {
  return `${book.title} by ${book.author} (${book.publishedYear})`;
}

/**
 * bookToRepr reproduces the SQLAlchemy `__repr__` behaviour:
 *   `"<Book(id=.., title='..', author='..', year=..)>"`
 *
 * Useful for debug logging.
 *
 * @param book - the book to render
 * @returns a developer-oriented debug string
 */
export function bookToRepr(book: Book): string {
  return `<Book(id=${book.id}, title='${book.title}', author='${book.author}', year=${book.publishedYear})>`;
}
