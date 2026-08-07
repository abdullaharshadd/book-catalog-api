import 'dotenv/config';
import { execSync } from 'child_process';

try {
  execSync('npx prisma generate', { stdio: 'inherit' });
} catch (err) {
  console.error('Failed to run prisma generate:', err);
}

// Import app modules AFTER prisma generate has run
const { createApp, start } = require('./app/main');

start().catch((err: unknown) => {
  console.error(`Failed to start server: ${String(err)}`);
  process.exit(1);
});

export default createApp();