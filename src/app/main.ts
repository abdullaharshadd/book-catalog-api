// app/main.ts
import express, { Express, Request, Response, NextFunction, Router } from 'express';
import { z } from 'zod';

import { config } from '../config';
import { logger } from '../logger';
import { PrismaClient, Prisma } from '@prisma/client';
import { initDb } from '../database';
import {
  bookCreateSchema,
  bookUpdateSchema,
  serializeBook,
  type BookResponse,
} from '../schemas';

/**
 * HttpError is a typed error carrying an HTTP status code and a detail message.
 *
 * MIGRATION: FastAPI's HTTPException carried a `detail` field surfaced via a
 * custom exception handler returning {"detail": ...}. We replicate that exact
 * response shape in the Express error middleware below.
 */
export class HttpError extends Error {
  public readonly status: number;
  public readonly detail: string;

  constructor(status: number, detail: string) {
    super(detail);
    this.status = status;
    this.detail = detail;
    this.name = 'HttpError';
  }
}

/**
 * createBooksRouter builds the /books resource router.
 *
 * Dependency injection: the PrismaClient is passed in via constructor-style
 * parameter rather than a global singleton, matching FastAPI's Depends pattern.
 *
 * MIGRATION: The source used two distinct SQLAlchemy session strategies — async
 * (get_db) for list_books and sync (get_sync_db) for the other handlers. Prisma
 * is uniformly async, so both strategies collapse into a single async client.
 * Behavior (queries, pagination, error mapping) is preserved exactly.
 */
export function createBooksRouter(prisma: PrismaClient): Router {
  const router = Router();

  // GET /books/ — list with pagination (was async endpoint).
  router.get('/', async (req: Request, res: Response, next: NextFunction) => {
    try {
      const skip = parseIntParam(req.query.skip, 0);
      let limit = parseIntParam(req.query.limit, 100);
      // Enforce reasonable limits (preserved: cap at 1000).
      limit = Math.min(limit, 1000);

      const books = await prisma.book.findMany({ skip, take: limit });

      logger.info(
        `Retrieved ${books.length} books (skip=${skip}, limit=${limit})`,
      );
      const payload: BookResponse[] = books.map(serializeBook);
      res.json(payload);
    } catch (err) {
      logger.error(`Error retrieving books: ${stringifyError(err)}`);
      next(
        new HttpError(500, 'Internal server error while retrieving books'),
      );
    }
  });

  // GET /books/:bookId — fetch a single book.
  router.get(
    '/:bookId',
    async (req: Request, res: Response, next: NextFunction) => {
      const bookId = Number(req.params.bookId);
      try {
        const book = await prisma.book.findUnique({ where: { id: bookId } });

        if (book === null) {
          logger.warn(`Book with ID ${bookId} not found`);
          throw new HttpError(404, `Book with ID ${bookId} not found`);
        }

        logger.info(`Retrieved book: ${book.title}`);
        res.json(serializeBook(book));
      } catch (err) {
        if (err instanceof HttpError) {
          next(err);
          return;
        }
        logger.error(`Error retrieving book ${bookId}: ${stringifyError(err)}`);
        next(
          new HttpError(500, 'Internal server error while retrieving book'),
        );
      }
    },
  );

  // POST /books/ — create a new book (201 Created).
  router.post('/', async (req: Request, res: Response, next: NextFunction) => {
    const parsed = bookCreateSchema.safeParse(req.body);
    if (!parsed.success) {
      next(new HttpError(422, 'Validation error'));
      return;
    }

    try {
      const book = await prisma.book.create({ data: parsed.data });

      logger.info(`Created new book: ${book.title} by ${book.author}`);
      res.status(201).json(serializeBook(book));
    } catch (err) {
      if (isUniqueConstraintError(err)) {
        logger.error(`Integrity error creating book: ${stringifyError(err)}`);
        next(
          new HttpError(
            400,
            'Book with this title and author already exists',
          ),
        );
        return;
      }
      logger.error(`Error creating book: ${stringifyError(err)}`);
      next(new HttpError(500, 'Internal server error while creating book'));
    }
  });

  // PUT /books/:bookId — partial update (only provided fields).
  router.put(
    '/:bookId',
    async (req: Request, res: Response, next: NextFunction) => {
      const bookId = Number(req.params.bookId);
      const parsed = bookUpdateSchema.safeParse(req.body);
      if (!parsed.success) {
        next(new HttpError(422, 'Validation error'));
        return;
      }

      try {
        const existing = await prisma.book.findUnique({
          where: { id: bookId },
        });

        if (existing === null) {
          logger.warn(`Book with ID ${bookId} not found for update`);
          throw new HttpError(404, `Book with ID ${bookId} not found`);
        }

        // Update only provided fields (exclude_unset semantics): Zod strips
        // undefined keys so only explicitly-set fields are applied.
        const updateData = stripUndefined(parsed.data);

        const updated = await prisma.book.update({
          where: { id: bookId },
          data: updateData,
        });

        logger.info(`Updated book: ${updated.title}`);
        res.json(serializeBook(updated));
      } catch (err) {
        if (err instanceof HttpError) {
          next(err);
          return;
        }
        if (isUniqueConstraintError(err)) {
          logger.error(
            `Integrity error updating book: ${stringifyError(err)}`,
          );
          next(
            new HttpError(
              400,
              'Book with this title and author already exists',
            ),
          );
          return;
        }
        logger.error(`Error updating book ${bookId}: ${stringifyError(err)}`);
        next(
          new HttpError(500, 'Internal server error while updating book'),
        );
      }
    },
  );

  // DELETE /books/:bookId — delete (204 No Content).
  router.delete(
    '/:bookId',
    async (req: Request, res: Response, next: NextFunction) => {
      const bookId = Number(req.params.bookId);
      try {
        const existing = await prisma.book.findUnique({
          where: { id: bookId },
        });

        if (existing === null) {
          logger.warn(`Book with ID ${bookId} not found for deletion`);
          throw new HttpError(404, `Book with ID ${bookId} not found`);
        }

        await prisma.book.delete({ where: { id: bookId } });

        logger.info(`Deleted book: ${existing.title}`);
        res.status(204).send();
      } catch (err) {
        if (err instanceof HttpError) {
          next(err);
          return;
        }
        logger.error(`Error deleting book ${bookId}: ${stringifyError(err)}`);
        next(
          new HttpError(500, 'Internal server error while deleting book'),
        );
      }
    },
  );

  return router;
}

/**
 * createApp wires up the Express application: root info, health check, the
 * books router and the centralized error handler.
 */
export function createApp(prisma: PrismaClient): Express {
  const app = express();
  app.use(express.json());

  // Root endpoint with API information.
  app.get('/', (_req: Request, res: Response) => {
    res.json({
      message: 'Welcome to Book Catalog API',
      version: '1.0.0',
      docs_url: '/docs',
    });
  });

  // Health check endpoint.
  app.get('/health', (_req: Request, res: Response) => {
    res.json({ status: 'healthy', service: 'book-catalog-api' });
  });

  app.use('/books', createBooksRouter(prisma));

  // Centralized error handler replicating FastAPI's {"detail": ...} shape.
  app.use(
    (err: unknown, _req: Request, res: Response, _next: NextFunction): void => {
      if (err instanceof HttpError) {
        res.status(err.status).json({ detail: err.detail });
        return;
      }
      logger.error(`Unhandled error: ${stringifyError(err)}`);
      res.status(500).json({ detail: 'Internal server error' });
    },
  );

  return app;
}

/**
 * start initializes the database (replacing FastAPI's startup event) and
 * begins listening on the configured port.
 *
 * MIGRATION: @app.on_event("startup") with init_db() ran schema bootstrap at
 * runtime. With Prisma this is normally handled by `prisma migrate`; initDb is
 * preserved for parity but verify production does not rely on runtime schema
 * creation.
 */
export async function start(): Promise<void> {
  const prisma = new PrismaClient();
  await initDb(prisma);
  logger.info('Database initialized successfully');

  const app = createApp(prisma);
  app.listen(config.port, () => {
    logger.info(`Book Catalog API listening on port ${config.port}`);
  });
}

// ---- helpers ----

function parseIntParam(value: unknown, fallback: number): number {
  if (typeof value !== 'string') return fallback;
  const n = Number.parseInt(value, 10);
  return Number.isNaN(n) ? fallback : n;
}

function stripUndefined<T extends Record<string, unknown>>(obj: T): Partial<T> {
  const out: Partial<T> = {};
  for (const [k, v] of Object.entries(obj)) {
    if (v !== undefined) {
      (out as Record<string, unknown>)[k] = v;
    }
  }
  return out;
}

function isUniqueConstraintError(err: unknown): boolean {
  return (
    err instanceof Prisma.PrismaClientKnownRequestError && err.code === 'P2002'
  );
}

function stringifyError(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

if (require.main === module) {
  start().catch((err) => {
    logger.error(`Fatal startup error: ${stringifyError(err)}`);
    process.exit(1);
  });
}
