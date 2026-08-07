import 'dotenv/config';
import { execSync } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

const fallbackDir = '/tmp/prisma';
const fallbackSchema = '/tmp/prisma/schema.prisma';

const schemaContent = `
generator client {
  provider = "prisma-client-js"
  output   = "/app/node_modules/.prisma/client"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model Book {
  id        Int      @id @default(autoincrement())
  title     String
  author    String
  genre     String?
  year      Int?
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}
`;

// Ensure the fallback dir exists
try {
  if (!fs.existsSync(fallbackDir)) {
    fs.mkdirSync(fallbackDir, { recursive: true });
  }
} catch (_) {}

// Write the schema
try {
  fs.writeFileSync(fallbackSchema, schemaContent, 'utf8');
} catch (err) {
  console.warn('[WARN] Could not write fallback schema:', err);
}

// Also try to write to /app/prisma/schema.prisma
try {
  const appPrismaDir = '/app/prisma';
  if (!fs.existsSync(appPrismaDir)) {
    fs.mkdirSync(appPrismaDir, { recursive: true });
  }
  fs.writeFileSync(path.join(appPrismaDir, 'schema.prisma'), schemaContent, 'utf8');
} catch (_) {}

// Try prisma generate with the written schema
const schemaPathsToTry = [
  fallbackSchema,
  '/app/prisma/schema.prisma',
  path.join(__dirname, '..', 'prisma', 'schema.prisma'),
];

let generated = false;
for (const sp of schemaPathsToTry) {
  if (!fs.existsSync(sp)) continue;
  try {
    execSync(`npx prisma generate --schema=${sp}`, { stdio: 'inherit' });
    console.log(`[INFO] prisma generate succeeded with schema: ${sp}`);
    generated = true;
    break;
  } catch (err) {
    console.warn(`[WARN] prisma generate failed with schema ${sp}:`, err);
  }
}

if (!generated) {
  // Last resort: try without --schema flag but from a directory that has schema.prisma
  try {
    execSync(`cd /tmp/prisma && npx prisma generate`, { stdio: 'inherit' });
    console.log('[INFO] prisma generate (cwd) succeeded');
    generated = true;
  } catch (err) {
    console.warn('[WARN] All prisma generate attempts failed; will try to use existing client');
  }
}

// Import app modules AFTER prisma generate has run
const { createApp, start } = require('./app/main');

start().catch((err: unknown) => {
  console.error(`[ERROR] Failed to start server: ${String(err)}`);
  process.exit(1);
});

export default createApp();