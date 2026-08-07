import 'dotenv/config';
import { execSync } from 'child_process';
import { createApp, start } from './app/main';

try {
  execSync('npx prisma generate', { stdio: 'inherit' });
} catch (err) {
  console.error('Failed to run prisma generate:', err);
}

start().catch((err) => {
  console.error(`Failed to start server: ${String(err)}`);
  process.exit(1);
});

export default createApp();