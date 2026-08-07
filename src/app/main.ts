import express, {
  type Express,
  type Request,
  type Response,
  type NextFunction,
} from 'express';
import { ZodError } from 'zod';
import { execSync } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

import { config } from '../config';
import { toBookResponse, bookCreateSchema, bookUpdateSchema } from './schemas';

const logger = {
  info: (msg: string): void => console.log(`[INFO] ${msg}`),
  warn: (msg: string): void => console.warn(`[WARN] ${msg}`),
  error: (msg: string): void => console.error(`[ERROR] ${msg}`),
};

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

const asyncHandler =
  (fn: (req: Request, res: Response, next: NextFunction) => Promise<unknown>) =>
  (req: Request, res: Response, next: NextFunction): void => {
    fn(req, res, next).catch(next);
  };

function isUniqueViolation(err: unknown): boolean {
  return (
    typeof err === 'object' &&
    err !== null &&
    'code' in err &&
    (err as { code?: unknown }).code === 'P2002'
  );
}

function tryRunCommand(cmd: string, cwd: string): boolean {
  try {
    execSync(cmd, { stdio: 'inherit', cwd });
    return true;
  } catch {
    return false;
  }
}

function runPrismaGenerate(): void {
  const cwdCandidates = [
    '/app',
    process.cwd(),
    path.join(__dirname, '..', '..'),
    path.join(__dirname, '..'),
    __dirname,
  ];

  const schemaCandidates = [
    'prisma/schema.prisma',
    'schema.prisma',
    '../prisma/schema.prisma',
  ];

  // First try: find schema file and use --schema flag
  for (const cwd of cwdCandidates) {
    for (const rel of schemaCandidates) {
      const abs = path.resolve(cwd, rel);
      try {
        if (fs.existsSync(abs)) {
          logger.info(`Found schema at ${abs}, running prisma generate...`);
          if (tryRunCommand(`npx prisma generate --schema="${abs}"`, cwd)) {
            logger.info('prisma generate succeeded');
            return;
          }
        }
      } catch {
        // continue
      }
    }
  }

  // Second try: run from each candidate cwd without explicit schema
  for (const cwd of cwdCandidates) {
    logger.info(`Trying prisma generate from cwd=${cwd}...`);
    if (tryRunCommand('npx prisma generate', cwd)) {
      logger.info('prisma generate succeeded');
      return;
    }
  }

  logger.warn('All prisma generate attempts failed; will try to use existing client');
}

function runPrismaDbPush(): void {
  const cwdCandidates = [
    '/app',
    process.cwd(),
    path.join(__dirname, '..', '..'),
    path.join(__dirname, '..'),
    __dirname,
  ];

  const schemaCandidates = [
    'prisma/schema.prisma',
    'schema.prisma',
    '../prisma/schema.prisma',
  ];

  for (const cwd of cwdCandidates) {
    for (const rel of schemaCandidates) {
      const abs = path.resolve(cwd, rel);
      try {
        if (fs.existsSync(abs)) {
          logger.info(`Found schema at ${abs}, running prisma db push...`);
          if (tryRunCommand(`npx prisma db push --accept-data-loss --schema="${abs}"`, cwd)) {
            logger.info('prisma db push succeeded');
            return;
          }
        }
      } catch {
        // continue
      }
    }
  }

  for (const cwd of cwdCandidates) {
    logger.info(`Trying prisma db push from cwd=${cwd}...`);
    if (tryRunCommand('npx prisma db push --accept-data-loss', cwd)) {
      logger.info('prisma db push succeeded');
      return;
    }
  }

  logger.warn('All prisma db push attempts failed');
}

function clearPrismaRequireCache(): void {
  const keysToDelete = Object.keys(require.cache).filter(
    (k) =>
      k.includes('@prisma') ||
      k.includes('.prisma') ||
      k.includes('prisma/client') ||
      k.includes('database'),
  );
  for (const key of keysToDelete) {
    delete require.cache[key];
  }
}

export function createBooksRouter(prisma: import('@prisma/client').PrismaClient): express.Router {
  const router = express.Router();

  router.get(
    '/',
    asyncHandler(async (req: Request, res: Response) => {
      try {
        const skip = Number.parseInt(String(req.query.skip ?? '0'), 10) || 0;
        let limit = Number.parseInt(String(req.query.limit ?? '100'), 10);
        if (Number.isNaN(limit)) limit = 100;
        limit = Math.min(limit, 1000);

        const books = await prisma.book.findMany({
          skip,
          take: limit,
          orderBy: { id: 'asc' },
        });

        logger.info(`Retrieved ${books.length} books (skip=${skip}, limit=${limit})`);
        res.status(200).json(books.map(toBookResponse));
      } catch (err) {
        logger.error(`Error retrieving books: ${String(err)}`);
        throw new HttpError(500, 'Internal server error while retrieving books');
      }
    }),
  );

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

  router.post(
    '/',
    asyncHandler(async (req: Request, res: Response) => {
      const parsed = bookCreateSchema.parse(req.body);
      try {
        const dbBook = await prisma.book.create({ data: parsed });
        logger.info(`Created new book: ${dbBook.title} by ${dbBook.author}`);
        res.status(201).json(toBookResponse(dbBook));
      } catch (err) {
        if (isUniqueViolation(err)) {
          logger.error(`Integrity error creating book: ${String(err)}`);
          throw new HttpError(400, 'Book with this title and author already exists');
        }
        logger.error(`Error creating book: ${String(err)}`);
        throw new HttpError(500, 'Internal server error while creating book');
      }
    }),
  );

  router.put(
    '/:book_id',
    asyncHandler(async (req: Request, res: Response) => {
      const bookId = Number.parseInt(req.params.book_id, 10);
      const updateData = bookUpdateSchema.parse(req.body);

      try {
        const existing = Number.isNaN(bookId)
          ? null
          : await prisma.book.findUnique({ where: { id: bookId } });

        if (existing === null) {
          logger.warn(`Book with ID ${req.params.book_id} not found for update`);
          throw new HttpError(404, `Book with ID ${bookId} not found`);
        }

        const data: Record<string, unknown> = {};
        for (const [key, value] of Object.entries(updateData)) {
          if (value !== undefined) data[key] = value;
        }

        const dbBook = await prisma.book.update({ where: { id: bookId }, data });

        logger.info(`Updated book: ${dbBook.title}`);
        res.status(200).json(toBookResponse(dbBook));
      } catch (err) {
        if (err instanceof HttpError) throw err;
        if (isUniqueViolation(err)) {
          logger.error(`Integrity error updating book: ${String(err)}`);
          throw new HttpError(400, 'Book with this title and author already exists');
        }
        logger.error(`Error updating book ${bookId}: ${String(err)}`);
        throw new HttpError(500, 'Internal server error while updating book');
      }
    }),
  );

  router.delete(
    '/:book_id',
    asyncHandler(async (req: Request, res: Response) => {
      const bookId = Number.parseInt(req.params.book_id, 10);
      try {
        const existing = Number.isNaN(bookId)
          ? null
          : await prisma.book.findUnique({ where: { id: bookId } });

        if (existing === null) {
          logger.warn(`Book with ID ${req.params.book_id} not found for deletion`);
          throw new HttpError(404, `Book with ID ${bookId} not found`);
        }

        await prisma.book.delete({ where: { id: bookId } });

        logger.info(`Deleted book: ${existing.title}`);
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

export function createApp(prisma: import('@prisma/client').PrismaClient): Express {
  const app = express();
  app.use(express.json());

  app.get('/', (_req: Request, res: Response) => {
    res.status(200).json({
      message: 'Welcome to Book Catalog API',
      version: '1.0.0',
      docs_url: '/docs',
    });
  });

  app.get('/health', (_req: Request, res: Response) => {
    res.status(200).json({ status: 'healthy', service: 'book-catalog-api' });
  });

  app.use('/books', createBooksRouter(prisma));

  app.use(
    (err: unknown, _req: Request, res: Response, _next: NextFunction): void => {
      if (err instanceof ZodError) {
        res.status(422).json({ detail: err.issues });
        return;
      }

      if (
        typeof err === 'object' &&
        err !== null &&
        'type' in err &&
        (err as { type?: unknown }).type === 'entity.parse.failed'
      ) {
        res.status(422).json({ detail: 'Invalid JSON body' });
        return;
      }

      if (err instanceof HttpError) {
        res.status(err.statusCode).json({ detail: err.detail });
        return;
      }

      logger.error(`Unhandled error: ${String(err)}`);
      res.status(500).json({ detail: 'Internal server error' });
    },
  );

  return app;
}

export async function start(): Promise<void> {
  // Step 1: run prisma generate before loading any prisma-dependent modules
  runPrismaGenerate();

  // Step 2: clear require cache so fresh client is picked up
  clearPrismaRequireCache();

  // Step 3: load database module (lazy, after generate)
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const db = require('./database') as typeof import('./database');

  // Step 4: push schema to DB
  runPrismaDbPush();

  // Step 5: initialize DB (connect, run any seed logic)
  await db.initDb();
  logger.info('Database initialized successfully');

  const app = createApp(db.prisma);
  const port = config.port;

  const server = app.listen(port, () => {
    logger.info(`Book Catalog API listening on port ${port}`);
  });

  const shutdown = async (): Promise<void> => {
    server.close();
    await db.closeDb();
    process.exit(0);
  };
  process.on('SIGINT', () => void shutdown());
  process.on('SIGTERM', () => void shutdown());
}

if (require.main === module) {
  start().catch((err) => {
    logger.error(`Failed to start server: ${String(err)}`);
    process.exit(1);
  });
}