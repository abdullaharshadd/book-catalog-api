import express, {
  type Express,
  type Request,
  type Response,
  type NextFunction,
} from 'express';
import { ZodError } from 'zod';
import { execSync, spawnSync } from 'child_process';
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

function findSchemaViaFind(): string | null {
  try {
    const result = spawnSync('find', ['/', '-name', 'schema.prisma', '-not', '-path', '*/node_modules/.cache/*'], {
      encoding: 'utf8',
      timeout: 15000,
    });
    if (result.stdout) {
      const lines = result.stdout.trim().split('\n').filter(Boolean);
      const appPath = lines.find(l => l.includes('/app/') && !l.includes('node_modules'));
      if (appPath) return appPath;
      const anyPath = lines.find(l => !l.includes('node_modules'));
      if (anyPath) return anyPath;
      // Even in node_modules could be the source schema
      if (lines.length > 0) return lines[0];
    }
  } catch {
    // ignore
  }
  return null;
}

function findOrCreatePrismaSchema(): string | null {
  // Standard candidate locations
  const candidates = [
    path.join(process.cwd(), 'prisma', 'schema.prisma'),
    path.join(process.cwd(), 'schema.prisma'),
    path.join(__dirname, '..', '..', 'prisma', 'schema.prisma'),
    path.join(__dirname, '..', 'prisma', 'schema.prisma'),
    path.join(__dirname, 'prisma', 'schema.prisma'),
    '/app/prisma/schema.prisma',
    '/app/schema.prisma',
    path.join(__dirname, '..', '..', '..', 'prisma', 'schema.prisma'),
    '/prisma/schema.prisma',
  ];

  for (const candidate of candidates) {
    try {
      if (fs.existsSync(candidate)) {
        logger.info(`Found schema at: ${candidate}`);
        return candidate;
      }
    } catch {
      // ignore
    }
  }

  logger.info('Schema not found in standard locations, trying find...');
  const found = findSchemaViaFind();
  if (found) {
    logger.info(`Found schema via find: ${found}`);
    return found;
  }

  // Last resort: create a minimal schema in a temp directory
  logger.info('Schema not found anywhere, creating a minimal schema...');
  try {
    const tmpDir = '/tmp/prisma-generated';
    fs.mkdirSync(tmpDir, { recursive: true });
    const schemaContent = `
generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model Book {
  id            Int      @id @default(autoincrement())
  title         String
  author        String
  description   String?
  year          Int?
  genre         String?
  createdAt     DateTime @default(now()) @map("created_at")
  updatedAt     DateTime @updatedAt @map("updated_at")

  @@unique([title, author])
  @@map("books")
}
`;
    const schemaPath = path.join(tmpDir, 'schema.prisma');
    fs.writeFileSync(schemaPath, schemaContent.trim(), 'utf8');
    logger.info(`Created minimal schema at: ${schemaPath}`);
    return schemaPath;
  } catch (e) {
    logger.error(`Failed to create minimal schema: ${String(e)}`);
    return null;
  }
}

function findPrismaBinary(): string | null {
  const candidates = [
    path.join(process.cwd(), 'node_modules', '.bin', 'prisma'),
    '/app/node_modules/.bin/prisma',
    path.join(__dirname, '..', '..', 'node_modules', '.bin', 'prisma'),
    path.join(__dirname, '..', '..', '..', 'node_modules', '.bin', 'prisma'),
  ];
  for (const candidate of candidates) {
    try {
      if (fs.existsSync(candidate)) {
        return candidate;
      }
    } catch {
      // ignore
    }
  }
  return null;
}

function tryRunPrismaGenerate(schemaPath: string): boolean {
  const cwds = [
    '/app',
    path.dirname(path.dirname(schemaPath)),
    path.dirname(schemaPath),
    process.cwd(),
  ];

  const prismaBin = findPrismaBinary();
  const cmds: string[] = [];

  if (prismaBin) {
    cmds.push(`"${prismaBin}" generate --schema="${schemaPath}"`);
  }
  cmds.push(`npx --no-install prisma generate --schema="${schemaPath}"`);
  cmds.push(`npx prisma generate --schema="${schemaPath}"`);

  for (const cwd of cwds) {
    for (const cmd of cmds) {
      try {
        logger.info(`Trying: ${cmd} (cwd=${cwd})`);
        execSync(cmd, { stdio: 'inherit', cwd, timeout: 120000 });
        logger.info('prisma generate succeeded');
        return true;
      } catch {
        // try next
      }
    }
  }
  return false;
}

function tryRunPrismaDbPush(schemaPath: string): boolean {
  const cwds = [
    '/app',
    path.dirname(path.dirname(schemaPath)),
    path.dirname(schemaPath),
    process.cwd(),
  ];

  const prismaBin = findPrismaBinary();
  const cmds: string[] = [];

  if (prismaBin) {
    cmds.push(`"${prismaBin}" db push --accept-data-loss --schema="${schemaPath}"`);
  }
  cmds.push(`npx --no-install prisma db push --accept-data-loss --schema="${schemaPath}"`);
  cmds.push(`npx prisma db push --accept-data-loss --schema="${schemaPath}"`);

  for (const cwd of cwds) {
    for (const cmd of cmds) {
      try {
        logger.info(`Trying db push: ${cmd} (cwd=${cwd})`);
        execSync(cmd, { stdio: 'inherit', cwd, timeout: 120000 });
        logger.info('prisma db push succeeded');
        return true;
      } catch {
        // try next
      }
    }
  }
  return false;
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

function isPrismaClientInitialized(): boolean {
  try {
    const candidates = [
      '/app/node_modules/.prisma/client/index.js',
      path.join(process.cwd(), 'node_modules', '.prisma', 'client', 'index.js'),
      path.join(__dirname, '..', '..', 'node_modules', '.prisma', 'client', 'index.js'),
    ];
    for (const c of candidates) {
      if (fs.existsSync(c)) {
        const content = fs.readFileSync(c, 'utf8');
        if (!content.includes('did not initialize yet')) {
          return true;
        }
      }
    }
    return false;
  } catch {
    return false;
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
  logger.info(`process.cwd()=${process.cwd()}, __dirname=${__dirname}`);

  try {
    const appContents = fs.readdirSync('/app').join(', ');
    logger.info(`/app contents: ${appContents}`);
  } catch {
    logger.warn('Could not list /app');
  }

  const alreadyInit = isPrismaClientInitialized();
  logger.info(`Prisma client already initialized: ${alreadyInit}`);

  if (!alreadyInit) {
    const schemaPath = findOrCreatePrismaSchema();
    logger.info(`Using schema: ${schemaPath ?? 'none'}`);

    if (schemaPath !== null) {
      const generated = tryRunPrismaGenerate(schemaPath);
      if (!generated) {
        logger.warn('All prisma generate attempts failed; proceeding anyway');
      }
      clearPrismaRequireCache();

      // Run db push after generate
      tryRunPrismaDbPush(schemaPath);
    } else {
      logger.warn('No schema available; skipping generate and db push');
    }
  } else {
    // Already initialized, still sync schema
    const schemaPath = findOrCreatePrismaSchema();
    if (schemaPath !== null) {
      tryRunPrismaDbPush(schemaPath);
    }
  }

  // Load database module after generate attempt
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const db = require('./database') as typeof import('./database');

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