// src/app/main.ts
import express, {
  Application,
  Request,
  Response,
  NextFunction,
  Router,
} from 'express';
import { z } from 'zod';
import { Prisma } from '@prisma/client';

import { prisma } from './database';
import { config } from './config';
import { logger } from './logger';
import {
  bookCreateSchema,
  bookUpdateSchema,
  toBookResponse,
} from './schemas';

/**
 * MIGRATION NOTES
 * ----------------
 * The source `app/main.py` was a FastAPI application. In the Node.js stack we
 * use Express + Prisma. Key mappings:
 *
 *   - FastAPI path operations (`@app.get/post/put/delete`) -> Express Router
 *     handlers grouped by resource.
 *   - FastAPI `Depends(get_db)` / `Depends(get_sync_db)` DI -> a single shared
 *     `prisma` client. The source mixed async (`AsyncSession`) and sync
 *     (`Session`) database access; Prisma is uniformly promise-based, so all
 *     handlers are async here. Business logic is preserved identically.
 *   - Pydantic schemas (`BookCreate`/`BookUpdate`/`BookResponse`) -> Zod
 *     schemas for validation + a `toBookResponse` serializer.
 *   - FastAPI `HTTPException` + custom exception handler -> a typed `HttpError`
 *     class plus centralized Express error middleware that returns a
 *     `{ detail: string }` JSON body (matching the original response shape).
 *   - `@app.on_event("startup") init_db()` -> Prisma manages schema via
 *     migrations; we simply verify connectivity on startup (see `startServer`).
 *   - SQLAlchemy `IntegrityError` (duplicate title+author) -> Prisma
 *     `P2002` unique-constraint violation.
 *   - The pagination `limit` cap of 1000 is preserved.
 */

/**
 * HttpError is the Node.js equivalent of FastAPI's HTTPException. It carries an
 * HTTP status code and a `detail` message that is rendered into the JSON
 * response body by the error-handling middleware.
 */
export class HttpError extends Error {
  public readonly statusCode: number;
  public readonly detail: string;

  constructor(statusCode: number, detail: string) {
    super(detail);
    this.statusCode = statusCode;
    this.detail = detail;
    this.name = 'HttpError';
  }
}

/**
 * asyncHandler wraps an async route handler so that any rejected promise is
 * forwarded to Express's error middleware (`next(err)`), avoiding repetitive
 * try/catch boilerplate at the call sites.
 */
function asyncHandler(
  fn: (req: Request, res: Response, next: NextFunction) => Promise<unknown>,
) {
  return (req: Request, res: Response, next: NextFunction): void => {
    fn(req, res, next).catch(next);
  };
}

/** Parsed query parameters for the list endpoint. */
const listQuerySchema = z.object({
  skip: z.coerce.number().int().min(0).default(0),
  limit: z.coerce.number().int().min(0).default(100),
});

/**
 * booksRouter groups all `/books` CRUD endpoints, mirroring the FastAPI route
 * handlers from `app/main.py`.
 */
export const booksRouter = Router();

/**
 * GET /books/ — Retrieve all books with pagination.
 *
 * - skip: Number of books to skip (default: 0)
 * - limit: Maximum number of books to return (default: 100, max: 1000)
 */
booksRouter.get(
  '/',
  asyncHandler(async (req: Request, res: Response) => {
    const parsed = listQuerySchema.safeParse(req.query);
    if (!parsed.success) {
      throw new HttpError(400, 'Invalid pagination parameters');
    }

    const { skip } = parsed.data;
    // Enforce reasonable limits (preserves the original 1000 cap).
    const limit = Math.min(parsed.data.limit, 1000);

    try {
      const books = await prisma.book.findMany({
        skip,
        take: limit,
      });

      logger.info(
        `Retrieved ${books.length} books (skip=${skip}, limit=${limit})`,
      );
      res.json(books.map(toBookResponse));
    } catch (err) {
      logger.error(
        `Error retrieving books: ${err instanceof Error ? err.message : String(err)}`,
      );
      throw new HttpError(500, 'Internal server error while retrieving books');
    }
  }),
);

/**
 * GET /books/:bookId — Retrieve a single book by its ID. Responds 404 if the
 * book does not exist.
 */
booksRouter.get(
  '/:bookId',
  asyncHandler(async (req: Request, res: Response) => {
    const bookId = Number(req.params.bookId);
    if (!Number.isInteger(bookId)) {
      throw new HttpError(404, `Book with ID ${req.params.bookId} not found`);
    }

    try {
      const book = await prisma.book.findUnique({ where: { id: bookId } });

      if (book === null) {
        logger.warn(`Book with ID ${bookId} not found`);
        throw new HttpError(404, `Book with ID ${bookId} not found`);
      }

      logger.info(`Retrieved book: ${book.title}`);
      res.json(toBookResponse(book));
    } catch (err) {
      if (err instanceof HttpError) throw err;
      logger.error(
        `Error retrieving book ${bookId}: ${err instanceof Error ? err.message : String(err)}`,
      );
      throw new HttpError(500, 'Internal server error while retrieving book');
    }
  }),
);

/**
 * POST /books/ — Create a new book. Returns 201 on success and 400 if a book
 * with the same title and author already exists (unique-constraint violation).
 */
booksRouter.post(
  '/',
  asyncHandler(async (req: Request, res: Response) => {
    const parsed = bookCreateSchema.safeParse(req.body);
    if (!parsed.success) {
      throw new HttpError(422, 'Validation error');
    }

    try {
      const dbBook = await prisma.book.create({ data: parsed.data });

      logger.info(`Created new book: ${dbBook.title} by ${dbBook.author}`);
      res.status(201).json(toBookResponse(dbBook));
    } catch (err) {
      if (
        err instanceof Prisma.PrismaClientKnownRequestError &&
        err.code === 'P2002'
      ) {
        logger.error(`Integrity error creating book: ${err.message}`);
        throw new HttpError(
          400,
          'Book with this title and author already exists',
        );
      }
      logger.error(
        `Error creating book: ${err instanceof Error ? err.message : String(err)}`,
      );
      throw new HttpError(500, 'Internal server error while creating book');
    }
  }),
);

/**
 * PUT /books/:bookId — Update an existing book. Only the fields provided in the
 * request body are modified (PATCH-like semantics, matching the source's
 * `exclude_unset`). Responds 404 if not found, 400 on unique-constraint clash.
 */
booksRouter.put(
  '/:bookId',
  asyncHandler(async (req: Request, res: Response) => {
    const bookId = Number(req.params.bookId);
    if (!Number.isInteger(bookId)) {
      throw new HttpError(404, `Book with ID ${req.params.bookId} not found`);
    }

    const parsed = bookUpdateSchema.safeParse(req.body);
    if (!parsed.success) {
      throw new HttpError(422, 'Validation error');
    }

    try {
      const existing = await prisma.book.findUnique({ where: { id: bookId } });
      if (existing === null) {
        logger.warn(`Book with ID ${bookId} not found for update`);
        throw new HttpError(404, `Book with ID ${bookId} not found`);
      }

      // Update only provided fields (exclude_unset equivalent: Zod strips
      // undefined keys, so `parsed.data` contains only supplied values).
      const updateData = parsed.data;

      const dbBook = await prisma.book.update({
        where: { id: bookId },
        data: updateData,
      });

      logger.info(`Updated book: ${dbBook.title}`);
      res.json(toBookResponse(dbBook));
    } catch (err) {
      if (err instanceof HttpError) throw err;
      if (
        err instanceof Prisma.PrismaClientKnownRequestError &&
        err.code === 'P2002'
      ) {
        logger.error(`Integrity error updating book: ${err.message}`);
        throw new HttpError(
          400,
          'Book with this title and author already exists',
        );
      }
      logger.error(
        `Error updating book ${bookId}: ${err instanceof Error ? err.message : String(err)}`,
      );
      throw new HttpError(500, 'Internal server error while updating book');
    }
  }),
);

/**
 * DELETE /books/:bookId — Delete a book by its ID. Returns 204 on success and
 * 404 if the book does not exist.
 */
booksRouter.delete(
  '/:bookId',
  asyncHandler(async (req: Request, res: Response) => {
    const bookId = Number(req.params.bookId);
    if (!Number.isInteger(bookId)) {
      throw new HttpError(404, `Book with ID ${req.params.bookId} not found`);
    }

    try {
      const existing = await prisma.book.findUnique({ where: { id: bookId } });
      if (existing === null) {
        logger.warn(`Book with ID ${bookId} not found for deletion`);
        throw new HttpError(404, `Book with ID ${bookId} not found`);
      }

      await prisma.book.delete({ where: { id: bookId } });

      logger.info(`Deleted book: ${existing.title}`);
      res.status(204).send();
    } catch (err) {
      if (err instanceof HttpError) throw err;
      logger.error(
        `Error deleting book ${bookId}: ${err instanceof Error ? err.message : String(err)}`,
      );
      throw new HttpError(500, 'Internal server error while deleting book');
    }
  }),
);

/**
 * errorHandler is the centralized Express error middleware. It mirrors the
 * FastAPI custom `http_exception_handler`, rendering `{ detail: string }` JSON
 * bodies with the appropriate status code.
 */
export function errorHandler(
  err: unknown,
  _req: Request,
  res: Response,
  _next: NextFunction,
): void {
  if (err instanceof HttpError) {
    res.status(err.statusCode).json({ detail: err.detail });
    return;
  }
  logger.error(
    `Unhandled error: ${err instanceof Error ? err.message : String(err)}`,
  );
  res.status(500).json({ detail: 'Internal server error' });
}

/**
 * createApp builds and configures the Express application with all routes and
 * middleware. It is exported (rather than a global singleton) so tests can
 * instantiate the app with supertest without binding to a port.
 */
export function createApp(): Application {
  const app = express();
  app.use(express.json());

  /**
   * GET / — Root endpoint with API information.
   */
  app.get('/', (_req: Request, res: Response) => {
    res.json({
      message: 'Welcome to Book Catalog API',
      version: '1.0.0',
      docs_url: '/docs',
    });
  });

  /**
   * GET /health — Health check endpoint.
   */
  app.get('/health', (_req: Request, res: Response) => {
    res.json({ status: 'healthy', service: 'book-catalog-api' });
  });

  // Group all book CRUD routes under /books.
  app.use('/books', booksRouter);

  // Centralized error handling must be registered last.
  app.use(errorHandler);

  return app;
}

/**
 * startServer verifies database connectivity (the Node.js analogue of the
 * FastAPI startup `init_db()` hook — Prisma manages schema via migrations) and
 * starts the HTTP server.
 */
export async function startServer(): Promise<void> {
  await prisma.$connect();
  logger.info('Database initialized successfully');

  const app = createApp();
  app.listen(config.port, () => {
    logger.info(`Book Catalog API listening on port ${config.port}`);
  });
}

// Only start the server when executed directly (not when imported by tests).
if (require.main === module) {
  startServer().catch((err) => {
    logger.error(
      `Failed to start server: ${err instanceof Error ? err.message : String(err)}`,
    );
    process.exit(1);
  });
}
