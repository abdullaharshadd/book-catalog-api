import 'dotenv/config';
import { execSync } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

// Try multiple possible schema locations
const possibleSchemaPaths = [
  path.join(__dirname, '..', 'prisma', 'schema.prisma'),
  path.join(process.cwd(), 'prisma', 'schema.prisma'),
  path.join(__dirname, 'prisma', 'schema.prisma'),
  '/app/prisma/schema.prisma',
];

let schemaPath: string | null = null;
for (const p of possibleSchemaPaths) {
  if (fs.existsSync(p)) {
    schemaPath = p;
    break;
  }
}

if (schemaPath) {
  try {
    execSync(`npx prisma generate --schema=${schemaPath}`, { stdio: 'inherit' });
  } catch (err) {
    console.warn('[WARN] prisma generate warning:', err);
  }
} else {
  // Try to find schema by searching recursively from /app
  try {
    const found = execSync('find /app -name "schema.prisma" 2>/dev/null | head -5', { encoding: 'utf8' }).trim();
    console.log('[INFO] Prisma schema search result:', found || 'not found');
    if (found) {
      const firstFound = found.split('\n')[0].trim();
      schemaPath = firstFound;
      execSync(`npx prisma generate --schema=${firstFound}`, { stdio: 'inherit' });
    } else {
      console.warn('[WARN] Could not find prisma schema file; skipping generate/push');
    }
  } catch (err) {
    console.warn('[WARN] prisma generate warning:', err);
  }
}

// Import app modules AFTER prisma generate has run
const { createApp, start } = require('./app/main');

start().catch((err: unknown) => {
  console.error(`[ERROR] Failed to start server: ${String(err)}`);
  process.exit(1);
});

export default createApp();