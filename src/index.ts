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
  console.warn('[WARN] Could not find prisma schema, skipping prisma generate');
}

// Import app modules AFTER prisma generate has run
const { createApp, start } = require('./app/main');

start().catch((err: unknown) => {
  console.error(`Failed to start server: ${String(err)}`);
  process.exit(1);
});

export default createApp();