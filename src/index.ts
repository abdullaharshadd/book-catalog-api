import 'dotenv/config';
import { execSync } from 'child_process';
import * as path from 'path';

const schemaPath = path.join(__dirname, '..', 'prisma', 'schema.prisma');

try {
  execSync(`npx prisma generate --schema=${schemaPath}`, { stdio: 'inherit' });
} catch (err) {
  console.warn('[WARN] prisma generate warning:', err);
}

// Import app modules AFTER prisma generate has run
const { createApp, start } = require('./app/main');

start().catch((err: unknown) => {
  console.error(`Failed to start server: ${String(err)}`);
  process.exit(1);
});

export default createApp();