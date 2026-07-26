import { defineConfig } from '@playwright/test';
import * as dotenv from 'dotenv';

dotenv.config({ path: '.env' });

export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  fullyParallel: true,
  workers: 8, // safe: per-test actors + unique data, no shared state (TEST-DESIGN.md)
  retries: 0,

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
