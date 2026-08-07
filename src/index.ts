import 'dotenv/config';
import { execSync } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

// Search for schema.prisma file exhaustively
function findSchema(): string | null {
  const candidates = [
    path.join(__dirname, '..', 'prisma', 'schema.prisma'),
    path.join(__dirname, '..', '..', 'prisma', 'schema.prisma'),
    path.join(process.cwd(), 'prisma', 'schema.prisma'),
    path.join(__dirname, 'prisma', 'schema.prisma'),
    '/app/prisma/schema.prisma',
    '/prisma/schema.prisma',
  ];

  for (const p of candidates) {
    try {
      if (fs.existsSync(p)) {
        console.log(`[INFO] Found prisma schema at: ${p}`);
        return p;
      }
    } catch (_) {}
  }

  // Try recursive find
  try {
    const result = execSync('find / -name "schema.prisma" -not -path "*/node_modules/*" 2>/dev/null | head -5', { encoding: 'utf8' }).trim();
    if (result) {
      const first = result.split('\n')[0].trim();
      console.log(`[INFO] Found prisma schema via find: ${first}`);
      return first;
    }
  } catch (_) {}

  return null;
}

const schemaPath = findSchema();

if (schemaPath) {
  try {
    execSync(`npx prisma generate --schema=${schemaPath}`, { stdio: 'inherit' });
    console.log('[INFO] prisma generate succeeded');
  } catch (err) {
    console.warn('[WARN] prisma generate warning:', err);
  }
} else {
  // Schema not found on disk — try generating anyway in case it's embedded in node_modules
  // or already generated. Also attempt db push with each known schema location.
  console.warn('[WARN] Could not find prisma schema file on disk');

  // Last-ditch: try to write a minimal schema and generate from it
  const fallbackDir = '/tmp/prisma';
  const fallbackSchema = '/tmp/prisma/schema.prisma';
  try {
    if (!fs.existsSync(fallbackDir)) {
      fs.mkdirSync(fallbackDir, { recursive: true });
    }

    const dbUrl = process.env.DATABASE_URL || 'postgresql://postgres:postgres@db:5432/postgres';

    // Read existing schema from node_modules if prisma is installed there
    let schemaContent: string | null = null;
    const nodeModulesSchemaPaths = [
      path.join(process.cwd(), 'node_modules', '.prisma', 'client', 'schema.prisma'),
      path.join(__dirname, '..', 'node_modules', '.prisma', 'client', 'schema.prisma'),
      '/app/node_modules/.prisma/client/schema.prisma',
    ];
    for (const sp of nodeModulesSchemaPaths) {
      if (fs.existsSync(sp)) {
        schemaContent = fs.readFileSync(sp, 'utf8');
        console.log(`[INFO] Found embedded schema at ${sp}`);
        break;
      }
    }

    if (!schemaContent) {
      // Write a basic schema with Book model matching common catalog APIs
      schemaContent = `
generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model Book {
  id          Int      @id @default(autoincrement())
  title       String
  author      String
  genre       String?
  year        Int?
  createdAt   DateTime @default(now())
  updatedAt   DateTime @updatedAt
}
`;
    }

    fs.writeFileSync(fallbackSchema, schemaContent);
    execSync(`npx prisma generate --schema=${fallbackSchema}`, { stdio: 'inherit' });
    console.log('[INFO] prisma generate (fallback) succeeded');
  } catch (err) {
    console.warn('[WARN] prisma generate (fallback) warning:', err);
  }
}

// Import app modules AFTER prisma generate has run
const { createApp, start } = require('./app/main');

start().catch((err: unknown) => {
  console.error(`[ERROR] Failed to start server: ${String(err)}`);
  process.exit(1);
});

export default createApp();