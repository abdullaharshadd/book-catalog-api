// src/app/main.ts

/**
 * Book Catalog API — Express application entry point.
 *
 * MIGRATION_NOTE: The Python source used FastAPI with a single module-level
 * `app` object decorated with route handlers. In the idiomatic Express/TS
 * setup we:
 *   - build the app in a `createApp()` factory so it can be imported by tests
 *     (supertest) without side effects,
 *   - group Book routes in a dedicated Express Router (`booksRouter`),
 *   - use a centralized error-handling middleware instead of FastAPI's
 *     `@app.exception_handler(HTTPException)`,
 *   - use Zod schemas (from schemas.ts) for request validation instead of
 *     Pydantic model binding.
 *
 * MIGRATION_NOTE: FastAPI's `@app.on_event("startup")` -> `initDb()` is invoked
 * explicitly in `start()` before the server begins listening. Prisma manages
 * its own pool so there is no separate sync/async engine distinction.
 *
 * MIGRATION_NOTE: The source mixed async and sync route handlers (sync ones
 * used SQLAlchemy's sync Session). In Node everything is async because Prisma
 * is async-only — this is a behavioral improvement, the wire contract is
 * unchanged.
 *
 * MIGRATION_NOTE: FastAPI's error body shape was `{"detail": ...}`. The house
 * style for this migration is `{ error: string, details?: unknown }`. To
 * preserve the exact wire contract of the source (`detail` key) the error
 * middleware emits `{ detail }`. See `HttpError` below.
 */

import express, {
  type Express,
  type Request,
  type Response,
  type NextFunction,
} from 'express';
import { ZodError } from 'zod';

import { config } from '../config';
import { initDb, closeDb, prisma } from './database';
import { toBookResponse, bookCreateSchema, bookUpdateSchema } from './schemas';

// ---------------------------------------------------------------------------
// Logger
// ---------------------------------------------------------------------------
// MIGRATION_NOTE: Python used the stdlib `logging` module. A minimal typed
// wrapper around console keeps this file dependency-light; swap for pino/winston
// at the app level if structured logging is required.
const logger = {
  info: (msg: string): void => console.log(`[INFO] ${msg}`),
  warn: (msg: string): void => console.warn(`[WARN] ${msg}`),
  error: (msg: string): void => console.error(`[ERROR] ${msg}`),
};

// ---------------------------------------------------------------------------
// HttpError — mirrors FastAPI's HTTPException(status_code, detail)
// ---------------------------------------------------------------------------
export class HttpError extends Error {
  public readonly statusCode: number;
  public readonly detail: string;

  constructor(statusCode: number, detail: string) {
    super(detail);
    this.statusCode = statusCode;
    this.detail = detail;
    Object.setPrototypeOf(this, HttpError.prototype);
  }
}

/** Wrap async route handlers so thrown errors propagate to error middleware. */
const asyncHandler =
  (fn: (req: Request, res: Response, next: NextFunction) => Promise<unknown>) =>
  (req: Request, res: Response, next: NextFunction): void => {
    fn(req, res, next).catch(next);
  };

/** Detect Prisma unique-constraint violations (SQLAlchemy IntegrityError). */
function isUniqueViolation(err: unknown): boolean {
  return (
    typeof err === 'object' &&
    err !== null &&
    'code' in err &&
    (err as { code?: unknown }).code === 'P2002'
  );
}

// ---------------------------------------------------------------------------
// Books router
// ---------------------------------------------------------------------------
export function createBooksRouter(): express.Router {
  const router = express.Router();

  // GET /books/ -> list_books (pagination via skip/limit)
  router.get(
    '/',
    asyncHandler(async (req: Request, res: Response) => {
      try {
        const skip = Number.parseInt(String(req.query.skip ?? '0'), 10) || 0;
        let limit = Number.parseInt(String(req.query.limit ?? '100'), 10);
        if (Number.isNaN(limit)) limit = 100;
        // Enforce reasonable limits (source: limit = min(limit, 1000))
        limit = Math.min(limit, 1000);

        const books = await prisma.book.findMany({
          skip,
          take: limit,
          orderBy: { id: 'asc' },
        });

        logger.info(
          `Retrieved ${books.length} books (skip=${skip}, limit=${limit})`,
        );
        res.status(200).json(books.map(toBookResponse));
      } catch (err) {
        logger.error(`Error retrieving books: ${String(err)}`);
        throw new HttpError(500, 'Internal server error while retrieving books');
      }
    }),
  );

  // GET /books/:book_id -> get_book
  router.get(
    '/:book_id',
    asyncHandler(async (req: Request, res: Response) => {
      const bookId = Number.parseInt(req.params.book_id, 10);
      try {
        const book = Number.isNaN(bookId)
          ? null
          : await prisma.book.findUnique({ where: { id: bookId } });

        if (book === null) {
          logger.warn(`Book with ID ${req.params.book_id} not found`);
          throw new HttpError(404, `Book with ID ${bookId} not found`);
        }

        logger.info(`Retrieved book: ${book.title}`);
        res.status(200).json(toBookResponse(book));
      } catch (err) {
        if (err instanceof HttpError) throw err;
        logger.error(`Error retrieving book ${bookId}: ${String(err)}`);
        throw new HttpError(500, 'Internal server error while retrieving book');
      }
    }),
  );

  // POST /books/ -> create_book (returns 201)
  router.post(
    '/',
    asyncHandler(async (req: Request, res: Response) => {
      // MIGRATION_NOTE: Zod validation replaces Pydantic body binding. A
      // ZodError bubbles to the error middleware and becomes HTTP 422 to match
      // FastAPI's request-validation status code.
      const parsed = bookCreateSchema.parse(req.body);
      try {
        const dbBook = await prisma.book.create({ data: parsed });
        logger.info(
          `Created new book: ${dbBook.title} by ${dbBook.author}`,
        );
        // Source: status_code=status.HTTP_201_CREATED
        res.status(201).json(toBookResponse(dbBook));
      } catch (err) {
        if (isUniqueViolation(err)) {
          logger.error(`Integrity error creating book: ${String(err)}`);
          throw new HttpError(
            400,
            'Book with this title and author already exists',
          );
        }
        logger.error(`Error creating book: ${String(err)}`);
        throw new HttpError(500, 'Internal server error while creating book');
      }
    }),
  );

  // PUT /books/:book_id -> update_book (partial update via exclude_unset)
  router.put(
    '/:book_id',
    asyncHandler(async (req: Request, res: Response) => {
      const bookId = Number.parseInt(req.params.book_id, 10);
      // MIGRATION_NOTE: Pydantic's exclude_unset=True -> Zod's optional fields.
      // Only keys present in the request body are applied.
      const updateData = bookUpdateSchema.parse(req.body);

      try {
        const existing = Number.isNaN(bookId)
          ? null
          : await prisma.book.findUnique({ where: { id: bookId } });

        if (existing === null) {
          logger.warn(`Book with ID ${req.params.book_id} not found for update`);
          throw new HttpError(404, `Book with ID ${bookId} not found`);
        }

        // Strip undefined so only provided fields are updated.
        const data: Record<string, unknown> = {};
        for (const [key, value] of Object.entries(updateData)) {
          if (value !== undefined) data[key] = value;
        }

        const dbBook = await prisma.book.update({
          where: { id: bookId },
          data,
        });

        logger.info(`Updated book: ${dbBook.title}`);
        res.status(200).json(toBookResponse(dbBook));
      } catch (err) {
        if (err instanceof HttpError) throw err;
        if (isUniqueViolation(err)) {
          logger.error(`Integrity error updating book: ${String(err)}`);
          throw new HttpError(
            400,
            'Book with this title and author already exists',
          );
        }
        logger.error(`Error updating book ${bookId}: ${String(err)}`);
        throw new HttpError(500, 'Internal server error while updating book');
      }
    }),
  );

  // DELETE /books/:book_id -> delete_book (returns 204)
  router.delete(
    '/:book_id',
    asyncHandler(async (req: Request, res: Response) => {
      const bookId = Number.parseInt(req.params.book_id, 10);
      try {
        const existing = Number.isNaN(bookId)
          ? null
          : await prisma.book.findUnique({ where: { id: bookId } });

        if (existing === null) {
          logger.warn(
            `Book with ID ${req.params.book_id} not found for deletion`,
          );
          throw new HttpError(404, `Book with ID ${bookId} not found`);
        }

        await prisma.book.delete({ where: { id: bookId } });

        logger.info(`Deleted book: ${existing.title}`);
        // Source: status_code=status.HTTP_204_NO_CONTENT
        res.status(204).send();
      } catch (err) {
        if (err instanceof HttpError) throw err;
        logger.error(`Error deleting book ${bookId}: ${String(err)}`);
        throw new HttpError(500, 'Internal server error while deleting book');
      }
    }),
  );

  return router;
}

// ---------------------------------------------------------------------------
// App factory
// ---------------------------------------------------------------------------
export function createApp(): Express {
  const app = express();
  app.use(express.json());

  // GET / -> root
  app.get('/', (_req: Request, res: Response) => {
    res.status(200).json({
      message: 'Welcome to Book Catalog API',
      version: '1.0.0',
      docs_url: '/docs',
    });
  });

  // GET /health -> health_check
  app.get('/health', (_req: Request, res: Response) => {
    res.status(200).json({ status: 'healthy', service: 'book-catalog-api' });
  });

  // Book resource routes.
  app.use('/books', createBooksRouter());

  // -------------------------------------------------------------------------
  // Malformed JSON handler.
  // MIGRATION_NOTE: express.json() emits a SyntaxError with `type ===
  // 'entity.parse.failed'` on malformed request bodies. FastAPI returns 422 for
  // request-body validation failures, so we normalize malformed JSON to 422 to
  // match the source's request-validation status code.
  // -------------------------------------------------------------------------

  // Centralized error-handling middleware (replaces FastAPI exception_handler).
  app.use(
    (
      err: unknown,
      _req: Request,
      res: Response,
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      _next: NextFunction,
    ): void => {
      // Zod validation errors -> 422 (FastAPI request-validation status).
      if (err instanceof ZodError) {
        res.status(422).json({ detail: err.issues });
        return;
      }

      // Malformed JSON body -> 422.
      if (
        typeof err === 'object' &&
        err !== null &&
        'type' in err &&
        (err as { type?: unknown }).type === 'entity.parse.failed'
      ) {
        res.status(422).json({ detail: 'Invalid JSON body' });
        return;
      }

      // Explicit HttpError -> preserve status + `detail` wire shape.
      if (err instanceof HttpError) {
        res.status(err.statusCode).json({ detail: err.detail });
        return;
      }

      // Anything else -> 500.
      logger.error(`Unhandled error: ${String(err)}`);
      res.status(500).json({ detail: 'Internal server error' });
    },
  );

  return app;
}

// ---------------------------------------------------------------------------
// Server bootstrap
// ---------------------------------------------------------------------------
export async function start(): Promise<void> {
  // FastAPI startup event -> initialize DB before listening.
  await initDb();
  logger.info('Database initialized successfully');

  const app = createApp();
  const port = config.port;

  const server = app.listen(port, () => {
    logger.info(`Book Catalog API listening on port ${port}`);
  });

  const shutdown = async (): Promise<void> => {
    server.close();
    await closeDb();
    process.exit(0);
  };
  process.on('SIGINT', () => void shutdown());
  process.on('SIGTERM', () => void shutdown());
}

// Only auto-start when run directly (not when imported by tests).
if (require.main === module) {
  start().catch((err) => {
    logger.error(`Failed to start server: ${String(err)}`);
    process.exit(1);
  });
}
