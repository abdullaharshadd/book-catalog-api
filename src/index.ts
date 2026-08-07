import 'dotenv/config';
import { createApp, start } from './app/main';

start().catch((err) => {
  console.error(`Failed to start server: ${String(err)}`);
  process.exit(1);
});

export default createApp();
