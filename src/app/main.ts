/**
 * Book Catalog API — Express application entrypoint.
 *
 * MIGRATION: The original `app/main.py` was a FastAPI application. In the
 * Node.js/TypeScript stack we use **Express** + **Prisma** + **Zod**. Key
 * mapping decisions:
 *
 *   - FastAPI path operation decorators (@app.get/post/...) -> Express Router
 *     handlers grouped by resource (see `createBookRouter`).
 *   - FastAPI `Depends(get_db)` dependency injection -> a shared `PrismaClient`
 *     injected into the route factory (constructor-style DI).
 *   - Pydantic schemas (BookCreate/BookUpdate/BookResponse) -> Zod schemas
 *     (see `./schemas`), validated at the route level.
 *   - The custom HTTPException handler + per-route try/except blocks are
 *     replaced by a centralized Express error middleware (`errorHandler`).
 *   - The FastAPI `@app.on_event("startup")` -> init_db() hook is removed.
 *     Prisma manages connections lazily; schema creation is handled by
 *     `prisma migrate`. We still expose a `connect()` for explicit startup.
 *   - The mixed async/sync handlers in the source are all `async` here, since
 *     Prisma is uniformly async.
 *   - `limit = min(limit, 1000)` cap is preserved in the list handler.
 *   - IntegrityError on duplicate (title, author) -> Prisma P2002 unique
 *     constraint violation -> HTTP 400.
 */

import express, {
  type Application,
  type NextFunction,
  type Request,
  type Response,
  type Router,
} from 'express';
import { Prisma, PrismaClient } from '@prisma/client';
import { z } from 'zod';

import { logger } from '../config/logger';
import {
  bookCreateSchema,
  bookUpdateSchema,
  listBooksQuerySchema,
} from './schemas';

/**
 * ApiError is a typed error carrying an HTTP status code and a client-facing
 * message. Thrown from route handlers and translated to a JSON response by the
 * centralized error middleware.
 *
 * MIGRATION: Replaces FastAPI's `HTTPException(status_code, detail)`.
 */
export class ApiError extends Error {
  public readonly statusCode: number;
  public readonly details?: unknown;

  constructor(statusCode: number, message: string, details?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.statusCode = statusCode;
    this.details = details;
  }
}

/**
 * asyncHandler wraps an async Express handler so that rejected promises are
 * forwarded to the error middleware instead of crashing the process.
 */
const asyncHandler =
  (
    fn: (req: Request, res: Response, next: NextFunction) => Promise<unknown>,
  ) =>
  (req: Request, res: Response, next: NextFunction): void => {
    fn(req, res, next).catch(next);
  };

/**
 * createBookRouter builds the `/books` resource router. The PrismaClient is
 * injected (constructor-style DI) rather than accessed as a global singleton.
 *
 * @param prisma - the shared Prisma client used for all DB access.
 * @returns an Express Router mounted at `/books`.
 */
export function createBookRouter(prisma: PrismaClient): Router {
  const router = express.Router();

  /**
   * GET /books — retrieve all books with pagination.
   *
   * Query params: `skip` (default 0), `limit` (default 100, capped at 1000).
   */
  router.get(
    '/',
    asyncHandler(async (req: Request, res: Response) => {
      const parsed = listBooksQuerySchema.safeParse(req.query);
      if (!parsed.success) {
        throw new ApiError(422, 'Invalid query parameters', parsed.error.flatten());
      }

      const { skip } = parsed.data;
      // Enforce reasonable limits (matches Python `min(limit, 1000)`).
      const limit = Math.min(parsed.data.limit, 1000);

      const books = await prisma.book.findMany({
        skip,
        take: limit,
      });

      logger.info(
        `Retrieved ${books.length} books (skip=${skip}, limit=${limit})`,
      );
      res.status(200).json(books);
    }),
  );

  /**
   * GET /books/:bookId — retrieve a single book by its ID.
   */
  router.get(
    '/:bookId',
    asyncHandler(async (req: Request, res: Response) => {
      const bookId = parseBookId(req.params.bookId);

      const book = await prisma.book.findUnique({ where: { id: bookId } });
      if (book === null) {
        logger.warn(`Book with ID ${bookId} not found`);
        throw new ApiError(404, `Book with ID ${bookId} not found`);
      }

      logger.info(`Retrieved book: ${book.title}`);
      res.status(200).json(book);
    }),
  );

  /**
   * POST /books — create a new book.
   */
  router.post(
    '/',
    asyncHandler(async (req: Request, res: Response) => {
      const parsed = bookCreateSchema.safeParse(req.body);
      if (!parsed.success) {
        throw new ApiError(422, 'Validation failed', parsed.error.flatten());
      }

      try {
        const book = await prisma.book.create({ data: parsed.data });
        logger.info(`Created new book: ${book.title} by ${book.author}`);
        res.status(201).json(book);
      } catch (err) {
        // P2002 == unique constraint violation on (title, author).
        if (
          err instanceof Prisma.PrismaClientKnownRequestError &&
          err.code === 'P2002'
        ) {
          logger.error(`Integrity error creating book: ${err.message}`);
          throw new ApiError(
            400,
            'Book with this title and author already exists',
          );
        }
        throw err;
      }
    }),
  );

  /**
   * PUT /books/:bookId — update an existing book (partial update semantics).
   *
   * MIGRATION: FastAPI's `book_update.dict(exclude_unset=True)` partial update
   * is reproduced by stripping undefined fields from the validated payload, so
   * only provided fields are written.
   */
  router.put(
    '/:bookId',
    asyncHandler(async (req: Request, res: Response) => {
      const bookId = parseBookId(req.params.bookId);

      const parsed = bookUpdateSchema.safeParse(req.body);
      if (!parsed.success) {
        throw new ApiError(422, 'Validation failed', parsed.error.flatten());
      }

      // Only include fields that were actually provided (exclude_unset).
      const updateData: Prisma.BookUpdateInput = {};
      for (const [field, value] of Object.entries(parsed.data)) {
        if (value !== undefined) {
          (updateData as Record<string, unknown>)[field] = value;
        }
      }

      const existing = await prisma.book.findUnique({ where: { id: bookId } });
      if (existing === null) {
        logger.warn(`Book with ID ${bookId} not found for update`);
        throw new ApiError(404, `Book with ID ${bookId} not found`);
      }

      try {
        const book = await prisma.book.update({
          where: { id: bookId },
          data: updateData,
        });
        logger.info(`Updated book: ${book.title}`);
        res.status(200).json(book);
      } catch (err) {
        if (
          err instanceof Prisma.PrismaClientKnownRequestError &&
          err.code === 'P2002'
        ) {
          logger.error(`Integrity error updating book: ${err.message}`);
          throw new ApiError(
            400,
            'Book with this title and author already exists',
          );
        }
        throw err;
      }
    }),
  );

  /**
   * DELETE /books/:bookId — delete a book by its ID. Returns 204 No Content.
   */
  router.delete(
    '/:bookId',
    asyncHandler(async (req: Request, res: Response) => {
      const bookId = parseBookId(req.params.bookId);

      const existing = await prisma.book.findUnique({ where: { id: bookId } });
      if (existing === null) {
        logger.warn(`Book with ID ${bookId} not found for deletion`);
        throw new ApiError(404, `Book with ID ${bookId} not found`);
      }

      await prisma.book.delete({ where: { id: bookId } });
      logger.info(`Deleted book: ${existing.title}`);
      res.status(204).send();
    }),
  );

  return router;
}

/**
 * parseBookId validates and converts a path parameter into a positive integer
 * book ID, throwing a 422 ApiError on malformed input.
 */
function parseBookId(raw: string): number {
  const id = Number(raw);
  if (!Number.isInteger(id) || id <= 0) {
    throw new ApiError(422, `Invalid book ID: ${raw}`);
  }
  return id;
}

/**
 * errorHandler is the centralized Express error middleware. It translates
 * ApiError instances into their declared status codes and falls back to a 500
 * for unexpected errors, mirroring the FastAPI custom exception handler plus
 * the per-route 500 fallbacks.
 *
 * Response shape: `{ error: string, details?: unknown }`.
 */
export function errorHandler(
  err: unknown,
  _req: Request,
  res: Response,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _next: NextFunction,
): void {
  if (err instanceof ApiError) {
    const body: { error: string; details?: unknown } = { error: err.message };
    if (err.details !== undefined) {
      body.details = err.details;
    }
    res.status(err.statusCode).json(body);
    return;
  }

  const message = err instanceof Error ? err.message : String(err);
  logger.error(`Unhandled error: ${message}`);
  res.status(500).json({ error: 'Internal server error' });
}

/**
 * createApp wires together the Express application: JSON body parsing, resource
 * routers, root/health endpoints, and the centralized error handler.
 *
 * @param prisma - the shared Prisma client injected into resource routers.
 * @returns a configured Express Application ready to listen.
 */
export function createApp(prisma: PrismaClient): Application {
  const app = express();
  app.use(express.json());

  /**
   * GET / — root endpoint with API information.
   */
  app.get('/', (_req: Request, res: Response) => {
    res.status(200).json({
      message: 'Welcome to Book Catalog API',
      version: '1.0.0',
      docs_url: '/docs',
    });
  });

  /**
   * GET /health — health check endpoint.
   */
  app.get('/health', (_req: Request, res: Response) => {
    res.status(200).json({ status: 'healthy', service: 'book-catalog-api' });
  });

  app.use('/books', createBookRouter(prisma));

  app.use(errorHandler);

  return app;
}
