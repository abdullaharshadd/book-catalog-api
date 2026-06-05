// src/app/main.ts
import express, { Express, Request, Response, NextFunction, Router } from 'express';
import { z } from 'zod';
import { PrismaClient, Prisma } from '@prisma/client';
import { logger } from './logger';
import { config } from './config';

// MIGRATION: The original FastAPI app used SQLAlchemy with a mixed async/sync
// session split that was artificial. Here all DB access uses Prisma, which is
// async throughout. Pydantic schemas (BookCreate, BookUpdate, BookResponse)
// are reimplemented as Zod schemas. The startup init_db() hook maps to running
// Prisma migrations (`prisma migrate deploy`) rather than runtime DB init.
//
// MIGRATION: The unique constraint on (title, author) implied by the
// IntegrityError handling MUST be declared in schema.prisma, e.g.:
//   @@unique([title, author])
// Prisma raises P2002 on unique-constraint violations, which we map to 400.

/**
 * BookCreate is the request schema for creating a new book.
 * All fields except summary are required.
 */
export const BookCreateSchema = z.object({
  title: z.string().min(1),
  author: z.string().min(1),
  published_year: z.number().int(),
  summary: z.string().optional().nullable(),
});
export type BookCreate = z.infer<typeof BookCreateSchema>;

/**
 * BookUpdate is the request schema for partial updates.
 * Only provided fields are applied (equivalent to Pydantic exclude_unset).
 */
export const BookUpdateSchema = z.object({
  title: z.string().min(1).optional(),
  author: z.string().min(1).optional(),
  published_year: z.number().int().optional(),
  summary: z.string().optional().nullable(),
});
export type BookUpdate = z.infer<typeof BookUpdateSchema>;

/**
 * ListBooksQuerySchema validates pagination query params.
 * limit is capped at 1000 to preserve the original behavior.
 */
export const ListBooksQuerySchema = z.object({
  skip: z.coerce.number().int().min(0).default(0),
  limit: z.coerce.number().int().min(0).default(100),
});

/**
 * HttpError is a typed error carrying an HTTP status code and a detail message.
 * It mirrors FastAPI's HTTPException and produces a { detail } JSON body.
 */
export class HttpError extends Error {
  public readonly status: number;
  public readonly detail: string;

  constructor(status: number, detail: string) {
    super(detail);
    this.status = status;
    this.detail = detail;
    Object.setPrototypeOf(this, HttpError.prototype);
  }
}

/**
 * serializeBook maps a Prisma Book row to the BookResponse shape.
 */
function serializeBook(book: {
  id: number;
  title: string;
  author: string;
  published_year: number;
  summary: string | null;
}) {
  return {
    id: book.id,
    title: book.title,
    author: book.author,
    published_year: book.published_year,
    summary: book.summary,
  };
}

/**
 * createBookRouter builds the /books resource router.
 * The PrismaClient is injected for testability (no global singleton).
 */
export function createBookRouter(prisma: PrismaClient): Router {
  const router = Router();

  // GET /books/ - list books with pagination
  router.get('/', async (req: Request, res: Response, next: NextFunction) => {
    try {
      const parsed = ListBooksQuerySchema.safeParse(req.query);
      if (!parsed.success) {
        throw new HttpError(422, 'Invalid query parameters');
      }
      const { skip } = parsed.data;
      // Enforce reasonable limits
      const limit = Math.min(parsed.data.limit, 1000);

      const books = await prisma.book.findMany({ skip, take: limit });

      logger.info(`Retrieved ${books.length} books (skip=${skip}, limit=${limit})`);
      res.json(books.map(serializeBook));
    } catch (err) {
      if (err instanceof HttpError) return next(err);
      logger.error(`Error retrieving books: ${String(err)}`);
      next(new HttpError(500, 'Internal server error while retrieving books'));
    }
  });

  // GET /books/:book_id - retrieve a single book
  router.get('/:book_id', async (req: Request, res: Response, next: NextFunction) => {
    const bookId = Number(req.params.book_id);
    try {
      if (!Number.isInteger(bookId)) {
        throw new HttpError(422, 'Invalid book ID');
      }

      const book = await prisma.book.findUnique({ where: { id: bookId } });

      if (book === null) {
        logger.warn(`Book with ID ${bookId} not found`);
        throw new HttpError(404, `Book with ID ${bookId} not found`);
      }

      logger.info(`Retrieved book: ${book.title}`);
      res.json(serializeBook(book));
    } catch (err) {
      if (err instanceof HttpError) return next(err);
      logger.error(`Error retrieving book ${bookId}: ${String(err)}`);
      next(new HttpError(500, 'Internal server error while retrieving book'));
    }
  });

  // POST /books/ - create a new book
  router.post('/', async (req: Request, res: Response, next: NextFunction) => {
    try {
      const parsed = BookCreateSchema.safeParse(req.body);
      if (!parsed.success) {
        throw new HttpError(422, 'Validation failed');
      }
      const data = parsed.data;

      const dbBook = await prisma.book.create({
        data: {
          title: data.title,
          author: data.author,
          published_year: data.published_year,
          summary: data.summary ?? null,
        },
      });

      logger.info(`Created new book: ${dbBook.title} by ${dbBook.author}`);
      res.status(201).json(serializeBook(dbBook));
    } catch (err) {
      if (err instanceof HttpError) return next(err);
      // P2002 = unique constraint violation (title + author)
      if (
        err instanceof Prisma.PrismaClientKnownRequestError &&
        err.code === 'P2002'
      ) {
        logger.error(`Integrity error creating book: ${String(err)}`);
        return next(new HttpError(400, 'Book with this title and author already exists'));
      }
      logger.error(`Error creating book: ${String(err)}`);
      next(new HttpError(500, 'Internal server error while creating book'));
    }
  });

  // PUT /books/:book_id - update an existing book (partial update)
  router.put('/:book_id', async (req: Request, res: Response, next: NextFunction) => {
    const bookId = Number(req.params.book_id);
    try {
      if (!Number.isInteger(bookId)) {
        throw new HttpError(422, 'Invalid book ID');
      }

      const parsed = BookUpdateSchema.safeParse(req.body);
      if (!parsed.success) {
        throw new HttpError(422, 'Validation failed');
      }

      const existing = await prisma.book.findUnique({ where: { id: bookId } });
      if (existing === null) {
        logger.warn(`Book with ID ${bookId} not found for update`);
        throw new HttpError(404, `Book with ID ${bookId} not found`);
      }

      // Update only provided fields (exclude_unset equivalent).
      const updateData: Prisma.BookUpdateInput = {};
      if (parsed.data.title !== undefined) updateData.title = parsed.data.title;
      if (parsed.data.author !== undefined) updateData.author = parsed.data.author;
      if (parsed.data.published_year !== undefined)
        updateData.published_year = parsed.data.published_year;
      if (parsed.data.summary !== undefined) updateData.summary = parsed.data.summary;

      const dbBook = await prisma.book.update({
        where: { id: bookId },
        data: updateData,
      });

      logger.info(`Updated book: ${dbBook.title}`);
      res.json(serializeBook(dbBook));
    } catch (err) {
      if (err instanceof HttpError) return next(err);
      if (
        err instanceof Prisma.PrismaClientKnownRequestError &&
        err.code === 'P2002'
      ) {
        logger.error(`Integrity error updating book: ${String(err)}`);
        return next(new HttpError(400, 'Book with this title and author already exists'));
      }
      logger.error(`Error updating book ${bookId}: ${String(err)}`);
      next(new HttpError(500, 'Internal server error while updating book'));
    }
  });

  // DELETE /books/:book_id - delete a book
  router.delete('/:book_id', async (req: Request, res: Response, next: NextFunction) => {
    const bookId = Number(req.params.book_id);
    try {
      if (!Number.isInteger(bookId)) {
        throw new HttpError(422, 'Invalid book ID');
      }

      const existing = await prisma.book.findUnique({ where: { id: bookId } });
      if (existing === null) {
        logger.warn(`Book with ID ${bookId} not found for deletion`);
        throw new HttpError(404, `Book with ID ${bookId} not found`);
      }

      await prisma.book.delete({ where: { id: bookId } });

      logger.info(`Deleted book: ${existing.title}`);
      res.status(204).send();
    } catch (err) {
      if (err instanceof HttpError) return next(err);
      logger.error(`Error deleting book ${bookId}: ${String(err)}`);
      next(new HttpError(500, 'Internal server error while deleting book'));
    }
  });

  return router;
}

/**
 * createApp wires up the Express application with all routes and the
 * centralized error handler. PrismaClient is injected for testability.
 */
export function createApp(prisma: PrismaClient): Express {
  const app = express();
  app.use(express.json());

  // Root endpoint with API information
  app.get('/', (_req: Request, res: Response) => {
    res.json({
      message: 'Welcome to Book Catalog API',
      version: '1.0.0',
      docs_url: '/docs',
    });
  });

  // Health check endpoint
  app.get('/health', (_req: Request, res: Response) => {
    res.json({ status: 'healthy', service: 'book-catalog-api' });
  });

  app.use('/books', createBookRouter(prisma));

  // Centralized error handler — produces FastAPI-compatible { detail } bodies.
  app.use((err: unknown, _req: Request, res: Response, _next: NextFunction) => {
    if (err instanceof HttpError) {
      res.status(err.status).json({ detail: err.detail });
      return;
    }
    logger.error(`Unhandled error: ${String(err)}`);
    res.status(500).json({ detail: 'Internal server error' });
  });

  return app;
}

// MIGRATION: FastAPI auto-generated OpenAPI docs at /docs and /redoc.
// Express has no built-in equivalent; integrate swagger-ui-express or
// @asteasolutions/zod-to-openapi if interactive docs are required.

/* istanbul ignore next */
if (require.main === module) {
  const prisma = new PrismaClient();
  const app = createApp(prisma);
  const port = config.port;
  app.listen(port, () => {
    logger.info(`Book Catalog API listening on port ${port}`);
  });
}
