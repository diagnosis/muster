import { defineConfig } from '@playwright/test';
import * as dotenv from 'dotenv';

dotenv.config({ path: '.env' });

export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  fullyParallel: false,
  workers: 1, // capacity/state-machine specs share DB state — keep it serial
  retries: process.env.CI ? 1 : 0,

  reporter: [
    ['list'],
    ['html', { outputFolder: 'playwright-report', open: 'never' }],
  ],

  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:8088',
  },

  // Uncomment once cmd/muster/main.go exposes a cheap health route and you're
  // happy letting Playwright spin the server up per run:
  // webServer: {
  //   command: 'cd ../muster && go run cmd/muster/main.go',
  //   url: 'http://localhost:8088/health',
  //   reuseExistingServer: !process.env.CI,
  //   timeout: 120_000,
  // },
});
